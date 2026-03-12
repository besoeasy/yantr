/**
 * Self-Install Bootstrap
 *
 * When the user runs the minimal install command:
 *
 *   docker run -v /var/run/docker.sock:/var/run/docker.sock ghcr.io/besoeasy/yantr
 *
 * This module detects that the container is NOT running with the full required
 * configuration (host network + volumes mount) and uses the Docker socket to:
 *
 *   1. Pull the latest image (if not already present)
 *   2. Remove any existing "yantr" container
 *   3. Launch a new "yantr" container with the full production configuration
 *   4. Exit this bootstrap container
 *
 * If the container is already running with the correct configuration (i.e. the
 * user ran the full command, or this is the properly-installed container), this
 * module is a no-op and normal startup continues.
 */

import Docker from "dockerode";
import os from "os";

const CONTAINER_NAME = process.env.YANTR_CONTAINER_NAME || "yantr";
const IMAGE = process.env.YANTR_IMAGE || "ghcr.io/besoeasy/yantr";
const SOCKET_PATH = process.env.DOCKER_SOCKET || "/var/run/docker.sock";

/**
 * Check whether we are already running as the fully-configured container.
 *
 * We consider the current container "fully configured" when ALL of the
 * following are true:
 *   - It is named CONTAINER_NAME (i.e. it was started by a previous install run)
 *   - It has NetworkMode === "host"
 *   - It has /var/lib/docker/volumes bind-mounted
 *
 * If any of these are missing we are the bootstrap (minimal) container and
 * should re-launch ourselves properly.
 */
async function isFullyConfigured(docker) {
  // Resolve our own container ID from the cgroup / hostname
  const hostname = os.hostname();

  let selfInfo = null;
  try {
    // In Docker, the container short-id is the hostname by default
    const container = docker.getContainer(hostname);
    selfInfo = await container.inspect();
  } catch {
    // Could not inspect by hostname — not inside a container at all (dev mode)
    return true;
  }

  const hostConfig = selfInfo?.HostConfig || {};
  const name = (selfInfo?.Name || "").replace(/^\//, "");

  const hasHostNetwork = hostConfig.NetworkMode === "host";
  const binds = hostConfig.Binds || [];
  const hasVolumeMount = binds.some((b) => b.includes("/var/lib/docker/volumes"));
  const isNamedCorrectly = name === CONTAINER_NAME;

  return hasHostNetwork && hasVolumeMount && isNamedCorrectly;
}

/**
 * Remove a container by name if it exists (stop it first if running).
 */
async function removeExisting(docker, name) {
  try {
    const container = docker.getContainer(name);
    const info = await container.inspect();
    if (info.State?.Running) {
      console.log(`[selfinstall] Stopping existing "${name}" container...`);
      await container.stop({ t: 5 });
    }
    console.log(`[selfinstall] Removing existing "${name}" container...`);
    await container.remove({ force: true });
  } catch (err) {
    if (err?.statusCode === 404) return; // doesn't exist, nothing to do
    throw err;
  }
}

/**
 * Run the self-install bootstrap.
 * Returns true if the bootstrap ran (caller should exit immediately after).
 * Returns false if we are already fully configured (normal startup continues).
 */
export async function runSelfInstallIfNeeded() {
  const docker = new Docker({ socketPath: SOCKET_PATH });

  let fullyConfigured = false;
  try {
    fullyConfigured = await isFullyConfigured(docker);
  } catch (err) {
    console.warn(`[selfinstall] Could not check configuration: ${err.message}`);
    return false;
  }

  if (fullyConfigured) {
    return false; // already correct — proceed with normal startup
  }

  console.log("[selfinstall] Minimal install detected. Launching fully-configured yantr container...");

  // Remove any existing container with this name
  await removeExisting(docker, CONTAINER_NAME);

  // Create and start the new container with full production config
  const container = await docker.createContainer({
    name: CONTAINER_NAME,
    Image: IMAGE,
    HostConfig: {
      NetworkMode: "host",
      RestartPolicy: { Name: "unless-stopped" },
      Binds: [
        `${SOCKET_PATH}:/var/run/docker.sock`,
        "/var/lib/docker/volumes:/var/lib/docker/volumes",
      ],
    },
    Env: buildEnv(),
  });

  await container.start();

  console.log(`[selfinstall] yantr container started successfully.`);
  console.log(`[selfinstall] Access the UI at http://localhost:5252`);
  console.log(`[selfinstall] This bootstrap container will now exit.`);

  return true; // signal to caller: exit
}

/**
 * Forward whitelisted env vars from the current process to the new container.
 * This lets users pass TZ, YANTR_* vars through the minimal install command.
 */
function buildEnv() {
  const passthrough = [
    "TZ",
    "YANTR_CONTAINER_NAME",
    "YANTR_IMAGE",
    "YANTR_SELFUPDATE",
    "YANTR_SELFUPDATE_INTERVAL",
    "DOCKER_SOCKET",
    "UI_BASE_PATH",
    "VITE_BASE_PATH",
  ];

  const env = [];
  for (const key of passthrough) {
    if (process.env[key] !== undefined) {
      env.push(`${key}=${process.env[key]}`);
    }
  }
  return env;
}
