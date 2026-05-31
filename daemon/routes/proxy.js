import path from 'path'
import { readFile } from 'fs/promises'
import { docker, appsDir, socketPath, log } from '../shared.js'
import {
  nonEmptyStringSchema,
  optionalStringSchema,
  positiveIntegerSchema,
  sendError,
} from '../api.js'
import { getBaseAppId } from '../utils.js'
import { resolveComposeCommand } from '../compose.js'
import { spawnProcess } from '../utils.js'
import {
  getProjectComposeRef,
  buildProjectComposeContent,
  parseCompose,
  stringifyCompose,
  writeProjectCompose,
  getComposeProcessEnv,
} from '../stack-compose.js'
import { hashPassword, reloadCaddyConfig, isRunning, getCaddyProxies, startCaddy } from '../caddy.js'

const enableProxySchema = {
  body: {
    type: 'object',
    required: ['projectId', 'servePort', 'targetPort'],
    additionalProperties: false,
    properties: {
      projectId: nonEmptyStringSchema,
      serviceName: optionalStringSchema,
      servePort: positiveIntegerSchema,
      targetPort: positiveIntegerSchema,
      authUser: optionalStringSchema,
      authPass: optionalStringSchema,
    },
  },
}

const disableProxySchema = {
  body: {
    type: 'object',
    required: ['projectId'],
    additionalProperties: false,
    properties: {
      projectId: nonEmptyStringSchema,
      serviceName: optionalStringSchema,
    },
  },
}

// ── Shared helpers ────────────────────────────────────────────────────────────

async function getBootstrappedCompose(projectId, projectContainers) {
  const baseAppId = getBaseAppId(projectId)
  const appPath = path.join(appsDir, baseAppId)
  const composeRef = await getProjectComposeRef(appPath, projectId)

  let composeContent
  if (composeRef.isProjectCompose) {
    composeContent = await readFile(composeRef.composePath, 'utf-8')
  } else {
    const baseContent = await readFile(composeRef.composePath, 'utf-8')
    const bootstrapped = parseCompose(buildProjectComposeContent(baseContent, { projectId, appId: baseAppId }))
    composeContent = stringifyCompose(bootstrapped)
  }

  return { appPath, composeContent }
}

async function redeployStack(appPath, projectId, composeFile) {
  const composeCmd = await resolveComposeCommand({ socketPath, log })
  const composeEnv = await getComposeProcessEnv(appPath, projectId, { DOCKER_HOST: `unix://${socketPath}` })
  return spawnProcess(
    composeCmd.command,
    [...composeCmd.args, '-p', projectId, '-f', composeFile, 'up', '-d'],
    { cwd: appPath, env: composeEnv }
  )
}

// ── Routes ────────────────────────────────────────────────────────────────────

