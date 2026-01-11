<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useToast } from 'vue-toastification'

const toast = useToast()

// State
const activeTab = ref('containers')
const containers = ref([])
const apps = ref([])
const imagesData = ref({})
const logsData = ref({})
const logFilter = ref('all')
const loading = ref(false)
const deploying = ref(null)
const deleting = ref(null)
const deletingImage = ref(null)
const deletingAllImages = ref(false)
const showEnvModal = ref(false)
const selectedApp = ref(null)
const envValues = ref({})
const appSearch = ref('')
const apiUrl = ref('')
const version = ref('loading...')
const showPortModal = ref(false)
const selectedAppForPorts = ref(null)
const musthaveapps = ['dufs', 'watchtower']

// Container detail view
const containerDetailView = ref(false)
const selectedContainer = ref(null)
const containerStats = ref(null)
const containerLogs = ref([])
let statsInterval = null

// Computed
const allAppsCount = computed(() => uninstalledApps.value.length)

const installedAppIds = computed(() => {
  const ids = new Set(containers.value.map(c => c.app.id))
  return ids
})

const customContainers = computed(() => {
  return containers.value.filter(c => !c.appLabels || !c.appLabels.name)
})

const yantraContainers = computed(() => {
  return containers.value.filter(c => c.appLabels && c.appLabels.name)
})

const installedApps = computed(() => {
  return yantraContainers.value.map(c => ({
    ...c.app,
    isInstalled: true,
    containerId: c.id,
    ports: c.ports,
    state: c.state,
    status: c.status,
    env: c.env
  }))
})

const uninstalledApps = computed(() => {
  const uninstalled = apps.value.filter(app => !installedAppIds.value.has(app.id))
  return shuffleWithSeed(uninstalled).map(app => ({
    ...app,
    isInstalled: false
  }))
})

const combinedApps = computed(() => {
  let combined = [...uninstalledApps.value]
  
  if (appSearch.value) {
    const search = appSearch.value.toLowerCase()
    combined = combined.filter(app => {
      return app.name.toLowerCase().includes(search) ||
        app.category.toLowerCase().includes(search) ||
        (app.description && app.description.toLowerCase().includes(search))
    })
  }
  
  return combined
})

// Helper Functions
function getDateHourSeed() {
  const now = new Date()
  const year = now.getFullYear()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  const hour = String(now.getHours()).padStart(2, '0')
  return `${year}-${month}-${day}-${hour}`
}

function seededRandom(seed) {
  let x = Math.sin(seed) * 10000
  return x - Math.floor(x)
}

function shuffleWithSeed(array) {
  const dateHourSeed = getDateHourSeed()
  let numericSeed = 0
  for (let i = 0; i < dateHourSeed.length; i++) {
    numericSeed += dateHourSeed.charCodeAt(i) * (i + 1)
  }
  
  const shuffled = [...array]
  
  for (let i = shuffled.length - 1; i > 0; i--) {
    const j = Math.floor(seededRandom(numericSeed + i) * (i + 1));
    [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]]
  }
  
  return shuffled
}

function appUrl(port, protocol = 'http') {
  const normalizedProtocol = protocol.replace('://', '').replace(':', '')
  let host = window.location.hostname || 'localhost'

  if (host.includes(':') && !host.startsWith('[')) {
    host = `[${host}]`
  }

  const portString = String(port ?? '').trim()
  const portMatch = portString.match(/\d+/)
  if (!portMatch) {
    return `${normalizedProtocol}://${host}`
  }

  return `${normalizedProtocol}://${host}:${portMatch[0]}`
}

