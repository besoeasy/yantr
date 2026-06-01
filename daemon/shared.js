/**
 * Shared state, helpers, and caches used across all route modules.
 * Import from here – do NOT import directly from main.js.
 */
import Docker from "dockerode";
import path from "path";
import { readFile, access, readdir } from "fs/promises";
import { fileURLToPath } from "url";
import { dirname } from "path";
import { getBaseAppId } from "./utils.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// ── Docker ─────────────────────────────────────────────────────────────────────
export const socketPath = process.env.DOCKER_SOCKET || "/var/run/docker.sock";
export const docker = new Docker({ socketPath });

// ── package.json ───────────────────────────────────────────────────────────────
export const packageJson = JSON.parse(
  await readFile(new URL("../package.json", import.meta.url), "utf-8")
);

// ── Directory helpers ──────────────────────────────────────────────────────────
export const daemonDir = __dirname;
export const appsDir = path.join(__dirname, "..", "apps");

// ── Logger (circular buffer) ───────────────────────────────────────────────────
export const MAX_LOGS = 1000;
export const logs = [];

export function log(level, message, ...args) {
  const timestamp = new Date().toISOString();
  const logEntry = { timestamp, level, message, args: args.length > 0 ? args : undefined };
  logs.push(logEntry);
  if (logs.length > MAX_LOGS) logs.shift();

  const formattedMessage = args.length > 0 ? `${message} ${args.join(" ")}` : message;
  if (level === "error") {
    console.error(formattedMessage);
  } else {
    console.log(formattedMessage);
  }
}

// ── Time helper ────────────────────────────────────────────────────────────────
export function nowMs() {
  return Date.now();
}

function isLikelyIpfsCid(value) {
  return /^Qm[1-9A-HJ-NP-Za-km-z]{44}$/.test(value) || /^b[a-z2-7]{20,}$/.test(value);
}

function normalizeAppLogo(logoRaw) {
  if (typeof logoRaw !== "string") return null;
  const trimmed = logoRaw.trim();
  if (!trimmed) return null;
  if (trimmed.includes("://")) return trimmed;
  return isLikelyIpfsCid(trimmed) ? `https://ipfs.io/ipfs/${trimmed}` : null;
}

// ── Concurrency helper ─────────────────────────────────────────────────────────
export async function mapWithConcurrency(items, limit, mapper) {
  const results = new Array(items.length);
  let nextIndex = 0;
  const workers = Array.from({ length: Math.max(1, limit) }, async () => {
    while (true) {
      const index = nextIndex++;
      if (index >= items.length) return;
      results[index] = await mapper(items[index], index);
    }
  });
  await Promise.all(workers);
  return results;
}

// ── Docker label helpers ───────────────────────────────────────────────────────
export function parseAppLabels(labels) {
  const appLabels = {};
  if (!labels || typeof labels !== "object") return appLabels;
  if (labels["yantr.app"])          appLabels.app         = labels["yantr.app"];
  if (labels["yantr.service"])      appLabels.service     = labels["yantr.service"];
  if (labels["yantr.caddy.master"]) appLabels.caddyMaster = labels["yantr.caddy.master"];
  return appLabels;
}

// ── Container env cache ────────────────────────────────────────────────────────
const CONTAINER_ENV_CACHE_TTL_MS = 60_000;
const containerEnvCache = new Map();

export async function getContainerEnv(containerId) {
  const cached = containerEnvCache.get(containerId);
  if (cached && cached.expiresAt > nowMs()) return cached.env;
  try {
    const containerObj = docker.getContainer(containerId);
    const info = await containerObj.inspect();
    const envVars = info?.Config?.Env || [];
    containerEnvCache.set(containerId, { env: envVars, expiresAt: nowMs() + CONTAINER_ENV_CACHE_TTL_MS });
    return envVars;
  } catch (error) {
    containerEnvCache.set(containerId, { env: [], expiresAt: nowMs() + 10_000 });
    throw error;
  }
}

// ── Apps catalog cache ─────────────────────────────────────────────────────────
export const APPS_CACHE_TTL_MS = 60_000;
let appsCatalogCache = { value: null, expiresAt: 0, inFlight: null };

