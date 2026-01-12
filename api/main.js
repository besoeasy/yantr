import express from "express";
import Docker from "dockerode";
import cors from "cors";
import fs from "fs";
import path from "path";
import { exec } from "child_process";
import util from "util";
import { fileURLToPath } from "url";
import { dirname } from "path";
import { createRequire } from "module";

import { startCleanupScheduler, cleanupExpiredApps } from "./cleanup.js";
import "./updater.js";

const require = createRequire(import.meta.url);
const packageJson = require("../package.json");
const fsPromises = fs.promises;
const execPromise = util.promisify(exec);

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const app = express();

// Docker socket path
const socketPath = process.env.DOCKER_SOCKET || "/var/run/docker.sock";

const docker = new Docker({ socketPath });

// System architecture cache
let systemArchitecture = null;

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
    const { stdout } = await execPromise("uname -m");
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
    const arch = await getSystemArchitecture();
    log("info", `🔍 Checking architecture support for ${imageName} on ${arch}`);

    // Try to inspect the image manifest to check supported platforms
    // First, try to pull manifest without actually pulling the image
    const command = `docker image inspect ${imageName} --format='{{.Architecture}}' 2>/dev/null || docker manifest inspect ${imageName} 2>/dev/null`;

    try {
      const { stdout } = await execPromise(command);
      const output = stdout.trim();

      // If we get architecture directly
      if (output && !output.includes("{") && !output.includes("[")) {
        const imageArch = output.toLowerCase();
        log("info", `  Image architecture: ${imageArch}`);

        // Check if architectures match
        if (imageArch === arch || (imageArch === "amd64" && arch === "amd64") || (imageArch === "arm64" && arch === "arm64")) {
          return { supported: true, imageArch, systemArch: arch };
        } else {
          return { supported: false, imageArch, systemArch: arch };
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

          return {
            supported: isSupported,
            imageArch: supportedArchs.join(", "),
            systemArch: arch,
          };
        } catch (parseError) {
          log("error", "  Failed to parse manifest:", parseError.message);
        }
      }
    } catch (inspectError) {
      // If inspection fails, the image might not be available locally or remotely
      log("info", `  Could not inspect image ${imageName}, will attempt pull`);
    }

    // If we can't determine from manifest, we'll try to pull and see if it fails
    return { supported: "unknown", imageArch: "unknown", systemArch: arch };
  } catch (error) {
    log("error", "❌ Error checking architecture support:", error.message);
    return { supported: "unknown", imageArch: "unknown", systemArch: arch };
  }
}

