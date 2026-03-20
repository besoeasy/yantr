import path from 'path'
import { readFile } from 'fs/promises'
import { docker, appsDir, socketPath, log } from '../shared.js'
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
import {
  isRunning,
  startTor,
  reloadOnionConfig,
  getOnionServices,
  createOnionService,
  removeOnionService,
} from '../tor-onion.js'

async function getBootstrappedCompose(projectId) {
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

export default async function torOnionRoutes(fastify) {
  fastify.get('/api/tor/onion', async (_request, reply) => {
    const services = await getOnionServices()
    return reply.send({
      success: true,
      services,
      torRunning: isRunning(),
    })
  })

  fastify.get('/api/tor/onion/:projectId', async (request, reply) => {
    const { projectId } = request.params
    const services = await getOnionServices()
    const service = services.find(s => s.projectId === projectId)
    
    if (!service) {
      return reply.code(404).send({
        success: false,
        error: 'No onion service configured for this app',
        onionUrl: null,
      })
    }
    
    return reply.send({
      success: true,
      projectId,
      onionAddress: service.onionAddress,
      onionUrl: service.onionUrl,
      targetPort: service.targetPort,
      virtualPort: service.virtualPort,
    })
  })

  fastify.get('/api/tor/onion/:projectId/open', async (request, reply) => {
    const { projectId } = request.params
    const services = await getOnionServices()
    const service = services.find(s => s.projectId === projectId)
    
    if (!service || !service.onionUrl) {
      return reply.code(404).send({
        success: false,
        error: 'Onion service not configured',
      })
    }
    
    return reply.redirect(service.onionUrl)
  })

  fastify.post('/api/tor/reload', async (_request, reply) => {
    if (!isRunning()) {
      const started = await startTor()
      if (!started) {
        return reply.send({
          success: false,
          error: 'Failed to start Tor',
          torRunning: false,
        })
      }
    }
    const result = await reloadOnionConfig()
    return reply.send({ success: true, torRunning: isRunning(), ...result })
  })

  fastify.post('/api/tor/onion/enable', async (request, reply) => {
    const {
      projectId,
      targetPort,
      virtualPort = 80,
    } = request.body || {}

    if (!projectId || !targetPort) {
      return reply.code(400).send({ success: false, error: 'projectId and targetPort are required' })
    }

    const allContainers = await docker.listContainers({ all: false })
    const projectContainers = allContainers.filter(c => c.Labels?.['com.docker.compose.project'] === projectId)
    if (!projectContainers.length) {
      return reply.code(404).send({ success: false, error: 'Stack not found or not running' })
    }

    const { appPath, composeContent } = await getBootstrappedCompose(projectId)
    const compose = parseCompose(composeContent)
    
    const serviceName = Object.keys(compose.services || {})[0]
    if (!serviceName) {
      return reply.code(400).send({ success: false, error: 'No services found in compose' })
    }

    const service = compose.services[serviceName]
    if (!service.labels || Array.isArray(service.labels)) service.labels = {}
    service.labels['yantr.tor.enabled'] = 'true'
    service.labels['yantr.tor.target.port'] = String(targetPort)
    service.labels['yantr.tor.virtual.port'] = String(virtualPort)
    service.labels['yantr.tor.service.name'] = serviceName

    const { composeFile } = await writeProjectCompose(appPath, projectId, stringifyCompose(compose))
    const { stdout, stderr, exitCode } = await redeployStack(appPath, projectId, composeFile)
    if (exitCode !== 0) {
      return reply.code(500).send({ success: false, error: `docker compose failed: ${stderr || stdout}` })
    }

    if (!isRunning()) {
      await startTor()
    }

    let onionAddress = null
    try {
      onionAddress = await createOnionService(projectId, targetPort, virtualPort)
    } catch (err) {
      log('warn', `[tor-onion] Failed to create onion: ${err.message}`)
    }

    return reply.send({
      success: true,
      message: 'Onion service enabled with dedicated circuit',
      onion: {
        projectId,
        serviceName,
        targetPort: Number(targetPort),
        virtualPort: Number(virtualPort),
        onionAddress,
        onionUrl: onionAddress ? `http://${onionAddress}.onion` : null,
      },
    })
  })

  fastify.post('/api/tor/onion/disable', async (request, reply) => {
    const { projectId } = request.body || {}
    if (!projectId) {
      return reply.code(400).send({ success: false, error: 'projectId is required' })
    }

    const { appPath, composeContent } = await getBootstrappedCompose(projectId)
    const compose = parseCompose(composeContent)
    
    for (const [svcName, service] of Object.entries(compose.services || {})) {
      if (service?.labels && typeof service.labels === 'object' && !Array.isArray(service.labels)) {
        delete service.labels['yantr.tor.enabled']
        delete service.labels['yantr.tor.target.port']
        delete service.labels['yantr.tor.virtual.port']
        delete service.labels['yantr.tor.service.name']
      }
    }

    const { composeFile } = await writeProjectCompose(appPath, projectId, stringifyCompose(compose))
    const { stdout, stderr, exitCode } = await redeployStack(appPath, projectId, composeFile)
    if (exitCode !== 0) {
      return reply.code(500).send({ success: false, error: `docker compose failed: ${stderr || stdout}` })
    }

    await removeOnionService(projectId)
    await reloadOnionConfig()

    return reply.send({
      success: true,
      message: `Onion service removed for '${projectId}'`,
    })
  })
}
