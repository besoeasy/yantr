import Docker from "dockerode";
import { exec } from "child_process";
import util from "util";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
import { dirname } from "path";

const execPromise = util.promisify(exec);
const fsPromises = fs.promises;

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
    
    log("info", `Checking ${containers.length} containers for expiration`);

    const now = Math.floor(Date.now() / 1000); // Current Unix timestamp

    for (const container of containers) {
      const labels = container.Labels || {};
      const expireAt = labels["yantra.expireAt"];
      const composeProject = labels["com.docker.compose.project"];
      const containerName = container.Names[0]?.replace("/", "") || "unknown";

      // Skip if no expiration label
      if (!expireAt) continue;

      // Parse expiration timestamp
      const expirationTime = parseInt(expireAt, 10);
      
      if (isNaN(expirationTime)) {
        log("warn", `Invalid expireAt timestamp for ${containerName}: ${expireAt}`);
        continue;
      }

      // Check if expired
      if (now >= expirationTime) {
        results.expired++;
        log("info", `Found expired app: ${containerName} (expired at ${new Date(expirationTime * 1000).toISOString()})`);

        try {
          // If part of a compose project, remove the entire stack
          if (composeProject) {
            const appsDir = path.join(__dirname, "..", "apps");
            const appPath = path.join(appsDir, composeProject);
            const composePath = path.join(appPath, "compose.yml");

            try {
              await fsPromises.access(composePath);
              log("info", `Removing compose stack: ${composeProject}`);

              // Execute docker compose down with volume removal
              const command = `docker compose down -v`;
              const { stdout, stderr } = await execPromise(command, {
                cwd: appPath,
                env: {
                  ...process.env,
                  DOCKER_HOST: `unix://${socketPath}`,
                },
              });

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

          // Remove associated volumes
          const removedVolumes = [];
          for (const volumeName of volumeNames) {
            try {
              const volume = docker.getVolume(volumeName);
              await volume.remove();
              removedVolumes.push(volumeName);
            } catch (err) {
              log("warn", `Failed to remove volume ${volumeName}: ${err.message}`);
            }
          }

          log("info", `Successfully removed container: ${containerName}`);
          results.removed.push({
            name: containerName,
            type: "container",
            volumes: removedVolumes,
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
  
  // Run immediately on start
  cleanupExpiredApps().catch((err) => {
    log("error", `Initial cleanup failed: ${err.message}`);
  });

  // Then run on interval
  setInterval(() => {
    cleanupExpiredApps().catch((err) => {
      log("error", `Scheduled cleanup failed: ${err.message}`);
    });
  }, intervalMs);
}

// Allow direct execution for testing
if (import.meta.url === `file://${process.argv[1]}`) {
  log("info", "Running cleanup manually");
  cleanupExpiredApps()
    .then((results) => {
      console.log("\nCleanup Results:");
      console.log(JSON.stringify(results, null, 2));
      process.exit(0);
    })
    .catch((err) => {
      console.error("Cleanup failed:", err);
      process.exit(1);
    });
}