// Helper function to extract image name from compose file
async function getImageFromCompose(composePath) {
  try {
    const content = await fsPromises.readFile(composePath, "utf8");
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

// GET /api/updates/pending - Check for pending updates
app.get("/api/updates/pending", async (req, res) => {
  try {
    const images = await docker.listImages({
      filters: { reference: ['ghcr.io/besoeasy/yantra'] }
    });

    const hasPendingUpdate = images.length > 1;

    res.json({
      success: true,
      hasPendingUpdate,
      imageCount: images.length,
      message: hasPendingUpdate 
        ? 'A new Yantra update is ready. Please reboot your device to apply it, or install the Watchtower app for automatic updates.'
        : 'You are running the latest version'
    });
  } catch (error) {
    log('error', 'Failed to check for pending updates:', error.message);
    res.status(500).json({
      success: false,
      error: 'Failed to check for pending updates'
    });
  }
});

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
app.get("/api/containers", async (req, res) => {
  // log('info', '📦 [GET /api/containers] Fetching all containers');
  try {
    const containers = await docker.listContainers({ all: true });
    // log('info', `📦 [GET /api/containers] Found ${containers.length} containers`);

    const formattedContainers = await Promise.all(
      containers.map(async (container) => {
        const appLabels = parseAppLabels(container.Labels);
        const composeProject = container.Labels["com.docker.compose.project"];

        // Fetch full container details to get environment variables
        let envVars = [];
        try {
          const containerObj = docker.getContainer(container.Id);
          const info = await containerObj.inspect();
          envVars = info.Config.Env || [];
        } catch (error) {
          log("error", `Failed to get env for container ${container.Id}:`, error.message);
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
            id: composeProject || container.Names[0]?.replace("/", "") || "unknown", // Add app ID
            name: appLabels.name || container.Names[0]?.replace("/", "") || "unknown",
            logo: appLabels.logo ? (appLabels.logo.includes("://") ? appLabels.logo : `https://dweb.link/ipfs/${appLabels.logo}`) : null,
            category: appLabels.category || "uncategorized",
            port: appLabels.port || null,
            description: appLabels.description || "",
            docs: appLabels.docs || null,
            website: appLabels.website || null,
          },
        };
      })
    );

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

    // log('info', `✅ [GET /api/containers] Returning ${filteredContainers.length} formatted containers (filtered from ${containers.length})`);
    res.json({
      success: true,
      count: filteredContainers.length,
      containers: filteredContainers,
    });
  } catch (error) {
    log("error", "❌ [GET /api/containers] Error:", error.message);
    res.status(500).json({
      success: false,
      error: error.message,
    });
  }
});

// GET /api/containers/:id - Get single container details
app.get("/api/containers/:id", async (req, res) => {
  // log('info', `🔍 [GET /api/containers/:id] Fetching container: ${req.params.id}`);
  try {
    const container = docker.getContainer(req.params.id);
    const info = await container.inspect();
    // log('info', `✅ [GET /api/containers/:id] Found container: ${info.Name}`);
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
        app: {
          name: appLabels.name || info.Name.replace("/", ""),
          logo: appLabels.logo
            ? appLabels.logo.includes("://")
              ? appLabels.logo
              : `https://dweb.link/ipfs/${appLabels.logo}`
            : "https://dweb.link/ipfs/QmVdbRUyvZpXCsVJAs7fo1PJPXaPHnWRtSCFx6jFTGaG5i",
          category: appLabels.category || "uncategorized",
          port: appLabels.port || null,
          description: appLabels.description || "",
          docs: appLabels.docs || null,
          website: appLabels.website || null,
        },
      },
    });
  } catch (error) {
    log("error", `❌ [GET /api/containers/:id] Container not found: ${req.params.id}`, error.message);
    res.status(404).json({
      success: false,
      error: "Container not found",
      message: error.message,
    });
  }
});

// GET /api/containers/:id/stats - Get real-time stats for a container
app.get("/api/containers/:id/stats", async (req, res) => {
  try {
    const container = docker.getContainer(req.params.id);
    const stats = await container.stats({ stream: false });

    // Calculate CPU percentage
    const cpuDelta = stats.cpu_stats.cpu_usage.total_usage - stats.precpu_stats.cpu_usage.total_usage;
    const systemDelta = stats.cpu_stats.system_cpu_usage - stats.precpu_stats.system_cpu_usage;
    const cpuPercent = systemDelta > 0 ? (cpuDelta / systemDelta) * stats.cpu_stats.online_cpus * 100 : 0;

    // Calculate memory usage
    const memoryUsage = stats.memory_stats.usage || 0;
    const memoryLimit = stats.memory_stats.limit || 0;
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
  } catch (error) {
    log("error", `❌ [GET /api/containers/:id/stats] Error:`, error.message);
    res.status(500).json({
      success: false,
      error: error.message,
    });
  }
});

// GET /api/containers/:id/logs - Get logs from a container
app.get("/api/containers/:id/logs", async (req, res) => {
  try {
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
  } catch (error) {
    log("error", `❌ [GET /api/containers/:id/logs] Error:`, error.message);
    res.status(500).json({
      success: false,
      error: error.message,
    });
  }
});

