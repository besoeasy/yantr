import { docker, packageJson, getPublicIpIdentityCached } from "./shared.js";

const TELEMETRY_TOPIC = process.env.YANTR_TELEMETRY_TOPIC || "https://ntfy.sh/yantr";
const PRESENCE_INTERVAL_MS = 24 * 60 * 60 * 1000;

function isEnabled() {
  return process.env.YANTR_TELEMETRY !== "false";
}

function formatOsName(operatingSystem) {
  if (!operatingSystem || operatingSystem === "unknown") return "unknown";
  return operatingSystem.replace("Debian GNU/Linux", "Debian");
}

function formatRamGb(memTotalBytes) {
  if (!memTotalBytes) return 0;
  return Math.round(memTotalBytes / (1024 ** 3));
}

async function countYantrStacks() {
  const containers = await docker.listContainers({ all: true });
  const projects = new Set();
  for (const container of containers) {
    const labels = container.Labels || {};
    if (labels["yantr.app"] && labels["com.docker.compose.project"]) {
      projects.add(labels["com.docker.compose.project"]);
    }
  }
  return projects.size;
}

export function ping(event, fields = {}) {
  if (!isEnabled()) return;

  const parts = Object.entries(fields)
    .filter(([, value]) => value !== null && value !== undefined && value !== "")
    .map(([key, value]) => `${key}=${value}`);
  const body = [event, ...parts].join(" ");

  fetch(TELEMETRY_TOPIC, {
    method: "POST",
    headers: {
      Title: event,
      Tags: event,
    },
    body,
  }).catch(() => {});
}

export async function sendPresence() {
  if (!isEnabled()) return;

  try {
    const [info, identity, stacks] = await Promise.all([
      docker.info(),
      getPublicIpIdentityCached().catch(() => null),
      countYantrStacks(),
    ]);

    ping("presence", {
      country: identity?.countryCode || "??",
      os: formatOsName(info.OperatingSystem || "unknown"),
      arch: info.Architecture || "unknown",
      cores: info.NCPU || 0,
      ram_gb: formatRamGb(info.MemTotal),
      stacks,
      v: packageJson.version || "dev",
    });
  } catch {
    // fire-and-forget
  }
}

export function trackInstall(appId) {
  ping("install", { app: appId });
}

export async function trackUpdatesForContainers(containerNames = []) {
  if (!isEnabled() || !containerNames.length) return;

  try {
    const wanted = new Set(containerNames.map((name) => String(name).replace(/^\//, "")));
    const containers = await docker.listContainers({ all: true });
    const appIds = new Set();

    for (const container of containers) {
      const name = container.Names?.[0]?.replace(/^\//, "");
      const appId = container.Labels?.["yantr.app"];
      if (name && wanted.has(name) && appId) appIds.add(appId);
    }

    for (const appId of appIds) ping("update", { app: appId });
  } catch {
    // fire-and-forget
  }
}

export function trackSelfUpdate(updatedCount) {
  ping("selfupdate", {
    updated: updatedCount,
    v: packageJson.version || "dev",
  });
}

export function startPresenceScheduler() {
  if (!isEnabled()) return;

  sendPresence();

  const timer = setInterval(() => {
    sendPresence();
  }, PRESENCE_INTERVAL_MS);

  if (timer.unref) timer.unref();
}