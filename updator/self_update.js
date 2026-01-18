import Docker from "dockerode";

const socketPath = process.env.DOCKER_SOCKET || "/var/run/docker.sock";
const docker = new Docker({ socketPath });

function getEnvBool(name, defaultValue = false) {
  const v = process.env[name];
  if (v == null) return defaultValue;
  return /^(1|true|yes|on)$/i.test(String(v).trim());
}

function env(name, defaultValue) {
  const v = process.env[name];
  return v == null || v === "" ? defaultValue : v;
}

async function sleep(ms) {
  await new Promise((r) => setTimeout(r, ms));
}

async function waitForHealthy(container, timeoutMs) {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const info = await container.inspect();
    const status = info?.State?.Health?.Status;
    if (!status) {
      // No HEALTHCHECK configured; treat "running" as good enough.
      if (info?.State?.Running) return;
    }
    if (status === "healthy") return;
    if (status === "unhealthy") throw new Error("container reported unhealthy");
    await sleep(500);
  }
  throw new Error(`health timeout after ${timeoutMs}ms`);
}

async function pullImage(imageRef) {
  return await new Promise((resolve, reject) => {
    docker.pull(imageRef, (err, stream) => {
      if (err) return reject(err);
      docker.modem.followProgress(stream, (err2, output) => {
        if (err2) return reject(err2);
        resolve(output);
      });
    });
  });
}

async function containerExists(name) {
  try {
    await docker.getContainer(name).inspect();
    return true;
  } catch {
    return false;
  }
}

function safeName(n) {
  return String(n || "").replace(/^\//, "");
}

async function main() {
  const enabled = getEnvBool("YANTRA_AUTO_UPDATE", true);
  if (!enabled) {
    process.stdout.write("YANTRA_AUTO_UPDATE disabled; skipping\n");
    return;
  }

  const containerName = env("YANTRA_CONTAINER_NAME", "yantra");
  const imageRef = env("YANTRA_IMAGE", "ghcr.io/besoeasy/yantra:latest");
  const healthTimeoutMs = Number(env("YANTRA_AUTO_UPDATE_HEALTH_TIMEOUT_MS", "60000"));

  if (!(await containerExists(containerName))) {
    process.stderr.write(`Target container '${containerName}' not found; skipping\n`);
    process.exitCode = 2;
    return;
  }

  const target = docker.getContainer(containerName);
  const currentInfo = await target.inspect();
  const currentImageId = currentInfo?.Image; // sha256:...

  process.stdout.write(`Pulling ${imageRef}...\n`);
  await pullImage(imageRef);

  const newImage = await docker.getImage(imageRef).inspect();
  const newImageId = newImage?.Id;

  if (!newImageId || !currentImageId) {
    process.stderr.write("Unable to determine image IDs; aborting\n");
    process.exitCode = 3;
    return;
  }

  if (newImageId === currentImageId) {
    process.stdout.write("No update available\n");
    return;
  }

  const ts = Date.now();
  const oldName = safeName(currentInfo?.Name) || containerName;
  const backupName = `${oldName}-old-${ts}`;
  const tmpName = `${oldName}-new-${ts}`;

  // Build create options from existing container.
  const createOptions = {
    name: tmpName,
    Image: imageRef,
    Cmd: currentInfo?.Config?.Cmd,
    Entrypoint: currentInfo?.Config?.Entrypoint,
    Env: currentInfo?.Config?.Env,
    WorkingDir: currentInfo?.Config?.WorkingDir,
    User: currentInfo?.Config?.User,
    Labels: currentInfo?.Config?.Labels,
    ExposedPorts: currentInfo?.Config?.ExposedPorts,
    HostConfig: {
      ...currentInfo?.HostConfig,
      // These are always re-derived by Docker and can cause create() to fail if present.
      PortBindings: currentInfo?.HostConfig?.PortBindings,
      Binds: currentInfo?.HostConfig?.Binds,
      Mounts: undefined,
    },
    NetworkingConfig: undefined,
  };

  // Preserve user-defined networks (best-effort).
  const networks = currentInfo?.NetworkSettings?.Networks || {};
  const endpointConfigs = {};
  for (const [netName, net] of Object.entries(networks)) {
    endpointConfigs[netName] = {
      Aliases: net?.Aliases,
    };
  }
  if (Object.keys(endpointConfigs).length > 0) {
    createOptions.NetworkingConfig = { EndpointsConfig: endpointConfigs };
  }

  process.stdout.write(
    `Update available (${currentImageId.slice(0, 19)} → ${newImageId.slice(0, 19)}). Recreating container...\n`
  );

  let tmpContainer;
  try {
    tmpContainer = await docker.createContainer(createOptions);
  } catch (e) {
    process.stderr.write(`Failed to create replacement container: ${e?.message || e}\n`);
    process.exitCode = 4;
    return;
  }

  try {
    process.stdout.write(`Stopping '${oldName}'...\n`);
    try {
      await target.stop({ t: 15 });
    } catch {}

    process.stdout.write(`Renaming '${oldName}' → '${backupName}'...\n`);
    await target.rename({ name: backupName });

    process.stdout.write(`Starting '${tmpName}'...\n`);
    await tmpContainer.start();

    process.stdout.write("Waiting for healthy...\n");
    await waitForHealthy(tmpContainer, Number.isFinite(healthTimeoutMs) ? healthTimeoutMs : 60000);

    process.stdout.write(`Renaming '${tmpName}' → '${oldName}'...\n`);
    await tmpContainer.rename({ name: oldName });

    process.stdout.write(`Removing backup '${backupName}'...\n`);
    await docker.getContainer(backupName).remove({ force: true });

    process.stdout.write("Update applied successfully\n");
  } catch (e) {
    process.stderr.write(`Update failed; attempting rollback: ${e?.message || e}\n`);

    try {
      await tmpContainer.stop({ t: 5 });
    } catch {}
    try {
      await tmpContainer.remove({ force: true });
    } catch {}

    try {
      const backup = docker.getContainer(backupName);
      await backup.inspect();
      await backup.rename({ name: oldName });
      await docker.getContainer(oldName).start();
      process.stderr.write("Rollback complete (old container restored)\n");
    } catch (e2) {
      process.stderr.write(`Rollback may be incomplete: ${e2?.message || e2}\n`);
    }

    process.exitCode = 5;
  }
}

await main();
