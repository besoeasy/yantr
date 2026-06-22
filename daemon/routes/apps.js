import path from "path";
import { readFile, access } from "fs/promises";
import YAML from "yaml";
import {
  nonEmptyStringSchema,
  positiveIntegerSchema,
  scalarMapSchema,
  sendError,
  stringOrIntegerSchema,
} from "../api.js";
import {
  docker, log, appsDir, socketPath,
  getAppsCatalogCached, checkImageArchitectureSupport, getImageFromCompose,
} from "../shared.js";
import { spawnProcess, NotFoundError, BadRequestError } from "../utils.js";
import { resolveComposeCommand } from "../compose.js";
import { buildProjectComposeContent, getComposeProcessEnv, writeProjectCompose, writeProjectEnv } from "../stack-compose.js";
import { trackInstall } from "../telemetry.js";

const deploySchema = {
  body: {
    type: "object",
    required: ["appId"],
    additionalProperties: false,
    properties: {
      appId: nonEmptyStringSchema,
      environment: scalarMapSchema,
      extraEnv: scalarMapSchema,
      expiresIn: positiveIntegerSchema,
      instanceId: positiveIntegerSchema,
      masterApp: nonEmptyStringSchema,
      customPortMappings: {
        type: "object",
        additionalProperties: stringOrIntegerSchema,
      },
    },
  },
};

export default async function appsRoutes(fastify) {

  // GET /api/apps
  fastify.get("/api/apps", async (request, reply) => {
    const forceRefresh = request.query.refresh === "1" || request.query.refresh === "true";
    const { apps, count } = await getAppsCatalogCached({ forceRefresh });
    return reply.send({ success: true, count, apps });
  });

  // GET /api/apps/:id/check-arch
  fastify.get("/api/apps/:id/check-arch", async (request, reply) => {
    const appId = request.params.id;
    const composePath = path.join(appsDir, appId, "compose.yml");
    try { await access(composePath); } catch { throw new NotFoundError(`App '${appId}' not found`); }

    const imageName = await getImageFromCompose(composePath);
    if (!imageName) throw new BadRequestError("Could not extract image name from compose file");

    const archCheck = await checkImageArchitectureSupport(imageName);
    return reply.send({ success: true, appId, image: imageName, supported: archCheck.supported, systemArch: archCheck.systemArch, imageArch: archCheck.imageArch });
  });

  // POST /api/deploy
  fastify.post("/api/deploy", { schema: deploySchema }, async (request, reply) => {
    log("info", "🚀 [POST /api/deploy] Deploy request received");
    try {
      const { appId, environment, extraEnv, expiresIn, customPortMappings, instanceId, masterApp } = request.body;
      log("info", `🚀 [POST /api/deploy] Deploying app: ${appId}${instanceId > 1 ? ` (Instance #${instanceId})` : ""}`);

    const appPath = path.join(appsDir, appId);
    const composePath = path.join(appPath, "compose.yml");

      let composeContent;
      try {
        composeContent = await readFile(composePath, "utf-8");
      } catch {
        return sendError(reply, 404, { code: "APP_NOT_FOUND", message: `App '${appId}' not found or has no compose.yml` });
      }

    // Architecture check
    const imageName = await getImageFromCompose(composePath);
      if (imageName) {
        const archCheck = await checkImageArchitectureSupport(imageName);
        if (archCheck.supported === false) {
          return sendError(reply, 400, {
            code: "ARCHITECTURE_NOT_SUPPORTED",
            message: `The image '${imageName}' does not support your system architecture (${archCheck.systemArch}). Image supports: ${archCheck.imageArch}`,
            details: { image: imageName, systemArch: archCheck.systemArch, imageArch: archCheck.imageArch },
          });
        }
      }

    // Check external networks
    let parsedCompose;
    try { parsedCompose = YAML.parse(composeContent); } catch { parsedCompose = null; }
    if (parsedCompose?.networks) {
      const missingNetworks = [];
      for (const [netName, netConfig] of Object.entries(parsedCompose.networks)) {
        if (netConfig?.external === true) {
          const name = netConfig.name || netName;
          const nets = await docker.listNetworks({ filters: { name: [name] } });
          if (!nets.some(n => n.Name === name)) missingNetworks.push(name);
        }
      }
          if (missingNetworks.length > 0) {
          const needed = missingNetworks.map(n => n.replace(/_network$/, "")).join(", ");
          return sendError(reply, 400, {
            code: "MISSING_NETWORKS",
            message: `Required network(s) ${missingNetworks.join(", ")} do not exist. Deploy ${needed} first.`,
            details: { missingNetworks },
          });
        }
    }

    const extraEnvEntries = extraEnv && typeof extraEnv === 'object'
      ? Object.entries(extraEnv).filter(([k, v]) => k.trim() && v !== null && v !== undefined && String(v).trim() !== '')
      : [];
    const projectName = (instanceId && instanceId > 1) ? `${appId}-${instanceId}` : appId;
    await writeProjectEnv(appPath, projectName, environment);
    const modifiedComposeContent = buildProjectComposeContent(composeContent, {
      projectId: projectName,
      appId,
      expiresIn,
      customPortMappings,
      masterApp: masterApp || null,
      extraEnv: Object.fromEntries(extraEnvEntries.map(([key, value]) => [key.trim(), value])),
    });
    const { composeFile } = await writeProjectCompose(appPath, projectName, modifiedComposeContent);

    const composeCmd = await resolveComposeCommand({ socketPath, log });
    try {
      const composeEnv = await getComposeProcessEnv(appPath, projectName, { DOCKER_HOST: `unix://${socketPath}` });
      const { stdout, stderr, exitCode } = await spawnProcess(
        composeCmd.command,
        [...composeCmd.args, "-p", projectName, "-f", composeFile, "up", "-d"],
        { cwd: appPath, env: composeEnv }
      );
      if (exitCode !== 0) throw new Error(`docker compose failed with exit code ${exitCode}: ${stderr}`);

      trackInstall(appId);
      return reply.send({ success: true, message: `App '${appId}' deployed successfully`, appId, output: stdout, warnings: stderr || null, temporary: !!expiresIn });
    } catch (error) {
      const isArchError = error.message?.includes("no matching manifest") || error.message?.includes("platform") || error.message?.includes("architecture");
      return sendError(reply, isArchError ? 400 : 500, {
        code: isArchError ? "ARCHITECTURE_NOT_SUPPORTED" : "DEPLOYMENT_FAILED",
        message: isArchError ? "This image does not support your system architecture" : error.message,
        details: error.stderr ? { stderr: error.stderr } : null,
      });
    }
  } catch (error) {
    log("error", "❌ [POST /api/deploy] Unexpected error:", error.message);
    return sendError(reply, 500, { code: "DEPLOYMENT_UNEXPECTED_ERROR", message: error.message });
  }
  });
}