export async function getAppsCatalogCached({ forceRefresh } = { forceRefresh: false }) {
  const cacheIsValid = appsCatalogCache.value && appsCatalogCache.expiresAt > nowMs();
  if (!forceRefresh && cacheIsValid) return appsCatalogCache.value;
  if (!forceRefresh && appsCatalogCache.inFlight) return await appsCatalogCache.inFlight;

  const loadPromise = (async () => {
    const apps = [];
    const entries = await readdir(appsDir, { withFileTypes: true });

    for (const entry of entries) {
      if (!entry.isDirectory()) continue;
      const appPath = path.join(appsDir, entry.name);
      const composePath = path.join(appPath, "compose.yml");
      const infoPath = path.join(appPath, "info.json");

      try {
        await access(composePath);
        await access(infoPath);

        const infoRaw = await readFile(infoPath, "utf-8");
        const info = JSON.parse(infoRaw);
        if (!info.name) continue;

        const composeContent = await readFile(composePath, "utf-8");
        let match;

        const envVars = [];
        const envRegex = /-\s+([A-Za-z_][A-Za-z0-9_]*)=\$\{([A-Za-z_][A-Za-z0-9_]*):?-?([^}]*)\}/g;
        while ((match = envRegex.exec(composeContent)) !== null) {
          envVars.push({ name: match[1], envVar: match[2], default: match[3] || "" });
        }
        const envKeyValueRegex = /^\s+([A-Za-z_][A-Za-z0-9_]*):\s*\$\{([A-Za-z_][A-Za-z0-9_]*):?-?([^}]*)\}/gm;
        while ((match = envKeyValueRegex.exec(composeContent)) !== null) {
          if (!envVars.find(v => v.envVar === match[2])) {
            envVars.push({ name: match[1], envVar: match[2], default: match[3] || "" });
          }
        }

        const portMappings = [];
        const fixedPortRegex = /-\s*["']?(\d+):(\d+)(?:\/(tcp|udp))?["']?/g;
        while ((match = fixedPortRegex.exec(composeContent)) !== null) {
          portMappings.push({ hostPort: match[1], containerPort: match[2], protocol: match[3] || "tcp", envVar: null });
        }
        const autoPortRegex = /-\s*["'](\d+)["'](?:\s|$)/g;
        while ((match = autoPortRegex.exec(composeContent)) !== null) {
          const port = match[1];
          if (!portMappings.some(p => p.containerPort === port)) {
            portMappings.push({ hostPort: port, containerPort: port, protocol: "tcp", envVar: null });
          }
        }

        const logoRaw = info.logo || null;
        const logoUrl = normalizeAppLogo(logoRaw);

        const envGeneratorsRaw =
          info.env_generators && typeof info.env_generators === "object"
            ? info.env_generators
            : info.envGenerators && typeof info.envGenerators === "object"
              ? info.envGenerators
              : {};

        const envGenerators = {};
        for (const [envVar, rule] of Object.entries(envGeneratorsRaw)) {
          if (!envVar || typeof envVar !== "string") continue;
          if (!rule || typeof rule !== "object" || Array.isArray(rule)) continue;
          envGenerators[envVar] = { ...rule };
        }

        apps.push({
          id: entry.name,
          name: info.name,
          logo: logoUrl,
          tags: Array.isArray(info.tags) ? info.tags : [],
          ports: Array.isArray(info.ports) ? info.ports : [],
          short_description: info.short_description || "",
          description: info.description || info.short_description || "",
          usecases: Array.isArray(info.usecases) ? info.usecases : [],
          website: info.website || null,
          customapp: !!info.customapp,
          path: appPath,
          composePath,
          environment: envVars,
          envGenerators,
          composePorts: portMappings,
        });
      } catch {
        // skip unreadable app
      }
    }
    return { apps, count: apps.length };
  })();

  if (!forceRefresh) appsCatalogCache.inFlight = loadPromise;
  try {
    const value = await loadPromise;
    appsCatalogCache.value = value;
    appsCatalogCache.expiresAt = nowMs() + APPS_CACHE_TTL_MS;
    return value;
  } finally {
    if (appsCatalogCache.inFlight === loadPromise) appsCatalogCache.inFlight = null;
  }
}