function getPorts(app) {
  if (!app.port) return []
  
  const portStr = String(app.port).trim()
  
  if (portStr.includes(',')) {
    return portStr.split(',').map(p => {
      const trimmed = p.trim()
      const match = trimmed.match(/^(\d+)(?:\s*\(([^-\)]+)\s*-\s*([^)]+)\))?$/)
      if (match) {
        return { 
          port: match[1], 
          protocol: match[2] ? match[2].trim().toLowerCase() : 'http',
          label: match[3] ? match[3].trim() : null 
        }
      }
      return { port: trimmed, protocol: 'http', label: null }
    }).filter(p => p.port)
  }
  
  const match = portStr.match(/^(\d+)(?:\s*\(([^-\)]+)\s*-\s*([^)]+)\))?$/)
  if (match) {
    return [{ 
      port: match[1], 
      protocol: match[2] ? match[2].trim().toLowerCase() : 'http',
      label: match[3] ? match[3].trim() : null 
    }]
  }
  
  return [{ port: portStr, protocol: 'http', label: null }]
}

function openApp(app) {
  let ports
  
  if (app.isInstalled && app.ports && app.ports.length > 0) {
    ports = app.ports
      .filter(p => p.PublicPort)
      .map(p => ({
        port: p.PublicPort,
        protocol: p.Type === 'tcp' ? 'http' : 'http',
        label: `Port ${p.PublicPort}`
      }))
    
    const uniquePorts = []
    const seenPorts = new Set()
    ports.forEach(p => {
      if (!seenPorts.has(p.port)) {
        seenPorts.add(p.port)
        uniquePorts.push(p)
      }
    })
    ports = uniquePorts
  } else {
    ports = getPorts(app)
  }
  
  if (ports.length === 0) return
  
  if (ports.length === 1) {
    window.open(appUrl(ports[0].port, ports[0].protocol), '_blank')
  } else {
    selectedAppForPorts.value = { ...app, parsedPorts: ports }
    showPortModal.value = true
  }
}

function closePortModal() {
  showPortModal.value = false
  selectedAppForPorts.value = null
}