export default async function proxyRoutes(fastify) {
  // GET /api/proxy — list all active proxy routes derived from container labels
  fastify.get('/api/proxy', async (_request, reply) => {
    const proxies = await getCaddyProxies()
    return reply.send({ success: true, proxies, caddyRunning: isRunning() })
  })

  // POST /api/proxy/reload — force full Caddyfile rebuild from current labels
  fastify.post('/api/proxy/reload', async (_request, reply) => {
    if (!isRunning()) await startCaddy()
    const result = await reloadCaddyConfig()
    return reply.send({ success: true, caddyRunning: isRunning(), ...result })
  })

  // POST /api/proxy/enable
  // Body: { projectId, servePort, targetPort, serviceName?, authUser?, authPass? }
  fastify.post('/api/proxy/enable', { schema: enableProxySchema }, async (request, reply) => {
    const {
      projectId,
      serviceName: requestedService,
      servePort,
      targetPort,
      authUser,
      authPass,
    } = request.body || {}

    if (!!authUser !== !!authPass) {
      return sendError(reply, 400, { code: 'PROXY_AUTH_FIELDS_MISMATCH', message: 'Both authUser and authPass must be provided together, or neither' })
    }

    const allContainers = await docker.listContainers({ all: false })
    const projectContainers = allContainers.filter(c => c.Labels?.['com.docker.compose.project'] === projectId)
    if (!projectContainers.length) {
      return sendError(reply, 404, { code: 'STACK_NOT_FOUND', message: 'Stack not found or not running' })
    }

    const availableServices = [...new Set(
      projectContainers.map(c => c.Labels?.['com.docker.compose.service']).filter(Boolean)
    )]
    const serviceName = requestedService || availableServices[0]
    if (!serviceName) {
      return sendError(reply, 400, { code: 'SERVICE_NAME_UNRESOLVED', message: 'Cannot determine a service name for this stack' })
    }

    let passHash = null
    if (authUser && authPass) {
      try {
        passHash = await hashPassword(authPass)
      } catch (err) {
        return sendError(reply, 500, { code: 'PROXY_PASSWORD_HASH_FAILED', message: `Failed to hash password: ${err.message}` })
      }
    }

    const { appPath, composeContent } = await getBootstrappedCompose(projectId, projectContainers)
    const compose = parseCompose(composeContent)
    const service = compose.services?.[serviceName]
    if (!service) {
      return sendError(reply, 400, { code: 'SERVICE_NOT_FOUND_IN_COMPOSE', message: `Service '${serviceName}' not found in compose` })
    }

    // Validate the container port exists on the service
    const serviceContainer = projectContainers.find(c => c.Labels?.['com.docker.compose.service'] === serviceName)
    const portBinding = (serviceContainer?.Ports || []).find(p => p.PrivatePort === Number(targetPort))
    if (!portBinding) {
      return sendError(reply, 400, { code: 'TARGET_PORT_NOT_FOUND', message: `Container port ${targetPort} not found on service '${serviceName}'` })
    }

    // Apply caddy labels — convert array labels to map if needed
    if (!service.labels || Array.isArray(service.labels)) service.labels = {}
    service.labels['yantr.caddy.enabled'] = 'true'
    service.labels['yantr.caddy.serve.port'] = String(servePort)
    service.labels['yantr.caddy.target.port'] = String(targetPort)
    if (passHash) {
      service.labels['yantr.caddy.auth.user'] = String(authUser)
      // Encode as hex to avoid '$' being treated as variable expansion by docker-compose.
      service.labels['yantr.caddy.auth.pass'] = Buffer.from(passHash).toString('hex')
    } else {
      delete service.labels['yantr.caddy.auth.user']
      delete service.labels['yantr.caddy.auth.pass']
    }

    const { composeFile } = await writeProjectCompose(appPath, projectId, stringifyCompose(compose))
    const { stdout, stderr, exitCode } = await redeployStack(appPath, projectId, composeFile)
    if (exitCode !== 0) {
      return sendError(reply, 500, { code: 'PROXY_REDEPLOY_FAILED', message: `docker compose failed: ${stderr || stdout}` })
    }

    if (!isRunning()) await startCaddy()
    else await reloadCaddyConfig()

    return reply.send({
      success: true,
      message: `Caddy proxy enabled: :${servePort} → localhost:${targetPort}`,
      proxied: {
        projectId,
        serviceName,
        servePort: Number(servePort),
        targetPort: Number(targetPort),
        authEnabled: !!passHash,
      },
      output: stdout,
    })
  })

  // POST /api/proxy/disable
  // Body: { projectId, serviceName? }
  fastify.post('/api/proxy/disable', { schema: disableProxySchema }, async (request, reply) => {
    const { projectId, serviceName: requestedService } = request.body || {}

    const allContainers = await docker.listContainers({ all: false })
    const projectContainers = allContainers.filter(c => c.Labels?.['com.docker.compose.project'] === projectId)
    if (!projectContainers.length) {
      return sendError(reply, 404, { code: 'STACK_NOT_FOUND', message: 'Stack not found or not running' })
    }

    const availableServices = [...new Set(
      projectContainers.map(c => c.Labels?.['com.docker.compose.service']).filter(Boolean)
    )]
    const serviceName = requestedService || availableServices[0]

    const { appPath, composeContent } = await getBootstrappedCompose(projectId, projectContainers)
    const compose = parseCompose(composeContent)
    const service = compose.services?.[serviceName]

    if (service?.labels && typeof service.labels === 'object' && !Array.isArray(service.labels)) {
      delete service.labels['yantr.caddy.enabled']
      delete service.labels['yantr.caddy.serve.port']
      delete service.labels['yantr.caddy.target.port']
      delete service.labels['yantr.caddy.auth.user']
      delete service.labels['yantr.caddy.auth.pass']
    }

    const { composeFile } = await writeProjectCompose(appPath, projectId, stringifyCompose(compose))
    const { stdout, stderr, exitCode } = await redeployStack(appPath, projectId, composeFile)
    if (exitCode !== 0) {
      return sendError(reply, 500, { code: 'PROXY_DISABLE_REDEPLOY_FAILED', message: `docker compose failed: ${stderr || stdout}` })
    }

    await reloadCaddyConfig()

    return reply.send({
      success: true,
      message: `Caddy proxy disabled for '${projectId}'`,
      output: stdout,
    })
  })
}