// GET /api/apps - List available apps from /apps directory
app.get("/api/apps", async (req, res) => {
  // log('info', '🏪 [GET /api/apps] Scanning apps directory');
  try {
    const appsDir = path.join(__dirname, "..", "apps");
    const apps = [];

    // Read all directories in /apps
    const entries = await fsPromises.readdir(appsDir, { withFileTypes: true });
    // log('info', `🏪 [GET /api/apps] Found ${entries.length} entries in apps directory`);

    for (const entry of entries) {
      if (entry.isDirectory()) {
        const appPath = path.join(appsDir, entry.name);
        const composePath = path.join(appPath, "compose.yml");

        // Check if compose.yml exists
        try {
          await fsPromises.access(composePath);

          // Read and parse compose file to extract labels
          const composeContent = await fsPromises.readFile(composePath, "utf8");
          const labels = {};

          // Simple regex to extract labels (works for both formats)
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
          // Match patterns like: - "8080:80", "53:53/tcp", "53:53/udp", "${PORT:-8080}:80"
          const portRegex = /-\s*["']?(?:\$\{([A-Z_]+):-)?(\d+)(?:\})?:(\d+)(?:\/(tcp|udp))?["']?/g;
          while ((match = portRegex.exec(composeContent)) !== null) {
            portMappings.push({
              hostPort: match[2], // The host port (left side)
              containerPort: match[3], // The container port (right side)
              protocol: match[4] || "tcp", // tcp/udp, defaults to tcp
              envVar: match[1] || null, // Environment variable if used
            });
          }

          // Parse yantra.port to identify named ports (ports with descriptions)
          const namedPorts = new Set();
          if (labels.port) {
            const portDescRegex = /(\d+)\s*\(([^-\)]+)\s*-\s*([^)]+)\)/g;
            let portMatch;
            while ((portMatch = portDescRegex.exec(labels.port)) !== null) {
              namedPorts.add(portMatch[1]); // Add the port number to named ports set
            }
          }

          // Mark which ports are named (have descriptions in yantra.port)
          portMappings.forEach((port) => {
            port.isNamed = namedPorts.has(port.hostPort);
          });

          apps.push({
            id: entry.name,
            name: labels.name || entry.name,
            logo: labels.logo
              ? labels.logo.includes("://")
                ? labels.logo
                : `https://dweb.link/ipfs/${labels.logo}`
              : "https://dweb.link/ipfs/QmVdbRUyvZpXCsVJAs7fo1PJPXaPHnWRtSCFx6jFTGaG5i",
            category: labels.category || "uncategorized",
            port: labels.port || null,
            description: labels.description || "",

            website: labels.website || null, // Map website label
            docs: labels.docs || null,
            path: appPath,
            composePath: composePath,
            environment: envVars,
            ports: portMappings,
          });
        } catch (err) {
          // Skip if compose.yml doesn't exist
          console.warn(`  ⚠ No compose.yml found for ${entry.name}`);
        }
      }
    }

    // log('info', `✅ [GET /api/apps] Returning ${apps.length} apps`);
    res.json({
      success: true,
      count: apps.length,
      apps: apps,
    });
  } catch (error) {
    log("error", "❌ [GET /api/apps] Error:", error.message);
    res.status(500).json({
      success: false,
      error: error.message,
    });
  }
});

// GET /api/apps/:id/check-arch - Check architecture compatibility for an app
app.get("/api/apps/:id/check-arch", async (req, res) => {
  // log('info', `🔍 [GET /api/apps/:id/check-arch] Checking architecture for: ${req.params.id}`);
  try {
    const appId = req.params.id;
    const appsDir = path.join(__dirname, "..", "apps");
    const appPath = path.join(appsDir, appId);
    const composePath = path.join(appPath, "compose.yml");

    // Verify compose file exists
    try {
      await fsPromises.access(composePath);
    } catch (err) {
      log("error", `❌ [GET /api/apps/:id/check-arch] App not found: ${appId}`);
      return res.status(404).json({
        success: false,
        error: `App '${appId}' not found`,
      });
    }

    // Get image name from compose file
    const imageName = await getImageFromCompose(composePath);
    if (!imageName) {
      log("error", `❌ [GET /api/apps/:id/check-arch] Could not extract image name from compose file`);
      return res.status(400).json({
        success: false,
        error: "Could not extract image name from compose file",
      });
    }

    // Check architecture support
    const archCheck = await checkImageArchitectureSupport(imageName);

    // log('info', `✅ [GET /api/apps/:id/check-arch] Architecture check complete`);
    res.json({
      success: true,
      appId: appId,
      image: imageName,
      supported: archCheck.supported,
      systemArch: archCheck.systemArch,
      imageArch: archCheck.imageArch,
    });
  } catch (error) {
    log("error", `❌ [GET /api/apps/:id/check-arch] Error:`, error.message);
    res.status(500).json({
      success: false,
      error: error.message,
    });
  }
});