function formatTimestamp(timestamp) {
  const date = new Date(timestamp)
  return date.toLocaleTimeString('en-US', {
    hour12: false,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

function formatBytes(bytes) {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function getPortLabel(publicPort) {
  if (!selectedContainer.value || !selectedContainer.value.app || !selectedContainer.value.app.port) {
    return null
  }
  
  const portStr = selectedContainer.value.app.port
  const regex = new RegExp(`${publicPort}\\s*\\(([^)]+)\\)`)
  const match = portStr.match(regex)
  
  if (match) {
    return match[1]
  }
  
  return null
}

function getLabeledPorts() {
  if (!selectedContainer.value || !selectedContainer.value.ports) {
    return []
  }
  
  const ports = []
  const portDescriptions = {}
  
  if (!selectedContainer.value.app || !selectedContainer.value.app.port) {
    return []
  }
  
  const portStr = selectedContainer.value.app.port
  const regex = /(\d+)\s*\(([^-\)]+)\s*-\s*([^)]+)\)/g
  let match
  
  while ((match = regex.exec(portStr)) !== null) {
    portDescriptions[match[1]] = {
      protocol: match[2].trim().toLowerCase(),
      label: match[3].trim()
    }
  }
  
  if (Object.keys(portDescriptions).length === 0) {
    return []
  }
  
  const portKeys = Object.keys(selectedContainer.value.ports)
  const portMap = {}
  portKeys.forEach(key => {
    const [privatePort, type] = key.split('/')
    const bindings = selectedContainer.value.ports[key]
    
    if (type === 'tcp' && bindings && bindings.length > 0) {
      const hostPort = bindings[0].HostPort
      if (hostPort) {
        portMap[privatePort] = hostPort
      }
    }
  })
  
  for (const [privatePort, description] of Object.entries(portDescriptions)) {
    const hostPort = portMap[privatePort]
    
    if (hostPort) {
      ports.push({
        port: hostPort,
        protocol: description.protocol,
        label: description.label
      })
    }
  }
  
  return ports
}

// API Functions
async function fetchVersion() {
  try {
    const response = await fetch(`${apiUrl.value}/api/version`)
    const data = await response.json()
    version.value = data.version || '0.0.0'
  } catch (error) {
    console.error('Failed to fetch version:', error)
    version.value = '0.0.0'
  }
}

async function refreshAll() {
  loading.value = true
  await Promise.all([
    fetchContainers(),
    fetchApps()
  ])
  loading.value = false
}

async function fetchContainers() {
  try {
    const response = await fetch(`${apiUrl.value}/api/containers`)
    const data = await response.json()
    if (data.success) {
      containers.value = data.containers.filter(c => c.state === 'running')
    }
  } catch (error) {
    console.error('Failed to fetch containers:', error)
  }
}

async function fetchApps() {
  try {
    const response = await fetch(`${apiUrl.value}/api/apps`)
    const data = await response.json()
    if (data.success) {
      apps.value = data.apps
    }
  } catch (error) {
    console.error('Failed to fetch apps:', error)
  }
}

async function fetchImages() {
  loading.value = true
  try {
    const response = await fetch(`${apiUrl.value}/api/images`)
    const data = await response.json()
    if (data.success) {
      imagesData.value = data
    }
  } catch (error) {
    console.error('Failed to fetch images:', error)
  } finally {
    loading.value = false
  }
}

async function fetchLogs() {
  loading.value = true
  try {
    const level = logFilter.value === 'all' ? '' : logFilter.value
    const url = level ? `${apiUrl.value}/api/logs?level=${level}` : `${apiUrl.value}/api/logs`
    const response = await fetch(url)
    const data = await response.json()
    if (data.success) {
      logsData.value = data
    }
  } catch (error) {
    console.error('Failed to fetch logs:', error)
  } finally {
    loading.value = false
  }
}

async function deployApp(appId) {
  const app = apps.value.find(a => a.id === appId)

  if (!app) return

  if (app.environment && app.environment.length > 0) {
    selectedApp.value = app
    envValues.value = {}
    app.environment.forEach(env => {
      envValues.value[env.envVar] = env.default
    })
    showEnvModal.value = true
  } else {
    await confirmDeploy(appId, {}, {})
  }
}

async function confirmDeploy(appId, environment, ports) {
  showEnvModal.value = false
  deploying.value = appId
  
  toast.info(`Deploying ${appId}... This may take a few minutes.`)
  
  try {
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 300000)
    
    const response = await fetch(`${apiUrl.value}/api/deploy`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ appId, environment }),
      signal: controller.signal
    })

    clearTimeout(timeoutId)
    const data = await response.json()

    if (data.success) {
      toast.success(`${appId} deployed successfully!`)
      await fetchContainers()
    } else {
      toast.error(`Deployment failed: ${data.error}`)
    }
  } catch (error) {
    if (error.name === 'AbortError') {
      toast.error(`Deployment timeout - ${appId} is taking longer than expected`)
    } else {
      toast.error(`Deployment failed: ${error.message}`)
    }
  } finally {
    deploying.value = null
    selectedApp.value = null
  }
}

function cancelDeploy() {
  showEnvModal.value = false
  selectedApp.value = null
  envValues.value = {}
}

async function deleteContainer(containerId, containerName) {
  if (!confirm(`Delete ${containerName}?\n\nThis will remove the container and all its volumes permanently.`)) return

  deleting.value = containerId
  try {
    const response = await fetch(`${apiUrl.value}/api/containers/${containerId}`, {
      method: 'DELETE'
    })
    const data = await response.json()

    if (data.success) {
      let message = `${containerName} deleted successfully!`
      if (data.volumesRemoved.length > 0) {
        message += `\n\nVolumes removed: ${data.volumesRemoved.join(', ')}`
      }
      toast.success(message)
      
      if (containerDetailView.value && selectedContainer.value?.id === containerId) {
        closeContainerDetail()
      }
      
      await fetchContainers()
    } else {
      toast.error(`Deletion failed: ${data.error}`)
    }
  } catch (error) {
    toast.error(`Deletion failed: ${error.message}`)
  } finally {
    deleting.value = null
  }
}

