import path from "path";
import { access } from "fs/promises";
import { idParamsSchema, sendError } from "../api.js";
import {
  docker, log, appsDir, socketPath,
  getAppsCatalogCached, getContainerEnv, mapWithConcurrency, parseAppLabels,
} from "../shared.js";
import { spawnProcess, getBaseAppId } from "../utils.js";
import { resolveComposeCommand } from "../compose.js";
import { getComposeProcessEnv, getProjectComposeRef, deleteProjectCompose } from "../stack-compose.js";

export default async function containersRoutes(fastify) {

  // GET /api/containers
  fastify.get("/api/containers", async (request, reply) => {
  const containers = await docker.listContainers({ all: true });
  const catalog = await getAppsCatalogCached();
  const catalogMap = new Map(catalog.apps.map(a => [a.id, a]));

  const formattedContainers = await mapWithConcurrency(containers, 8, async (container) => {
    const appLabels = parseAppLabels(container.Labels);
    const composeProject = container.Labels["com.docker.compose.project"];
    let envVars = [];
    try {
      envVars = await getContainerEnv(container.Id);
    } catch (error) {
      log("error", `Failed to get env for container ${container.Id}:`, error.message);
    }

    const baseAppId = getBaseAppId(composeProject) || container.Names[0]?.replace("/", "") || "unknown";
    const appId = appLabels.app || baseAppId;
    const catalogEntry = catalogMap.get(appId) || null;

    return {
      id: container.Id,
      name: container.Names[0]?.replace("/", "") || "unknown",
      image: container.Image,
      imageId: container.ImageID,
      state: container.State,
      status: container.Status,
      created: container.Created,
      ports: container.Ports,
      labels: container.Labels,
      appLabels,
      env: envVars,
      app: {
        id: appId,
        projectId: composeProject || container.Names[0]?.replace("/", "") || "unknown",
        service: appLabels.service || container.Names[0]?.replace("/", "") || "unknown",
        name: catalogEntry?.name || appLabels.service || container.Names[0]?.replace("/", "") || "unknown",
        logo: catalogEntry?.logo || null,
        tags: catalogEntry?.tags || [],
        ports: Array.isArray(catalogEntry?.ports) ? catalogEntry.ports : [],
        short_description: catalogEntry?.short_description || "",
        description: catalogEntry?.description || "",
        usecases: catalogEntry?.usecases || [],
        website: catalogEntry?.website || null,
      },
    };
  });

  const yantrProjects = new Set();
  formattedContainers.forEach(c => {
    const project = c.labels?.["com.docker.compose.project"];
    if (c.appLabels?.app && project) yantrProjects.add(project);
  });

  const filteredContainers = formattedContainers.filter(c => {
    const hasYantrLabel = !!(c.appLabels?.app);
    const project = c.labels?.["com.docker.compose.project"];
    const isPartOfYantrStack = project && yantrProjects.has(project);
    return hasYantrLabel || !isPartOfYantrStack;
  });

    return reply.send({ success: true, count: filteredContainers.length, containers: filteredContainers });
  });

  // GET /api/containers/:id
  fastify.get("/api/containers/:id", async (request, reply) => {
    const container = docker.getContainer(request.params.id);
    const info = await container.inspect();
  const appLabels = parseAppLabels(info.Config.Labels);
  const composeProject = info.Config.Labels["com.docker.compose.project"];

  const catalog = await getAppsCatalogCached();
  const catalogMap = new Map(catalog.apps.map(a => [a.id, a]));
  const baseAppId = getBaseAppId(composeProject) || info.Name.replace("/", "") || "unknown";
  const appId = appLabels.app || baseAppId;
  const catalogEntry = catalogMap.get(appId) || null;
  const allPorts = info.NetworkSettings.Ports;

    return reply.send({
      success: true,
      container: {
        id: info.Id,
        name: info.Name.replace("/", ""),
        image: info.Config.Image,
        imageId: info.Image,
        state: info.State.Status || (info.State.Running ? "running" : "stopped"),
        stateDetails: info.State,
        created: info.Created,
        ports: allPorts,
        mounts: info.Mounts,
        env: info.Config.Env,
        labels: appLabels,
        expireAt: info.Config.Labels?.["yantr.expireAt"] || null,
        app: {
          id: appId,
          projectId: composeProject || info.Name.replace("/", "") || "unknown",
          service: appLabels.service || info.Name.replace("/", ""),
          name: catalogEntry?.name || appLabels.service || info.Name.replace("/", ""),
          logo: catalogEntry?.logo || null,
          tags: catalogEntry?.tags || [],
          ports: Array.isArray(catalogEntry?.ports) ? catalogEntry.ports : [],
          short_description: catalogEntry?.short_description || "",
          description: catalogEntry?.description || "",
          usecases: catalogEntry?.usecases || [],
          website: catalogEntry?.website || null,
        },
      },
    });
  });

  // GET /api/containers/:id/stats
  fastify.get("/api/containers/:id/stats", async (request, reply) => {
    const container = docker.getContainer(request.params.id);
  const stats = await container.stats({ stream: false });

  const cpuDelta = stats.cpu_stats.cpu_usage.total_usage - stats.precpu_stats.cpu_usage.total_usage;
  const systemDelta = stats.cpu_stats.system_cpu_usage - stats.precpu_stats.system_cpu_usage;
  const cpuPercent = systemDelta > 0 ? (cpuDelta / systemDelta) * stats.cpu_stats.online_cpus * 100 : 0;

  const memoryRawUsage = stats?.memory_stats?.usage || 0;
  const memoryLimit = stats?.memory_stats?.limit || 0;
  const memStats = stats?.memory_stats?.stats || {};
  const memoryCache =
    typeof memStats.inactive_file === "number" ? memStats.inactive_file
    : typeof memStats.cache === "number" ? memStats.cache
    : 0;
  const memoryUsage = Math.max(0, memoryRawUsage - memoryCache);
  const memoryPercent = memoryLimit > 0 ? (memoryUsage / memoryLimit) * 100 : 0;

  let networkRx = 0, networkTx = 0;
  if (stats.networks) {
    Object.values(stats.networks).forEach(n => { networkRx += n.rx_bytes || 0; networkTx += n.tx_bytes || 0; });
  }

  let blockRead = 0, blockWrite = 0;
  if (stats.blkio_stats?.io_service_bytes_recursive) {
    stats.blkio_stats.io_service_bytes_recursive.forEach(io => {
      if (io.op === "Read") blockRead += io.value;
      if (io.op === "Write") blockWrite += io.value;
    });
  }

    return reply.send({
      success: true,
      stats: {
        cpu: { percent: cpuPercent.toFixed(2), usage: stats.cpu_stats.cpu_usage.total_usage },
        memory: { usage: memoryUsage, rawUsage: memoryRawUsage, cache: memoryCache, limit: memoryLimit, percent: memoryPercent.toFixed(2) },
        network: { rx: networkRx, tx: networkTx },
        blockIO: { read: blockRead, write: blockWrite },
      },
    });
  });

  // GET /api/containers/:id/logs
  fastify.get("/api/containers/:id/logs", async (request, reply) => {
    const container = docker.getContainer(request.params.id);
    const tailLines = request.query.tail || 100;
    const rawLogs = await container.logs({ stdout: true, stderr: true, tail: tailLines, timestamps: true });
    const logString = rawLogs.toString("utf8");
    const lines = logString.split("\n").filter(l => l.trim()).map(l => l.substring(8));
    return reply.send({ success: true, logs: lines });
  });

  // DELETE /api/containers/:id
  fastify.delete("/api/containers/:id", { schema: { params: idParamsSchema } }, async (request, reply) => {
    log("info", `🗑️  [DELETE /api/containers/:id] Remove request for: ${request.params.id}`);
    try {
      const container = docker.getContainer(request.params.id);
      const info = await container.inspect();
      const containerName = info.Name.replace("/", "");
      const labels = info.Config.Labels || {};
      const composeProject = labels["com.docker.compose.project"];

      if (composeProject) {
        const baseAppId = getBaseAppId(composeProject);
        const appPath = path.join(appsDir, baseAppId);

        try {
          const { composePath, composeFile } = await getProjectComposeRef(appPath, composeProject);
          await access(composePath);
          const composeCmd = await resolveComposeCommand({ socketPath });
          const composeEnv = await getComposeProcessEnv(appPath, composeProject, { DOCKER_HOST: `unix://${socketPath}` });
          const { stdout, stderr, exitCode } = await spawnProcess(
            composeCmd.command,
            [...composeCmd.args, "-p", composeProject, "-f", composeFile, "down"],
            { cwd: appPath, env: composeEnv }
          );
          if (exitCode !== 0) throw new Error(`docker compose down failed: ${stderr}`);
          await deleteProjectCompose(appPath, composeProject);
          return reply.send({ success: true, message: `App stack '${composeProject}' removed successfully`, container: containerName, stackRemoved: true, volumesRemoved: [], volumesFailed: [], output: stdout });
        } catch (err) {
          log("info", `⚠️  Compose file not found for ${composeProject}, falling back to single container deletion`);
        }
      }

      const volumeNames = info.Mounts.filter(m => m.Type === "volume").map(m => m.Name);
      if (info.State.Running) await container.stop();
      await container.remove();

      return reply.send({ success: true, message: `Container '${containerName}' removed successfully`, container: containerName, volumesPreserved: volumeNames });
    } catch (error) {
      log("error", `❌ [DELETE /api/containers/:id] Error:`, error.message);
      return sendError(reply, 500, { code: "CONTAINER_REMOVE_FAILED", message: "Failed to remove container", details: { cause: error.message } });
    }
  });

}
