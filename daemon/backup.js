import Docker from "dockerode";
import { readFile, writeFile, unlink, mkdir, stat } from "fs/promises";
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

    // List directories in the backup bucket (each backup is a directory)
    const { stdout, stderr, exitCode } = await spawnProcess(
      "rclone",
      ["lsjson", `s3:${s3Config.bucket}/yantra-backups/`],
      { env: rcloneEnv }
    );

    if (exitCode !== 0) {
      // If directory doesn't exist yet (no backups created), return empty array
      if (stderr.includes("directory not found") || stderr.includes("not found")) {
        log?.("info", "No backups directory found yet - returning empty list");
        return [];
      }
      throw new Error(`rclone failed: ${stderr}`);
    }

    const items = JSON.parse(stdout || "[]");

    // Filter directories (backups) and get metadata
    const backups = [];
    for (const item of items) {
      if (item.IsDir && item.Name.startsWith("backup-")) {
        try {
          // Try to read metadata file
          const metaResult = await spawnProcess(
            "rclone",
            ["cat", `s3:${s3Config.bucket}/yantra-backups/${item.Name}/metadata.json`],
            { env: rcloneEnv }
          );

          if (metaResult.exitCode === 0) {
            const metadata = JSON.parse(metaResult.stdout);
            backups.push(metadata);
          }
        } catch (err) {
          log?.("warn", `Failed to read metadata for ${item.Name}:`, err.message);
        }
      }
    }

    return backups.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp));
  } catch (err) {
    throw new Error(`Failed to list backups: ${err.message}`);
  }
}

// Create backup of volumes
export async function createBackup(volumes, s3Config, name, log) {
  const jobId = generateJobId();
  const backupId = `backup-${Date.now()}`;
  const tmpDir = "/tmp";

  backupJobs.set(jobId, {
    status: "in-progress",
    progress: 0,
    backupId,
    volumes,
    startedAt: new Date().toISOString(),
  });

  // Run backup asynchronously
  (async () => {
    try {
      log?.("info", `[Backup ${jobId}] Starting backup of ${volumes.length} volume(s)`);

      const volumeBackups = [];
      const totalVolumes = volumes.length;

      for (let i = 0; i < volumes.length; i++) {
        const volumeName = volumes[i];
        log?.("info", `[Backup ${jobId}] Processing volume ${i + 1}/${totalVolumes}: ${volumeName}`);

        // Update progress
        backupJobs.set(jobId, {
          ...backupJobs.get(jobId),
          progress: Math.floor((i / totalVolumes) * 90),
          currentVolume: volumeName,
        });

        const tarFileName = `${volumeName}.tar.gz`;
        const tarPath = path.join(tmpDir, tarFileName);

        // Step 1: Create tar of volume using temporary container
        log?.("info", `[Backup ${jobId}] Creating tar archive of ${volumeName}`);

        const container = await docker.createContainer({
          Image: "alpine:latest",
          Cmd: ["tar", "czf", `/backup/${tarFileName}`, "-C", "/data", "."],
          HostConfig: {
            Binds: [
              `${volumeName}:/data:ro`,
              `${tmpDir}:/backup`,
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
            `s3:${s3Config.bucket}/yantra-backups/${backupId}/`,
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
        await unlink(tarPath);
      }

      // Step 3: Upload metadata
      log?.("info", `[Backup ${jobId}] Creating metadata`);

      const metadata = {
        id: backupId,
        name: name || `Backup ${new Date().toLocaleString()}`,
        timestamp: new Date().toISOString(),
        volumes: volumeBackups,
        totalSize: volumeBackups.reduce((sum, v) => sum + v.size, 0),
      };

      const metadataPath = path.join(tmpDir, "metadata.json");
      await writeFile(metadataPath, JSON.stringify(metadata, null, 2), "utf-8");

      const rcloneEnv = createRcloneEnv(s3Config);
      const metaUpload = await spawnProcess(
        "rclone",
        [
          "copy",
          metadataPath,
          `s3:${s3Config.bucket}/yantra-backups/${backupId}/`,
        ],
        { env: rcloneEnv }
      );

      if (metaUpload.exitCode !== 0) {
        throw new Error(`Failed to upload metadata: ${metaUpload.stderr}`);
      }

      await unlink(metadataPath);

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
    }
  })();

  return { jobId, backupId, status: "started" };
}

// Restore backup from S3
export async function restoreBackup(backupId, s3Config, volumesToRestore, overwrite, log) {
  const jobId = generateJobId();
  const tmpDir = "/tmp";

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

      // Step 1: Download metadata
      const rcloneEnv = createRcloneEnv(s3Config);
      const metadataPath = path.join(tmpDir, `metadata-${jobId}.json`);

      const metaDownload = await spawnProcess(
        "rclone",
        [
          "copy",
          `s3:${s3Config.bucket}/yantra-backups/${backupId}/metadata.json`,
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
            `s3:${s3Config.bucket}/yantra-backups/${backupId}/${tarFileName}`,
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
      ["purge", `s3:${s3Config.bucket}/yantra-backups/${backupId}/`],
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
