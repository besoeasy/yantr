/**
 * Auto-Update
 *
 * Two update functions:
 *
 * 1. runUpdate(containerNames) — runs watchtower --run-once targeting specific
 *    container names. Called on-demand from the UI per stack.
 *
 * 2. runSelfUpdate() — runs watchtower --run-once targeting only the yantr
 *    container by name so yantr can keep itself current.
 *    Container name: YANTR_CONTAINER_NAME (default: "yantr")
 */

import { spawnProcess } from "./utils.js";
import { socketPath } from "./shared.js";

let log = () => {};

async function runWatchtower(containerArgs, label) {
  try {
    const { exitCode, stdout, stderr } = await spawnProcess("docker", [
      "run", "--rm",
      "-v", `${socketPath}:/var/run/docker.sock`,
      "-e", "DOCKER_API_VERSION=1.44",
      "containrrr/watchtower",
      "--run-once",
      "--cleanup",
      ...containerArgs,
    ], { env: { ...process.env, DOCKER_HOST: `unix://${socketPath}`, DOCKER_API_VERSION: "1.44" }, timeout: 10 * 60 * 1000 });

    if (exitCode === 0) {
      // Watchtower logs 'msg="Updated <name>"' for each container it actually pulls+restarts.
      // Count those lines to distinguish "pulled new image" from "already up to date".
      const allOutput = stdout + stderr;
      const updatedMatches = allOutput.match(/msg="Updated /g);
      const updatedCount = updatedMatches ? updatedMatches.length : 0;
      log("info", `🔄 [${label}] Completed — ${updatedCount} container(s) updated`);
      return { exitCode, stdout, stderr, updatedCount };
    } else {
      log("warn", `🔄 [${label}] Finished with exit code ${exitCode}: ${stderr.trim()}`);
      return { exitCode, stdout, stderr, updatedCount: 0 };
    }
  } catch (e) {
    log("error", `🔄 [${label}] Failed: ${e.message}`);
    return { error: e.message };
  }
}

// Update a specific set of containers by name (called per-stack from the UI)
export async function runUpdate(containerNames = []) {
  if (!containerNames.length) return { error: "No container names provided" };
  log("info", `🔄 [UPDATE] Updating containers: ${containerNames.join(", ")}`);
  return runWatchtower(containerNames, "UPDATE");
}

// Update yantr itself by container name
export async function runSelfUpdate() {
  const containerName = process.env.YANTR_CONTAINER_NAME || "yantr";
  log("info", `🔄 [SELFUPDATE] Checking for yantr update (container: ${containerName})...`);
  return runWatchtower([containerName], "SELFUPDATE");
}

export function initAutoUpdate(logger) {
  log = logger;
  if (process.env.YANTR_SELFUPDATE !== "false") {
    const intervalHours = parseFloat(process.env.YANTR_SELFUPDATE_INTERVAL) || 1;
    setTimeout(() => {
      runSelfUpdate();
    }, 5 * 60 * 1000);
    setInterval(() => {
      if (Math.random() < 0.5) runSelfUpdate();
    }, intervalHours * 60 * 60 * 1000);
    log("info", `🔄 [SELFUPDATE] Scheduler started — every ${intervalHours}h with 50% chance (first run in 10m)`);
  } else {
    log("info", "🔄 [SELFUPDATE] Disabled via YANTR_SELFUPDATE=false");
  }
}

