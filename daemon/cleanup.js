import Docker from "dockerode";
import { readFile } from "fs/promises";
import { spawn } from "child_process";
import { resolveComposeCommand } from "./compose.js";
import path from "path";
import { fileURLToPath } from "url";
import { dirname } from "path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Docker socket path
const socketPath = process.env.DOCKER_SOCKET || "/var/run/docker.sock";
const docker = new Docker({ socketPath });

/**
 * Helper function to spawn a process and capture output
 */
function spawnProcess(command, args, options = {}) {
  return new Promise((resolve, reject) => {
    const proc = spawn(command, args, {
      ...options,
      env: options.env || process.env,
    });

    let stdout = '';
    let stderr = '';

    if (proc.stdout) {
      proc.stdout.on('data', (data) => {
        stdout += data.toString();
      });
    }

    if (proc.stderr) {
      proc.stderr.on('data', (data) => {
        stderr += data.toString();
      });
    }

    proc.on('close', (code) => {
      resolve({ stdout, stderr, exitCode: code });
    });

    proc.on('error', (err) => {
      reject(err);
    });
  });
}

// Compose resolver moved to compose.js

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

    // Early exit if no containers have expiration labels
    const hasExpiredContainers = containers.some(c => c.Labels?.["yantra.expireAt"]);
    if (!hasExpiredContainers) {
      log("info", "No temporary apps found, skipping cleanup");
      return results;
    }

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
              const fs = await import("fs/promises");
              try {
                await fs.access(composePath);
              } catch {
                throw new Error("Compose file not found");
              }
              
              log("info", `Removing compose stack: ${composeProject}`);

              // Execute docker compose down without volume removal
              const composeCmd = await resolveComposeCommand({ socketPath });
              const { stdout, stderr, exitCode } = await spawnProcess(
                composeCmd.command,
                [...composeCmd.args, "down"],
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
