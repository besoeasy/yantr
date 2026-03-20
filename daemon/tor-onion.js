/**
 * Tor Onion Service manager with per-app circuit isolation.
 *
 * Each app gets its own dedicated Tor hidden service with separate HiddenServiceDir
 * to prevent traffic correlation attacks and maintain true anonymity.
 *
 * Label schema:
 *   yantr.tor.enabled        = "true"
 *   yantr.tor.target.port    = "<container port>"
 *   yantr.tor.virtual.port   = "<virtual port, default 80>"
 */

import { spawn } from 'child_process'
import { log, docker } from './shared.js'
import { spawnProcess } from './utils.js'
import net from 'node:net'
import fs from 'node:fs/promises'
import path from 'node:path'

let torProcess = null
const TOR_BINARY = 'tor'
const CONTROL_PORT = 9051
const HashedControlPassword = process.env.TOR_CONTROL_PASSWORD_HASH || ''

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms))
}

async function waitForControlPort(retries = 30, delayMs = 500) {
  for (let i = 0; i < retries; i++) {
    try {
      await new Promise((resolve, reject) => {
        const socket = net.connect(CONTROL_PORT, '127.0.0.1', () => {
          socket.end()
          resolve()
        })
        socket.on('error', reject)
        socket.setTimeout(1000, () => {
          socket.destroy()
          reject(new Error('timeout'))
        })
      })
      return true
    } catch {
      await sleep(delayMs)
    }
  }
  return false
}

function createControlConnection() {
  return new Promise((resolve, reject) => {
    const socket = net.connect(CONTROL_PORT, '127.0.0.1', () => resolve(socket))
    socket.on('error', reject)
  })
}

async function sendTorCommand(socket, command) {
  return new Promise((resolve, reject) => {
    let response = ''
    const onData = (chunk) => {
      response += chunk.toString()
      if (response.includes('250 OK') || response.includes('250 ') || response.includes('514 ')) {
        socket.removeListener('data', onData)
        resolve(response)
      }
    }
    socket.on('data', onData)
    socket.write(command + '\r\n')
    setTimeout(() => resolve(response), 500)
  })
}

async function authenticate(socket) {
  if (!HashedControlPassword) {
    await sendTorCommand(socket, 'AUTHENTICATE')
    return
  }
  await sendTorCommand(socket, `AUTHENTICATE "${process.env.TOR_CONTROL_PASSWORD || ''}"`)
}

function getOnionAddressDir(projectId) {
  return `/var/lib/tor/hidden_service_${projectId.replace(/[^a-zA-Z0-9]/g, '_')}`
}

async function readOnionHostname(hsDir) {
  try {
    const hostnameFile = path.join(hsDir, 'hostname')
    const content = await fs.readFile(hostnameFile, 'utf-8')
    const match = content.trim().match(/^([a-z2-7]{16,56})\.onion/)
    return match ? match[1] : null
  } catch {
    return null
  }
}