// POST /api/deploy - Deploy a compose file from /apps directory
app.post("/api/deploy", async (req, res) => {
  log("info", "🚀 [POST /api/deploy] Deploy request received");
  try {
    const { appId, environment, expiresIn } = req.body;
    log("info", `🚀 [POST /api/deploy] Deploying app: ${appId}`);
    if (environment) {
      log("info", `🚀 [POST /api/deploy] Custom environment:`, environment);
    }
    if (expiresIn) {
      log("info", `🚀 [POST /api/deploy] Temporary installation: ${expiresIn} hours`);
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
      composeContent = await fsPromises.readFile(composePath, "utf8");
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
        await fsPromises.writeFile(envPath, envContent);
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
        await fsPromises.writeFile(tempComposePath, modifiedComposeContent);
        log("info", `🚀 [POST /api/deploy] Created temporary compose file with expiration labels`);
      }
    }

    // Deploy using docker compose (Docker will auto-assign ports)
    const composeFile = expiresIn ? ".compose.tmp.yml" : "compose.yml";
    const command = `docker compose -f "${composeFile}" up -d`;
    log("info", `🚀 [POST /api/deploy] Executing: ${command}`);

    try {
      const { stdout, stderr } = await execPromise(command, {
        cwd: appPath,
        env: {
          ...process.env,
          DOCKER_HOST: `unix://${socketPath}`,
        },
      });

      log("info", `✅ [POST /api/deploy] Deployment successful for ${appId}`);
      log("info", `   stdout: ${stdout.trim()}`);
      if (stderr) log("info", `   stderr: ${stderr.trim()}`);

      // Cleanup temporary compose file if it exists
      if (expiresIn) {
        try {
          const tempComposePath = path.join(appPath, ".compose.tmp.yml");
          await fsPromises.unlink(tempComposePath);
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
  // log('info', '🖼️  [GET /api/images] Fetching all images');
  try {
    const images = await docker.listImages();
    const containers = await docker.listContainers({ all: true });

    // log('info', `🖼️  [GET /api/images] Found ${images.length} images`);

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

    // log('info', `✅ [GET /api/images] Returning ${formattedImages.length} images (${unusedImages.length} unused)`);
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
      const appPath = path.join(appsDir, composeProject);
      const composePath = path.join(appPath, "compose.yml");

      try {
        await fsPromises.access(composePath);
        log("info", `🗑️  [DELETE /api/containers/:id] Found managed app at ${appPath}, shutting down stack...`);

        // Execute docker compose down
        const command = `docker compose down -v`; // -v removes named volumes declared in 'volumes' section

        const { stdout, stderr } = await execPromise(command, {
          cwd: appPath,
          env: {
            ...process.env,
            DOCKER_HOST: `unix://${socketPath}`,
          },
        });

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

    // Remove volumes
    const removedVolumes = [];
    const failedVolumes = [];

    for (const volumeName of volumeNames) {
      try {
        log("info", `🗑️  [DELETE /api/containers/:id] Removing volume: ${volumeName}`);
        const volume = docker.getVolume(volumeName);
        await volume.remove();
        removedVolumes.push(volumeName);
      } catch (err) {
        log("error", `⚠️  [DELETE /api/containers/:id] Failed to remove volume ${volumeName}:`, err.message);
        failedVolumes.push({ name: volumeName, error: err.message });
      }
    }

    log("info", `✅ [DELETE /api/containers/:id] Successfully removed ${containerName}`);
    res.json({
      success: true,
      message: `Container '${containerName}' removed successfully`,
      container: containerName,
      volumesRemoved: removedVolumes,
      volumesFailed: failedVolumes,
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

// Root endpoint
app.get("/", (req, res) => {
  // log('info', '🏠 [GET /] Root endpoint accessed');
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

// GET /api/cleanup - Manually trigger cleanup of expired apps
app.get("/api/cleanup", async (req, res) => {
  log("info", "🧹 [GET /api/cleanup] Manual cleanup triggered");
  try {
    const results = await cleanupExpiredApps();
    res.json({
      success: true,
      results,
    });
  } catch (error) {
    log("error", "❌ [GET /api/cleanup] Error:", error.message);
    res.status(500).json({
      success: false,
      error: error.message,
    });
  }
});

// Catch-all route to serve Vue.js app for client-side routing (must be last)
if (process.env.NODE_ENV === "production") {
  app.get("*", (req, res) => {
    res.sendFile(path.join(__dirname, "..", "dist", "index.html"));
  });
}

// Error handling middleware
app.use((err, req, res, next) => {
  log("error", err.stack);
  res.status(500).json({
    success: false,
    error: "Internal server error",
    message: err.message,
  });
});

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

  // Start cleanup scheduler (runs every 15 minutes to handle temporary installations)
  log("info", "🧹 Starting automatic cleanup scheduler");
  startCleanupScheduler(15);
});
