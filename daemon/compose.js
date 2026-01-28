import { spawn } from "child_process";

let composeCommandCache = null;
let composeCommandLogged = false;

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

function getComposeEnv(socketPath) {
  return {
    ...process.env,
    DOCKER_HOST: socketPath ? `unix://${socketPath}` : process.env.DOCKER_HOST,
  };
}

export async function resolveComposeCommand({ socketPath, log } = {}) {
  if (composeCommandCache) {
    return composeCommandCache;
  }

  const env = getComposeEnv(socketPath);

  try {
    const { exitCode } = await spawnProcess("docker", ["compose", "version"], { env });
    if (exitCode === 0) {
      composeCommandCache = { command: "docker", args: ["compose"], display: "docker compose" };
      if (log && !composeCommandLogged) {
        composeCommandLogged = true;
        log("info", `🧩 [COMPOSE] Using ${composeCommandCache.display}`);
      }
      return composeCommandCache;
    }
  } catch (err) {
    // ignore and try docker-compose
  }

  try {
    const { exitCode } = await spawnProcess("docker-compose", ["version"], { env });
    if (exitCode === 0) {
      composeCommandCache = { command: "docker-compose", args: [], display: "docker-compose" };
      if (log && !composeCommandLogged) {
        composeCommandLogged = true;
        log("info", `🧩 [COMPOSE] Using ${composeCommandCache.display}`);
      }
      return composeCommandCache;
    }
  } catch (err) {
    // ignore and fail below
  }

  throw new Error("docker compose is not available (docker compose or docker-compose not found)");
}
