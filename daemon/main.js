import express from "express";
import Docker from "dockerode";
import cors from "cors";
import path from "path";
import { readFile, writeFile, unlink } from "fs/promises";
import { resolveComposeCommand } from "./compose.js";
import { fileURLToPath } from "url";
import { dirname } from "path";
import { 
  spawnProcess,
  errorHandler,
  asyncHandler,
  NotFoundError,
  BadRequestError,
  ConflictError,
  DockerError 
} from "./utils.js";

import { startCleanupScheduler } from "./cleanup.js";
import {
  getS3Config,
  saveS3Config,
  listBackups,
  createBackup,
  restoreBackup,
  deleteBackup,
  getBackupJobStatus,
  getRestoreJobStatus,
  getAllBackupJobs,
  getAllRestoreJobs,
} from "./backup.js";

// Import package.json
const packageJson = JSON.parse(await readFile(new URL("../package.json", import.meta.url), "utf-8"));

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const app = express();

// Docker socket path
const socketPath = process.env.DOCKER_SOCKET || "/var/run/docker.sock";

const docker = new Docker({ socketPath });

// Compose resolver moved to compose.js

// System architecture cache
let systemArchitecture = null;

// Container env cache (avoid N+1 inspect on polling-heavy endpoints)
const CONTAINER_ENV_CACHE_TTL_MS = 60_000;
const containerEnvCache = new Map(); // id -> { env: string[], expiresAt: number }

// Image architecture support cache
const IMAGE_ARCH_CACHE_TTL_MS = 60 * 60_000;
const imageArchCache = new Map(); // imageName -> { value: any, expiresAt: number }

// Apps catalog cache (reading/parsing many compose.yml files can be expensive)
const APPS_CACHE_TTL_MS = 60_000;
let appsCatalogCache = {
  value: null,
  expiresAt: 0,
  inFlight: null,
};

// Public IP/geo cache (avoid hammering external services)
const IP_IDENTITY_CACHE_TTL_MS = 5 * 60_000;
let ipIdentityCache = {
  value: null,
  expiresAt: 0,
  inFlight: null,
};

async function fetchJsonWithTimeout(url, timeoutMs = 6000) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const res = await fetch(url, {
      headers: { "user-agent": "yantra-daemon" },
      signal: controller.signal,
    });
    if (!res.ok) {
      throw new Error(`Request failed: ${res.status} ${res.statusText}`);
    }
    return await res.json();
  } finally {
    clearTimeout(timer);
  }
}

async function getPublicIpIdentityCached({ forceRefresh } = { forceRefresh: false }) {
  const cacheIsValid = ipIdentityCache.value && ipIdentityCache.expiresAt > nowMs();
  if (!forceRefresh && cacheIsValid) {
    return ipIdentityCache.value;
  }

  if (!forceRefresh && ipIdentityCache.inFlight) {
    return await ipIdentityCache.inFlight;
  }

  const loadPromise = (async () => {
    const fetchedAt = new Date().toISOString();

    const normalize = (source, raw) => {
      if (!raw || typeof raw !== "object") return null;

      // Provider: ipwho.is
      if (source === "ipwho.is") {
        if (raw.success === false) return null;
        const connection = raw.connection || {};
        return {
          ip: raw.ip || null,
          city: raw.city || null,
          region: raw.region || null,
          country: raw.country || null,
          countryCode: raw.country_code || null,
          isp: connection.isp || null,
          org: connection.org || null,
          asn: connection.asn || null,
          timezone: raw.timezone?.id || null,
          latitude: typeof raw.latitude === "number" ? raw.latitude : null,
          longitude: typeof raw.longitude === "number" ? raw.longitude : null,
        };
      }

      // Provider: ipapi.co/json
      if (source === "ipapi.co") {
        // ipapi.co can return { error: true, reason: '...' }
        if (raw.error) return null;
        return {
          ip: raw.ip || null,
          city: raw.city || null,
          region: raw.region || null,
          country: raw.country_name || null,
          countryCode: raw.country_code || null,
          isp: raw.org || raw.isp || null,
          org: raw.org || null,
          asn: raw.asn || null,
          timezone: raw.timezone || null,
          latitude: typeof raw.latitude === "number" ? raw.latitude : null,
          longitude: typeof raw.longitude === "number" ? raw.longitude : null,
        };
      }

      // Provider: ipinfo.io/json (no key required but rate-limited)
      if (source === "ipinfo.io") {
        // ipinfo can return { error: { title, message } }
        if (raw.error) return null;
        const loc = typeof raw.loc === "string" ? raw.loc.split(",") : [];
        const lat = loc.length === 2 ? Number(loc[0]) : null;
        const lon = loc.length === 2 ? Number(loc[1]) : null;
        return {
          ip: raw.ip || null,
          city: raw.city || null,
          region: raw.region || null,
          country: raw.country || null,
          countryCode: raw.country || null,
          isp: raw.org || null,
          org: raw.org || null,
          asn: null,
          timezone: raw.timezone || null,
          latitude: Number.isFinite(lat) ? lat : null,
          longitude: Number.isFinite(lon) ? lon : null,
        };
      }

      // Provider: ifconfig.co/json
      if (source === "ifconfig.co") {
        // ifconfig.co may include fields like: ip, city, region_name, country, country_iso
        return {
          ip: raw.ip || null,
          city: raw.city || null,
          region: raw.region_name || raw.region || null,
          country: raw.country || null,
          countryCode: raw.country_iso || raw.country_code || null,
          isp: raw.asn_org || raw.organization || null,
          org: raw.asn_org || raw.organization || null,
          asn: raw.asn || null,
          timezone: raw.time_zone || raw.timezone || null,
          latitude: typeof raw.latitude === "number" ? raw.latitude : null,
          longitude: typeof raw.longitude === "number" ? raw.longitude : null,
        };
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
          return {
            ...normalized,
            source: provider.source,
            fetchedAt,
            cacheTtlMs: IP_IDENTITY_CACHE_TTL_MS,
          };
        }
        errors.push(`${provider.source}: invalid response`);
      } catch (e) {
        errors.push(`${provider.source}: ${e?.message || String(e)}`);
      }
    }

    throw new Error(`Failed to resolve public IP (${errors.join("; ")})`);
  })();

  if (!forceRefresh) {
    ipIdentityCache.inFlight = loadPromise;
  }

  try {
    const value = await loadPromise;
    ipIdentityCache.value = value;
    ipIdentityCache.expiresAt = nowMs() + IP_IDENTITY_CACHE_TTL_MS;
    return value;
  } finally {
    if (ipIdentityCache.inFlight === loadPromise) {
      ipIdentityCache.inFlight = null;
    }
  }
}

function nowMs() {
  return Date.now();
}

