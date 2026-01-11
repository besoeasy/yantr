<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'vue-toastification'

const route = useRoute()
const router = useRouter()
const toast = useToast()

const selectedContainer = ref(null)
const containerStats = ref(null)
const containerLogs = ref([])
const deleting = ref(false)
const apiUrl = ref('')
let statsInterval = null

const getLabeledPorts = computed(() => {
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
})

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

function formatBytes(bytes) {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

async function fetchContainerDetail() {
  try {
    const response = await fetch(`${apiUrl.value}/api/containers/${route.params.id}`)
    const data = await response.json()
    
    if (data.success) {
      selectedContainer.value = data.container
    } else {
      toast.error('Container not found')
      router.push('/containers')
    }
  } catch (error) {
    console.error('Failed to fetch container details:', error)
    toast.error('Failed to load container details')
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

async function deleteContainer() {
  if (!confirm(`Delete ${selectedContainer.value.name}?\n\nThis will remove the container and all its volumes permanently.`)) return

  deleting.value = true
  try {
    const response = await fetch(`${apiUrl.value}/api/containers/${selectedContainer.value.id}`, {
      method: 'DELETE'
    })
    const data = await response.json()

    if (data.success) {
      let message = `${selectedContainer.value.name} deleted successfully!`
      if (data.volumesRemoved.length > 0) {
        message += `\n\nVolumes removed: ${data.volumesRemoved.join(', ')}`
      }
      toast.success(message)
      router.push('/containers')
    } else {
      toast.error(`Deletion failed: ${data.error}`)
    }
  } catch (error) {
    toast.error(`Deletion failed: ${error.message}`)
  } finally {
    deleting.value = false
  }
}

onMounted(async () => {
  await fetchContainerDetail()
  await Promise.all([
    fetchContainerStats(),
    fetchContainerLogs()
  ])
  
  // Start polling stats every 2 seconds
  statsInterval = setInterval(() => {
    fetchContainerStats()
  }, 2000)
})

onUnmounted(() => {
  if (statsInterval) {
    clearInterval(statsInterval)
  }
})
</script>

<template>
  <div class="p-6 md:p-10 lg:p-12">
    <div v-if="!selectedContainer" class="text-center py-16">
      <div class="text-gray-500 font-medium">Loading container details...</div>
    </div>
    
    <div v-else>
      <!-- Header with back button -->
      <div class="mb-8">
        <router-link to="/containers"
          class="flex items-center gap-2 text-gray-600 hover:text-gray-900 mb-4 transition-colors">
          <i class="fas fa-arrow-left"></i>
          <span>Back to Containers</span>
        </router-link>
        
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-4">
            <img v-if="selectedContainer.app.logo" :src="selectedContainer.app.logo" :alt="selectedContainer.name"
              class="w-16 h-16 rounded-xl object-cover">
            <span v-else class="text-5xl">🐳</span>
            <div>
              <h2 class="text-4xl font-bold text-gray-900">{{ selectedContainer.name }}</h2>
              <div class="text-sm text-gray-500 font-mono mt-2">{{ selectedContainer.id.substring(0, 12) }}</div>
            </div>
          </div>
          <div class="flex items-center gap-3">
            <div :class="selectedContainer.state === 'running' ? 'bg-green-500' : 'bg-gray-400'" 
              class="px-4 py-2 rounded-full text-white text-sm font-semibold flex items-center gap-2">
              <span class="w-2 h-2 bg-white rounded-full animate-pulse"></span>
              {{ selectedContainer.state }}
            </div>
            <button @click="deleteContainer"
              :disabled="deleting"
              class="px-4 py-2.5 bg-red-500 hover:bg-red-600 disabled:bg-gray-300 disabled:cursor-not-allowed text-white rounded-xl text-sm font-semibold transition-all">
              {{ deleting ? 'Deleting...' : '🗑️ Delete' }}
            </button>
          </div>
        </div>
      </div>

      <!-- Quick Access Ports (from yantra.port labels) -->
      <div v-if="getLabeledPorts.length > 0" class="mb-6">
        <h3 class="text-xl font-bold mb-4 text-gray-900">Quick Access</h3>
        
        <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
          <a v-for="(portInfo, index) in getLabeledPorts" :key="index"
            :href="appUrl(portInfo.port, portInfo.protocol)"
            target="_blank"
            class="group flex items-center gap-3 bg-white hover:bg-gray-50 rounded-xl p-4 border border-gray-200 hover:border-gray-300 transition-all">
            
            <div class="flex-shrink-0 w-10 h-10 rounded-lg bg-gray-100 group-hover:bg-indigo-500 flex items-center justify-center transition-colors">
              <i class="fas fa-external-link-alt text-gray-600 group-hover:text-white text-sm transition-colors"></i>
            </div>
            
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-1">
                <span class="text-lg font-bold text-gray-900 font-mono">{{ portInfo.port }}</span>
                <span class="text-xs px-2 py-0.5 bg-gray-100 text-gray-600 rounded font-medium uppercase">
                  {{ portInfo.protocol }}
                </span>
              </div>
              <div class="text-sm text-gray-600 truncate">{{ portInfo.label }}</div>
            </div>
            
            <i class="fas fa-arrow-right text-gray-400 group-hover:text-gray-600 text-sm transition-colors"></i>
          </a>
        </div>
      </div>

      <!-- Container Info Grid -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        <!-- Basic Info Card -->
        <div class="bg-gradient-to-br from-white to-gray-50 rounded-2xl p-6 border border-gray-200 shadow-sm">
          <h3 class="text-xl font-bold mb-6 flex items-center gap-2">
            <div class="w-10 h-10 rounded-lg bg-gradient-to-br from-blue-500 to-blue-600 flex items-center justify-center">
              <i class="fas fa-info-circle text-white"></i>
            </div>
            <span>Container Info</span>
          </h3>
          <div class="space-y-4">
            <div v-if="selectedContainer.app.description" class="bg-white rounded-xl p-4 border border-gray-100">
              <div class="flex items-center gap-2 mb-2">
                <i class="fas fa-file-alt text-gray-400"></i>
                <div class="text-xs text-gray-500 font-semibold uppercase">Description</div>
              </div>
              <div class="text-sm text-gray-900 leading-relaxed">{{ selectedContainer.app.description }}</div>
            </div>
            
            <div v-if="selectedContainer.app.category && selectedContainer.app.category !== 'uncategorized'" 
              class="bg-white rounded-xl p-4 border border-gray-100">
              <div class="flex items-center gap-2 mb-2">
                <i class="fas fa-tags text-gray-400"></i>
                <div class="text-xs text-gray-500 font-semibold uppercase">Categories</div>
              </div>
              <div class="flex flex-wrap gap-2">
                <span v-for="cat in selectedContainer.app.category.split(',')" :key="cat"
                  class="px-3 py-1.5 rounded-lg bg-gradient-to-r from-blue-500 to-blue-600 text-white text-xs font-semibold shadow-sm">
                  {{ cat.trim() }}
                </span>
              </div>
            </div>
            
            <div class="bg-white rounded-xl p-4 border border-gray-100">
              <div class="flex items-center gap-2 mb-2">
                <i class="fas fa-cube text-gray-400"></i>
                <div class="text-xs text-gray-500 font-semibold uppercase">Docker Image</div>
              </div>
              <div class="text-sm font-mono text-gray-900 bg-gray-50 px-3 py-2 rounded-lg border border-gray-200">
                {{ selectedContainer.image }}
              </div>
            </div>
            
            <div class="bg-white rounded-xl p-4 border border-gray-100">
              <div class="flex items-center gap-2 mb-2">
                <i class="fas fa-heartbeat text-gray-400"></i>
                <div class="text-xs text-gray-500 font-semibold uppercase">Status</div>
              </div>
              <div class="flex items-center gap-2">
                <div :class="selectedContainer.state === 'running' ? 'bg-green-500' : 'bg-gray-400'" 
                  class="w-3 h-3 rounded-full animate-pulse"></div>
                <span class="text-sm font-semibold text-gray-900 capitalize">{{ selectedContainer.status }}</span>
              </div>
            </div>
            
            <div v-if="selectedContainer.app.website" class="bg-white rounded-xl p-4 border border-gray-100">
              <div class="flex items-center gap-2 mb-2">
                <i class="fas fa-globe text-gray-400"></i>
                <div class="text-xs text-gray-500 font-semibold uppercase">Website</div>
              </div>
              <a :href="selectedContainer.app.website" target="_blank"
                class="inline-flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 text-white rounded-lg text-sm font-semibold transition-all shadow-sm">
                Visit Website
                <i class="fas fa-external-link-alt text-xs"></i>
              </a>
            </div>
          </div>
        </div>

        <!-- Resource Stats Card -->
        <div class="bg-white rounded-2xl p-6 border border-gray-200">
          <h3 class="text-xl font-bold mb-4 flex items-center gap-2">
            <i class="fas fa-chart-line text-green-500"></i>
            <span>Resource Usage</span>
          </h3>
          <div v-if="containerStats" class="space-y-4">
            <!-- CPU Usage -->
            <div>
              <div class="flex justify-between items-center mb-2">
                <span class="text-xs text-gray-500 font-semibold uppercase">CPU</span>
                <span class="text-sm font-bold text-gray-900">{{ containerStats.cpu.percent }}%</span>
              </div>
              <div class="w-full bg-gray-200 rounded-full h-2">
                <div class="bg-blue-500 h-2 rounded-full transition-all duration-300" 
                  :style="`width: ${Math.min(containerStats.cpu.percent, 100)}%`"></div>
              </div>
            </div>
            
            <!-- Memory Usage -->
            <div>
              <div class="flex justify-between items-center mb-2">
                <span class="text-xs text-gray-500 font-semibold uppercase">Memory</span>
                <span class="text-sm font-bold text-gray-900">
                  {{ formatBytes(containerStats.memory.usage) }} / {{ formatBytes(containerStats.memory.limit) }}
                  ({{ containerStats.memory.percent }}%)
                </span>
              </div>
              <div class="w-full bg-gray-200 rounded-full h-2">
                <div class="bg-green-500 h-2 rounded-full transition-all duration-300" 
                  :style="`width: ${Math.min(containerStats.memory.percent, 100)}%`"></div>
              </div>
            </div>
            
            <!-- Network I/O -->
            <div class="grid grid-cols-2 gap-4 pt-2">
              <div class="bg-blue-50 rounded-lg p-3">
                <div class="text-xs text-blue-600 font-semibold uppercase mb-1">Network RX</div>
                <div class="text-lg font-bold text-blue-700">{{ formatBytes(containerStats.network.rx) }}</div>
              </div>
              <div class="bg-purple-50 rounded-lg p-3">
                <div class="text-xs text-purple-600 font-semibold uppercase mb-1">Network TX</div>
                <div class="text-lg font-bold text-purple-700">{{ formatBytes(containerStats.network.tx) }}</div>
              </div>
            </div>
            
            <!-- Block I/O -->
            <div class="grid grid-cols-2 gap-4">
              <div class="bg-green-50 rounded-lg p-3">
                <div class="text-xs text-green-600 font-semibold uppercase mb-1">Block Read</div>
                <div class="text-lg font-bold text-green-700">{{ formatBytes(containerStats.blockIO.read) }}</div>
              </div>
              <div class="bg-orange-50 rounded-lg p-3">
                <div class="text-xs text-orange-600 font-semibold uppercase mb-1">Block Write</div>
                <div class="text-lg font-bold text-orange-700">{{ formatBytes(containerStats.blockIO.write) }}</div>
              </div>
            </div>
          </div>
          <div v-else class="text-center py-8 text-gray-500">
            <i class="fas fa-spinner fa-spin text-2xl mb-2"></i>
            <div class="text-sm">Loading stats...</div>
          </div>
        </div>
      </div>

      <!-- Environment Variables -->
      <div v-if="selectedContainer.env && selectedContainer.env.length > 0" class="bg-white rounded-2xl p-6 border border-gray-200 mb-6">
        <h3 class="text-xl font-bold mb-4 flex items-center gap-2">
          <i class="fas fa-cog text-purple-500"></i>
          <span>Environment Variables</span>
        </h3>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3 max-h-64 overflow-y-auto">
          <div v-for="(envVar, index) in selectedContainer.env" :key="index" 
            class="text-xs font-mono bg-gray-50 px-3 py-2 rounded border border-gray-200">
            <span class="text-purple-600 font-semibold">{{ envVar.split('=')[0] }}</span>
            <span class="text-gray-500">=</span>
            <span class="text-gray-700">{{ envVar.split('=').slice(1).join('=') }}</span>
          </div>
        </div>
      </div>

      <!-- Container Logs -->
      <div class="bg-white rounded-2xl p-6 border border-gray-200">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-xl font-bold flex items-center gap-2">
            <i class="fas fa-terminal text-gray-700"></i>
            <span>Container Logs</span>
          </h3>
          <button @click="fetchContainerLogs"
            class="px-3 py-1.5 bg-gray-100 hover:bg-gray-200 text-gray-700 rounded-lg text-sm font-medium transition-all">
            <i class="fas fa-sync-alt mr-1"></i>
            Refresh
          </button>
        </div>
        <div class="bg-gray-900 rounded-xl overflow-hidden">
          <div class="max-h-96 overflow-y-auto font-mono text-xs p-4">
            <div v-if="containerLogs.length === 0" class="text-gray-500 text-center py-8">
              No logs available
            </div>
            <div v-else>
              <div v-for="(log, index) in containerLogs" :key="index" 
                class="text-gray-300 hover:bg-gray-800 px-2 py-1 rounded">
                {{ log }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