async function deleteImage(imageId, imageName) {
  if (!confirm(`Delete image ${imageName}?\n\nThis will permanently remove the image from your system.`)) return

  deletingImage.value = imageId
  try {
    const response = await fetch(`${apiUrl.value}/api/images/${imageId}`, {
      method: 'DELETE'
    })
    const data = await response.json()

    if (data.success) {
      toast.success(`Image deleted successfully!`)
      await fetchImages()
    } else {
      toast.error(`Deletion failed: ${data.error}\n${data.message}`)
    }
  } catch (error) {
    toast.error(`Deletion failed: ${error.message}`)
  } finally {
    deletingImage.value = null
  }
}

async function deleteAllUnusedImages() {
  const count = imagesData.value.unusedImages?.length || 0
  if (!count) return
  
  if (!confirm(`Delete all ${count} unused images?\n\nThis will free up ${imagesData.value.unusedSize} MB of disk space.\n\nThis action cannot be undone.`)) return

  deletingAllImages.value = true
  let deleted = 0
  let failed = 0

  try {
    for (const image of imagesData.value.unusedImages) {
      try {
        const response = await fetch(`${apiUrl.value}/api/images/${image.id}`, {
          method: 'DELETE'
        })
        const data = await response.json()
        
        if (data.success) {
          deleted++
        } else {
          failed++
        }
      } catch (error) {
        failed++
      }
    }

    if (deleted > 0) {
      toast.success(`Successfully deleted ${deleted} unused image${deleted > 1 ? 's' : ''}!${failed > 0 ? `\n${failed} failed.` : ''}`)
      await fetchImages()
    } else {
      toast.error(`Failed to delete images`)
    }
  } catch (error) {
    toast.error(`Deletion failed: ${error.message}`)
  } finally {
    deletingAllImages.value = false
  }
}

// Container detail functions
async function viewContainerDetail(container) {
  containerDetailView.value = true
  containerStats.value = null
  containerLogs.value = []
  
  try {
    const response = await fetch(`${apiUrl.value}/api/containers/${container.id}`)
    const data = await response.json()
    
    if (data.success) {
      selectedContainer.value = data.container
    } else {
      selectedContainer.value = container
    }
  } catch (error) {
    console.error('Failed to fetch container details:', error)
    selectedContainer.value = container
  }
  
  await Promise.all([
    fetchContainerStats(),
    fetchContainerLogs()
  ])
  
  statsInterval = setInterval(() => {
    fetchContainerStats()
  }, 2000)
}

function closeContainerDetail() {
  containerDetailView.value = false
  selectedContainer.value = null
  containerStats.value = null
  containerLogs.value = []
  
  if (statsInterval) {
    clearInterval(statsInterval)
    statsInterval = null
  }
}

async function fetchContainerStats() {
  if (!selectedContainer.value) return
  
  try {
    const response = await fetch(`${apiUrl.value}/api/containers/${selectedContainer.value.id}/stats`)
    const data = await response.json()
    
    if (data.success) {
      containerStats.value = data.stats
    }
  } catch (error) {
    console.error('Failed to fetch container stats:', error)
  }
}

async function fetchContainerLogs() {
  if (!selectedContainer.value) return
  
  try {
    const response = await fetch(`${apiUrl.value}/api/containers/${selectedContainer.value.id}/logs?tail=200`)
    const data = await response.json()
    
    if (data.success) {
      containerLogs.value = data.logs
    }
  } catch (error) {
    console.error('Failed to fetch container logs:', error)
  }
}

// Lifecycle
onMounted(() => {
  fetchVersion()
  refreshAll()
  // Auto-refresh every 10 seconds
  setInterval(() => {
    if (activeTab.value === 'apps') {
      fetchContainers()
    }
  }, 10000)
})

onUnmounted(() => {
  if (statsInterval) {
    clearInterval(statsInterval)
  }
})
</script>