async function getAppsCatalogCached({ forceRefresh } = { forceRefresh: false }) {
  const cacheIsValid = appsCatalogCache.value && appsCatalogCache.expiresAt > nowMs();
  if (!forceRefresh && cacheIsValid) {
    return appsCatalogCache.value;
  }

  if (!forceRefresh && appsCatalogCache.inFlight) {
    return await appsCatalogCache.inFlight;
  }

  const loadPromise = (async () => {
    const appsDir = path.join(__dirname, "..", "apps");
    const apps = [];

    const fs = await import("fs/promises");
    const entries = await fs.readdir(appsDir, { withFileTypes: true });

    for (const entry of entries) {
      if (!entry.isDirectory()) continue;

      const appPath = path.join(appsDir, entry.name);
      const composePath = path.join(appPath, "compose.yml");

      try {
        // Check if compose.yml exists using fs
        const fs = await import("fs/promises");
        try {
          await fs.access(composePath);
        } catch {
          continue; // File doesn't exist, skip
        }

        const composeContent = await readFile(composePath, "utf-8");
        const labels = {};

        // Format 1: yantra.name: "value"
        const labelRegex = /yantra\.([\w-]+):\s*["'](.+?)["']/g;
        let match;
        while ((match = labelRegex.exec(composeContent)) !== null) {
          labels[match[1]] = match[2];
        }

        // Format 2: - "yantra.name=value"
        const arrayLabelRegex = /-\s*["']yantra\.([\w-]+)=(.+?)["']/g;
        while ((match = arrayLabelRegex.exec(composeContent)) !== null) {
          labels[match[1]] = match[2];
        }

        // Extract environment variables
        const envVars = [];
        const envRegex = /-\s+([A-Z_]+)=\$\{([A-Z_]+):?-?([^}]*)\}/g;
        while ((match = envRegex.exec(composeContent)) !== null) {
          envVars.push({
            name: match[1],
            envVar: match[2],
            default: match[3] || "",
          });
        }

        // Extract port mappings
        const portMappings = [];

        // Match fixed port mappings: - "8080:80", "53:53/tcp", "53:53/udp"
        const fixedPortRegex = /-\s*["']?(\d+):(\d+)(?:\/(tcp|udp))?["']?/g;
        while ((match = fixedPortRegex.exec(composeContent)) !== null) {
          portMappings.push({
            hostPort: match[1],
            containerPort: match[2],
            protocol: match[3] || "tcp",
            envVar: null,
          });
        }

        // Match auto-assigned ports: - "9091", - "8080"
        const autoPortRegex = /-\s*["'](\d+)["'](?:\s|$)/g;
        while ((match = autoPortRegex.exec(composeContent)) !== null) {
          const port = match[1];
          if (!portMappings.some((p) => p.containerPort === port)) {
            portMappings.push({
              hostPort: port,
              containerPort: port,
              protocol: "tcp",
              envVar: null,
            });
          }
        }

        // Parse yantra.port to identify named ports
        const namedPorts = new Set();
        if (labels.port) {
          const portDescRegex = /(\d+)\s*\(([^-\)]+)\s*-\s*([^)]+)\)/g;
          let portMatch;
          while ((portMatch = portDescRegex.exec(labels.port)) !== null) {
            namedPorts.add(portMatch[1]);
          }
        }

        portMappings.forEach((port) => {
          port.isNamed = namedPorts.has(port.hostPort);
        });

        apps.push({
          id: entry.name,
          name: labels.name || entry.name,
          logo: labels.logo
            ? labels.logo.includes("://")
              ? labels.logo
              : `https://ipfs.io/ipfs/${labels.logo}`
            : "https://ipfs.io/ipfs/QmVdbRUyvZpXCsVJAs7fo1PJPXaPHnWRtSCFx6jFTGaG5i",
          category: labels.category || "uncategorized",
          port: labels.port || null,
          description: labels.description || "",

          website: labels.website || null,
          docs: labels.docs || null,
          path: appPath,
          composePath: composePath,
          environment: envVars,
          ports: portMappings,
        });
      } catch (err) {
        // Skip apps with missing/unreadable compose.yml
      }
    }

    return { apps, count: apps.length };
  })();

  if (!forceRefresh) {
    appsCatalogCache.inFlight = loadPromise;
  }

  try {
    const value = await loadPromise;
    appsCatalogCache.value = value;
    appsCatalogCache.expiresAt = nowMs() + APPS_CACHE_TTL_MS;
    return value;
  } finally {
    if (appsCatalogCache.inFlight === loadPromise) {
      appsCatalogCache.inFlight = null;
    }
  }
}

async function mapWithConcurrency(items, limit, mapper) {
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

async function getContainerEnv(containerId) {
  const cached = containerEnvCache.get(containerId);
  if (cached && cached.expiresAt > nowMs()) {
    return cached.env;
  }

  try {
    const containerObj = docker.getContainer(containerId);
    const info = await containerObj.inspect();
    const envVars = info?.Config?.Env || [];
    containerEnvCache.set(containerId, { env: envVars, expiresAt: nowMs() + CONTAINER_ENV_CACHE_TTL_MS });
    return envVars;
  } catch (error) {
    // Negative-cache briefly to avoid hammering Docker on repeated failures.
    containerEnvCache.set(containerId, { env: [], expiresAt: nowMs() + 10_000 });
    throw error;
  }
}

// Logs storage - circular buffer for last 1000 logs
const MAX_LOGS = 1000;
const logs = [];

// Logger utility function
function log(level, message, ...args) {
  const timestamp = new Date().toISOString();
  const logEntry = {
    timestamp,
    level,
    message,
    args: args.length > 0 ? args : undefined,
  };

  // Add to logs array (circular buffer)
  logs.push(logEntry);
  if (logs.length > MAX_LOGS) {
    logs.shift(); // Remove oldest log
  }

  // Also log to console
  const formattedMessage = args.length > 0 ? `${message} ${args.join(" ")}` : message;
  if (level === "error") {
    console.error(formattedMessage);
  } else {
    console.log(formattedMessage);
  }
}

log("info", "� Using Docker socket:", socketPath);


// Helper function to get system architecture
async function getSystemArchitecture() {
  if (systemArchitecture) {
    return systemArchitecture;
  }

  try {
    const { stdout, stderr, exitCode } = await spawnProcess("uname", ["-m"]);
    if (exitCode !== 0) {
      throw new Error(stderr || `uname exited with code ${exitCode}`);
    }
    const arch = stdout.trim();

    // Map common architecture names to Docker platform names
    const archMap = {
      x86_64: "amd64",
      aarch64: "arm64",
      armv7l: "arm/v7",
      armv6l: "arm/v6",
      i386: "386",
      i686: "386",
    };

    systemArchitecture = archMap[arch] || arch;
    log("info", `🏗️  Detected system architecture: ${arch} (${systemArchitecture})`);
    return systemArchitecture;
  } catch (error) {
    log("error", "❌ Failed to detect system architecture:", error.message);
    // Default to amd64 if detection fails
    systemArchitecture = "amd64";
    return systemArchitecture;
  }
}

// Helper function to check if image supports current architecture
async function checkImageArchitectureSupport(imageName) {
  try {
    const cached = imageArchCache.get(imageName);
    if (cached && cached.expiresAt > nowMs()) {
      return cached.value;
    }

    const arch = await getSystemArchitecture();
    log("info", `🔍 Checking architecture support for ${imageName} on ${arch}`);

    // Try to inspect the image manifest to check supported platforms
    // First, try to pull manifest without actually pulling the image
    const command = `docker image inspect ${imageName} --format='{{.Architecture}}' 2>/dev/null || docker manifest inspect ${imageName} 2>/dev/null`;

    try {
      const { stdout, stderr, exitCode } = await spawnProcess("sh", ["-c", command]);
      if (exitCode !== 0) {
        throw new Error(stderr || `shell command exited with code ${exitCode}`);
      }
      const output = stdout.trim();

      // If we get architecture directly
      if (output && !output.includes("{") && !output.includes("[")) {
        const imageArch = output.toLowerCase();
        log("info", `  Image architecture: ${imageArch}`);

        // Check if architectures match
        if (imageArch === arch || (imageArch === "amd64" && arch === "amd64") || (imageArch === "arm64" && arch === "arm64")) {
          const value = { supported: true, imageArch, systemArch: arch };
          imageArchCache.set(imageName, { value, expiresAt: nowMs() + IMAGE_ARCH_CACHE_TTL_MS });
          return value;
        } else {
          const value = { supported: false, imageArch, systemArch: arch };
          imageArchCache.set(imageName, { value, expiresAt: nowMs() + IMAGE_ARCH_CACHE_TTL_MS });
          return value;
        }
      }

      // If we get a manifest (JSON), parse it for supported platforms
      if (output.includes("{")) {
        try {
          const manifest = JSON.parse(output);
          const manifests = manifest.manifests || [];

          const supportedArchs = manifests.map((m) => m.platform?.architecture).filter((a) => a);

          log("info", `  Supported architectures: ${supportedArchs.join(", ")}`);

          const isSupported = supportedArchs.some((a) => a === arch || (a === "amd64" && arch === "amd64") || (a === "arm64" && arch === "arm64"));

          const value = {
            supported: isSupported,
            imageArch: supportedArchs.join(", "),
            systemArch: arch,
          };
          imageArchCache.set(imageName, { value, expiresAt: nowMs() + IMAGE_ARCH_CACHE_TTL_MS });
          return value;
        } catch (parseError) {
          log("error", "  Failed to parse manifest:", parseError.message);
        }
      }
    } catch (inspectError) {
      // If inspection fails, the image might not be available locally or remotely
      log("info", `  Could not inspect image ${imageName}, will attempt pull`);
    }

    // If we can't determine from manifest, we'll try to pull and see if it fails
    const value = { supported: "unknown", imageArch: "unknown", systemArch: arch };
    imageArchCache.set(imageName, { value, expiresAt: nowMs() + 10 * 60_000 });
    return value;
  } catch (error) {
    log("error", "❌ Error checking architecture support:", error.message);
    return { supported: "unknown", imageArch: "unknown", systemArch: arch };
  }
}

// Helper function to extract image name from compose file
async function getImageFromCompose(composePath) {
  try {
    const content = await readFile(composePath, "utf-8");
    const imageMatch = content.match(/image:\s*([^\s\n]+)/);
    if (imageMatch) {
      return imageMatch[1];
    }
    return null;
  } catch (error) {
    log("error", "❌ Error reading compose file:", error.message);
    return null;
  }
}

// Middleware
app.use(cors());
app.use(express.json());

// Serve static files from Vue.js dist folder in production
if (process.env.NODE_ENV === "production") {
  const distPath = path.join(__dirname, "..", "dist");
  app.use(express.static(distPath));

  log("info", `📦 Serving Vue.js app from: ${distPath}`);
}

// Helper function to parse app labels
function parseAppLabels(labels) {
  const appLabels = {};

  // Safety check for labels
  if (!labels || typeof labels !== "object") {
    return appLabels;
  }

  for (const [key, value] of Object.entries(labels)) {
    if (key.startsWith("yantra.")) {
      const labelName = key.replace("yantra.", "");
      appLabels[labelName] = value;
    }
  }

  return appLabels;
}

// GET /api/version - Get app version
app.get("/api/version", (req, res) => {
  res.json({
    success: true,
    version: packageJson.version,
  });
});

// GET /api/health - Health check endpoint
app.get("/api/health", (req, res) => {
  res.json({
    success: true,
    status: "ok",
    timestamp: new Date().toISOString(),
    version: packageJson.version,
  });
});

// GET /api/network/identity - Public IP + geo/ISP resolved by daemon (machine IP)
app.get("/api/network/identity", asyncHandler(async (req, res) => {
  const force = String(req.query.force || "").toLowerCase() === "true";
  const identity = await getPublicIpIdentityCached({ forceRefresh: force });
  res.json({
    success: true,
    identity,
  });
}));

// GET /api/logs - Get stored logs
app.get("/api/logs", (req, res) => {
  const limit = parseInt(req.query.limit) || MAX_LOGS;
  const level = req.query.level; // Optional filter by level

  let filteredLogs = logs;
  if (level) {
    filteredLogs = logs.filter((log) => log.level === level);
  }

  res.json({
    success: true,
    count: filteredLogs.length,
    maxLogs: MAX_LOGS,
    logs: filteredLogs.slice(-limit).reverse(), // Most recent first
  });
});

// GET /api/containers - List all containers with their labels
app.get("/api/containers", asyncHandler(async (req, res) => {
  const containers = await docker.listContainers({ all: true });

    const formattedContainers = await mapWithConcurrency(containers, 8, async (container) => {
      const appLabels = parseAppLabels(container.Labels);
      const composeProject = container.Labels["com.docker.compose.project"];

      // Fetch full container details to get environment variables
      let envVars = [];
      try {
        envVars = await getContainerEnv(container.Id);
      } catch (error) {
        log("error", `Failed to get env for container ${container.Id}:`, error.message);
      }

      // Extract base app ID from compose project (remove -2, -3 suffix for instances)
      let baseAppId = composeProject || container.Names[0]?.replace("/", "") || "unknown";
      if (composeProject && composeProject.match(/-\d+$/)) {
        baseAppId = composeProject.replace(/-\d+$/, '');
      }

      return {
        id: container.Id,
        name: container.Names[0]?.replace("/", "") || "unknown",
        image: container.Image,
        imageId: container.ImageID,
        state: container.State,
        status: container.Status,
        created: container.Created,
        ports: container.Ports,
        labels: container.Labels, // Keep original labels for filtering
        appLabels: appLabels,
        env: envVars,
        // Add computed fields for easier UI access
        app: {
          id: baseAppId, // Base app ID without instance suffix
          projectId: composeProject || container.Names[0]?.replace("/", "") || "unknown", // Full project ID with instance suffix
          name: appLabels.name || container.Names[0]?.replace("/", "") || "unknown",
          logo: appLabels.logo ? (appLabels.logo.includes("://") ? appLabels.logo : `https://ipfs.io/ipfs/${appLabels.logo}`) : null,
          category: appLabels.category || "uncategorized",
          port: appLabels.port || null,
          description: appLabels.description || "",
          docs: appLabels.docs || null,
          website: appLabels.website || null,
        },
      };
    });

    // Filter out auxiliary containers (sidecars)
    // Identify stacks that have at least one explicit Yantra app
    const yantraProjects = new Set();
    formattedContainers.forEach((c) => {
      const project = c.labels ? c.labels["com.docker.compose.project"] : null;
      if (c.appLabels && c.appLabels.name && project) {
        yantraProjects.add(project);
      }
    });

    // Filter: Show container IF:
    // 1. It has a visible Yantra name label
    // 2. OR it does NOT belong to a project that has a Yantra app (unmanaged/external containers)
    const filteredContainers = formattedContainers.filter((c) => {
      const hasYantraLabel = !!(c.appLabels && c.appLabels.name);
      const project = c.labels ? c.labels["com.docker.compose.project"] : null;
      const isPartOfYantraStack = project && yantraProjects.has(project);

      return hasYantraLabel || !isPartOfYantraStack;
    });

    res.json({
      success: true,
      count: filteredContainers.length,
      containers: filteredContainers,
    });
}));

// GET /api/containers/:id - Get single container details
app.get("/api/containers/:id", asyncHandler(async (req, res) => {
  const container = docker.getContainer(req.params.id);
  const info = await container.inspect();
  const appLabels = parseAppLabels(info.Config.Labels);
  const composeProject = info.Config.Labels["com.docker.compose.project"];

    // If part of a compose project, get ports from all containers in the stack
    let allPorts = info.NetworkSettings.Ports;
    if (composeProject) {
      try {
        const allContainers = await docker.listContainers({ all: false });
        const projectContainers = allContainers.filter((c) => c.Labels["com.docker.compose.project"] === composeProject);

        // Aggregate all ports from the stack
        const portMap = {};
        for (const pc of projectContainers) {
          if (pc.Ports) {
            pc.Ports.forEach((port) => {
              if (port.PublicPort && port.PrivatePort) {
                const key = `${port.PrivatePort}/${port.Type}`;
                if (!portMap[key]) {
                  portMap[key] = [
                    {
                      HostIp: port.IP || "0.0.0.0",
                      HostPort: String(port.PublicPort),
                    },
                  ];
                }
              }
            });
          }
        }
        allPorts = portMap;
      } catch (err) {
        log("error", "Failed to aggregate stack ports:", err.message);
      }
    }

    res.json({
      success: true,
      container: {
        id: info.Id,
        name: info.Name.replace("/", ""),
        image: info.Config.Image,
        imageId: info.Image, // SHA256 of the image
        state: info.State.Status || (info.State.Running ? "running" : "stopped"),
        stateDetails: info.State,
        created: info.Created,
        ports: allPorts,
        mounts: info.Mounts,
        env: info.Config.Env,
        labels: appLabels,
        expireAt: info.Config.Labels?.["yantra.expireAt"] || null,
        app: {
          name: appLabels.name || info.Name.replace("/", ""),
          logo: appLabels.logo
            ? appLabels.logo.includes("://")
              ? appLabels.logo
              : `https://ipfs.io/ipfs/${appLabels.logo}`
            : "https://ipfs.io/ipfs/QmVdbRUyvZpXCsVJAs7fo1PJPXaPHnWRtSCFx6jFTGaG5i",
          category: appLabels.category || "uncategorized",
          port: appLabels.port || null,
          description: appLabels.description || "",
          docs: appLabels.docs || null,
          website: appLabels.website || null,
        },
      },
    });
}));

// GET /api/containers/:id/stats - Get real-time stats for a container
app.get("/api/containers/:id/stats", asyncHandler(async (req, res) => {
  const container = docker.getContainer(req.params.id);
  const stats = await container.stats({ stream: false });

    // Calculate CPU percentage
    const cpuDelta = stats.cpu_stats.cpu_usage.total_usage - stats.precpu_stats.cpu_usage.total_usage;
    const systemDelta = stats.cpu_stats.system_cpu_usage - stats.precpu_stats.system_cpu_usage;
    const cpuPercent = systemDelta > 0 ? (cpuDelta / systemDelta) * stats.cpu_stats.online_cpus * 100 : 0;

    // Calculate memory usage
    // Docker's raw `memory_stats.usage` includes page cache/buffers, which often makes UI
    // appear "higher" than what users expect from tools like `docker stats`.
    // Prefer an "excluding cache" metric when available.
    const memoryRawUsage = stats?.memory_stats?.usage || 0;
    const memoryLimit = stats?.memory_stats?.limit || 0;
    const memStats = stats?.memory_stats?.stats || {};
    const memoryCache =
      typeof memStats.inactive_file === "number"
        ? memStats.inactive_file // cgroup v2
        : typeof memStats.cache === "number"
          ? memStats.cache // cgroup v1
          : 0;
    const memoryUsage = Math.max(0, memoryRawUsage - memoryCache);
    const memoryPercent = memoryLimit > 0 ? (memoryUsage / memoryLimit) * 100 : 0;

    // Network I/O
    let networkRx = 0;
    let networkTx = 0;
    if (stats.networks) {
      Object.values(stats.networks).forEach((network) => {
        networkRx += network.rx_bytes || 0;
        networkTx += network.tx_bytes || 0;
      });
    }

    // Block I/O
    let blockRead = 0;
    let blockWrite = 0;
    if (stats.blkio_stats && stats.blkio_stats.io_service_bytes_recursive) {
      stats.blkio_stats.io_service_bytes_recursive.forEach((io) => {
        if (io.op === "Read") blockRead += io.value;
        if (io.op === "Write") blockWrite += io.value;
      });
    }

    res.json({
      success: true,
      stats: {
        cpu: {
          percent: cpuPercent.toFixed(2),
          usage: stats.cpu_stats.cpu_usage.total_usage,
        },
        memory: {
          usage: memoryUsage,
          rawUsage: memoryRawUsage,
          cache: memoryCache,
          limit: memoryLimit,
          percent: memoryPercent.toFixed(2),
        },
        network: {
          rx: networkRx,
          tx: networkTx,
        },
        blockIO: {
          read: blockRead,
          write: blockWrite,
        },
      },
    });
}));

// GET /api/containers/:id/logs - Get logs from a container
app.get("/api/containers/:id/logs", asyncHandler(async (req, res) => {
  const container = docker.getContainer(req.params.id);
  const tailLines = req.query.tail || 100;

    const logs = await container.logs({
      stdout: true,
      stderr: true,
      tail: tailLines,
      timestamps: true,
    });

    // Parse logs (Docker logs have a header we need to strip)
    const logString = logs.toString("utf8");
    const lines = logString
      .split("\n")
      .filter((line) => line.trim())
      .map((line) => {
        // Strip Docker log header (8 bytes)
        const cleanLine = line.substring(8);
        return cleanLine;
      });

    res.json({
      success: true,
      logs: lines,
    });
}));

// GET /api/apps - List available apps from /apps directory
app.get("/api/apps", asyncHandler(async (req, res) => {
  const forceRefresh = req.query.refresh === "1" || req.query.refresh === "true";
  const { apps, count } = await getAppsCatalogCached({ forceRefresh });
  res.json({
    success: true,
    count,
    apps: apps,
  });
}));

// GET /api/apps/:id/check-arch - Check architecture compatibility for an app
app.get("/api/apps/:id/check-arch", asyncHandler(async (req, res) => {
  const appId = req.params.id;
  const appsDir = path.join(__dirname, "..", "apps");
  const appPath = path.join(appsDir, appId);
  const composePath = path.join(appPath, "compose.yml");

  // Verify compose file exists
  try {
    await fsPromises.access(composePath);
  } catch (err) {
    throw new NotFoundError(`App '${appId}' not found`);
  }

  // Get image name from compose file
  const imageName = await getImageFromCompose(composePath);
  if (!imageName) {
    throw new BadRequestError("Could not extract image name from compose file");
  }

  // Check architecture support
  const archCheck = await checkImageArchitectureSupport(imageName);

  res.json({
    success: true,
    appId: appId,
    image: imageName,
    supported: archCheck.supported,
    systemArch: archCheck.systemArch,
    imageArch: archCheck.imageArch,
  });
}));

// POST /api/deploy - Deploy a compose file from /apps directory
app.post("/api/deploy", async (req, res) => {
  log("info", "🚀 [POST /api/deploy] Deploy request received");
  try {
    const { appId, environment, expiresIn, customPortMappings, instanceId } = req.body;
    log("info", `🚀 [POST /api/deploy] Deploying app: ${appId}${instanceId > 1 ? ` (Instance #${instanceId})` : ''}`);
    if (environment) {
      log("info", `🚀 [POST /api/deploy] Custom environment:`, environment);
    }
    if (expiresIn) {
      log("info", `🚀 [POST /api/deploy] Temporary installation: ${expiresIn} hours`);
    }
    if (customPortMappings) {
      log("info", `🚀 [POST /api/deploy] Custom port mappings:`, customPortMappings);
    }
    if (instanceId) {
      log("info", `🚀 [POST /api/deploy] Instance ID: ${instanceId}`);
    }

    if (!appId) {
      log("error", "❌ [POST /api/deploy] No appId provided");
      return res.status(400).json({
        success: false,
        error: "appId is required",
      });
    }

    const appsDir = path.join(__dirname, "..", "apps");
    const appPath = path.join(appsDir, appId);
    const composePath = path.join(appPath, "compose.yml");
    log("info", `🚀 [POST /api/deploy] Compose path: ${composePath}`);

    // Verify compose file exists
    let composeContent;
    try {
      composeContent = await readFile(composePath, "utf-8");
      log("info", `🚀 [POST /api/deploy] Compose file exists, proceeding with deployment`);
    } catch (err) {
      log("error", `❌ [POST /api/deploy] Compose file not found for ${appId}`);
      return res.status(404).json({
        success: false,
        error: `App '${appId}' not found or has no compose.yml`,
      });
    }

    // Check architecture compatibility
    const imageName = await getImageFromCompose(composePath);
    if (imageName) {
      log("info", `🚀 [POST /api/deploy] Checking architecture support for image: ${imageName}`);
      const archCheck = await checkImageArchitectureSupport(imageName);

      if (archCheck.supported === false) {
        log("error", `❌ [POST /api/deploy] Architecture not supported for ${appId}`);
        return res.status(400).json({
          success: false,
          error: "Architecture not supported",
          message: `The image '${imageName}' does not support your system architecture (${archCheck.systemArch}). Image supports: ${archCheck.imageArch}`,
          details: {
            image: imageName,
            systemArch: archCheck.systemArch,
            imageArch: archCheck.imageArch,
          },
        });
      } else if (archCheck.supported === "unknown") {
        log("info", `⚠️  [POST /api/deploy] Could not verify architecture, attempting deployment anyway`);
      } else {
        log("info", `✅ [POST /api/deploy] Architecture compatible (${archCheck.systemArch})`);
      }
    }

    // Write .env file if environment variables provided
    if (environment && Object.keys(environment).length > 0) {
      const envPath = path.join(appPath, ".env");
      // Filter out empty values - don't write vars with empty/whitespace-only values
      const envContent = Object.entries(environment)
        .filter(([key, value]) => value !== null && value !== undefined && String(value).trim() !== "")
        .map(([key, value]) => `${key}=${value}`)
        .join("\n");

      // Only create .env file if there are non-empty variables
      if (envContent.length > 0) {
        await writeFile(envPath, envContent, "utf-8");
        log("info", `🚀 [POST /api/deploy] Created .env file with custom variables`);
      }
    }

    // Add expiration label if temporary installation
    let modifiedComposeContent = composeContent;
    if (expiresIn) {
      const expiresInHours = parseFloat(expiresIn);
      if (!isNaN(expiresInHours) && expiresInHours > 0) {
        const expireAtTimestamp = Math.floor(Date.now() / 1000) + Math.floor(expiresInHours * 3600);
        const expireAtDate = new Date(expireAtTimestamp * 1000).toISOString();
        log("info", `🚀 [POST /api/deploy] App will expire at: ${expireAtDate}`);

        // Inject expiration labels into all services in compose file
        const lines = composeContent.split("\n");
        const result = [];
        let inLabelsSection = false;
        let baseIndentLevel = 0;

        for (let i = 0; i < lines.length; i++) {
          const line = lines[i];
          result.push(line);

          // Detect when we enter a labels section
          if (line.trim().startsWith("labels:")) {
            inLabelsSection = true;
            baseIndentLevel = line.search(/\S/);
            // Add expiration labels after 'labels:' line with proper YAML indentation
            // Labels should always be indented 2 spaces more than the 'labels:' key
            const labelIndent = " ".repeat(baseIndentLevel + 2);
            result.push(`${labelIndent}yantra.expireAt: "${expireAtTimestamp}"`);
            result.push(`${labelIndent}yantra.temporary: "true"`);
          } else if (inLabelsSection) {
            // Check if we've left the labels section
            const currentIndent = line.search(/\S/);
            if (line.trim() && currentIndent <= baseIndentLevel) {
              inLabelsSection = false;
            }
          }
        }

        modifiedComposeContent = result.join("\n");

        // Write modified compose to a temporary file
        const tempComposePath = path.join(appPath, ".compose.tmp.yml");
        await writeFile(tempComposePath, modifiedComposeContent, "utf-8");
        log("info", `🚀 [POST /api/deploy] Created temporary compose file with expiration labels`);
      }
    }

    // Apply instance-specific naming (container names and volumes)
    if (instanceId && instanceId > 1) {
      log("info", `🚀 [POST /api/deploy] Applying instance-specific naming for instance #${instanceId}`);

      const lines = modifiedComposeContent.split("\n");
      const result = [];
      let inVolumesSection = false;
      let inServiceVolumesSection = false;

      for (let i = 0; i < lines.length; i++) {
        let line = lines[i];

        // Track if we're in the main volumes section (at root level)
        if (line.match(/^volumes:\s*$/)) {
          inVolumesSection = true;
          inServiceVolumesSection = false;
        } else if (line.match(/^[a-z]+:\s*$/) && !line.startsWith(' ')) {
          // Hit another root-level section, exit volumes section
          inVolumesSection = false;
        }

        // Track if we're in a service's volumes subsection
        if (line.match(/^\s+volumes:\s*$/)) {
          inServiceVolumesSection = true;
        } else if (line.match(/^\s+[a-z_]+:\s*$/) && !line.match(/^\s+volumes:/)) {
          // Hit another service property, exit service volumes section
          inServiceVolumesSection = false;
        }

        // Match container_name: appname
        const containerNameMatch = line.match(/^(\s*container_name:\s*)(.+)$/);
        if (containerNameMatch) {
          const indent = containerNameMatch[1];
          const containerName = containerNameMatch[2].trim();
          line = `${indent}${containerName}-${instanceId}`;
          log("info", `🚀 [POST /api/deploy] Renamed container: ${containerName} → ${containerName}-${instanceId}`);
        }

        // Match volume definitions ONLY in the volumes section
        const volumeDefMatch = line.match(/^(\s+)([a-zA-Z0-9_-]+):(\s*)$/);
        if (volumeDefMatch && inVolumesSection) {
          const indent = volumeDefMatch[1];
          const volumeName = volumeDefMatch[2];
          const suffix = volumeDefMatch[3];
          line = `${indent}${volumeName}_${instanceId}:${suffix}`;
          log("info", `🚀 [POST /api/deploy] Renamed volume: ${volumeName} → ${volumeName}_${instanceId}`);
        }

        // Match volume name property: name: volumename
        const volumeNameMatch = line.match(/^(\s+name:\s+)(.+)$/);
        if (volumeNameMatch && inVolumesSection) {
          const indent = volumeNameMatch[1];
          const volumeName = volumeNameMatch[2].trim();
          line = `${indent}${volumeName}_${instanceId}`;
          log("info", `🚀 [POST /api/deploy] Renamed volume name property: ${volumeName} → ${volumeName}_${instanceId}`);
        }

        // Match volume references in service volumes: - volumename:/path
        const volumeRefMatch = line.match(/^(\s*-\s+)([a-zA-Z0-9_-]+)(:.+)$/);
        if (volumeRefMatch && inServiceVolumesSection) {
          const indent = volumeRefMatch[1];
          const volumeName = volumeRefMatch[2];
          const path = volumeRefMatch[3];
          line = `${indent}${volumeName}_${instanceId}${path}`;
          log("info", `🚀 [POST /api/deploy] Renamed volume reference: ${volumeName} → ${volumeName}_${instanceId}`);
        }

        result.push(line);
      }

      modifiedComposeContent = result.join("\n");

      // Write to temporary file
      const tempComposePath = path.join(appPath, ".compose.tmp.yml");
      await writeFile(tempComposePath, modifiedComposeContent, "utf-8");
      log("info", `🚀 [POST /api/deploy] Created temporary compose file with instance naming`);
    }

    // Apply custom port mappings if provided
    if (customPortMappings && Object.keys(customPortMappings).length > 0) {
      log("info", `🚀 [POST /api/deploy] Applying custom port mappings`);

      // Replace port mappings in compose content
      const lines = modifiedComposeContent.split("\n");
      const result = [];

      for (let i = 0; i < lines.length; i++) {
        let line = lines[i];

        // Match fixed port mappings: - "8080:80", "53:53/tcp", etc.
        const fixedPortMatch = line.match(/^(\s*-\s*)(["']?)(\d+):(\d+)(\/(?:tcp|udp))?(["']?)$/);

        if (fixedPortMatch) {
          const indent = fixedPortMatch[1];
          const openQuote = fixedPortMatch[2];
          const hostPort = fixedPortMatch[3];
          const containerPort = fixedPortMatch[4];
          const protocol = fixedPortMatch[5] || '';
          const closeQuote = fixedPortMatch[6];
          const protocolName = protocol.replace('/', '') || 'tcp';

          // Check if this port has a custom mapping
          const mappingKey = `${hostPort}/${protocolName}`;
          if (customPortMappings[mappingKey]) {
            const newHostPort = customPortMappings[mappingKey];
            line = `${indent}${openQuote}${newHostPort}:${containerPort}${protocol}${closeQuote}`;
            log("info", `🚀 [POST /api/deploy] Replaced fixed port ${hostPort} with ${newHostPort}`);
          }
        }

        // Match auto-assigned ports: - "9091"
        const autoPortMatch = line.match(/^(\s*-\s*)(["'])(\d+)["']$/);

        if (autoPortMatch) {
          const indent = autoPortMatch[1];
          const quote = autoPortMatch[2];
          const port = autoPortMatch[3];
          const protocolName = 'tcp'; // Auto-assigned ports default to tcp

          // Check if this port has a custom mapping
          const mappingKey = `${port}/${protocolName}`;
          if (customPortMappings[mappingKey]) {
            const newHostPort = customPortMappings[mappingKey];
            line = `${indent}${quote}${newHostPort}:${port}${quote}`;
            log("info", `🚀 [POST /api/deploy] Replaced auto-assigned port ${port} with ${newHostPort}`);
          }
        }

        result.push(line);
      }

      modifiedComposeContent = result.join("\n");

      // Write to temporary file
      const tempComposePath = path.join(appPath, ".compose.tmp.yml");
      await writeFile(tempComposePath, modifiedComposeContent, "utf-8");
      log("info", `🚀 [POST /api/deploy] Created temporary compose file with custom ports`);
    }

    // Deploy using docker compose (Docker will auto-assign ports or use custom mappings)
    const composeFile = (expiresIn || customPortMappings || (instanceId && instanceId > 1)) ? ".compose.tmp.yml" : "compose.yml";

    // Set unique project name for multi-instance deployments
    const projectName = (instanceId && instanceId > 1) ? `${appId}-${instanceId}` : appId;

    const composeCmd = await resolveComposeCommand({ socketPath, log });
    const command = `${composeCmd.display} -p "${projectName}" -f "${composeFile}" up -d`;
    log("info", `🚀 [POST /api/deploy] Executing: ${command}`);

    try {
      const { stdout, stderr, exitCode } = await spawnProcess(
        composeCmd.command,
        [...composeCmd.args, "-p", projectName, "-f", composeFile, "up", "-d"],
        {
          cwd: appPath,
          env: {
            ...process.env,
            DOCKER_HOST: `unix://${socketPath}`,
          },
        }
      );

      if (exitCode !== 0) {
        throw new Error(`docker compose failed with exit code ${exitCode}: ${stderr}`);
      }

      log("info", `✅ [POST /api/deploy] Deployment successful for ${appId}`);
      log("info", `   stdout: ${stdout.trim()}`);
      if (stderr) log("info", `   stderr: ${stderr.trim()}`);

      // Cleanup temporary compose file if it exists
      if (expiresIn || customPortMappings || (instanceId && instanceId > 1)) {
        try {
          const tempComposePath = path.join(appPath, ".compose.tmp.yml");
          const fs = await import("fs/promises");
          await fs.unlink(tempComposePath);
          log("info", `🚀 [POST /api/deploy] Cleaned up temporary compose file`);
        } catch (err) {
          log("warn", `⚠️  [POST /api/deploy] Failed to cleanup temp file: ${err.message}`);
        }
      }

      res.json({
        success: true,
        message: `App '${appId}' deployed successfully`,
        appId: appId,
        output: stdout,
        warnings: stderr || null,
        temporary: !!expiresIn,
      });
    } catch (error) {
      log("error", `❌ [POST /api/deploy] Deployment failed for ${appId}:`, error.message);
      log("error", `   stderr: ${error.stderr}`);

      // Cleanup temporary compose file on error
      if (composeFile === ".compose.tmp.yml") {
        const tempComposePath = path.join(appPath, ".compose.tmp.yml");
        try {
          await unlink(tempComposePath);
          log("info", `🚀 [POST /api/deploy] Cleaned up temporary compose file after error`);
        } catch (cleanupError) {
          log("error", `⚠️ [POST /api/deploy] Failed to cleanup temp file:`, cleanupError.message);
        }
      }

      // Check if the error is architecture-related
      const isArchError =
        error.stderr && (error.stderr.includes("no matching manifest") || error.stderr.includes("platform") || error.stderr.includes("architecture"));

      res.status(500).json({
        success: false,
        error: isArchError ? "Architecture not supported" : "Deployment failed",
        message: isArchError ? "This image does not support your system architecture" : error.message,
        stderr: error.stderr,
      });
    }
  } catch (error) {
    log("error", "❌ [POST /api/deploy] Unexpected error:", error.message);
    res.status(500).json({
      success: false,
      error: error.message,
    });
  }
});

// GET /api/images - List all images with usage status
app.get("/api/images", async (req, res) => {
  try {
    const images = await docker.listImages();
    const containers = await docker.listContainers({ all: true });

    // Create a set of image IDs used by containers
    const usedImageIds = new Set(containers.map((c) => c.ImageID));

    const formattedImages = images.map((image) => {
      const isUsed = usedImageIds.has(image.Id);
      const repoTags = image.RepoTags || ["<none>:<none>"];
      const size = (image.Size / (1024 * 1024)).toFixed(2); // Convert to MB

      return {
        id: image.Id,
        shortId: image.Id.substring(7, 19),
        tags: repoTags,
        created: image.Created,
        size: size,
        sizeBytes: image.Size,
        isUsed: isUsed,
        containers: image.Containers || 0,
      };
    });

    // Sort by size (largest first) and separate used from unused
    const usedImages = formattedImages.filter((img) => img.isUsed).sort((a, b) => b.sizeBytes - a.sizeBytes);
    const unusedImages = formattedImages.filter((img) => !img.isUsed).sort((a, b) => b.sizeBytes - a.sizeBytes);

    const totalSize = formattedImages.reduce((sum, img) => sum + img.sizeBytes, 0);
    const unusedSize = unusedImages.reduce((sum, img) => sum + img.sizeBytes, 0);

    res.json({
      success: true,
      total: formattedImages.length,
      used: usedImages.length,
      unused: unusedImages.length,
      totalSize: (totalSize / (1024 * 1024)).toFixed(2),
      unusedSize: (unusedSize / (1024 * 1024)).toFixed(2),
      images: formattedImages,
      usedImages: usedImages,
      unusedImages: unusedImages,
    });
  } catch (error) {
    log("error", "❌ [GET /api/images] Error:", error.message);
    res.status(500).json({
      success: false,
      error: error.message,
    });
  }
});

// GET /api/image-details/:appId - Get detailed metadata for app's Docker image(s)
app.get("/api/image-details/:appId", async (req, res) => {
  try {
    const appId = req.params.appId;
    const appsDir = path.join(__dirname, "..", "apps");
    const appPath = path.join(appsDir, appId);
    const composePath = path.join(appPath, "compose.yml");

    // Read compose file to extract image names
    let composeContent;
    try {
      composeContent = await readFile(composePath, "utf-8");
    } catch (err) {
      return res.status(404).json({
        success: false,
        error: "App compose file not found",
      });
    }

    // Extract all image references from compose file
    const imageMatches = composeContent.matchAll(/image:\s*([^\s\n]+)/g);
    const imageNames = [...new Set([...imageMatches].map(match => match[1]))];

    if (imageNames.length === 0) {
      return res.json({
        success: true,
        images: [],
      });
    }

    // Get locally available images
    const localImages = await docker.listImages();

    const imageDetails = await Promise.all(imageNames.map(async (imageName) => {
      // Check if image is available locally
      const localImage = localImages.find(img =>
        img.RepoTags && img.RepoTags.some(tag => tag === imageName || tag.includes(imageName.split(':')[0]))
      );

      if (localImage) {
        // Image is available locally - get full details
        try {
          const image = docker.getImage(localImage.Id);
          const info = await image.inspect();
          const createdDate = new Date(info.Created);
          const now = new Date();
          const diffMs = now - createdDate;
          const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

          let relativeTime = '';
          if (diffDays === 0) {
            const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
            relativeTime = diffHours === 0 ? 'just now' : `${diffHours} hour${diffHours !== 1 ? 's' : ''} ago`;
          } else if (diffDays === 1) {
            relativeTime = 'yesterday';
          } else if (diffDays < 30) {
            relativeTime = `${diffDays} day${diffDays !== 1 ? 's' : ''} ago`;
          } else if (diffDays < 365) {
            const diffMonths = Math.floor(diffDays / 30);
            relativeTime = `${diffMonths} month${diffMonths !== 1 ? 's' : ''} ago`;
          } else {
            const diffYears = Math.floor(diffDays / 365);
            relativeTime = `${diffYears} year${diffYears !== 1 ? 's' : ''} ago`;
          }

          return {
            name: imageName,
            id: localImage.Id,
            shortId: localImage.Id.substring(7, 19),
            tags: localImage.RepoTags || [imageName],
            digest: info.RepoDigests ? info.RepoDigests[0] : 'N/A',
            created: info.Created,
            createdDate: createdDate.toLocaleDateString('en-US', {
              year: 'numeric',
              month: 'short',
              day: 'numeric'
            }),
            relativeTime: relativeTime,
            architecture: info.Architecture || 'unknown',
            os: info.Os || 'unknown',
            size: (localImage.Size / (1024 * 1024)).toFixed(2),
            sizeBytes: localImage.Size,
            isLocal: true,
          };
        } catch (err) {
          log("error", `❌ Error inspecting image ${imageName}:`, err.message);
        }
      }

      // Image not available locally - return basic info from compose
      return {
        name: imageName,
        id: null,
        shortId: 'Not pulled',
        tags: [imageName],
        digest: 'N/A',
        created: null,
        createdDate: 'Not available',
        relativeTime: 'Image not pulled yet',
        architecture: 'unknown',
        os: 'unknown',
        size: 'N/A',
        sizeBytes: 0,
        isLocal: false,
      };
    }));

    res.json({
      success: true,
      images: imageDetails.filter(img => img !== undefined),
    });
  } catch (error) {
    log("error", `❌ [GET /api/image-details] Error:`, error.message);
    res.status(500).json({
      success: false,
      error: error.message,
    });
  }
});

// DELETE /api/images/:id - Remove an image
app.delete("/api/images/:id", async (req, res) => {
  log("info", `🗑️  [DELETE /api/images/:id] Remove request for image: ${req.params.id}`);
  try {
    const image = docker.getImage(req.params.id);
    const info = await image.inspect();

    log("info", `🗑️  [DELETE /api/images/:id] Image tags: ${info.RepoTags}`);

    // Remove image
    log("info", `🗑️  [DELETE /api/images/:id] Removing image...`);
    await image.remove({ force: false }); // force: false means won't delete if in use

    log("info", `✅ [DELETE /api/images/:id] Successfully removed image`);
    res.json({
      success: true,
      message: "Image removed successfully",
      imageId: req.params.id,
      tags: info.RepoTags,
    });
  } catch (error) {
    log("error", `❌ [DELETE /api/images/:id] Error:`, error.message);
    res.status(500).json({
      success: false,
      error: "Failed to remove image",
      message: error.message,
    });
  }
});



// DELETE /api/containers/:id - Remove container (or stack if part of an app)
app.delete("/api/containers/:id", async (req, res) => {
  log("info", `🗑️  [DELETE /api/containers/:id] Remove request for: ${req.params.id}`);
  try {
    const container = docker.getContainer(req.params.id);
    const info = await container.inspect();
    const containerName = info.Name.replace("/", "");
    const labels = info.Config.Labels || {};
    const composeProject = labels["com.docker.compose.project"];

    log("info", `🗑️  [DELETE /api/containers/:id] Container: ${containerName}, Project: ${composeProject || "none"}`);

    // If part of a compose project, check if it's a managed app
    if (composeProject) {
      const appsDir = path.join(__dirname, "..", "apps");

      // Extract base app ID from project (remove -2, -3 suffix for instances)
      let baseAppId = composeProject;
      if (composeProject.match(/-\d+$/)) {
        baseAppId = composeProject.replace(/-\d+$/, '');
      }

      const appPath = path.join(appsDir, baseAppId); // Use base app ID for folder lookup
      const composePath = path.join(appPath, "compose.yml");

      try {
        const fs = await import("fs/promises");
        try {
          await fs.access(composePath);
        } catch {
          throw new Error("Compose file not found");
        }
        
        log("info", `🗑️  [DELETE /api/containers/:id] Found managed app at ${appPath}, shutting down stack...`);

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

        log("info", `✅ [DELETE /api/containers/:id] Stack removal successful`);
        return res.json({
          success: true,
          message: `App stack '${composeProject}' removed successfully`,
          container: containerName,
          stackRemoved: true,
          volumesRemoved: [], // Stack deletion handles volumes, return empty to satisfy UI
          volumesFailed: [],
          output: stdout,
        });
      } catch (err) {
        log("info", `⚠️  [DELETE /api/containers/:id] Compose file not found at ${composePath}, falling back to single container deletion`);
      }
    }

    // Fallback: Single container deletion (for unmanaged containers)

    // Get volume names from mounts
    const volumeNames = info.Mounts.filter((mount) => mount.Type === "volume").map((mount) => mount.Name);

    log("info", `🗑️  [DELETE /api/containers/:id] Found ${volumeNames.length} volumes:`, volumeNames);

    // Stop container if running
    if (info.State.Running) {
      log("info", `🗑️  [DELETE /api/containers/:id] Stopping container...`);
      await container.stop();
    }

    // Remove container
    log("info", `🗑️  [DELETE /api/containers/:id] Removing container...`);
    await container.remove();

    // Volumes are preserved - they can be deleted manually from the Volumes page
    log("info", `✅ [DELETE /api/containers/:id] Successfully removed ${containerName}. Volumes preserved: ${volumeNames.join(', ')}`);
    res.json({
      success: true,
      message: `Container '${containerName}' removed successfully`,
      container: containerName,
      volumesPreserved: volumeNames,
    });
  } catch (error) {
    log("error", `❌ [DELETE /api/containers/:id] Error:`, error.message);
    res.status(500).json({
      success: false,
      error: "Failed to remove container",
      message: error.message,
    });
  }
});

// POST /api/system/prune - Prune unused resources
app.post("/api/system/prune", async (req, res) => {
  log("info", "🧹 [POST /api/system/prune] Prune request received");
  try {
    const { images, volumes } = req.body;
    const results = {
      images: { count: 0, spaceReclaimed: 0 },
      volumes: { count: 0, spaceReclaimed: 0 }
    };

    // Prune images if requested
    if (images) {
      log("info", "🧹 [POST /api/system/prune] Pruning images...");
      try {
        // Prune all unused images (dangling=false means all unused)
        const pruned = await docker.pruneImages({ filters: { dangling: { "false": true } } });
        results.images.count = pruned.ImagesDeleted?.length || 0;
        results.images.spaceReclaimed = pruned.SpaceReclaimed || 0;
        log("info", `✅ [POST /api/system/prune] Pruned ${results.images.count} images, reclaimed ${results.images.spaceReclaimed} bytes`);
      } catch (err) {
        log("error", "❌ [POST /api/system/prune] Failed to prune images:", err.message);
      }
    }

    // Prune volumes if requested
    // Logic updated to match frontend "Delete All" behavior: explicitly identifying and deleting unused volumes
    if (volumes) {
      log("info", "🧹 [POST /api/system/prune] Pruning volumes (iterative mode)...");
      try {
        // 1. Get all volumes
        const volList = await docker.listVolumes();
        const allVolumes = volList.Volumes || [];

        // 2. Identify used volumes
        const containers = await docker.listContainers({ all: true });
        const usedVolumeNames = new Set();
        containers.forEach((container) => {
          if (container.Mounts) {
            container.Mounts.forEach((mount) => {
              if (mount.Type === "volume" && mount.Name) {
                usedVolumeNames.add(mount.Name);
              }
            });
          }
        });

        // 3. Filter unused volumes
        const unusedVolumes = allVolumes.filter(v => !usedVolumeNames.has(v.Name));
        log("info", `🧹 [POST /api/system/prune] Found ${unusedVolumes.length} unused volumes out of ${allVolumes.length} total.`);

        // 4. Delete each unused volume
        for (const vol of unusedVolumes) {
          try {
            const volume = docker.getVolume(vol.Name);

            // Get size before deleting (best effort)
            let size = 0;
            // Note: dockerode df() is heavy, we might skip precise size calculation per volume here for speed
            // or we could try to estimate. For now, we'll just count it. 

            await volume.remove();
            results.volumes.count++;
            // We assume some space reclaimed, but without precise size it's hard. 
            // The frontend might calculate total expected reclaimable, so 0 here might confuse it?
            // Let's try to get simple inspection or just ignore exact bytes for now, 
            // as reliability is more important than the exact byte report in the toast.

          } catch (err) {
            log("warn", `⚠️ [POST /api/system/prune] Failed to remove volume ${vol.Name}: ${err.message}`);
          }
        }

        log("info", `✅ [POST /api/system/prune] Successfully removed ${results.volumes.count} volumes.`);

      } catch (err) {
        log("error", "❌ [POST /api/system/prune] Failed to prune volumes:", err.message);
      }
    }

    res.json({
      success: true,
      results
    });
  } catch (error) {
    log("error", "❌ [POST /api/system/prune] Error:", error.message);
    res.status(500).json({
      success: false,
      error: error.message
    });
  }
});

// Root endpoint
app.get("/", (req, res) => {
  res.json({
    name: "Yantra API",
    version: packageJson.version,
    description: "Lightweight Docker dashboard for self-hosting",
    endpoints: {
      version: "/api/version",
      containers: "/api/containers",
      container: "/api/containers/:id",
      apps: "/api/apps",
      deploy: "POST /api/deploy",
    },
  });
});

// GET /api/ports/used - Get all currently used ports from Docker containers
app.get("/api/ports/used", async (req, res) => {
  log("info", "🔌 [GET /api/ports/used] Fetching all used ports");
  try {
    const containers = await docker.listContainers({ all: false }); // Only running containers
    const usedPorts = new Set();

    containers.forEach((container) => {
      if (container.Ports && container.Ports.length > 0) {
        container.Ports.forEach((port) => {
          if (port.PublicPort) {
            usedPorts.add(port.PublicPort);
          }
        });
      }
    });

    const portArray = Array.from(usedPorts).sort((a, b) => a - b);
    log("info", `✅ [GET /api/ports/used] Found ${portArray.length} used ports`);

    res.json({
      success: true,
      count: portArray.length,
      ports: portArray,
    });
  } catch (error) {
    log("error", "❌ [GET /api/ports/used] Error:", error.message);
    res.status(500).json({
      success: false,
      error: error.message,
    });
  }
});

// GET /api/volumes - List all Docker volumes
app.get("/api/volumes", async (req, res) => {
  try {
    const volumes = await docker.listVolumes();
    const volumeList = volumes.Volumes || [];

    // Get volume sizes using dockerode df() method
    let volumeSizes = {};
    try {
      const dfData = await docker.df();

      // Extract volume sizes from the df response
      if (dfData.Volumes && Array.isArray(dfData.Volumes)) {
        dfData.Volumes.forEach(vol => {
          if (vol.Name && vol.UsageData && vol.UsageData.Size !== undefined) {
            // Size is already in bytes
            volumeSizes[vol.Name] = vol.UsageData.Size;
          }
        });
      }
    } catch (dfError) {
      log("warn", "⚠️  [GET /api/volumes] Could not get volume sizes:", dfError.message);
    }

    // Get all containers to check which volumes are in use
    const containers = await docker.listContainers({ all: true });

    // Check if any volume has an active dufs browser container
    const browsedVolumes = new Set();
    // Track which volumes are mounted by containers
    const usedVolumeNames = new Set();

    containers.forEach((container) => {
      if (container.Labels && container.Labels["yantra.volume-browser"]) {
        browsedVolumes.add(container.Labels["yantra.volume-browser"]);
      }

      // Add all mounted volumes to the used set
      if (container.Mounts) {
        container.Mounts.forEach((mount) => {
          if (mount.Type === "volume" && mount.Name) {
            usedVolumeNames.add(mount.Name);
          }
        });
      }
    });

    const enrichedVolumes = volumeList.map((vol) => {
      const sizeBytes = volumeSizes[vol.Name] || 0;
      const sizeMB = (sizeBytes / (1024 * 1024)).toFixed(2);

      return {
        name: vol.Name,
        driver: vol.Driver,
        mountpoint: vol.Mountpoint,
        createdAt: vol.CreatedAt,
        labels: vol.Labels || {},
        isBrowsing: browsedVolumes.has(vol.Name),
        isUsed: usedVolumeNames.has(vol.Name),
        size: sizeMB,
        sizeBytes: sizeBytes,
      };
    });

    // Separate used and unused volumes (sort by size)
    const usedVolumes = enrichedVolumes.filter(v => v.isUsed).sort((a, b) => b.sizeBytes - a.sizeBytes);
    const unusedVolumes = enrichedVolumes.filter(v => !v.isUsed).sort((a, b) => b.sizeBytes - a.sizeBytes);

    // Calculate total sizes
    const totalSize = enrichedVolumes.reduce((sum, vol) => sum + vol.sizeBytes, 0);
    const unusedSize = unusedVolumes.reduce((sum, vol) => sum + vol.sizeBytes, 0);

    res.json({
      success: true,
      total: enrichedVolumes.length,
      used: usedVolumes.length,
      unused: unusedVolumes.length,
      totalSize: (totalSize / (1024 * 1024)).toFixed(2),
      unusedSize: (unusedSize / (1024 * 1024)).toFixed(2),
      volumes: enrichedVolumes,
      usedVolumes: usedVolumes,
      unusedVolumes: unusedVolumes,
    });
  } catch (error) {
    log("error", "❌ [GET /api/volumes] Error:", error.message);
    res.status(500).json({
      success: false,
      error: error.message,
    });
  }
});

// POST /api/volumes/:name/browse - Start a dufs container to browse a volume
app.post("/api/volumes/:name/browse", async (req, res) => {
  const volumeName = req.params.name;
  const expiryMinutes = parseInt(req.body.expiryMinutes) || 0; // 0 means no expiry
  log("info", `🔍 [POST /api/volumes/${volumeName}/browse] Starting volume browser${expiryMinutes > 0 ? ` with ${expiryMinutes}m expiry` : ' (no expiry)'}`);

  try {
    // Check if volume exists
    const volumes = await docker.listVolumes();
    const volume = volumes.Volumes?.find((v) => v.Name === volumeName);

    if (!volume) {
      return res.status(404).json({
        success: false,
        error: "Volume not found",
      });
    }

    // Check if a browser container already exists for this volume
    const containers = await docker.listContainers({ all: true });
    const existingBrowser = containers.find(
      (c) => c.Labels && c.Labels["yantra.volume-browser"] === volumeName
    );

    if (existingBrowser) {
      const container = docker.getContainer(existingBrowser.Id);
      const inspect = await container.inspect();

      // If stopped, start it
      if (inspect.State.Status !== "running") {
        await container.start();
        log("info", `▶️ [POST /api/volumes/${volumeName}/browse] Started existing browser`);
      }

      const port = inspect.NetworkSettings.Ports["5000/tcp"]?.[0]?.HostPort;
      return res.json({
        success: true,
        port: port ? parseInt(port) : null,
        containerId: existingBrowser.Id,
        message: "Browser container already exists and is now running",
      });
    }

    // Pull the dufs image if not already present
    const imageName = "sigoden/dufs:latest";
    log("info", `📥 [POST /api/volumes/${volumeName}/browse] Pulling ${imageName} if needed`);

    try {
      await docker.getImage(imageName).inspect();
    } catch (error) {
      // Image doesn't exist locally, pull it
      log("info", `📥 [POST /api/volumes/${volumeName}/browse] Image not found, pulling...`);
      await new Promise((resolve, reject) => {
        docker.pull(imageName, (err, stream) => {
          if (err) return reject(err);
          docker.modem.followProgress(stream, (err, output) => {
            if (err) return reject(err);
            resolve(output);
          });
        });
      });
      log("info", `✅ [POST /api/volumes/${volumeName}/browse] Image pulled successfully`);
    }

    // Create new browser container
    const containerName = `yantra-v-${volumeName}`;

    // Prepare labels
    const labels = {
      "yantra.volume-browser": volumeName,
      "yantra.managed": "true",
    };

    // Add expiry label if specified
    if (expiryMinutes > 0) {
      const expiryTimestamp = Math.floor(Date.now() / 1000) + (expiryMinutes * 60);
      labels["yantra.expireAt"] = expiryTimestamp.toString();
    }

    const container = await docker.createContainer({
      Image: imageName,
      name: containerName,
      Cmd: ["/data", "--enable-cors", "--allow-all"],
      Labels: labels,
      HostConfig: {
        Binds: [`${volumeName}:/data`],
        PortBindings: {
          "5000/tcp": [{ HostPort: "" }], // Random port
        },
        RestartPolicy: {
          Name: "no",
        },
      },
      ExposedPorts: {
        "5000/tcp": {},
      },
    });

    await container.start();
    const inspect = await container.inspect();
    const port = inspect.NetworkSettings.Ports["5000/tcp"]?.[0]?.HostPort;

    log("info", `✅ [POST /api/volumes/${volumeName}/browse] Browser started on port ${port}`);
    res.json({
      success: true,
      port: port ? parseInt(port) : null,
      containerId: container.id,
      message: "Volume browser started successfully",
    });
  } catch (error) {
    log("error", `❌ [POST /api/volumes/${volumeName}/browse] Error:`, error.message);
    res.status(500).json({
      success: false,
      error: error.message,
    });
  }
});

// DELETE /api/volumes/:name/browse - Stop and remove the dufs browser container
app.delete("/api/volumes/:name/browse", async (req, res) => {
  const volumeName = req.params.name;
  log("info", `🛑 [DELETE /api/volumes/${volumeName}/browse] Stopping volume browser`);

  try {
    const containers = await docker.listContainers({ all: true });
    const browserContainer = containers.find(
      (c) => c.Labels && c.Labels["yantra.volume-browser"] === volumeName
    );

    if (!browserContainer) {
      return res.status(404).json({
        success: false,
        error: "No browser container found for this volume",
      });
    }

    const container = docker.getContainer(browserContainer.Id);
    const inspect = await container.inspect();

    // Stop if running
    if (inspect.State.Running) {
      await container.stop();
      log("info", `⏸️ [DELETE /api/volumes/${volumeName}/browse] Container stopped`);
    }

    // Remove container
    await container.remove();
    log("info", `✅ [DELETE /api/volumes/${volumeName}/browse] Browser removed`);

    res.json({
      success: true,
      message: "Volume browser stopped and removed successfully",
    });
  } catch (error) {
    log("error", `❌ [DELETE /api/volumes/${volumeName}/browse] Error:`, error.message);
    res.status(500).json({
      success: false,
      error: error.message,
    });
  }
});

// DELETE /api/volumes/:name - Remove a Docker volume
app.delete("/api/volumes/:name", async (req, res) => {
  const volumeName = req.params.name;
  log("info", `🗑️  [DELETE /api/volumes/:name] Remove request for volume: ${volumeName}`);

  try {
    const volume = docker.getVolume(volumeName);

    // Try to inspect the volume first to ensure it exists
    try {
      await volume.inspect();
    } catch (err) {
      return res.status(404).json({
        success: false,
        error: "Volume not found",
        message: `Volume '${volumeName}' does not exist`,
      });
    }

    // Remove the volume
    log("info", `🗑️  [DELETE /api/volumes/:name] Removing volume...`);
    await volume.remove();

    log("info", `✅ [DELETE /api/volumes/:name] Successfully removed volume: ${volumeName}`);
    res.json({
      success: true,
      message: `Volume '${volumeName}' removed successfully`,
      volume: volumeName,
    });
  } catch (error) {
    log("error", `❌ [DELETE /api/volumes/:name] Error:`, error.message);

    // Check if volume is in use
    const isInUseError = error.message && error.message.includes("in use");

    res.status(500).json({
      success: false,
      error: isInUseError ? "Volume is in use" : "Failed to remove volume",
      message: isInUseError
        ? `Volume '${volumeName}' is currently in use by a container and cannot be deleted`
        : error.message,
    });
  }
});

// POST /api/ports/suggest - Suggest available ports for an app
app.post("/api/ports/suggest", async (req, res) => {
  log("info", "💡 [POST /api/ports/suggest] Suggesting ports for app");
  try {
    const { appId, ports: appPorts } = req.body;

    if (!appId || !appPorts || !Array.isArray(appPorts)) {
      return res.status(400).json({
        success: false,
        error: "appId and ports array are required",
      });
    }

    // Get all currently used ports
    const containers = await docker.listContainers({ all: false });
    const usedPorts = new Set();

    containers.forEach((container) => {
      if (container.Ports && container.Ports.length > 0) {
        container.Ports.forEach((port) => {
          if (port.PublicPort) {
            usedPorts.add(port.PublicPort);
          }
        });
      }
    });

    // Port suggestion algorithm: starts from 5255
    const START_PORT = 5255;
    let currentPort = START_PORT;

    const suggestedPorts = appPorts.map((port) => {
      // Only suggest for named ports (ports with descriptions in yantra.port label)
      if (!port.isNamed) {
        // Keep original port for unnamed ports (like BitTorrent ports, etc.)
        return {
          ...port,
          suggestedPort: port.hostPort,
          isOriginal: true,
        };
      }

      // Find next available port starting from currentPort
      while (usedPorts.has(currentPort)) {
        currentPort++;
      }

      const suggested = currentPort;
      usedPorts.add(currentPort); // Mark as used for next iteration
      currentPort++;

      return {
        ...port,
        suggestedPort: suggested,
        isOriginal: false,
      };
    });

    log("info", `✅ [POST /api/ports/suggest] Suggested ports for ${appId}:`, suggestedPorts);

    res.json({
      success: true,
      appId,
      suggestions: suggestedPorts,
    });
  } catch (error) {
    log("error", "❌ [POST /api/ports/suggest] Error:", error.message);
    res.status(500).json({
      success: false,
      error: error.message,
    });
  }
});

// GET /api/system/info - Get host system information
app.get("/api/system/info", async (req, res) => {
  log("info", "💻 [GET /api/system/info] Fetching host system information");
  try {
    const info = await docker.info();
    
    // Get disk usage via df() for accurate storage metrics
    let storageInfo = {
      driver: info.Driver || 'unknown',
      total: null,
      used: null,
      available: null,
    };
    
    try {
      const df = await docker.df();
      
      // Calculate total storage from images, containers, and volumes
      const imagesSize = df.Images?.reduce((sum, img) => sum + (img.Size || 0), 0) || 0;
      const containersSize = df.Containers?.reduce((sum, c) => sum + (c.SizeRw || 0), 0) || 0;
      const volumesSize = df.Volumes?.reduce((sum, v) => sum + (v.UsageData?.Size || 0), 0) || 0;
      
      storageInfo.used = imagesSize + containersSize + volumesSize;
      
      // If we have filesystem info from DriverStatus, use it for total
      if (info.DriverStatus) {
        storageInfo.total = extractStorageInfo(info.DriverStatus, 'Data Space Total');
        const driverUsed = extractStorageInfo(info.DriverStatus, 'Data Space Used');
        if (driverUsed) storageInfo.used = driverUsed;
        storageInfo.available = extractStorageInfo(info.DriverStatus, 'Data Space Available');
      }
    } catch (dfError) {
      log("warn", `⚠️ [GET /api/system/info] Could not fetch df data: ${dfError.message}`);
    }

    // Extract relevant system information
    const systemInfo = {
      cpu: {
        cores: info.NCPU || 0,
      },
      memory: {
        total: info.MemTotal || 0,
      },
      storage: storageInfo,
      docker: {
        version: info.ServerVersion || 'unknown',
        containers: {
          total: info.Containers || 0,
          running: info.ContainersRunning || 0,
          paused: info.ContainersPaused || 0,
          stopped: info.ContainersStopped || 0,
        },
        images: info.Images || 0,
      },
      os: {
        type: info.OSType || 'unknown',
        name: info.OperatingSystem || 'unknown',
        arch: info.Architecture || 'unknown',
        kernel: info.KernelVersion || 'unknown',
      },
      name: info.Name || 'unknown',
    };

    const storageMsg = storageInfo.used ? `, ${formatBytes(storageInfo.used)} storage used` : '';
    log("info", `✅ [GET /api/system/info] System info retrieved: ${systemInfo.cpu.cores} cores, ${formatBytes(systemInfo.memory.total)} RAM${storageMsg}`);

    res.json({
      success: true,
      info: systemInfo,
    });
  } catch (error) {
    log("error", "❌ [GET /api/system/info] Error:", error.message);
    res.status(500).json({
      success: false,
      error: error.message,
    });
  }
});

// Helper function to extract storage info from DriverStatus array
function extractStorageInfo(driverStatus, key) {
  if (!Array.isArray(driverStatus)) return null;
  
  const entry = driverStatus.find(([k]) => k === key);
  if (!entry || !entry[1]) return null;
  
  // Parse values like "107.4GB" or "50.5 GB"
  const match = entry[1].match(/([\d.]+)\s*([KMGT]?B)/i);
  if (!match) return null;
  
  const value = parseFloat(match[1]);
  const unit = match[2].toUpperCase();
  
  const multipliers = {
    'B': 1,
    'KB': 1024,
    'MB': 1024 * 1024,
    'GB': 1024 * 1024 * 1024,
    'TB': 1024 * 1024 * 1024 * 1024,
  };
  
  return value * (multipliers[unit] || 1);
}

function formatBytes(bytes) {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
}


// ============================================
// BACKUP & RESTORE API ENDPOINTS
// ============================================

// GET /api/backup/config - Get S3 configuration
app.get("/api/backup/config", asyncHandler(async (req, res) => {
  const config = await getS3Config();

  if (!config) {
    return res.json({
      success: true,
      configured: false,
    });
  }

  // Return config without exposing secrets
  res.json({
    success: true,
    configured: true,
    config: {
      provider: config.provider,
      bucket: config.bucket,
      region: config.region,
      endpoint: config.endpoint,
    },
  });
}));

// POST /api/backup/config - Save S3 configuration
app.post("/api/backup/config", asyncHandler(async (req, res) => {
  log("info", "💾 [POST /api/backup/config] Saving S3 configuration");

  const { provider, endpoint, bucket, accessKey, secretKey, region } = req.body;

  if (!bucket || !accessKey || !secretKey) {
    return res.status(400).json({
      success: false,
      error: "bucket, accessKey, and secretKey are required",
    });
  }

  const config = {
    provider: provider || "AWS",
    endpoint,
    bucket,
    accessKey,
    secretKey,
    region: region || "us-east-1",
  };

  await saveS3Config(config);

  log("info", "✅ [POST /api/backup/config] S3 configuration saved");

  res.json({
    success: true,
    message: "S3 configuration saved successfully",
  });
}));

// GET /api/backup/list - List all backups from S3
app.get("/api/backup/list", asyncHandler(async (req, res) => {
  log("info", "📋 [GET /api/backup/list] Listing backups");

  const config = await getS3Config();

  if (!config) {
    return res.status(400).json({
      success: false,
      error: "S3 not configured. Please configure S3 settings first.",
    });
  }

  const backups = await listBackups(config, log);

  log("info", `✅ [GET /api/backup/list] Found ${backups.length} backup(s)`);

  res.json({
    success: true,
    count: backups.length,
    backups,
  });
}));

// POST /api/backup/create - Create a new backup
app.post("/api/backup/create", asyncHandler(async (req, res) => {
  log("info", "💾 [POST /api/backup/create] Creating backup");

  const { volumes, name } = req.body;

  if (!volumes || !Array.isArray(volumes) || volumes.length === 0) {
    return res.status(400).json({
      success: false,
      error: "volumes array is required and must not be empty",
    });
  }

  const config = await getS3Config();

  if (!config) {
    return res.status(400).json({
      success: false,
      error: "S3 not configured. Please configure S3 settings first.",
    });
  }

  // Verify volumes exist
  const allVolumes = await docker.listVolumes();
  const volumeNames = allVolumes.Volumes?.map(v => v.Name) || [];

  for (const volumeName of volumes) {
    if (!volumeNames.includes(volumeName)) {
      return res.status(404).json({
        success: false,
        error: `Volume '${volumeName}' not found`,
      });
    }
  }

  const result = await createBackup(volumes, config, name, log);

  log("info", `✅ [POST /api/backup/create] Backup job started: ${result.jobId}`);

  res.json({
    success: true,
    ...result,
  });
}));

// GET /api/backup/jobs - Get all backup jobs
app.get("/api/backup/jobs", (req, res) => {
  const jobs = getAllBackupJobs();

  res.json({
    success: true,
    count: jobs.length,
    jobs,
  });
});

// GET /api/backup/jobs/:jobId - Get backup job status
app.get("/api/backup/jobs/:jobId", (req, res) => {
  const job = getBackupJobStatus(req.params.jobId);

  if (!job) {
    return res.status(404).json({
      success: false,
      error: "Job not found",
    });
  }

  res.json({
    success: true,
    job,
  });
});

// POST /api/backup/:backupId/restore - Restore a backup
app.post("/api/backup/:backupId/restore", asyncHandler(async (req, res) => {
  const backupId = req.params.backupId;
  log("info", `🔄 [POST /api/backup/${backupId}/restore] Starting restore`);

  const { volumes, overwrite } = req.body;

  const config = await getS3Config();

  if (!config) {
    return res.status(400).json({
      success: false,
      error: "S3 not configured. Please configure S3 settings first.",
    });
  }

  const result = await restoreBackup(backupId, config, volumes, overwrite, log);

  log("info", `✅ [POST /api/backup/${backupId}/restore] Restore job started: ${result.jobId}`);

  res.json({
    success: true,
    ...result,
  });
}));

// GET /api/restore/jobs - Get all restore jobs
app.get("/api/restore/jobs", (req, res) => {
  const jobs = getAllRestoreJobs();

  res.json({
    success: true,
    count: jobs.length,
    jobs,
  });
});

// GET /api/restore/jobs/:jobId - Get restore job status
app.get("/api/restore/jobs/:jobId", (req, res) => {
  const job = getRestoreJobStatus(req.params.jobId);

  if (!job) {
    return res.status(404).json({
      success: false,
      error: "Job not found",
    });
  }

  res.json({
    success: true,
    job,
  });
});

// DELETE /api/backup/:backupId - Delete a backup
app.delete("/api/backup/:backupId", asyncHandler(async (req, res) => {
  const backupId = req.params.backupId;
  log("info", `🗑️ [DELETE /api/backup/${backupId}] Deleting backup`);

  const config = await getS3Config();

  if (!config) {
    return res.status(400).json({
      success: false,
      error: "S3 not configured",
    });
  }

  await deleteBackup(backupId, config, log);

  log("info", `✅ [DELETE /api/backup/${backupId}] Backup deleted successfully`);

  res.json({
    success: true,
    message: "Backup deleted successfully",
  });
}));

// ============================================
// END BACKUP & RESTORE API ENDPOINTS
// ============================================

// Catch-all route to serve Vue.js app for client-side routing (must be last)
if (process.env.NODE_ENV === "production") {
  app.get("*", (req, res) => {
    res.sendFile(path.join(__dirname, "..", "dist", "index.html"));
  });
}

// Error handling middleware (must be last)
app.use(errorHandler);

// Start server
const PORT = 5252;
app.listen(PORT, "0.0.0.0", () => {
  log("info", "\n" + "=".repeat(50));
  log("info", "🚀 Yantra API Server Started");
  log("info", "=".repeat(50));
  log("info", `📡 Port: ${PORT}`);
  log("info", `🔌 Socket: ${socketPath}`);
  log("info", `📂 Apps directory: ${path.join(__dirname, "..", "apps")}`);
  log("info", `🌐 Access: http://localhost:${PORT}`);
  log("info", "=".repeat(50) + "\n");

  resolveComposeCommand({ socketPath, log }).catch((err) => {
    log("warn", `⚠️  [COMPOSE] ${err.message}`);
  });

  // Start cleanup scheduler (runs every 15 minutes to handle temporary installations)
  log("info", "🧹 Starting automatic cleanup scheduler");
  startCleanupScheduler(11);
});
