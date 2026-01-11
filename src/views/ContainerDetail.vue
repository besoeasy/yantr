<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'vue-toastification'
import { ArrowLeft, ExternalLink, ArrowRight, Info, FileText, Tags, Box, Activity, Globe, TrendingUp, Loader2, Settings, Terminal, RefreshCw, Trash2, Cpu, MemoryStick, HardDrive, Network, Play, Square, RotateCcw } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const toast = useToast()

const selectedContainer = ref(null)
const containerStats = ref(null)
const containerLogs = ref([])
const deleting = ref(false)
const apiUrl = ref('')
const refreshingLogs = ref(false)
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
  
  refreshingLogs.value = true
  try {
    const response = await fetch(`${apiUrl.value}/api/containers/${selectedContainer.value.id}/logs?tail=200`)
    const data = await response.json()
    
    if (data.success) {
      containerLogs.value = data.logs
    }
  } catch (error) {
    console.error('Failed to fetch container logs:', error)
  } finally {
    setTimeout(() => {
      refreshingLogs.value = false
    }, 300)
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
  <div class="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800">
    <div class="max-w-7xl mx-auto p-4 sm:p-6 lg:p-8">
      <!-- Loading State -->
      <div v-if="!selectedContainer" class="flex items-center justify-center py-20">
        <div class="text-center">
          <Loader2 :size="48" class="animate-spin text-indigo-500 mx-auto mb-4" />
          <div class="text-gray-600 dark:text-gray-400 font-medium">Loading container details...</div>
        </div>
      </div>
      
      <!-- Main Content -->
      <div v-else class="space-y-6 animate-in fade-in duration-500">
        <!-- Header Section -->
        <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-lg border border-gray-200 dark:border-gray-700 p-6 transition-all hover:shadow-xl">
          <!-- Back Button -->
          <router-link to="/containers"
            class="inline-flex items-center gap-2 text-gray-600 dark:text-gray-400 hover:text-indigo-600 dark:hover:text-indigo-400 mb-6 transition-all group">
            <ArrowLeft :size="18" class="group-hover:-translate-x-1 transition-transform" />
            <span class="font-medium">Back to Containers</span>
          </router-link>
          
          <!-- Container Header -->
          <div class="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-6">
            <div class="flex items-center gap-4">
              <!-- Logo -->
              <div class="relative">
                <img v-if="selectedContainer.app.logo" 
                  :src="selectedContainer.app.logo" 
                  :alt="selectedContainer.name"
                  class="w-16 h-16 sm:w-20 sm:h-20 rounded-2xl object-cover shadow-md ring-4 ring-white dark:ring-gray-800">
                <div v-else class="w-16 h-16 sm:w-20 sm:h-20 rounded-2xl bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-3xl shadow-md ring-4 ring-white dark:ring-gray-800">
                  🐳
                </div>
                <!-- Status Indicator Badge -->
                <div :class="[
                  'absolute -bottom-2 -right-2 w-8 h-8 rounded-full flex items-center justify-center shadow-lg ring-4 ring-white dark:ring-gray-800',
                  selectedContainer.state === 'running' ? 'bg-green-500' : 'bg-gray-400'
                ]">
                  <div class="w-3 h-3 bg-white rounded-full animate-pulse"></div>
                </div>
              </div>
              
              <!-- Title and ID -->
              <div class="min-w-0">
                <h1 class="text-2xl sm:text-3xl lg:text-4xl font-bold text-gray-900 dark:text-white mb-1 truncate">
                  {{ selectedContainer.name }}
                </h1>
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="text-xs sm:text-sm text-gray-500 dark:text-gray-400 font-mono bg-gray-100 dark:bg-gray-700 px-3 py-1 rounded-lg">
                    {{ selectedContainer.id.substring(0, 12) }}
                  </span>
                  <span :class="[
                    'px-3 py-1 rounded-lg text-xs font-semibold uppercase tracking-wide',
                    selectedContainer.state === 'running' 
                      ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' 
                      : 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300'
                  ]">
                    {{ selectedContainer.state }}
                  </span>
                </div>
              </div>
            </div>
            
            <!-- Action Button -->
            <button @click="deleteContainer"
              :disabled="deleting"
              class="flex items-center justify-center gap-2 px-6 py-3 bg-gradient-to-r from-red-500 to-red-600 hover:from-red-600 hover:to-red-700 disabled:from-gray-300 disabled:to-gray-400 disabled:cursor-not-allowed text-white rounded-xl font-semibold shadow-lg hover:shadow-xl disabled:shadow-none transition-all transform hover:scale-105 disabled:transform-none">
              <Trash2 :size="18" :class="{ 'animate-pulse': deleting }" />
              {{ deleting ? 'Deleting...' : 'Delete Container' }}
            </button>
          </div>
        </div>

        <!-- Quick Access Section -->
        <div v-if="getLabeledPorts.length > 0" 
          class="bg-white dark:bg-gray-800 rounded-2xl shadow-lg border border-gray-200 dark:border-gray-700 p-6 transition-all hover:shadow-xl">
          <div class="flex items-center gap-3 mb-4">
            <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center shadow-md">
              <ExternalLink :size="20" class="text-white" />
            </div>
            <h2 class="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white">Quick Access</h2>
          </div>
          
          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            <a v-for="(portInfo, index) in getLabeledPorts" :key="index"
              :href="appUrl(portInfo.port, portInfo.protocol)"
              target="_blank"
              class="group relative flex items-center gap-3 bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-700 dark:to-gray-600 hover:from-indigo-50 hover:to-purple-50 dark:hover:from-indigo-900/30 dark:hover:to-purple-900/30 rounded-xl p-4 border border-gray-200 dark:border-gray-600 hover:border-indigo-300 dark:hover:border-indigo-500 transition-all shadow-sm hover:shadow-md transform hover:scale-105">
              
              <div class="flex-shrink-0 w-12 h-12 rounded-xl bg-white dark:bg-gray-800 group-hover:bg-gradient-to-br group-hover:from-indigo-500 group-hover:to-purple-600 flex items-center justify-center transition-all shadow-sm">
                <Globe :size="20" class="text-gray-600 dark:text-gray-300 group-hover:text-white transition-colors" />
              </div>
              
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2 mb-1">
                  <span class="text-lg font-bold text-gray-900 dark:text-white font-mono">{{ portInfo.port }}</span>
                  <span class="text-xs px-2 py-1 bg-white dark:bg-gray-800 text-gray-600 dark:text-gray-300 rounded-md font-medium uppercase shadow-sm">
                    {{ portInfo.protocol }}
                  </span>
                </div>
                <div class="text-sm text-gray-600 dark:text-gray-400 truncate">{{ portInfo.label }}</div>
              </div>
              
              <ArrowRight :size="18" class="text-gray-400 group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-all group-hover:translate-x-1" />
            </a>
          </div>
        </div>

        <!-- Info and Stats Grid -->
        <div class="grid grid-cols-1 xl:grid-cols-2 gap-6">
          <!-- Container Info Card -->
          <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-lg border border-gray-200 dark:border-gray-700 p-6 transition-all hover:shadow-xl">
            <div class="flex items-center gap-3 mb-6">
              <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-blue-500 to-cyan-600 flex items-center justify-center shadow-md">
                <Info :size="20" class="text-white" />
              </div>
              <h2 class="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white">Container Info</h2>
            </div>
            
            <div class="space-y-4">
              <!-- Description -->
              <div v-if="selectedContainer.app.description" 
                class="bg-gradient-to-br from-blue-50 to-cyan-50 dark:from-blue-900/20 dark:to-cyan-900/20 rounded-xl p-4 border border-blue-200 dark:border-blue-800 transition-all hover:shadow-md">
                <div class="flex items-center gap-2 mb-2">
                  <FileText :size="16" class="text-blue-600 dark:text-blue-400" />
                  <div class="text-xs text-blue-700 dark:text-blue-300 font-semibold uppercase tracking-wide">Description</div>
                </div>
                <p class="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">{{ selectedContainer.app.description }}</p>
              </div>
              
              <!-- Categories -->
              <div v-if="selectedContainer.app.category && selectedContainer.app.category !== 'uncategorized'" 
                class="bg-gradient-to-br from-purple-50 to-pink-50 dark:from-purple-900/20 dark:to-pink-900/20 rounded-xl p-4 border border-purple-200 dark:border-purple-800 transition-all hover:shadow-md">
                <div class="flex items-center gap-2 mb-3">
                  <Tags :size="16" class="text-purple-600 dark:text-purple-400" />
                  <div class="text-xs text-purple-700 dark:text-purple-300 font-semibold uppercase tracking-wide">Categories</div>
                </div>
                <div class="flex flex-wrap gap-2">
                  <span v-for="cat in selectedContainer.app.category.split(',')" :key="cat"
                    class="px-3 py-1.5 rounded-lg bg-gradient-to-r from-purple-500 to-pink-600 text-white text-xs font-semibold shadow-sm hover:shadow-md transition-all transform hover:scale-105">
                    {{ cat.trim() }}
                  </span>
                </div>
              </div>
              
              <!-- Docker Image -->
              <div class="bg-gradient-to-br from-gray-50 to-slate-50 dark:from-gray-700 dark:to-slate-700 rounded-xl p-4 border border-gray-200 dark:border-gray-600 transition-all hover:shadow-md">
                <div class="flex items-center gap-2 mb-2">
                  <Box :size="16" class="text-gray-600 dark:text-gray-400" />
                  <div class="text-xs text-gray-700 dark:text-gray-300 font-semibold uppercase tracking-wide">Docker Image</div>
                </div>
                <div class="text-sm font-mono text-gray-900 dark:text-white bg-white dark:bg-gray-800 px-3 py-2.5 rounded-lg border border-gray-200 dark:border-gray-600 break-all shadow-sm">
                  {{ selectedContainer.image }}
                </div>
              </div>
              
              <!-- Image ID -->
              <div v-if="selectedContainer.imageId" 
                class="bg-gradient-to-br from-gray-50 to-slate-50 dark:from-gray-700 dark:to-slate-700 rounded-xl p-4 border border-gray-200 dark:border-gray-600 transition-all hover:shadow-md">
                <div class="flex items-center gap-2 mb-2">
                  <Box :size="16" class="text-gray-600 dark:text-gray-400" />
                  <div class="text-xs text-gray-700 dark:text-gray-300 font-semibold uppercase tracking-wide">Image SHA256</div>
                </div>
                <div class="text-xs font-mono text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 px-3 py-2.5 rounded-lg border border-gray-200 dark:border-gray-600 break-all shadow-sm">
                  {{ selectedContainer.imageId.replace('sha256:', '') }}
                </div>
              </div>
              
              <!-- Status -->
              <div class="bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-900/20 dark:to-emerald-900/20 rounded-xl p-4 border border-green-200 dark:border-green-800 transition-all hover:shadow-md">
                <div class="flex items-center gap-2 mb-2">
                  <Activity :size="16" class="text-green-600 dark:text-green-400" />
                  <div class="text-xs text-green-700 dark:text-green-300 font-semibold uppercase tracking-wide">Status</div>
                </div>
                <div class="flex items-center gap-2">
                  <div :class="[
                    'w-3 h-3 rounded-full',
                    selectedContainer.state === 'running' ? 'bg-green-500 animate-pulse' : 'bg-gray-400'
                  ]"></div>
                  <span class="text-sm font-semibold text-gray-900 dark:text-white capitalize">{{ selectedContainer.status }}</span>
                </div>
              </div>
              
              <!-- Website Link -->
              <div v-if="selectedContainer.app.website" 
                class="bg-gradient-to-br from-indigo-50 to-blue-50 dark:from-indigo-900/20 dark:to-blue-900/20 rounded-xl p-4 border border-indigo-200 dark:border-indigo-800 transition-all hover:shadow-md">
                <div class="flex items-center gap-2 mb-3">
                  <Globe :size="16" class="text-indigo-600 dark:text-indigo-400" />
                  <div class="text-xs text-indigo-700 dark:text-indigo-300 font-semibold uppercase tracking-wide">Documentation</div>
                </div>
                <a :href="selectedContainer.app.website" target="_blank"
                  class="inline-flex items-center gap-2 px-4 py-2.5 bg-gradient-to-r from-indigo-500 to-blue-600 hover:from-indigo-600 hover:to-blue-700 text-white rounded-lg font-semibold transition-all shadow-md hover:shadow-lg transform hover:scale-105 group">
                  Visit Website
                  <ExternalLink :size="16" class="group-hover:translate-x-1 group-hover:-translate-y-1 transition-transform" />
                </a>
              </div>
            </div>
          </div>

          <!-- Resource Stats Card -->
          <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-lg border border-gray-200 dark:border-gray-700 p-6 transition-all hover:shadow-xl">
            <div class="flex items-center gap-3 mb-6">
              <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-emerald-500 to-teal-600 flex items-center justify-center shadow-md">
                <TrendingUp :size="20" class="text-white" />
              </div>
              <h2 class="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white">Resource Usage</h2>
            </div>
            
            <div v-if="containerStats" class="space-y-6">
              <!-- CPU Usage -->
              <div class="bg-gradient-to-br from-blue-50 to-cyan-50 dark:from-blue-900/20 dark:to-cyan-900/20 rounded-xl p-4 border border-blue-200 dark:border-blue-800">
                <div class="flex items-center justify-between mb-3">
                  <div class="flex items-center gap-2">
                    <Cpu :size="18" class="text-blue-600 dark:text-blue-400" />
                    <span class="text-sm font-semibold text-gray-700 dark:text-gray-300 uppercase tracking-wide">CPU Usage</span>
                  </div>
                  <span class="text-lg font-bold text-blue-600 dark:text-blue-400">{{ containerStats.cpu.percent }}%</span>
                </div>
                <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3 overflow-hidden shadow-inner">
                  <div class="bg-gradient-to-r from-blue-500 to-cyan-600 h-3 rounded-full transition-all duration-500 ease-out shadow-sm" 
                    :style="`width: ${Math.min(containerStats.cpu.percent, 100)}%`"></div>
                </div>
              </div>
              
              <!-- Memory Usage -->
              <div class="bg-gradient-to-br from-purple-50 to-pink-50 dark:from-purple-900/20 dark:to-pink-900/20 rounded-xl p-4 border border-purple-200 dark:border-purple-800">
                <div class="flex items-center justify-between mb-3">
                  <div class="flex items-center gap-2">
                    <MemoryStick :size="18" class="text-purple-600 dark:text-purple-400" />
                    <span class="text-sm font-semibold text-gray-700 dark:text-gray-300 uppercase tracking-wide">Memory</span>
                  </div>
                  <span class="text-lg font-bold text-purple-600 dark:text-purple-400">{{ containerStats.memory.percent }}%</span>
                </div>
                <div class="text-xs text-gray-600 dark:text-gray-400 mb-2 font-medium">
                  {{ formatBytes(containerStats.memory.usage) }} / {{ formatBytes(containerStats.memory.limit) }}
                </div>
                <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3 overflow-hidden shadow-inner">
                  <div class="bg-gradient-to-r from-purple-500 to-pink-600 h-3 rounded-full transition-all duration-500 ease-out shadow-sm" 
                    :style="`width: ${Math.min(containerStats.memory.percent, 100)}%`"></div>
                </div>
              </div>
              
              <!-- Network I/O -->
              <div class="grid grid-cols-2 gap-3">
                <div class="bg-gradient-to-br from-green-50 to-emerald-50 dark:from-green-900/20 dark:to-emerald-900/20 rounded-xl p-4 border border-green-200 dark:border-green-800 transition-all hover:shadow-md">
                  <div class="flex items-center gap-2 mb-2">
                    <Network :size="16" class="text-green-600 dark:text-green-400" />
                    <div class="text-xs text-green-700 dark:text-green-300 font-semibold uppercase tracking-wide">Net RX</div>
                  </div>
                  <div class="text-xl font-bold text-green-700 dark:text-green-400">{{ formatBytes(containerStats.network.rx) }}</div>
                </div>
                <div class="bg-gradient-to-br from-orange-50 to-amber-50 dark:from-orange-900/20 dark:to-amber-900/20 rounded-xl p-4 border border-orange-200 dark:border-orange-800 transition-all hover:shadow-md">
                  <div class="flex items-center gap-2 mb-2">
                    <Network :size="16" class="text-orange-600 dark:text-orange-400" />
                    <div class="text-xs text-orange-700 dark:text-orange-300 font-semibold uppercase tracking-wide">Net TX</div>
                  </div>
                  <div class="text-xl font-bold text-orange-700 dark:text-orange-400">{{ formatBytes(containerStats.network.tx) }}</div>
                </div>
              </div>
              
              <!-- Block I/O -->
              <div class="grid grid-cols-2 gap-3">
                <div class="bg-gradient-to-br from-teal-50 to-cyan-50 dark:from-teal-900/20 dark:to-cyan-900/20 rounded-xl p-4 border border-teal-200 dark:border-teal-800 transition-all hover:shadow-md">
                  <div class="flex items-center gap-2 mb-2">
                    <HardDrive :size="16" class="text-teal-600 dark:text-teal-400" />
                    <div class="text-xs text-teal-700 dark:text-teal-300 font-semibold uppercase tracking-wide">Disk Read</div>
                  </div>
                  <div class="text-xl font-bold text-teal-700 dark:text-teal-400">{{ formatBytes(containerStats.blockIO.read) }}</div>
                </div>
                <div class="bg-gradient-to-br from-rose-50 to-pink-50 dark:from-rose-900/20 dark:to-pink-900/20 rounded-xl p-4 border border-rose-200 dark:border-rose-800 transition-all hover:shadow-md">
                  <div class="flex items-center gap-2 mb-2">
                    <HardDrive :size="16" class="text-rose-600 dark:text-rose-400" />
                    <div class="text-xs text-rose-700 dark:text-rose-300 font-semibold uppercase tracking-wide">Disk Write</div>
                  </div>
                  <div class="text-xl font-bold text-rose-700 dark:text-rose-400">{{ formatBytes(containerStats.blockIO.write) }}</div>
                </div>
              </div>
            </div>
            
            <div v-else class="flex flex-col items-center justify-center py-12">
              <Loader2 :size="32" class="animate-spin text-emerald-500 mb-3" />
              <div class="text-sm text-gray-600 dark:text-gray-400 font-medium">Loading statistics...</div>
            </div>
          </div>
        </div>

        <!-- Environment Variables -->
        <div v-if="selectedContainer.env && selectedContainer.env.length > 0" 
          class="bg-white dark:bg-gray-800 rounded-2xl shadow-lg border border-gray-200 dark:border-gray-700 p-6 transition-all hover:shadow-xl">
          <div class="flex items-center gap-3 mb-4">
            <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-violet-500 to-purple-600 flex items-center justify-center shadow-md">
              <Settings :size="20" class="text-white" />
            </div>
            <h2 class="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white">Environment Variables</h2>
          </div>
          <div class="grid grid-cols-1 lg:grid-cols-2 gap-3 max-h-80 overflow-y-auto custom-scrollbar">
            <div v-for="(envVar, index) in selectedContainer.env" :key="index" 
              class="text-sm font-mono bg-gradient-to-br from-gray-50 to-slate-50 dark:from-gray-700 dark:to-slate-700 px-4 py-3 rounded-lg border border-gray-200 dark:border-gray-600 hover:shadow-md transition-all break-all">
              <span class="text-violet-600 dark:text-violet-400 font-semibold">{{ envVar.split('=')[0] }}</span>
              <span class="text-gray-500 dark:text-gray-400">=</span>
              <span class="text-gray-700 dark:text-gray-300">{{ envVar.split('=').slice(1).join('=') }}</span>
            </div>
          </div>
        </div>

        <!-- Container Logs -->
        <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-lg border border-gray-200 dark:border-gray-700 p-6 transition-all hover:shadow-xl">
          <div class="flex items-center justify-between mb-4 flex-wrap gap-4">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-gray-700 to-gray-900 flex items-center justify-center shadow-md">
                <Terminal :size="20" class="text-white" />
              </div>
              <h2 class="text-xl sm:text-2xl font-bold text-gray-900 dark:text-white">Container Logs</h2>
            </div>
            <button @click="fetchContainerLogs"
              :disabled="refreshingLogs"
              class="inline-flex items-center gap-2 px-4 py-2.5 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 disabled:opacity-50 text-gray-700 dark:text-gray-300 rounded-lg font-medium transition-all shadow-sm hover:shadow-md transform hover:scale-105 disabled:transform-none">
              <RefreshCw :size="16" :class="{ 'animate-spin': refreshingLogs }" />
              Refresh
            </button>
          </div>
          <div class="bg-gray-900 dark:bg-black rounded-xl overflow-hidden shadow-inner border border-gray-700">
            <div class="max-h-96 overflow-y-auto font-mono text-xs leading-relaxed custom-scrollbar">
              <div v-if="containerLogs.length === 0" class="flex flex-col items-center justify-center text-gray-500 py-12">
                <Terminal :size="48" class="mb-3 opacity-30" />
                <div class="text-sm">No logs available</div>
              </div>
              <div v-else class="p-4 space-y-1">
                <div v-for="(log, index) in containerLogs" :key="index" 
                  class="text-gray-300 hover:bg-gray-800 px-3 py-1.5 rounded transition-colors">
                  {{ log }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.custom-scrollbar::-webkit-scrollbar-track {
  background: rgba(0, 0, 0, 0.1);
  border-radius: 4px;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.3);
  border-radius: 4px;
}

.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.5);
}

@keyframes fade-in {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.animate-in {
  animation: fade-in 0.5s ease-out;
}
</style>