<template>
  <div class="min-h-screen flex bg-white text-gray-900">
    <!-- Minimal Sidebar -->
    <aside class="bg-white flex flex-col items-center border-r border-gray-200 w-20 py-6 px-2 fixed h-screen">
      <!-- Logo -->
      <h1 class="text-lg font-bold text-gray-900 mb-8 uppercase leading-tight text-center">
        Yan<br/>tra
      </h1>

      <!-- Navigation -->
      <nav class="flex flex-col items-center gap-3 mb-auto">
        <!-- Apps Tab -->
        <button 
          @click="activeTab = 'apps'"
          :class="activeTab === 'apps' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-100'"
          class="relative w-12 h-12 rounded-full flex items-center justify-center transition-all smooth-shadow"
          title="Apps">
          <i class="fas fa-box text-lg"></i>
          <span v-if="allAppsCount > 0" 
            class="absolute -top-1 -right-1 w-5 h-5 bg-blue-500 text-white text-xs font-bold rounded-full flex items-center justify-center">
            {{ allAppsCount }}
          </span>
        </button>

        <!-- Containers Tab -->
        <button 
          @click="activeTab = 'containers'"
          :class="activeTab === 'containers' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-100'"
          class="relative w-12 h-12 rounded-full flex items-center justify-center transition-all smooth-shadow"
          title="Containers">
          <i class="fas fa-cubes text-lg"></i>
          <span v-if="containers.length > 0" 
            class="absolute -top-1 -right-1 w-5 h-5 bg-purple-500 text-white text-xs font-bold rounded-full flex items-center justify-center">
            {{ containers.length }}
          </span>
        </button>

        <!-- Images Tab -->
        <button 
          @click="activeTab = 'images'; fetchImages()"
          :class="activeTab === 'images' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-100'"
          class="relative w-12 h-12 rounded-full flex items-center justify-center transition-all smooth-shadow"
          title="Images">
          <i class="fas fa-images text-lg"></i>
          <span v-if="imagesData.unused > 0" 
            class="absolute -top-1 -right-1 w-5 h-5 bg-orange-500 text-white text-xs font-bold rounded-full flex items-center justify-center">
            {{ imagesData.unused }}
          </span>
        </button>

        <!-- Logs Tab -->
        <button 
          @click="activeTab = 'logs'; fetchLogs()"
          :class="activeTab === 'logs' ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-100'"
          class="relative w-12 h-12 rounded-full flex items-center justify-center transition-all smooth-shadow"
          title="Logs">
          <i class="fas fa-clipboard-list text-lg"></i>
        </button>
      </nav>

      <!-- Bottom Actions -->
      <div class="flex flex-col items-center gap-3 mt-4">
        <!-- GitHub -->
        <a 
          href="https://github.com/besoeasy/Yantra" 
          target="_blank"
          class="w-12 h-12 rounded-full flex items-center justify-center text-gray-600 hover:bg-gray-100 transition-all"
          title="View on GitHub">
          <i class="fab fa-github text-lg"></i>
        </a>

        <!-- Report Issue -->
        <a 
          href="https://github.com/besoeasy/Yantra/issues/new" 
          target="_blank"
          class="w-12 h-12 rounded-full flex items-center justify-center text-gray-600 hover:bg-gray-100 transition-all"
          title="Report Issue">
          <i class="fas fa-exclamation-triangle text-lg"></i>
        </a>
      </div>
    </aside>

    <!-- Main Content -->
    <main class="ml-20 flex-1 min-h-screen">
      <div class="p-6 md:p-10 lg:p-12">
        <!-- Tabs content will be added in the next message due to length -->
        
        <!-- Placeholder for now -->
        <div v-if="activeTab === 'apps'">
          <h2 class="text-5xl font-bold mb-12 text-gray-900 tracking-tight">App Store</h2>
          <p class="text-gray-600 mb-8 -mt-8">Browse and install apps</p>
          <!-- Rest of apps content... -->
        </div>

        <div v-if="activeTab === 'containers'">
          <h2 class="text-5xl font-bold mb-12 text-gray-900 tracking-tight">Containers</h2>
          <p class="text-gray-600 mb-8 -mt-8">All running Docker containers</p>
          <!-- Rest of containers content... -->
        </div>

        <div v-if="activeTab === 'images'">
          <h2 class="text-5xl font-bold mb-12 text-gray-900 tracking-tight">Images</h2>
          <!-- Rest of images content... -->
        </div>

        <div v-if="activeTab === 'logs'">
          <h2 class="text-3xl font-bold mb-8 text-gray-800">Application Logs</h2>
          <!-- Rest of logs content... -->
        </div>
      </div>
    </main>

    <!-- Modals -->
    <!-- Environment Variables Modal -->
    <div v-if="showEnvModal"
      class="fixed inset-0 bg-black/30 backdrop-blur-sm flex items-center justify-center z-50"
      @click.self="cancelDeploy">
      <div class="glass-dark rounded-3xl p-8 max-w-md w-full mx-4 smooth-shadow-lg">
        <h2 class="text-2xl font-bold mb-2 text-gray-900">Configure {{ selectedApp?.name }}</h2>
        <p class="text-sm text-gray-600 mb-6">Set environment variables for deployment</p>

        <div class="space-y-4 mb-8">
          <div v-for="env in selectedApp?.environment" :key="env.envVar" class="space-y-2">
            <label class="block text-sm font-semibold text-gray-700">
              {{ env.name }}
              <span v-if="env.default" class="text-gray-400 text-xs font-normal ml-1">(default: {{ env.default }})</span>
            </label>
            <input v-model="envValues[env.envVar]" type="text" :placeholder="env.default"
              class="w-full px-4 py-3 glass rounded-xl text-gray-900 placeholder-gray-400 transition-all">
          </div>
        </div>

        <div class="flex gap-3">
          <button @click="confirmDeploy(selectedApp.id, envValues, {})"
            class="flex-1 px-5 py-3 bg-gradient-to-r from-green-500 to-green-600 hover:from-green-600 hover:to-green-700 text-white rounded-xl font-semibold transition-all smooth-shadow">
            Deploy
          </button>
          <button @click="cancelDeploy"
            class="px-5 py-3 glass hover:bg-white/90 text-gray-700 rounded-xl font-medium transition-all">
            Cancel
          </button>
        </div>
      </div>
    </div>

    <!-- Port Selection Modal -->
    <div v-if="showPortModal"
      class="fixed inset-0 bg-black/30 backdrop-blur-sm flex items-center justify-center z-50"
      @click.self="closePortModal">
      <div class="glass-dark rounded-3xl p-8 max-w-2xl w-full mx-4 smooth-shadow-lg max-h-[90vh] overflow-y-auto">
        <h2 class="text-2xl font-bold mb-2 text-gray-900">{{ selectedAppForPorts?.name }} - Select Port</h2>
        <p class="text-sm text-gray-600 mb-6">This app has multiple ports available</p>

        <div class="space-y-3 mb-6">
          <a v-for="(portObj, index) in selectedAppForPorts?.parsedPorts" :key="index" 
            :href="appUrl(portObj.port, portObj.protocol)" 
            target="_blank"
            @click="closePortModal"
            class="block px-5 py-4 bg-white hover:bg-blue-50 border border-gray-200 hover:border-blue-300 rounded-xl transition-all cursor-pointer group">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center">
                  <i class="fas fa-external-link-alt text-blue-600 text-sm"></i>
                </div>
                <div>
                  <div class="font-semibold text-gray-900 group-hover:text-blue-600 transition-colors">
                    <span v-if="portObj.label">{{ portObj.label }}</span>
                    <span v-else>Port {{ portObj.port }}</span>
                  </div>
                  <div class="text-xs text-gray-500 font-mono">{{ appUrl(portObj.port, portObj.protocol) }}</div>
                </div>
              </div>
              <i class="fas fa-arrow-right text-gray-400 group-hover:text-blue-600 transition-colors"></i>
            </div>
          </a>
        </div>

        <button @click="closePortModal"
          class="w-full px-5 py-3 glass hover:bg-white/90 text-gray-700 rounded-xl font-medium transition-all">
          Cancel
        </button>
      </div>
    </div>
  </div>
</template>
