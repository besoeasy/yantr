import Docker from "dockerode";
import { readFile, access } from "fs/promises";
import { resolveComposeCommand } from "./compose.js";
import { spawnProcess, getBaseAppId } from "./utils.js";
import path from "path";
import { fileURLToPath } from "url";
import { dirname } from "path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Docker socket path
const socketPath = process.env.DOCKER_SOCKET || "/var/run/docker.sock";
const docker = new Docker({ socketPath });

/**
 * Logger utility for cleanup operations
 */
function log(level, message, ...args) {
  const timestamp = new Date().toISOString();
  const formattedMessage = args.length > 0 ? `${message} ${args.join(" ")}` : message;
  const prefix = level === "error" ? "❌" : level === "info" ? "🧹" : "⚠️ ";

  if (level === "error") {
    console.error(`[${timestamp}] ${prefix} [CLEANUP] ${formattedMessage}`);
  } else {
    console.log(`[${timestamp}] ${prefix} [CLEANUP] ${formattedMessage}`);
  }
}

/**
 * Build a Set of volumes currently in use by containers (excluding specified containers)
 * Follows the batch operation pattern from /api/system/prune endpoint
 * @param {Set<string>} excludeContainerIds - Container IDs to exclude from check
 * @returns {Promise<Set<string>>} Set of volume names in use
 */
async function getUsedVolumes(excludeContainerIds = new Set()) {
  try {
    const containers = await docker.listContainers({ all: true });
    const usedVolumes = new Set();

    for (const container of containers) {
      // Skip excluded containers (ones being removed)
      if (excludeContainerIds.has(container.Id)) {
        continue;
      }

      // Collect all volume mounts
      if (container.Mounts) {
        for (const mount of container.Mounts) {
          if (mount.Type === "volume") {
            usedVolumes.add(mount.Name);
          }
        }
      }
    }

    return usedVolumes;
  } catch (error) {
    log("warn", `Failed to check volume usage: ${error.message}`);
    // Err on the side of caution - return empty Set will cause all volumes to be skipped
    // since we can't verify safety
    return new Set();
  }
}

/**
 * Check and remove expired temporary apps
 * @returns {Promise<Object>} Results of cleanup operation
 */