// ── System architecture helpers ────────────────────────────────────────────────
let systemArchitecture = null;
const IMAGE_ARCH_CACHE_TTL_MS = 60 * 60_000;
const imageArchCache = new Map();

import { spawnProcess } from "./utils.js";

export async function getSystemArchitecture() {
  if (systemArchitecture) return systemArchitecture;
  try {
    const { stdout, stderr, exitCode } = await spawnProcess("uname", ["-m"]);
    if (exitCode !== 0) throw new Error(stderr || `uname exited with code ${exitCode}`);
    const arch = stdout.trim();
    const archMap = { x86_64: "amd64", aarch64: "arm64", armv7l: "arm/v7", armv6l: "arm/v6", i386: "386", i686: "386" };
    systemArchitecture = archMap[arch] || arch;
    log("info", `🏗️  Detected system architecture: ${arch} (${systemArchitecture})`);
    return systemArchitecture;
  } catch (error) {
    log("error", "❌ Failed to detect system architecture:", error.message);
    systemArchitecture = "amd64";
    return systemArchitecture;
  }
}

export async function checkImageArchitectureSupport(imageName) {
  try {
    const cached = imageArchCache.get(imageName);
    if (cached && cached.expiresAt > nowMs()) return cached.value;

    const arch = await getSystemArchitecture();
    log("info", `🔍 Checking architecture support for ${imageName} on ${arch}`);

    const command = `docker image inspect ${imageName} --format='{{.Architecture}}' 2>/dev/null || docker manifest inspect ${imageName} 2>/dev/null`;
    try {
      const { stdout, stderr, exitCode } = await spawnProcess("sh", ["-c", command]);
      if (exitCode !== 0) throw new Error(stderr || `shell command exited with code ${exitCode}`);
      const output = stdout.trim();

      if (output && !output.includes("{") && !output.includes("[")) {
        const imageArch = output.toLowerCase();
        const supported = imageArch === arch || (imageArch === "amd64" && arch === "amd64") || (imageArch === "arm64" && arch === "arm64");
        const value = { supported, imageArch, systemArch: arch };
        imageArchCache.set(imageName, { value, expiresAt: nowMs() + IMAGE_ARCH_CACHE_TTL_MS });
        return value;
      }

      if (output.includes("{")) {
        try {
          const manifest = JSON.parse(output);
          const supportedArchs = (manifest.manifests || []).map(m => m.platform?.architecture).filter(Boolean);
          const isSupported = supportedArchs.some(a => a === arch || (a === "amd64" && arch === "amd64") || (a === "arm64" && arch === "arm64"));
          const value = { supported: isSupported, imageArch: supportedArchs.join(", "), systemArch: arch };
          imageArchCache.set(imageName, { value, expiresAt: nowMs() + IMAGE_ARCH_CACHE_TTL_MS });
          return value;
        } catch {}
      }
    } catch {}

    const value = { supported: "unknown", imageArch: "unknown", systemArch: arch };
    imageArchCache.set(imageName, { value, expiresAt: nowMs() + 10 * 60_000 });
    return value;
  } catch (error) {
    log("error", "❌ Error checking architecture support:", error.message);
    const arch = systemArchitecture || "amd64";
    return { supported: "unknown", imageArch: "unknown", systemArch: arch };
  }
}

export async function getImageFromCompose(composePath) {
  try {
    const content = await readFile(composePath, "utf-8");
    const imageMatch = content.match(/image:\s*([^\s\n]+)/);
    return imageMatch ? imageMatch[1] : null;
  } catch (error) {
    log("error", "❌ Error reading compose file:", error.message);
    return null;
  }
}

// ── Public IP cache ────────────────────────────────────────────────────────────
const IP_IDENTITY_CACHE_TTL_MS = 5 * 60_000;
let ipIdentityCache = { value: null, expiresAt: 0, inFlight: null };

export async function fetchJsonWithTimeout(url, timeoutMs = 6000) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const res = await fetch(url, { headers: { "user-agent": "yantr-daemon" }, signal: controller.signal });
    if (!res.ok) throw new Error(`Request failed: ${res.status} ${res.statusText}`);
    return await res.json();
  } finally {
    clearTimeout(timer);
  }
}

