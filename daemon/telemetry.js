import { createHash } from "node:crypto";
import { docker, packageJson, getPublicIpIdentityCached } from "./shared.js";

const TELEMETRY_TOPIC = process.env.YANTR_TELEMETRY_TOPIC || "https://ntfy.sh/yantr";
const PRESENCE_INTERVAL_MS = 24 * 60 * 60 * 1000;
const MACHINE_ID_ROUNDS = 10_000;
const MACHINE_ID_SALT = "yantr:machine-id:v1";

function deriveMachineIdFromIp(ip) {
  if (!ip || typeof ip !== "string") return null;

  let digest = createHash("sha256")
    .update(MACHINE_ID_SALT)
    .update("\0")
    .update(ip.trim())
    .digest();

  for (let round = 0; round < MACHINE_ID_ROUNDS; round++) {
    digest = createHash("sha256").update(digest).digest();
  }

  return digest.toString("hex");
}

async function getMachineIdForTelemetry() {
  const identity = await getPublicIpIdentityCached().catch(() => null);
  return deriveMachineIdFromIp(identity?.ip);
}

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

async function countYantrWorkload() {
  const containers = await docker.listContainers({ all: true });
  const projects = new Set();
  let running = 0;

  for (const container of containers) {
    const labels = container.Labels || {};
    if (!labels["yantr.app"]) continue;
    if (container.State === "running") running++;
    if (labels["com.docker.compose.project"]) {
      projects.add(labels["com.docker.compose.project"]);
    }
  }

  return { stacks: projects.size, running };
}

function formatMachineLine(mid) {
  if (!mid) return null;
  return `**Machine ID** \`${mid}\``;
}

function formatTimestampLine() {
  return `_${new Date().toISOString()}_`;
}

function formatPresenceBody(mid, fields) {
  return [
    `**Host** ${fields.os} · ${fields.arch} · ${fields.cores} cores · ${fields.ram_gb} GB RAM`,
    `**Workload** ${fields.stacks} stacks · ${fields.running} running`,
    `**Version** ${fields.v}`,
    formatMachineLine(mid),
    formatTimestampLine(),
  ].filter(Boolean).join("\n");
}

function formatInstallBody(mid, fields) {
  return [
    `**App** \`${fields.app}\``,
    formatMachineLine(mid),
    formatTimestampLine(),
  ].filter(Boolean).join("\n");
}

function formatUpdateBody(mid, fields) {
  return [
    `**App** \`${fields.app}\``,
    formatMachineLine(mid),
    formatTimestampLine(),
  ].filter(Boolean).join("\n");
}

function formatSelfUpdateBody(mid, fields) {
  return [
    `**Containers updated** ${fields.updated}`,
    `**Version** ${fields.v}`,
    formatMachineLine(mid),
    formatTimestampLine(),
  ].filter(Boolean).join("\n");
}

function formatGenericBody(mid, fields) {
  const lines = Object.entries(fields)
    .filter(([, value]) => value !== null && value !== undefined && value !== "")
    .map(([key, value]) => `- **${key}** ${value}`);

  if (mid) lines.push(`- **Machine ID** \`${mid}\``);
  lines.push(`- ${formatTimestampLine()}`);

  return lines.join("\n");
}

const EVENT_FORMATTERS = {
  presence: {
    title: (fields) => `Presence · ${fields.country || "??"}`,
    body: formatPresenceBody,
    priority: "2",
  },
  install: {
    title: (fields) => `Installed · ${fields.app}`,
    body: formatInstallBody,
    priority: "3",
  },
  update: {
    title: (fields) => `Updated · ${fields.app}`,
    body: formatUpdateBody,
    priority: "3",
  },
  selfupdate: {
    title: (fields) => `Self-update · v${fields.v}`,
    body: formatSelfUpdateBody,
    priority: "3",
  },
};

function buildTelemetryMessage(event, mid, fields) {
  const formatter = EVENT_FORMATTERS[event];

  if (formatter) {
    return {
      title: formatter.title(fields),
      body: formatter.body(mid, fields),
      priority: formatter.priority,
    };
  }

  return {
    title: `Yantr · ${event}`,
    body: formatGenericBody(mid, fields),
    priority: "3",
  };
}

export function ping(event, fields = {}) {
  if (!isEnabled()) return;

  void (async () => {
    const mid = await getMachineIdForTelemetry();
    const { title, body, priority } = buildTelemetryMessage(event, mid, fields);

    fetch(TELEMETRY_TOPIC, {
      method: "POST",
      headers: {
        Title: title,
        Priority: priority,
        Markdown: "yes",
      },
      body,
    }).catch(() => {});
  })();
}

export async function sendPresence() {
  if (!isEnabled()) return;

  try {
    const [info, identity, workload] = await Promise.all([
      docker.info(),
      getPublicIpIdentityCached().catch(() => null),
      countYantrWorkload(),
    ]);

    ping("presence", {
      country: identity?.countryCode || "??",
      os: formatOsName(info.OperatingSystem || "unknown"),
      arch: info.Architecture || "unknown",
      cores: info.NCPU || 0,
      ram_gb: formatRamGb(info.MemTotal),
      stacks: workload.stacks,
      running: workload.running,
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