export async function cleanupExpiredApps() {
  log("info", "Starting cleanup check for expired temporary apps");

  const results = {
    checked: 0,
    expired: 0,
    removed: [],
    failed: [],
    timestamp: new Date().toISOString(),
  };

  try {
    // Get all containers
    const containers = await docker.listContainers({ all: true });
    results.checked = containers.length;

    // Early exit if no containers have expiration labels
    const hasExpiredContainers = containers.some(c => c.Labels?.["yantr.expireAt"]);
    if (!hasExpiredContainers) {
      log("info", "No temporary apps found, skipping cleanup");
      return results;
    }

    log("info", `Checking ${containers.length} containers for expiration`);

    const now = Math.floor(Date.now() / 1000); // Current Unix timestamp

    // First pass: identify expired containers
    const expiredContainers = [];
    for (const container of containers) {
      const labels = container.Labels || {};
      const expireAt = labels["yantr.expireAt"];

      if (!expireAt) continue;

      const expirationTime = parseInt(expireAt, 10);
      if (isNaN(expirationTime)) {
        const containerName = container.Names[0]?.replace("/", "") || "unknown";
        log("warn", `Invalid expireAt timestamp for ${containerName}: ${expireAt}`);
        continue;
      }

      if (now >= expirationTime) {
        expiredContainers.push({ container, expirationTime });
      }
    }

    if (expiredContainers.length === 0) {
      log("info", "No expired containers found");
      return results;
    }

    results.expired = expiredContainers.length;
    log("info", `Found ${expiredContainers.length} expired container(s)`);

    // Build Set of container IDs being removed (for volume check optimization)
    const expiredContainerIds = new Set(expiredContainers.map(e => e.container.Id));

    // Batch operation: get all volumes in use by non-expired containers (single API call)
    const usedVolumes = await getUsedVolumes(expiredContainerIds);
    log("info", `Found ${usedVolumes.size} volume(s) in use by other containers`);

    // Second pass: remove expired containers and their volumes
    for (const { container, expirationTime } of expiredContainers) {
      const labels = container.Labels || {};
      const composeProject = labels["com.docker.compose.project"];
      const containerName = container.Names[0]?.replace("/", "") || "unknown";

      log("info", `Processing expired app: ${containerName} (expired at ${new Date(expirationTime * 1000).toISOString()})`);

      try {
        // If part of a compose project, remove the entire stack
        if (composeProject) {
          const appsDir = path.join(__dirname, "..", "apps");
          const baseAppId = getBaseAppId(composeProject);
          const appPath = path.join(appsDir, baseAppId);
          const composePath = path.join(appPath, "compose.yml");

          try {
            try {
              await access(composePath);
            } catch {
              throw new Error("Compose file not found");
            }

            log("info", `Removing compose stack: ${composeProject}`);

            // Execute docker compose down with project name to target specific instance
            const composeCmd = await resolveComposeCommand({ socketPath });
            const { stdout, stderr, exitCode } = await spawnProcess(
              composeCmd.command,
              [...composeCmd.args, "-p", composeProject, "down"],
              {
                cwd: appPath,
                env: {
                  ...process.env,
                  DOCKER_HOST: `unix://${socketPath}`,
                },
              }
            );

            if (exitCode !== 0) {
              throw new Error(`docker compose down failed: ${stderr}`);
            }

            log("info", `Successfully removed stack: ${composeProject}`);
            results.removed.push({
              name: composeProject,
              type: "stack",
              containers: [containerName],
              expiredAt: new Date(expirationTime * 1000).toISOString(),
            });

            // Skip individual container processing since stack was removed
            continue;
          } catch (err) {
            log("warn", `Compose file not found for ${composeProject}, falling back to individual removal`);
          }
        }

        // Fallback: Individual container removal
        log("info", `Removing individual container: ${containerName}`);
        const containerObj = docker.getContainer(container.Id);
        const info = await containerObj.inspect();

        // Get volume names before removal
        const volumeNames = info.Mounts
          .filter((mount) => mount.Type === "volume")
          .map((mount) => mount.Name);

        // Stop if running
        if (info.State.Running) {
          await containerObj.stop();
        }

        // Remove container
        await containerObj.remove();

        // Remove associated volumes (O(1) lookup using pre-built Set)
        const removedVolumes = [];
        const skippedVolumes = [];
        for (const volumeName of volumeNames) {
          try {
            // Check if volume is being used by other containers (O(1) Set lookup)
            if (usedVolumes.has(volumeName)) {
              log("info", `Skipping volume ${volumeName} (shared with other containers)`);
              skippedVolumes.push(volumeName);
              continue;
            }

            // Safe to remove - volume is only used by this container
            const volume = docker.getVolume(volumeName);
            await volume.remove();
            removedVolumes.push(volumeName);
            log("info", `Removed volume ${volumeName}`);
          } catch (err) {
            log("warn", `Failed to remove volume ${volumeName}: ${err.message}`);
          }
        }

        log("info", `Successfully removed container: ${containerName}`);
        results.removed.push({
          name: containerName,
          type: "container",
          volumes: removedVolumes,
          volumesSkipped: skippedVolumes,
          expiredAt: new Date(expirationTime * 1000).toISOString(),
        });

      } catch (error) {
        log("error", `Failed to remove ${containerName}: ${error.message}`);
        results.failed.push({
          name: containerName,
          error: error.message,
          expiredAt: new Date(expirationTime * 1000).toISOString(),
        });
      }
    }

    log("info", `Cleanup complete: ${results.removed.length} removed, ${results.failed.length} failed`);
    return results;

  } catch (error) {
    log("error", `Cleanup check failed: ${error.message}`);
    return {
      ...results,
      error: error.message,
    };
  }
}

/**
 * Start automatic cleanup scheduler
 * @param {number} intervalMinutes - How often to run cleanup (default: 60 minutes)
 */
export function startCleanupScheduler(intervalMinutes = 60) {
  const intervalMs = intervalMinutes * 60 * 1000;

  log("info", `Starting cleanup scheduler (runs every ${intervalMinutes} minutes)`);

  // Run initial cleanup on startup
  cleanupExpiredApps().catch((err) => {
    log("error", `Initial cleanup failed: ${err.message}`);
  });

  // Run cleanup on interval
  setInterval(() => {
    cleanupExpiredApps().catch((err) => {
      log("error", `Scheduled cleanup failed: ${err.message}`);
    });
  }, intervalMs);
}