export async function getPublicIpIdentityCached({ forceRefresh } = { forceRefresh: false }) {
  const cacheIsValid = ipIdentityCache.value && ipIdentityCache.expiresAt > nowMs();
  if (!forceRefresh && cacheIsValid) return ipIdentityCache.value;
  if (!forceRefresh && ipIdentityCache.inFlight) return await ipIdentityCache.inFlight;

  const loadPromise = (async () => {
    const fetchedAt = new Date().toISOString();

    const normalize = (source, raw) => {
      if (!raw || typeof raw !== "object") return null;
      if (source === "ipwho.is") {
        if (raw.success === false) return null;
        const connection = raw.connection || {};
        return { ip: raw.ip || null, city: raw.city || null, region: raw.region || null, country: raw.country || null, countryCode: raw.country_code || null, isp: connection.isp || null, org: connection.org || null, asn: connection.asn || null, timezone: raw.timezone?.id || null, latitude: typeof raw.latitude === "number" ? raw.latitude : null, longitude: typeof raw.longitude === "number" ? raw.longitude : null };
      }
      if (source === "ipapi.co") {
        if (raw.error) return null;
        return { ip: raw.ip || null, city: raw.city || null, region: raw.region || null, country: raw.country_name || null, countryCode: raw.country_code || null, isp: raw.org || raw.isp || null, org: raw.org || null, asn: raw.asn || null, timezone: raw.timezone || null, latitude: typeof raw.latitude === "number" ? raw.latitude : null, longitude: typeof raw.longitude === "number" ? raw.longitude : null };
      }
      if (source === "ipinfo.io") {
        if (raw.error) return null;
        const loc = typeof raw.loc === "string" ? raw.loc.split(",") : [];
        const lat = loc.length === 2 ? Number(loc[0]) : null;
        const lon = loc.length === 2 ? Number(loc[1]) : null;
        return { ip: raw.ip || null, city: raw.city || null, region: raw.region || null, country: raw.country || null, countryCode: raw.country || null, isp: raw.org || null, org: raw.org || null, asn: null, timezone: raw.timezone || null, latitude: Number.isFinite(lat) ? lat : null, longitude: Number.isFinite(lon) ? lon : null };
      }
      if (source === "ifconfig.co") {
        return { ip: raw.ip || null, city: raw.city || null, region: raw.region_name || raw.region || null, country: raw.country || null, countryCode: raw.country_iso || raw.country_code || null, isp: raw.asn_org || raw.organization || null, org: raw.asn_org || raw.organization || null, asn: raw.asn || null, timezone: raw.time_zone || raw.timezone || null, latitude: typeof raw.latitude === "number" ? raw.latitude : null, longitude: typeof raw.longitude === "number" ? raw.longitude : null };
      }
      return null;
    };

    const providers = [
      { source: "ipwho.is", url: "https://ipwho.is/" },
      { source: "ipapi.co", url: "https://ipapi.co/json/" },
      { source: "ipinfo.io", url: "https://ipinfo.io/json" },
      { source: "ifconfig.co", url: "https://ifconfig.co/json" },
    ];

    const errors = [];
    for (const provider of providers) {
      try {
        const raw = await fetchJsonWithTimeout(provider.url, 6000);
        const normalized = normalize(provider.source, raw);
        if (normalized?.ip) {
          return { ...normalized, source: provider.source, fetchedAt, cacheTtlMs: IP_IDENTITY_CACHE_TTL_MS };
        }
        errors.push(`${provider.source}: invalid response`);
      } catch (e) {
        errors.push(`${provider.source}: ${e?.message || String(e)}`);
      }
    }
    throw new Error(`Failed to resolve public IP (${errors.join("; ")})`);
  })();

  if (!forceRefresh) ipIdentityCache.inFlight = loadPromise;
  try {
    const value = await loadPromise;
    ipIdentityCache.value = value;
    ipIdentityCache.expiresAt = nowMs() + IP_IDENTITY_CACHE_TTL_MS;
    return value;
  } finally {
    if (ipIdentityCache.inFlight === loadPromise) ipIdentityCache.inFlight = null;
  }
}
