import path from "path";
import { readFile } from "fs/promises";
import { nonEmptyStringSchema, projectIdParamsSchema, sendError } from "../api.js";
import { docker, getAppsCatalogCached, parseAppLabels, appsDir, socketPath, log } from "../shared.js";
import { getBaseAppId } from "../utils.js";
import { resolveComposeCommand } from "../compose.js";
import { spawnProcess } from "../utils.js";
import {
  applyCurrentPublishedPorts,
  buildProjectComposeContent,
  getComposeProcessEnv,
  parseDockerPortInput,
  parseCompose,
  setServicePortBindings,
  stringifyCompose,
  getProjectComposeRef,
  writeProjectCompose,
} from "../stack-compose.js";

const stackPortBodySchema = {
  type: "object",
  required: ["portMapping"],
  additionalProperties: false,
  properties: {
    serviceName: { type: "string" },
    portMapping: nonEmptyStringSchema,
  },
};

export default async function stacksRoutes(fastify) {
  async function listProjectContainers(projectId) {
    const allContainers = await docker.listContainers({ all: true });
    return allContainers.filter(c => c.Labels?.["com.docker.compose.project"] === projectId);
  }

  function collectPublishedPortsByService(containers) {
    const servicePorts = {};
    for (const container of containers) {
      const composeService = container.Labels?.["com.docker.compose.service"];
      if (!composeService) continue;
      if (!servicePorts[composeService]) servicePorts[composeService] = [];
      for (const port of (container.Ports || [])) {
        if (!port.PublicPort) continue;
        const key = `${port.PublicPort}:${port.PrivatePort}:${port.Type}`;
        if (servicePorts[composeService].some(existing => `${existing.hostPort}:${existing.containerPort}:${existing.protocol}` === key)) continue;
        servicePorts[composeService].push({ hostPort: port.PublicPort, containerPort: port.PrivatePort, protocol: port.Type });
      }
    }
    return servicePorts;
  }

  function findPublishedPort(servicePorts, serviceName, containerPort, protocol) {
    return (servicePorts[serviceName] || []).find((binding) =>
      Number(binding.containerPort) === Number(containerPort)
      && String(binding.protocol).toLowerCase() === String(protocol).toLowerCase()
    ) || null;
  }

  // GET /api/stacks/:projectId
  fastify.get("/api/stacks/:projectId", async (request, reply) => {
    const { projectId } = request.params;
    const catalog = await getAppsCatalogCached();
    const catalogMap = new Map(catalog.apps.map(a => [a.id, a]));

    const projectContainers = await listProjectContainers(projectId);

    if (projectContainers.length === 0) {
      return reply.code(404).send({ success: false, error: "Stack not found or no containers" });
    }

    const baseAppId = getBaseAppId(projectId);
    const catalogEntry = catalogMap.get(baseAppId) || null;

    // Build published ports (deduplicated by key)
    const publishedPortsMap = new Map();
    for (const c of projectContainers) {
      const svcLabel = c.Labels["yantr.service"] || c.Names[0]?.replace("/", "") || "unknown";
      for (const p of (c.Ports || [])) {
        if (p.PublicPort) {
          const key = `${p.PublicPort}:${p.PrivatePort}:${p.Type}:${svcLabel}`;
          if (!publishedPortsMap.has(key)) {
            publishedPortsMap.set(key, { hostPort: p.PublicPort, containerPort: p.PrivatePort, protocol: p.Type, service: svcLabel });
          }
        }
      }
    }
    const publishedPorts = [...publishedPortsMap.values()].sort((a, b) => a.hostPort - b.hostPort);

    const services = await Promise.all(projectContainers.map(async c => {
      const appLabels = parseAppLabels(c.Labels);

      const mountsMap = new Map();
      for (const m of (c.Mounts || [])) {
        if (!mountsMap.has(m.Destination)) {
          mountsMap.set(m.Destination, { type: m.Type, source: m.Source || "", destination: m.Destination, mode: m.Mode || "", name: m.Name || null });
        }
      }

      return {
        id: c.Id,
        name: c.Names[0]?.replace("/", "") || "unknown",
        composeService: c.Labels?.["com.docker.compose.service"] || null,
        image: c.Image,
        state: c.State,
        status: c.Status,
        created: c.Created,
        rawPorts: c.Ports || [],
        mounts: [...mountsMap.values()],
        service: c.Labels["yantr.service"] || appLabels.service || c.Names[0]?.replace("/", "") || "unknown",
        hasYantrLabel: !!(appLabels.app),
      };
    }));

    services.sort((a, b) => {
      if (a.hasYantrLabel !== b.hasYantrLabel) return a.hasYantrLabel ? -1 : 1;
      return a.service.localeCompare(b.service);
    });

    // Derive caddy proxy info from labels on the project containers themselves
    const caddyProxies = projectContainers
      .filter(c => c.Labels?.["yantr.caddy.enabled"] === "true")
      .map(c => ({
        servePort: Number(c.Labels["yantr.caddy.serve.port"]) || null,
        targetPort: Number(c.Labels["yantr.caddy.target.port"]) || null,
        authEnabled: !!(c.Labels["yantr.caddy.auth.user"]),
        authUser: c.Labels["yantr.caddy.auth.user"] || null,
        service: c.Labels?.["com.docker.compose.service"] || null,
      }))
      .filter(p => p.servePort);

    return reply.send({
      success: true,
      stack: {
        projectId,
        appId: baseAppId,
        app: catalogEntry ? { name: catalogEntry.name, logo: catalogEntry.logo, tags: catalogEntry.tags, ports: Array.isArray(catalogEntry.ports) ? catalogEntry.ports : [], short_description: catalogEntry.short_description, website: catalogEntry.website, customapp: !!catalogEntry.customapp } : null,
        publishedPorts,
        services,
        caddyProxies,
      },
    });
  });

  fastify.post("/api/stacks/:projectId/ports", { schema: { params: projectIdParamsSchema, body: stackPortBodySchema } }, async (request, reply) => {
    const { projectId } = request.params;
    const requestedServiceName = String(request.body?.serviceName || "").trim();
    const parsedPort = parseDockerPortInput(request.body?.portMapping);

    if (!parsedPort) {
      return sendError(reply, 400, { code: "INVALID_PORT_MAPPING", message: "portMapping must be a valid Docker port format like 9000, 9000:9000, or 53:53/udp" });
    }

    const projectContainers = await listProjectContainers(projectId);
    if (projectContainers.length === 0) {
      return sendError(reply, 404, { code: "STACK_NOT_FOUND", message: "Stack not found or no containers" });
    }

    const availableServices = [...new Set(projectContainers.map((container) => container.Labels?.["com.docker.compose.service"]).filter(Boolean))];
    let serviceName = requestedServiceName;
    if (!serviceName && availableServices.length === 1) {
      serviceName = availableServices[0];
    }
    if (!serviceName) {
      return sendError(reply, 400, { code: "SERVICE_NAME_REQUIRED", message: "serviceName is required for multi-service stacks" });
    }
    if (!availableServices.includes(serviceName)) {
      return sendError(reply, 400, { code: "SERVICE_NOT_IN_STACK", message: `Service '${serviceName}' is not part of this stack` });
    }

    const { containerPort, hostPort, protocol, hasExplicitHostPort } = parsedPort;

    const baseAppId = getBaseAppId(projectId);
    const appPath = path.join(appsDir, baseAppId);
    const composeRef = await getProjectComposeRef(appPath, projectId);
    let composeContent;

    if (composeRef.isProjectCompose) {
      composeContent = await readFile(composeRef.composePath, "utf-8");
    } else {
      const baseComposeContent = await readFile(composeRef.composePath, "utf-8");
      const bootstrappedCompose = parseCompose(buildProjectComposeContent(baseComposeContent, { projectId, appId: baseAppId }));
      applyCurrentPublishedPorts(bootstrappedCompose, collectPublishedPortsByService(projectContainers));
      composeContent = stringifyCompose(bootstrappedCompose);
    }

    const compose = parseCompose(composeContent);
    const service = compose.services?.[serviceName];
    if (!service) {
      return sendError(reply, 400, { code: "SERVICE_NOT_FOUND_IN_COMPOSE", message: `Service '${serviceName}' not found in stack compose` });
    }

    if (hasExplicitHostPort) {
      const allRunningContainers = await docker.listContainers({ all: false });
      const conflictingContainer = allRunningContainers.find((container) =>
        (container.Ports || []).some((port) => {
          if (port.PublicPort !== hostPort || String(port.Type).toLowerCase() !== protocol) return false;
          const sameMapping = container.Labels?.["com.docker.compose.project"] === projectId
            && container.Labels?.["com.docker.compose.service"] === serviceName
            && port.PrivatePort === containerPort;
          return !sameMapping;
        })
      );

      if (conflictingContainer) {
        return sendError(reply, 409, {
          code: "PORT_ALREADY_IN_USE",
          message: `Port ${hostPort}/${protocol} is already in use by ${conflictingContainer.Names?.[0]?.replace("/", "") || "another container"}`,
        });
      }
    }

    const currentBindings = collectPublishedPortsByService(projectContainers)[serviceName] || [];
    const nextBindings = currentBindings.filter((binding) => !(Number(binding.containerPort) === containerPort && String(binding.protocol).toLowerCase() === protocol));
    nextBindings.push({ containerPort, hostPort, protocol });
    setServicePortBindings(service, nextBindings);

    const nextComposeContent = stringifyCompose(compose);
    const { composeFile } = await writeProjectCompose(appPath, projectId, nextComposeContent);

    const composeCmd = await resolveComposeCommand({ socketPath, log });
    const composeEnv = await getComposeProcessEnv(appPath, projectId, { DOCKER_HOST: `unix://${socketPath}` });
    const { stdout, stderr, exitCode } = await spawnProcess(
      composeCmd.command,
      [...composeCmd.args, "-p", projectId, "-f", composeFile, "up", "-d"],
      { cwd: appPath, env: composeEnv }
    );

    if (exitCode !== 0) {
      return sendError(reply, 500, { code: "STACK_REDEPLOY_FAILED", message: `docker compose failed: ${stderr || stdout}` });
    }

    const updatedContainers = await listProjectContainers(projectId);
    const updatedServicePorts = collectPublishedPortsByService(updatedContainers);
    const publishedPort = findPublishedPort(updatedServicePorts, serviceName, containerPort, protocol);
    const effectiveHostPort = publishedPort?.hostPort ?? hostPort;
    const effectivePortMapping = effectiveHostPort != null
      ? `${effectiveHostPort}:${containerPort}${protocol === "tcp" ? "" : `/${protocol}`}`
      : `${containerPort}${protocol === "tcp" ? "" : `/${protocol}`}`;

    return reply.send({
      success: true,
      message: `Opened ${effectivePortMapping} on service '${serviceName}'`,
      port: {
        serviceName,
        hostPort: effectiveHostPort,
        containerPort,
        protocol,
        portMapping: effectivePortMapping,
        requestedPortMapping: String(request.body?.portMapping || "").trim(),
        autoAssigned: !hasExplicitHostPort,
      },
      output: stdout,
      warnings: stderr || null,
    });
  });
}