export async function getOnionServices() {
  try {
    const containers = await docker.listContainers({ all: false })
    const services = []
    
    for (const c of containers) {
      const labels = c.Labels || {}
      if (labels['yantr.tor.enabled'] !== 'true') continue
      
      const targetPort = Number(labels['yantr.tor.target.port'])
      const virtualPort = Number(labels['yantr.tor.virtual.port']) || 80
      const serviceName = labels['yantr.tor.service.name'] || labels['yantr.service'] || 'unknown'
      
      if (!targetPort) continue
      
      const hsDir = getOnionAddressDir(labels['com.docker.compose.project'] || c.Id.slice(0, 12))
      const onionAddress = await readOnionHostname(hsDir)
      
      services.push({
        containerName: (c.Names[0] || '').replace(/^\//, ''),
        containerId: c.Id,
        projectId: labels['com.docker.compose.project'] || null,
        serviceName,
        targetPort,
        virtualPort,
        onionAddress,
        onionUrl: onionAddress ? `http://${onionAddress}.onion` : null,
      })
    }
    return services
  } catch (err) {
    log('error', `[tor-onion] Failed to get onion services: ${err.message}`)
    return []
  }
}

async function createOnionService(projectId, targetPort, virtualPort = 80) {
  if (!torProcess) {
    throw new Error('Tor not running')
  }
  
  try {
    const socket = await createControlConnection()
    await authenticate(socket)
    
    const hsDir = getOnionAddressDir(projectId)
    await fs.mkdir(hsDir, { recursive: true })
    
    const config = `HiddenServiceDir ${hsDir}
HiddenServicePort ${virtualPort} 127.0.0.1:${targetPort}
HiddenServiceVersion 3
HiddenServiceAuthorizeClient stealth ${projectId}_client`

    await sendTorCommand(socket, `RESETCONF`)
    await sendTorCommand(socket, `SETCONF ${config.replace(/\n/g, ' ')}`)
    
    socket.end()
    
    const onionAddress = await readOnionHostname(hsDir)
    log('info', `[tor-onion] Created onion service for ${projectId}: ${onionAddress}`)
    
    return onionAddress
  } catch (err) {
    log('error', `[tor-onion] Failed to create onion service: ${err.message}`)
    throw err
  }
}

async function removeOnionService(projectId) {
  if (!torProcess) return
  
  try {
    const socket = await createControlConnection()
    await authenticate(socket)
    
    const hsDir = getOnionAddressDir(projectId)
    await sendTorCommand(socket, `DROPHIDDENSERVICES`)
    socket.end()
    
    try {
      await fs.rm(hsDir, { recursive: true, force: true })
    } catch {}
    
    log('info', `[tor-onion] Removed onion service for ${projectId}`)
  } catch (err) {
    log('warn', `[tor-onion] Error removing onion service: ${err.message}`)
  }
}

export async function reloadOnionConfig() {
  const services = await getOnionServices()
  log('info', `[tor-onion] ${services.length} onion service(s) configured`)
  return { services }
}

function buildTorrc() {
  return `
SOCKSPort 9050
ControlPort 9051
DataDirectory /var/lib/tor/data
RunAsDaemon 1
GeoIPFile /usr/share/tor/geoip
GeoIPv6File /usr/share/tor/geoip6
CircuitBuildTimeout 60
NumEntryGuards 8
MaxCircuitDirtiness 600
`
}

export async function startTor() {
  if (torProcess) return true
  
  try {
    await fs.mkdir('/var/lib/tor/data', { recursive: true })
    
    const torrcPath = '/var/lib/tor/torrc'
    await fs.writeFile(torrcPath, buildTorrc())
    
    torProcess = spawn(TOR_BINARY, ['-f', torrcPath], {
      stdio: ['ignore', 'pipe', 'pipe'],
      detached: false,
    })

    torProcess.stdout?.on('data', d => {
      const msg = d.toString().trim()
      if (msg) log('info', `[tor] ${msg}`)
    })
    torProcess.stderr?.on('data', d => {
      const msg = d.toString().trim()
      if (msg && !msg.includes('Oct')) log('info', `[tor] ${msg}`)
    })

    torProcess.on('exit', code => {
      log('info', `[tor] process exited (code ${code})`)
      torProcess = null
    })

    torProcess.on('error', err => {
      if (err.code === 'ENOENT') {
        log('warn', '⚠️  tor binary not found — install with: apt install tor')
      } else {
        log('error', `[tor] error: ${err.message}`)
      }
      torProcess = null
    })

    const ready = await waitForControlPort()
    if (!ready) {
      log('warn', '⚠️  Tor control port not ready')
      return false
    }

    log('info', '🧅 Tor started with circuit isolation')
    
    const services = await getOnionServices()
    for (const svc of services) {
      if (svc.projectId && !svc.onionAddress) {
        try {
          await createOnionService(svc.projectId, svc.targetPort, svc.virtualPort)
        } catch {}
      }
    }
    
    return true
  } catch (err) {
    log('error', `Failed to start Tor: ${err.message}`)
    return false
  }
}

export function isRunning() {
  return torProcess !== null
}

export function stopTor() {
  if (torProcess) {
    torProcess.kill()
    torProcess = null
  }
}

export { createOnionService, removeOnionService, getOnionAddressDir }
