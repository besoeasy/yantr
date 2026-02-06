import Docker from "dockerode";
import { readFile, writeFile, unlink, mkdir, stat, mkdtemp, rm } from "fs/promises";
import path from "path";
import { fileURLToPath } from "url";
import { dirname } from "path";
import { spawnProcess } from "./utils.js";
import { resolveComposeCommand } from "./compose.js";
import { Client as MinioClient } from "minio";

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
  const configPath = path.join(__dirname, "..", "backup-config.json");

  try {
    const content = await readFile(configPath, "utf-8");
    return JSON.parse(content);
  } catch (err) {
    return null;
  }
}

// Save S3 config
export async function saveS3Config(config) {
  const configPath = path.join(__dirname, "..", "backup-config.json");
  await writeFile(configPath, JSON.stringify(config, null, 2), "utf-8");
}

// Create MinIO client from S3 config
function createMinioClient(s3Config) {
  // Parse endpoint to extract host and port
  let endPoint = s3Config.endpoint;
  let port = 9000;
  let useSSL = true;

  if (endPoint) {
    // Remove protocol if present
    endPoint = endPoint.replace(/^https?:\/\//, "");
    useSSL = s3Config.endpoint.startsWith("https://");
    
    // Extract port if present
    if (endPoint.includes(":")) {
      const parts = endPoint.split(":");
      endPoint = parts[0];
      port = parseInt(parts[1], 10);
    } else {
      port = useSSL ? 443 : 80;
    }
  }

  return new MinioClient({
    endPoint,
    port,
    useSSL,
    accessKey: s3Config.accessKey,
    secretKey: s3Config.secretKey,
    region: s3Config.region || "us-east-1",
  });
}

function getComposeEnv(socketPath) {
  return {
    ...process.env,
    DOCKER_HOST: socketPath ? `unix://${socketPath}` : process.env.DOCKER_HOST,
  };
}

async function startRestoredApps(metadata, log) {
  const apps = Array.isArray(metadata?.apps) ? metadata.apps : [];
  const appConfigs = Array.isArray(metadata?.appConfigs) ? metadata.appConfigs : [];
  const appIds = new Set();

  for (const app of apps) {
    if (app?.appId) {
      appIds.add(app.appId);
    }
  }

  for (const appConfig of appConfigs) {
    if (appConfig?.appId) {
      appIds.add(appConfig.appId);
    }
  }

  if (appIds.size === 0) {
    return;
  }

  const socketPath = process.env.DOCKER_SOCKET || "/var/run/docker.sock";
  const compose = await resolveComposeCommand({ socketPath, log });
  const env = getComposeEnv(socketPath);
  const appsDir = path.join(__dirname, "..", "apps");

  for (const appId of appIds) {
    const composePath = path.join(appsDir, appId, "compose.yml");

    try {
      await stat(composePath);
    } catch (err) {
      log?.("warn", `[Restore] Compose file not found for ${appId}, skipping start`);
      continue;
    }

    const { exitCode, stderr } = await spawnProcess(
      compose.command,
      [...compose.args, "-f", composePath, "up", "-d"],
      { env }
    );

    if (exitCode !== 0) {
      log?.("warn", `[Restore] Failed to start ${appId}: ${stderr}`);
    }
  }
}

// List backups from S3
export async function listBackups(s3Config, log) {
  try {
    const minioClient = createMinioClient(s3Config);
    const bucket = s3Config.bucket;
    const prefix = "yantra-backup/";

    // List app directories in yantra-backup/
    const appFolders = new Set();
    const stream = minioClient.listObjects(bucket, prefix, false);

    for await (const obj of stream) {
      if (obj.prefix) {
        // This is a directory/prefix
        const appName = obj.prefix.replace(prefix, "").replace(/\/$/, "");
        if (appName) appFolders.add(appName);
      }
    }


    // Read metadata from each app folder
    const backups = [];
    for (const appName of appFolders) {

      try {
        const metadataPath = `${prefix}${appName}/metadata.json`;
        
        // Get metadata.json
        const chunks = [];
        const dataStream = await minioClient.getObject(bucket, metadataPath);
        
        for await (const chunk of dataStream) {
          chunks.push(chunk);
        }
        
        const metadataContent = Buffer.concat(chunks).toString("utf-8");
        const metadata = JSON.parse(metadataContent);
        backups.push(metadata);
      } catch (err) {
        log?.("error", `Failed to read metadata for ${appName}:`, err.message);
      }
    }

    return backups.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp));
  } catch (err) {
    // If bucket doesn't exist or is empty, return empty array
    if (err.code === "NoSuchBucket" || err.code === "NoSuchKey") {
      log?.("info", "No backups found - bucket or prefix doesn't exist yet");
      return [];
    }
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

      const minioClient = createMinioClient(s3Config);
      const expectedKeys = [];

      for (let i = 0; i < volumeList.length; i++) {
        const volumeName = volumeList[i];
        log?.("info", `[Backup ${jobId}] Processing volume ${i + 1}/${totalVolumes}: ${volumeName}`);

        // Update progress
        backupJobs.set(jobId, {
          ...backupJobs.get(jobId),
          progress: Math.floor((i / totalVolumes) * 90),
          currentVolume: volumeName,
        });

        const tarFileName = `${volumeName}.tar`;
        const tarPath = path.join(jobTmpDir, tarFileName);

        // Step 1: Create tar of volume using temporary container
        log?.("info", `[Backup ${jobId}] Creating tar archive of ${volumeName}`);

        const container = await docker.createContainer({
          Image: "alpine:latest",
          Cmd: ["tar", "cf", `/backup/${tarFileName}`, "-C", "/data", "."],
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

        // Step 2: Upload to S3 using MinIO client
        log?.("info", `[Backup ${jobId}] Uploading ${volumeName} to S3`);

        await minioClient.fPutObject(
          s3Config.bucket,
          `yantra-backup/${backupId}/${tarFileName}`,
          tarPath
        );

        log?.("info", `[Backup ${jobId}] Uploaded ${volumeName} successfully`);

        volumeBackups.push({
          name: volumeName,
          size: sizeBytes,
          tarFileName,
        });
        expectedKeys.push(`yantra-backup/${backupId}/${tarFileName}`);

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

        const appTarFileName = `app-${appId}.tar`;
        const appTarPath = path.join(jobTmpDir, appTarFileName);

        const tarResult = await spawnProcess(
          "tar",
          ["cf", appTarPath, "-C", appPath, "."]
        );

        if (tarResult.exitCode !== 0) {
          throw new Error(`Failed to archive app ${appId}: ${tarResult.stderr}`);
        }

        const appStats = await stat(appTarPath);

        await minioClient.fPutObject(
          s3Config.bucket,
          `yantra-backup/${backupId}/app-configs/${appTarFileName}`,
          appTarPath
        );

        appConfigBackups.push({
          appId,
          tarFileName: appTarFileName,
          size: appStats.size,
        });
        expectedKeys.push(`yantra-backup/${backupId}/app-configs/${appTarFileName}`);

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

      await minioClient.fPutObject(
        s3Config.bucket,
        `yantra-backup/${backupId}/metadata.json`,
        metadataPath
      );
      expectedKeys.push(`yantra-backup/${backupId}/metadata.json`);

      for (const key of expectedKeys) {
        await minioClient.statObject(s3Config.bucket, key);
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
  let jobTmpDir = null;
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
      // Create job-specific temp directory
      jobTmpDir = await mkdtemp(path.join(tmpDir, `restore-${jobId}-`));
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
      const minioClient = createMinioClient(s3Config);
      const metadataPath = path.join(jobTmpDir, "metadata.json");

      await minioClient.fGetObject(
        s3Config.bucket,
        `yantra-backup/${backupId}/metadata.json`,
        metadataPath
      );

      const metadataContent = await readFile(metadataPath, "utf-8");
      const metadata = JSON.parse(metadataContent);
      await unlink(metadataPath);

      const shouldRestoreApps = restoreApps === true || (restoreApps === undefined && !volumesToRestore);
      const appConfigs = Array.isArray(metadata.appConfigs) ? metadata.appConfigs : [];

      if (shouldRestoreApps && appConfigs.length > 0) {
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

          const tarPath = path.join(jobTmpDir, tarFileName);
          
          await minioClient.fGetObject(
            s3Config.bucket,
            `yantra-backup/${backupId}/app-configs/${tarFileName}`,
            tarPath
          );

          await mkdir(appPath, { recursive: true });

          const extract = await spawnProcess("tar", ["xf", tarPath, "-C", appPath]);

          if (extract.exitCode !== 0) {
            throw new Error(`Failed to extract app config ${appId}: ${extract.stderr}`);
          }

          try {
            await unlink(tarPath);
          } catch (err) {
            log?.("warn", `[Restore ${jobId}] Failed to remove app config tar ${tarFileName}:`, err.message);
          }

        }
      }

      // Determine which volumes to restore
      const volumes = volumesToRestore || metadata.volumes.map(v => v.name);
      const totalVolumes = volumes.length;

      for (let i = 0; i < volumes.length; i++) {
        const volumeName = volumes[i];
        const volumeBackup = metadata.volumes.find(v => v.name === volumeName);

        if (!volumeBackup) {
          log?.("warn", `[Restore ${jobId}] Volume ${volumeName} not found in backup, skipping`);
          continue;
        }

        // Update progress
        restoreJobs.set(jobId, {
          ...restoreJobs.get(jobId),
          progress: Math.floor((i / totalVolumes) * 90),
          currentVolume: volumeName,
        });

        const tarFileName = volumeBackup.tarFileName;
        const tarPath = path.join(jobTmpDir, tarFileName);

        // Step 2: Download tar from S3

        await minioClient.fGetObject(
          s3Config.bucket,
          `yantra-backup/${backupId}/${tarFileName}`,
          tarPath
        );

        // Step 3: Ensure volume exists
        try {
          await docker.getVolume(volumeName).inspect();

          if (!overwrite) {
            log?.("warn", `[Restore ${jobId}] Skipping ${volumeName} (already exists, overwrite=false)`);
            await unlink(tarPath);
            continue;
          }
        } catch (err) {
          // Volume doesn't exist, create it
          await docker.createVolume({ Name: volumeName });
        }

        // Step 4: Extract tar into volume
        const container = await docker.createContainer({
          Image: "alpine:latest",
          Cmd: ["tar", "xf", `/backup/${tarFileName}`, "-C", "/data"],
          HostConfig: {
            Binds: [
              `${volumeName}:/data`,
              `${jobTmpDir}:/backup`,
            ],
            AutoRemove: true,
          },
        });

        await container.start();
        await container.wait();

        // Cleanup
        await unlink(tarPath);
      }

      if (shouldRestoreApps) {
        await startRestoredApps(metadata, log);
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
    } finally {
      // Cleanup job temp directory
      if (jobTmpDir) {
        try {
          await rm(jobTmpDir, { recursive: true, force: true });
        } catch (err) {
          log?.("warn", `[Restore ${jobId}] Failed to cleanup temp directory:`, err.message);
        }
      }
    }
  })();

  return { jobId, status: "started" };
}

// Delete backup from S3
export async function deleteBackup(backupId, s3Config, log) {
  try {
    log?.("info", `[Backup Delete] Deleting backup ${backupId}`);

    const minioClient = createMinioClient(s3Config);
    const bucket = s3Config.bucket;
    const prefix = `yantra-backup/${backupId}/`;

    // List all objects with this prefix
    const objectsList = [];
    const stream = minioClient.listObjects(bucket, prefix, true);

    for await (const obj of stream) {
      objectsList.push(obj.name);
    }

    // Delete all objects
    if (objectsList.length > 0) {
      await minioClient.removeObjects(bucket, objectsList);
      log?.("info", `[Backup Delete] Deleted ${objectsList.length} object(s)`);
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
