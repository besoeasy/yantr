import Docker from "dockerode";
import { readFile, writeFile, unlink, mkdir, stat, mkdtemp, rm } from "fs/promises";
import path from "path";
import { fileURLToPath } from "url";
import { dirname } from "path";
import { spawnProcess } from "./utils.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const docker = new Docker({ socketPath: process.env.DOCKER_SOCKET || "/var/run/docker.sock" });

// Backup job tracking
const backupJobs = new Map();
const restoreJobs = new Map();

// Generate unique job ID
function generateJobId() {
  return `job-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
}

// Get S3 config from environment or config file
export async function getS3Config() {
  const configPath = path.join(__dirname, "..", ".backup-config.json");

  try {
    const content = await readFile(configPath, "utf-8");
    return JSON.parse(content);
  } catch (err) {
    return null;
  }
}

// Save S3 config
export async function saveS3Config(config) {
  const configPath = path.join(__dirname, "..", ".backup-config.json");
  await writeFile(configPath, JSON.stringify(config, null, 2), "utf-8");
}

// Create rclone environment variables from S3 config
function createRcloneEnv(s3Config) {
  const env = {
    ...process.env,
    RCLONE_CONFIG_S3_TYPE: "s3",
    RCLONE_CONFIG_S3_PROVIDER: s3Config.provider || "AWS",
    RCLONE_CONFIG_S3_ACCESS_KEY_ID: s3Config.accessKey,
    RCLONE_CONFIG_S3_SECRET_ACCESS_KEY: s3Config.secretKey,
    RCLONE_CONFIG_S3_REGION: s3Config.region || "us-east-1",
    RCLONE_CONFIG_S3_FORCE_PATH_STYLE: "true",
  };

  if (s3Config.endpoint) {
    env.RCLONE_CONFIG_S3_ENDPOINT = s3Config.endpoint;
  }

  return env;
}

// List backups from S3
export async function listBackups(s3Config, log) {
  try {
    const rcloneEnv = createRcloneEnv(s3Config);

    // List app directories in yantra-backup/
    const { stdout, stderr, exitCode } = await spawnProcess(
      "rclone",
      ["lsjson", `s3:${s3Config.bucket}/yantra-backup/`],
      { env: rcloneEnv }
    );

    if (exitCode !== 0) {
      // If directory doesn't exist yet (no backups created), return empty array
      if (stderr.includes("directory not found") || stderr.includes("not found")) {
        log?.("info", "No backup directory found yet - returning empty list");
        return [];
      }
      throw new Error(`rclone failed: ${stderr}`);
    }

    const items = JSON.parse(stdout || "[]");
    log?.("info", `Found ${items.length} app backup folder(s)`);

    // Read metadata from each app folder
    const backups = [];
    for (const item of items) {
      if (!item.IsDir) continue;

      const appName = item.Name || item.Path?.replace(/\/$/, "") || "";
      if (!appName) continue;

      log?.("info", `Reading backup for app: ${appName}`);

      try {
        // First, list what's actually in this folder to debug
        const lsResult = await spawnProcess(
          "rclone",
          ["ls", `s3:${s3Config.bucket}/yantra-backup/${appName}/`],
          { env: rcloneEnv }
        );
        log?.("info", `Files in ${appName} folder:`, lsResult.stdout || "(empty)");

        const metaResult = await spawnProcess(
          "rclone",
          ["cat", `s3:${s3Config.bucket}/yantra-backup/${appName}/metadata.json`],
          { env: rcloneEnv }
        );

        if (metaResult.exitCode === 0) {
          const metadata = JSON.parse(metaResult.stdout);
          backups.push(metadata);
          log?.("info", `Successfully loaded backup for: ${appName}`);
        } else {
          log?.("error", `Failed to read metadata for ${appName} - exitCode: ${metaResult.exitCode}, stderr: ${metaResult.stderr}, stdout: ${metaResult.stdout}`);
        }
      } catch (err) {
        log?.("error", `Exception reading metadata for ${appName}:`, err.message);
      }
    }

    return backups.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp));
  } catch (err) {
    throw new Error(`Failed to list backups: ${err.message}`);
  }
}

// Create backup of volumes
export async function createBackup({ volumes, s3Config, name, log, appConfigs = [], apps = [] }) {
  const jobId = generateJobId();
  // Use first app name as backup folder, or fallback to "volumes" for manual volume backups
  const backupId = (apps?.[0]?.appId || appConfigs?.[0]?.appId || name || "volumes").replace(/[^a-z0-9-_]/gi, "-");
  const tmpDir = "/tmp";
  let jobTmpDir = null;
  const volumeList = Array.isArray(volumes) ? volumes : [];

  backupJobs.set(jobId, {
    status: "in-progress",
    progress: 0,
    backupId,
    volumes: volumeList,
    apps,
    startedAt: new Date().toISOString(),
  });

  // Run backup asynchronously
  (async () => {
    try {
      log?.("info", `[Backup ${jobId}] Starting backup of ${volumeList.length} volume(s)`);

      // Ensure alpine image exists for tar operations
      try {
        await docker.getImage("alpine:latest").inspect();
        log?.("info", `[Backup ${jobId}] Alpine image already available`);
      } catch (err) {
        log?.("info", `[Backup ${jobId}] Pulling alpine:latest image...`);
        await new Promise((resolve, reject) => {
          docker.pull("alpine:latest", (err, stream) => {
            if (err) return reject(err);
            docker.modem.followProgress(stream, (err, output) => {
              if (err) return reject(err);
              resolve(output);
            });
          });
        });
        log?.("info", `[Backup ${jobId}] Alpine image pulled successfully`);
      }

      jobTmpDir = await mkdtemp(path.join(tmpDir, "yantra-backup-"));
      const volumeBackups = [];
      const appConfigBackups = [];
      const totalVolumes = volumeList.length;

      for (let i = 0; i < volumeList.length; i++) {
        const volumeName = volumeList[i];
        log?.("info", `[Backup ${jobId}] Processing volume ${i + 1}/${totalVolumes}: ${volumeName}`);

        // Update progress
        backupJobs.set(jobId, {
          ...backupJobs.get(jobId),
          progress: Math.floor((i / totalVolumes) * 90),
          currentVolume: volumeName,
        });

        const tarFileName = `${volumeName}.tar.gz`;
        const tarPath = path.join(jobTmpDir, tarFileName);

        // Step 1: Create tar of volume using temporary container
        log?.("info", `[Backup ${jobId}] Creating tar archive of ${volumeName}`);

        const container = await docker.createContainer({
          Image: "alpine:latest",
          Cmd: ["tar", "czf", `/backup/${tarFileName}`, "-C", "/data", "."],
          HostConfig: {
            Binds: [
              `${volumeName}:/data:ro`,
              `${jobTmpDir}:/backup`,
            ],
            AutoRemove: true,
          },
        });

        await container.start();
        await container.wait();

        // Get tar file size
        const stats = await stat(tarPath);
        const sizeBytes = stats.size;

        log?.("info", `[Backup ${jobId}] Created tar: ${(sizeBytes / (1024 * 1024)).toFixed(2)} MB`);

        // Step 2: Upload to S3 using rclone
        log?.("info", `[Backup ${jobId}] Uploading ${volumeName} to S3`);

        const rcloneEnv = createRcloneEnv(s3Config);
        const { stdout, stderr, exitCode } = await spawnProcess(
          "rclone",
          [
            "copy",
            tarPath,
            `s3:${s3Config.bucket}/yantra-backup/${backupId}/`,
            "--progress",
          ],
          { env: rcloneEnv }
        );

        if (exitCode !== 0) {
          throw new Error(`rclone upload failed: ${stderr}`);
        }

        log?.("info", `[Backup ${jobId}] Uploaded ${volumeName} successfully`);

        volumeBackups.push({
          name: volumeName,
          size: sizeBytes,
          tarFileName,
        });

        // Cleanup tar file
        try {
          await unlink(tarPath);
        } catch (err) {
          log?.("warn", `[Backup ${jobId}] Failed to remove temp file ${tarFileName}:`, err.message);
        }
      }

      const uniqueAppConfigs = new Map();
      for (const appConfig of appConfigs) {
        if (!appConfig || !appConfig.appId || !appConfig.appPath) continue;
        if (!uniqueAppConfigs.has(appConfig.appId)) {
          uniqueAppConfigs.set(appConfig.appId, appConfig);
        }
      }

      const appConfigList = Array.from(uniqueAppConfigs.values());
      const totalAppConfigs = appConfigList.length;

      if (totalAppConfigs > 0) {
        log?.("info", `[Backup ${jobId}] Backing up ${totalAppConfigs} app config(s)`);
      }

      for (let i = 0; i < totalAppConfigs; i++) {
        const appConfig = appConfigList[i];
        const appId = appConfig.appId;
        const appPath = appConfig.appPath;

        log?.("info", `[Backup ${jobId}] Archiving app config ${i + 1}/${totalAppConfigs}: ${appId}`);

        const appTarFileName = `app-${appId}.tar.gz`;
        const appTarPath = path.join(jobTmpDir, appTarFileName);

        const tarResult = await spawnProcess(
          "tar",
          ["czf", appTarPath, "-C", appPath, "."]
        );

        if (tarResult.exitCode !== 0) {
          throw new Error(`Failed to archive app ${appId}: ${tarResult.stderr}`);
        }

        const appStats = await stat(appTarPath);

        const rcloneEnv = createRcloneEnv(s3Config);
        const uploadResult = await spawnProcess(
          "rclone",
          [
            "copy",
            appTarPath,
            `s3:${s3Config.bucket}/yantra-backup/${backupId}/app-configs/`,
          ],
          { env: rcloneEnv }
        );

        if (uploadResult.exitCode !== 0) {
          throw new Error(`Failed to upload app config ${appId}: ${uploadResult.stderr}`);
        }

        appConfigBackups.push({
          appId,
          tarFileName: appTarFileName,
          size: appStats.size,
        });

        try {
          await unlink(appTarPath);
        } catch (err) {
          log?.("warn", `[Backup ${jobId}] Failed to remove app config tar ${appTarFileName}:`, err.message);
        }
      }

      // Step 3: Upload metadata
      log?.("info", `[Backup ${jobId}] Creating metadata`);

      const metadata = {
        id: backupId,
        name: name || `Backup ${new Date().toLocaleString()}`,
        timestamp: new Date().toISOString(),
        volumes: volumeBackups,
        totalSize:
          volumeBackups.reduce((sum, v) => sum + v.size, 0) +
          appConfigBackups.reduce((sum, v) => sum + v.size, 0),
        appConfigs: appConfigBackups,
        apps,
      };

      const metadataPath = path.join(jobTmpDir, "metadata.json");
      await writeFile(metadataPath, JSON.stringify(metadata, null, 2), "utf-8");

      const rcloneEnv = createRcloneEnv(s3Config);
      const metaUpload = await spawnProcess(
        "rclone",
        [
          "copy",
          metadataPath,
          `s3:${s3Config.bucket}/yantra-backup/${backupId}/`,
        ],
        { env: rcloneEnv }
      );

      if (metaUpload.exitCode !== 0) {
        throw new Error(`Failed to upload metadata: ${metaUpload.stderr}`);
      }

      try {
        await unlink(metadataPath);
      } catch (err) {
        log?.("warn", `[Backup ${jobId}] Failed to remove metadata file:`, err.message);
      }

      // Mark as completed
      backupJobs.set(jobId, {
        ...backupJobs.get(jobId),
        status: "completed",
        progress: 100,
        completedAt: new Date().toISOString(),
        metadata,
      });

      log?.("info", `[Backup ${jobId}] Backup completed successfully`);
    } catch (err) {
      log?.("error", `[Backup ${jobId}] Backup failed:`, err.message);
      backupJobs.set(jobId, {
        ...backupJobs.get(jobId),
        status: "failed",
        error: err.message,
      });
    } finally {
      if (jobTmpDir) {
        try {
          await rm(jobTmpDir, { recursive: true, force: true });
        } catch (err) {
          log?.("warn", `[Backup ${jobId}] Failed to clean temp directory:`, err.message);
        }
      }
    }
  })();

  return { jobId, backupId, status: "started" };
}

// Restore backup from S3
export async function restoreBackup(backupId, s3Config, volumesToRestore, overwrite, log, restoreApps) {
  const jobId = generateJobId();
  const tmpDir = "/tmp";
  const appsDir = path.join(__dirname, "..", "apps");

  restoreJobs.set(jobId, {
    status: "in-progress",
    progress: 0,
    backupId,
    startedAt: new Date().toISOString(),
  });

  // Run restore asynchronously
  (async () => {
    try {
      log?.("info", `[Restore ${jobId}] Starting restore from backup ${backupId}`);

      // Ensure alpine image exists for tar operations
      try {
        await docker.getImage("alpine:latest").inspect();
        log?.("info", `[Restore ${jobId}] Alpine image already available`);
      } catch (err) {
        log?.("info", `[Restore ${jobId}] Pulling alpine:latest image...`);
        await new Promise((resolve, reject) => {
          docker.pull("alpine:latest", (err, stream) => {
            if (err) return reject(err);
            docker.modem.followProgress(stream, (err, output) => {
              if (err) return reject(err);
              resolve(output);
            });
          });
        });
        log?.("info", `[Restore ${jobId}] Alpine image pulled successfully`);
      }

      // Step 1: Download metadata
      const rcloneEnv = createRcloneEnv(s3Config);
      const metadataPath = path.join(tmpDir, `metadata-${jobId}.json`);

      const metaDownload = await spawnProcess(
        "rclone",
        [
          "copy",
          `s3:${s3Config.bucket}/yantra-backup/${backupId}/metadata.json`,
          tmpDir,
          "--progress",
        ],
        { env: rcloneEnv }
      );

      if (metaDownload.exitCode !== 0) {
        throw new Error(`Failed to download metadata: ${metaDownload.stderr}`);
      }

      // Rename downloaded file
      await spawnProcess("mv", [path.join(tmpDir, "metadata.json"), metadataPath]);

      const metadataContent = await readFile(metadataPath, "utf-8");
      const metadata = JSON.parse(metadataContent);
      await unlink(metadataPath);

      const shouldRestoreApps = restoreApps === true || (restoreApps === undefined && !volumesToRestore);
      const appConfigs = Array.isArray(metadata.appConfigs) ? metadata.appConfigs : [];

      if (shouldRestoreApps && appConfigs.length > 0) {
        log?.("info", `[Restore ${jobId}] Restoring ${appConfigs.length} app config(s)`);

        for (const appConfig of appConfigs) {
          const appId = appConfig.appId;
          const tarFileName = appConfig.tarFileName;

          if (!appId || !tarFileName) {
            log?.("warn", `[Restore ${jobId}] Skipping invalid app config entry`);
            continue;
          }

          const appPath = path.join(appsDir, appId);
          let appPathExists = false;

          try {
            await stat(appPath);
            appPathExists = true;
          } catch (err) {
            appPathExists = false;
          }

          if (appPathExists && !overwrite) {
            log?.("warn", `[Restore ${jobId}] Skipping app ${appId} (already exists, overwrite=false)`);
            continue;
          }

          log?.("info", `[Restore ${jobId}] Downloading app config for ${appId}`);

          const download = await spawnProcess(
            "rclone",
            [
              "copy",
              `s3:${s3Config.bucket}/yantra-backup/${backupId}/app-configs/${tarFileName}`,
              tmpDir,
              "--progress",
            ],
            { env: rcloneEnv }
          );

          if (download.exitCode !== 0) {
            throw new Error(`Failed to download app config ${appId}: ${download.stderr}`);
          }

          const tarPath = path.join(tmpDir, tarFileName);
          await mkdir(appPath, { recursive: true });

          const extract = await spawnProcess(
            "tar",
            ["xzf", tarPath, "-C", appPath]
          );

          if (extract.exitCode !== 0) {
            throw new Error(`Failed to extract app config ${appId}: ${extract.stderr}`);
          }

          try {
            await unlink(tarPath);
          } catch (err) {
            log?.("warn", `[Restore ${jobId}] Failed to remove app config tar ${tarFileName}:`, err.message);
          }

          log?.("info", `[Restore ${jobId}] Restored app config for ${appId}`);
        }
      }

      // Determine which volumes to restore
      const volumes = volumesToRestore || metadata.volumes.map(v => v.name);
      const totalVolumes = volumes.length;

      log?.("info", `[Restore ${jobId}] Restoring ${totalVolumes} volume(s)`);

      for (let i = 0; i < volumes.length; i++) {
        const volumeName = volumes[i];
        const volumeBackup = metadata.volumes.find(v => v.name === volumeName);

        if (!volumeBackup) {
          log?.("warn", `[Restore ${jobId}] Volume ${volumeName} not found in backup, skipping`);
          continue;
        }

        log?.("info", `[Restore ${jobId}] Restoring volume ${i + 1}/${totalVolumes}: ${volumeName}`);

        // Update progress
        restoreJobs.set(jobId, {
          ...restoreJobs.get(jobId),
          progress: Math.floor((i / totalVolumes) * 90),
          currentVolume: volumeName,
        });

        const tarFileName = volumeBackup.tarFileName;
        const tarPath = path.join(tmpDir, tarFileName);

        // Step 2: Download tar from S3
        log?.("info", `[Restore ${jobId}] Downloading ${volumeName} from S3`);

        const download = await spawnProcess(
          "rclone",
          [
            "copy",
            `s3:${s3Config.bucket}/yantra-backup/${backupId}/${tarFileName}`,
            tmpDir,
            "--progress",
          ],
          { env: rcloneEnv }
        );

        if (download.exitCode !== 0) {
          throw new Error(`Failed to download ${volumeName}: ${download.stderr}`);
        }

        // Step 3: Ensure volume exists
        try {
          await docker.getVolume(volumeName).inspect();
          log?.("info", `[Restore ${jobId}] Volume ${volumeName} exists`);

          if (!overwrite) {
            log?.("warn", `[Restore ${jobId}] Skipping ${volumeName} (already exists, overwrite=false)`);
            await unlink(tarPath);
            continue;
          }
        } catch (err) {
          // Volume doesn't exist, create it
          log?.("info", `[Restore ${jobId}] Creating volume ${volumeName}`);
          await docker.createVolume({ Name: volumeName });
        }

        // Step 4: Extract tar into volume
        log?.("info", `[Restore ${jobId}] Extracting archive into ${volumeName}`);

        const container = await docker.createContainer({
          Image: "alpine:latest",
          Cmd: ["tar", "xzf", `/backup/${tarFileName}`, "-C", "/data"],
          HostConfig: {
            Binds: [
              `${volumeName}:/data`,
              `${tmpDir}:/backup`,
            ],
            AutoRemove: true,
          },
        });

        await container.start();
        await container.wait();

        log?.("info", `[Restore ${jobId}] Restored ${volumeName} successfully`);

        // Cleanup
        await unlink(tarPath);
      }

      // Mark as completed
      restoreJobs.set(jobId, {
        ...restoreJobs.get(jobId),
        status: "completed",
        progress: 100,
        completedAt: new Date().toISOString(),
      });

      log?.("info", `[Restore ${jobId}] Restore completed successfully`);
    } catch (err) {
      log?.("error", `[Restore ${jobId}] Restore failed:`, err.message);
      restoreJobs.set(jobId, {
        ...restoreJobs.get(jobId),
        status: "failed",
        error: err.message,
      });
    }
  })();

  return { jobId, status: "started" };
}

// Delete backup from S3
export async function deleteBackup(backupId, s3Config, log) {
  try {
    log?.("info", `[Backup Delete] Deleting backup ${backupId}`);

    const rcloneEnv = createRcloneEnv(s3Config);
    const { stdout, stderr, exitCode } = await spawnProcess(
      "rclone",
      ["purge", `s3:${s3Config.bucket}/yantra-backup/${backupId}/`],
      { env: rcloneEnv }
    );

    if (exitCode !== 0) {
      throw new Error(`rclone delete failed: ${stderr}`);
    }

    log?.("info", `[Backup Delete] Backup ${backupId} deleted successfully`);
    return { success: true };
  } catch (err) {
    throw new Error(`Failed to delete backup: ${err.message}`);
  }
}

// Get job status
export function getBackupJobStatus(jobId) {
  return backupJobs.get(jobId) || null;
}

export function getRestoreJobStatus(jobId) {
  return restoreJobs.get(jobId) || null;
}

// Get all jobs
export function getAllBackupJobs() {
  return Array.from(backupJobs.entries()).map(([id, job]) => ({ id, ...job }));
}

export function getAllRestoreJobs() {
  return Array.from(restoreJobs.entries()).map(([id, job]) => ({ id, ...job }));
}
