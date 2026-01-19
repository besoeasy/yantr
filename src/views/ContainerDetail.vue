<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'vue-toastification'
import { ArrowLeft, ExternalLink, ArrowRight, Info, FileText, Tags, Box, Activity, Globe, TrendingUp, Loader2, Settings, Terminal, RefreshCw, Trash2, Cpu, MemoryStick, HardDrive, Network, Play, Square, RotateCcw, Eye, Folder, Lock, ShieldCheck, FolderOpen, Database, Share2 } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const toast = useToast()

const selectedContainer = ref(null)
const containerStats = ref(null)
const containerLogs = ref([])
const deleting = ref(false)
const apiUrl = ref('')
const refreshingLogs = ref(false)
const browsingVolume = ref({})
const showVolumeMenu = ref({})
let statsInterval = null

// Extract volume names from container mounts
const containerVolumes = computed(() => {
  if (!selectedContainer.value?.mounts) return []
  
  return selectedContainer.value.mounts
    .filter(mount => mount.Type === 'volume')
    .map(mount => ({
      name: mount.Name,
      destination: mount.Destination,
      rw: mount.RW
    }))
})

// Get all port mappings from the container with labels
const allPortMappings = computed(() => {
  if (!selectedContainer.value || !selectedContainer.value.ports) {
    return []
  }
  
  // Parse port labels from app metadata
  const portLabels = {}
  if (selectedContainer.value.app?.port) {
    const portStr = selectedContainer.value.app.port
    const regex = /(\d+)\s*\(([^-\)]+)\s*-\s*([^)]+)\)/g
    let match
    
    while ((match = regex.exec(portStr)) !== null) {
      portLabels[match[1]] = {
        protocol: match[2].trim().toLowerCase(),
        label: match[3].trim()
      }
    }
  }
  
  const mappings = []
  const portKeys = Object.keys(selectedContainer.value.ports)
  
  portKeys.forEach(key => {
    const [privatePort, type] = key.split('/')
    const bindings = selectedContainer.value.ports[key]
    
    if (bindings && bindings.length > 0) {
      bindings.forEach(binding => {
        if (binding.HostPort) {
          const label = portLabels[privatePort] || portLabels[binding.HostPort]
          mappings.push({
            containerPort: privatePort,
            hostPort: binding.HostPort,
            hostIp: binding.HostIp || '0.0.0.0',
            protocol: type,
            label: label?.label || null,
            labeledProtocol: label?.protocol || null
          })
        }
      })
    } else {
      // Port exposed but not bound to host
      const label = portLabels[privatePort]
      mappings.push({
        containerPort: privatePort,
        hostPort: null,
        hostIp: null,
        protocol: type,
        label: label?.label || null,
        labeledProtocol: label?.protocol || null
      })
    }
  })
  
  // Sort by host port, then by container port
  return mappings.sort((a, b) => {
    if (a.hostPort && b.hostPort) {
      return parseInt(a.hostPort) - parseInt(b.hostPort)
    }
    if (a.hostPort && !b.hostPort) return -1
    if (!a.hostPort && b.hostPort) return 1
    return parseInt(a.containerPort) - parseInt(b.containerPort)
  })
})

// Separate labeled and unlabeled ports
const labeledPorts = computed(() => allPortMappings.value.filter(m => m.label))
const unlabeledPorts = computed(() => allPortMappings.value.filter(m => !m.label))

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

// Get protocol icon and color
function getProtocolInfo(protocol, labeledProtocol) {
  const proto = (labeledProtocol || protocol || 'tcp').toLowerCase()
  
  const protocolMap = {
    'http': { icon: Globe, color: 'text-blue-600', bg: 'bg-blue-100', label: 'HTTP' },
    'https': { icon: Lock, color: 'text-green-600', bg: 'bg-green-100', label: 'HTTPS' },
    'smb': { icon: Share2, color: 'text-purple-600', bg: 'bg-purple-100', label: 'SMB' },
    'ftp': { icon: FolderOpen, color: 'text-orange-600', bg: 'bg-orange-100', label: 'FTP' },
    'sftp': { icon: ShieldCheck, color: 'text-teal-600', bg: 'bg-teal-100', label: 'SFTP' },
    'webdav': { icon: Database, color: 'text-indigo-600', bg: 'bg-indigo-100', label: 'WebDAV' },
    'ssh': { icon: Terminal, color: 'text-gray-700', bg: 'bg-gray-100', label: 'SSH' },
    'tcp': { icon: Network, color: 'text-blue-600', bg: 'bg-blue-100', label: 'TCP' },
    'udp': { icon: Network, color: 'text-cyan-600', bg: 'bg-cyan-100', label: 'UDP' },
  }
  
  return protocolMap[proto] || { icon: Network, color: 'text-gray-600', bg: 'bg-gray-100', label: proto.toUpperCase() }
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
      router.push('/')
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
      router.push('/home')
    } else {
      toast.error(`Deletion failed: ${data.error}`)
    }
  } catch (error) {
    toast.error(`Deletion failed: ${error.message}`)
  } finally {
    deleting.value = false
  }
}

async function browseVolume(volumeName, expiryMinutes = 60) {
  browsingVolume.value[volumeName] = true
  showVolumeMenu.value[volumeName] = false
  try {
    const response = await fetch(`${apiUrl.value}/api/volumes/${volumeName}/browse`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ expiryMinutes }),
    })
    const data = await response.json()
    if (data.success) {
      const expiryText = expiryMinutes > 0 ? ` (expires in ${expiryMinutes}m)` : ' (no expiry)'
      toast.success(`Volume browser started on port ${data.port}${expiryText}`)
      
      // Open browser in new tab
      if (data.port) {
        const host = window.location.hostname || 'localhost'
        const url = `http://${host}:${data.port}`
        window.open(url, '_blank')
      }
    }
  } catch (error) {
    toast.error('Failed to start volume browser')
    console.error(error)
  } finally {
    delete browsingVolume.value[volumeName]
  }
}

function toggleVolumeMenu(volumeName) {
  showVolumeMenu.value[volumeName] = !showVolumeMenu.value[volumeName]
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
  <div class="min-h-screen bg-gray-50">
    <div class="max-w-7xl mx-auto p-4 sm:p-6 lg:p-8">
      <!-- Loading State -->
      <div v-if="!selectedContainer" class="flex items-center justify-center py-16 sm:py-20">
        <div class="text-center">
          <Loader2 :size="40" class="sm:w-12 sm:h-12 animate-spin text-gray-700 mx-auto mb-3 sm:mb-4" />
          <div class="text-gray-600 font-medium text-sm sm:text-base">Loading container details...</div>
        </div>
      </div>
      
      <!-- Main Content -->
      <div v-else class="space-y-4 sm:space-y-6 animate-in fade-in duration-500">
        <!-- Back Button -->
        <router-link to="/apps"
          class="inline-flex items-center gap-2 text-gray-600 hover:text-indigo-600 transition-all group touch-manipulation active:scale-95">
          <ArrowLeft :size="16" class="w-4 h-4 sm:w-5 sm:h-5 group-hover:-translate-x-1 transition-transform" />
          <span class="font-medium text-sm sm:text-base">Back to Apps</span>
        </router-link>
        
        <!-- Header Section -->
        <div class="bg-white rounded-xl sm:rounded-2xl shadow-sm border border-gray-200 p-4 sm:p-6 transition-shadow hover:shadow-md">
          <!-- Container Header -->
          <div class="flex flex-col sm:flex-col lg:flex-row lg:items-center lg:justify-between gap-4 sm:gap-6">
            <div class="flex items-center gap-3 sm:gap-4">
              <!-- Logo -->
              <div class="relative">
                <img v-if="selectedContainer.app.logo" 
                  :src="selectedContainer.app.logo" 
                  :alt="selectedContainer.name"
                  class="w-14 h-14 sm:w-16 sm:h-16 lg:w-20 lg:h-20 rounded-xl sm:rounded-2xl object-cover shadow-md ring-2 sm:ring-4 ring-white">
                <div v-else class="w-14 h-14 sm:w-16 sm:h-16 lg:w-20 lg:h-20 rounded-xl sm:rounded-2xl bg-gray-900 flex items-center justify-center text-2xl sm:text-3xl shadow-md ring-2 sm:ring-4 ring-white">
                  🐳
                </div>
                <!-- Status Indicator Badge -->
                <div :class="[
                  'absolute -bottom-1 -right-1 sm:-bottom-2 sm:-right-2 w-6 h-6 sm:w-8 sm:h-8 rounded-full flex items-center justify-center shadow-lg ring-2 sm:ring-4 ring-white',
                  selectedContainer.state === 'running' ? 'bg-green-500' : 'bg-gray-400'
                ]">
                  <div class="w-2 h-2 sm:w-3 sm:h-3 bg-white rounded-full" :class="{ 'motion-safe:animate-pulse': selectedContainer.state === 'running' }"></div>
                </div>
              </div>
              
              <!-- Title and ID -->
              <div class="min-w-0 flex-1">
                <h1 class="text-xl sm:text-2xl lg:text-3xl xl:text-4xl font-bold text-gray-900 mb-1 truncate">
                  {{ selectedContainer.name }}
                </h1>
                <div class="flex items-center gap-1.5 sm:gap-2 flex-wrap">
                  <span class="text-[10px] sm:text-xs lg:text-sm text-gray-500 font-mono bg-gray-100 px-2 sm:px-3 py-0.5 sm:py-1 rounded-md sm:rounded-lg">
                    {{ selectedContainer.id.substring(0, 12) }}
                  </span>
                  <span :class="[
                    'px-2 sm:px-3 py-0.5 sm:py-1 rounded-md sm:rounded-lg text-[10px] sm:text-xs font-semibold uppercase tracking-wide',
                    selectedContainer.state === 'running' 
                      ? 'bg-green-100 text-green-700' 
                      : 'bg-gray-100 text-gray-700'
                  ]">
                    {{ selectedContainer.state }}
                  </span>
                </div>
              </div>
            </div>
            
            <!-- Action Button -->
            <button @click="deleteContainer"
              :disabled="deleting"
              class="flex items-center justify-center gap-2 px-4 sm:px-6 py-2.5 sm:py-3 bg-red-600 hover:bg-red-700 disabled:bg-gray-300 disabled:cursor-not-allowed text-white rounded-xl font-semibold shadow-sm hover:shadow-md disabled:shadow-none transition-all active:scale-95 touch-manipulation text-sm sm:text-base w-full sm:w-auto">
              <Trash2 :size="16" class="w-4 h-4 sm:w-5 sm:h-5" :class="{ 'animate-pulse': deleting }" />
              {{ deleting ? 'Deleting...' : 'Delete Container' }}
            </button>
          </div>
        </div>

        <!-- Ports & Access Section -->
        <div v-if="allPortMappings.length > 0" 
          class="bg-white rounded-xl sm:rounded-2xl shadow-sm border border-gray-200 p-4 sm:p-6 transition-shadow hover:shadow-md">
          <div class="flex items-center gap-2.5 sm:gap-3 mb-3 sm:mb-4">
            <div class="w-8 h-8 sm:w-10 sm:h-10 rounded-lg sm:rounded-xl bg-indigo-50 flex items-center justify-center border border-indigo-100">
              <Network :size="16" class="sm:w-5 sm:h-5 text-indigo-600" />
            </div>
            <h2 class="text-lg sm:text-xl lg:text-2xl font-bold text-gray-900">Ports & Access</h2>
            <span class="text-xs sm:text-sm text-gray-500 font-medium">({{ allPortMappings.length }})</span>
          </div>
          
          <!-- Ports Table -->
          <div class="overflow-x-auto">
            <table class="w-full">
              <thead>
                <tr class="border-b-2 border-gray-200">
                  <th class="text-left py-3 px-2 sm:px-4 text-xs sm:text-sm font-semibold text-gray-700 uppercase tracking-wide">
                    <div class="flex items-center gap-2">
                      <Network :size="14" class="text-indigo-600" />
                      Protocol
                    </div>
                  </th>
                  <th class="text-left py-3 px-2 sm:px-4 text-xs sm:text-sm font-semibold text-gray-700 uppercase tracking-wide">
                    Host Port
                  </th>
                  <th class="text-left py-3 px-2 sm:px-4 text-xs sm:text-sm font-semibold text-gray-700 uppercase tracking-wide hidden md:table-cell">
                    Container Port
                  </th>
                  <th class="text-left py-3 px-2 sm:px-4 text-xs sm:text-sm font-semibold text-gray-700 uppercase tracking-wide hidden lg:table-cell">
                    Description
                  </th>
                  <th class="text-center py-3 px-2 sm:px-4 text-xs sm:text-sm font-semibold text-gray-700 uppercase tracking-wide">
                    Access
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(mapping, index) in allPortMappings" :key="index"
                  class="border-b border-gray-100 transition-colors animate-in"
                  :class="[
                    mapping.hostPort && mapping.protocol === 'tcp' 
                      ? 'hover:bg-indigo-50/30 cursor-pointer' 
                      : 'opacity-60'
                  ]"
                  :style="{ animationDelay: `${Math.min(index * 20, 400)}ms` }"
                  @click="mapping.hostPort && mapping.protocol === 'tcp' ? window.open(appUrl(mapping.hostPort, mapping.labeledProtocol || 'http'), '_blank') : null">
                  
                  <!-- Protocol Column -->
                  <td class="py-3 px-2 sm:px-4">
                    <div class="flex items-center gap-2">
                      <component
                        :is="getProtocolInfo(mapping.protocol, mapping.labeledProtocol).icon"
                        :size="18"
                        :class="getProtocolInfo(mapping.protocol, mapping.labeledProtocol).color"
                      />
                      <span class="text-[9px] sm:text-xs font-bold uppercase tracking-wider px-2 py-1 rounded whitespace-nowrap"
                        :class="[getProtocolInfo(mapping.protocol, mapping.labeledProtocol).bg, getProtocolInfo(mapping.protocol, mapping.labeledProtocol).color]">
                        {{ getProtocolInfo(mapping.protocol, mapping.labeledProtocol).label }}
                      </span>
                    </div>
                  </td>
                  
                  <!-- Host Port Column -->
                  <td class="py-3 px-2 sm:px-4">
                    <div class="flex flex-col gap-1">
                      <span v-if="mapping.hostPort" class="text-xl sm:text-2xl font-black text-gray-900 font-mono leading-none tracking-tight">
                        {{ mapping.hostPort }}
                      </span>
                      <span v-else class="text-sm text-gray-400 font-medium">Not bound</span>
                      <span class="text-[10px] sm:text-xs text-gray-500 font-mono md:hidden">
                        → :{{ mapping.containerPort }}
                      </span>
                    </div>
                  </td>
                  
                  <!-- Container Port Column (hidden on mobile) -->
                  <td class="py-3 px-2 sm:px-4 hidden md:table-cell">
                    <span class="text-sm sm:text-base text-gray-700 font-mono">
                      :{{ mapping.containerPort }}
                    </span>
                  </td>
                  
                  <!-- Description Column (hidden on mobile/tablet) -->
                  <td class="py-3 px-2 sm:px-4 hidden lg:table-cell">
                    <span v-if="mapping.label" class="text-xs sm:text-sm text-gray-600">
                      {{ mapping.label }}
                    </span>
                    <span v-else class="text-xs sm:text-sm text-gray-400 italic">
                      —
                    </span>
                  </td>
                  
                  <!-- Access Column -->
                  <td class="py-3 px-2 sm:px-4 text-center">
                    <a v-if="mapping.hostPort && mapping.protocol === 'tcp'"
                      :href="appUrl(mapping.hostPort, mapping.labeledProtocol || 'http')"
                      target="_blank"
                      @click.stop
                      class="inline-flex items-center justify-center gap-1.5 px-3 sm:px-4 py-1.5 sm:py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg font-semibold shadow-sm hover:shadow-md transition-all active:scale-95 text-xs sm:text-sm group">
                      <span class="hidden sm:inline">Open</span>
                      <ExternalLink :size="14" class="group-hover:translate-x-0.5 group-hover:-translate-y-0.5 transition-transform" />
                    </a>
                    <span v-else class="text-xs text-gray-400 font-medium">
                      N/A
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Volumes Section -->
        <div v-if="containerVolumes.length > 0" 
          class="bg-white rounded-xl sm:rounded-2xl shadow-sm border border-gray-200 p-4 sm:p-6 transition-shadow hover:shadow-md">
          <div class="flex items-center gap-2.5 sm:gap-3 mb-3 sm:mb-4">
            <div class="w-8 h-8 sm:w-10 sm:h-10 rounded-lg sm:rounded-xl bg-indigo-50 flex items-center justify-center border border-indigo-100">
              <HardDrive :size="16" class="sm:w-5 sm:h-5 text-indigo-600" />
            </div>
            <h2 class="text-lg sm:text-xl lg:text-2xl font-bold text-gray-900">Volumes</h2>
            <span class="text-xs sm:text-sm text-gray-500 font-medium">({{ containerVolumes.length }})</span>
          </div>
          
          <!-- Table -->
          <div class="overflow-x-auto">
            <table class="w-full">
              <thead>
                <tr class="border-b border-gray-200">
                  <th class="text-left py-3 px-2 sm:px-4 text-xs sm:text-sm font-semibold text-gray-700 uppercase tracking-wide">
                    <div class="flex items-center gap-2">
                      <Folder :size="14" class="text-indigo-600" />
                      Volume Name
                    </div>
                  </th>
                  <th class="text-left py-3 px-2 sm:px-4 text-xs sm:text-sm font-semibold text-gray-700 uppercase tracking-wide hidden md:table-cell">
                    Mount Path
                  </th>
                  <th class="text-center py-3 px-2 sm:px-4 text-xs sm:text-sm font-semibold text-gray-700 uppercase tracking-wide">
                    Access
                  </th>
                  <th class="text-right py-3 px-2 sm:px-4 text-xs sm:text-sm font-semibold text-gray-700 uppercase tracking-wide">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="volume in containerVolumes" :key="volume.name"
                  class="border-b border-gray-100 hover:bg-indigo-50/30 transition-colors">
                  <td class="py-3 px-2 sm:px-4">
                    <div class="flex flex-col gap-1">
                      <span class="text-sm sm:text-base font-semibold text-gray-900 break-all" :title="volume.name">
                        {{ volume.name }}
                      </span>
                      <span class="text-xs text-gray-600 font-mono bg-gray-50 px-2 py-1 rounded border border-gray-200 md:hidden break-all" :title="volume.destination">
                        {{ volume.destination }}
                      </span>
                    </div>
                  </td>
                  <td class="py-3 px-2 sm:px-4 hidden md:table-cell">
                    <span class="text-xs sm:text-sm text-gray-600 font-mono bg-gray-50 px-2 py-1 rounded border border-gray-200 inline-block" :title="volume.destination">
                      {{ volume.destination }}
                    </span>
                  </td>
                  <td class="py-3 px-2 sm:px-4 text-center">
                    <span :class="[
                      'px-2 sm:px-3 py-1 rounded-md text-[10px] sm:text-xs font-semibold uppercase inline-block',
                      volume.rw ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600'
                    ]">
                      {{ volume.rw ? 'RW' : 'RO' }}
                    </span>
                  </td>
                  <td class="py-3 px-2 sm:px-4 text-right">
                    <div v-if="!showVolumeMenu[volume.name]" class="inline-flex gap-2">
                      <button
                        @click="showVolumeMenu[volume.name] = true"
                        :disabled="browsingVolume[volume.name]"
                        class="inline-flex items-center gap-1.5 sm:gap-2 px-2.5 sm:px-4 py-1.5 sm:py-2 bg-indigo-600 hover:bg-indigo-700 disabled:bg-gray-300 text-white rounded-lg font-semibold shadow-sm hover:shadow-md disabled:shadow-none transition-all active:scale-95 disabled:transform-none disabled:cursor-not-allowed text-xs sm:text-sm">
                        <Loader2 v-if="browsingVolume[volume.name]" :size="14" class="animate-spin" />
                        <Eye v-else :size="14" />
                        <span class="hidden sm:inline">Browse</span>
                      </button>
                    </div>
                    
                    <!-- Two option buttons -->
                    <div v-else class="inline-flex gap-2 animate-in fade-in">
                      <button
                        @click="browseVolume(volume.name, 60)"
                        :disabled="browsingVolume[volume.name]"
                        :title="'Temporary (1 hour)'"
                        class="inline-flex items-center gap-1.5 px-3 py-2 bg-orange-500 hover:bg-orange-600 disabled:bg-gray-300 text-white rounded-lg font-semibold shadow-sm hover:shadow-md disabled:shadow-none transition-all active:scale-95 disabled:transform-none disabled:cursor-not-allowed text-xs sm:text-sm group">
                        <span class="text-base">⏱️</span>
                        <span class="hidden lg:inline">Temporary</span>
                      </button>
                      <button
                        @click="browseVolume(volume.name, 0)"
                        :disabled="browsingVolume[volume.name]"
                        :title="'Permanent (no expiry)'"
                        class="inline-flex items-center gap-1.5 px-3 py-2 bg-indigo-600 hover:bg-indigo-700 disabled:bg-gray-300 text-white rounded-lg font-semibold shadow-sm hover:shadow-md disabled:shadow-none transition-all active:scale-95 disabled:transform-none disabled:cursor-not-allowed text-xs sm:text-sm group">
                        <span class="text-base">♾️</span>
                        <span class="hidden lg:inline">Permanent</span>
                      </button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Info and Stats (Stacked) -->
        <div class="space-y-4 sm:space-y-6">
          <!-- Container Info Card -->
          <div class="bg-white rounded-xl sm:rounded-2xl shadow-sm border border-gray-200 p-4 sm:p-6 transition-shadow hover:shadow-md">
            <div class="flex items-center gap-2.5 sm:gap-3 mb-4 sm:mb-6">
              <div class="w-8 h-8 sm:w-10 sm:h-10 rounded-lg sm:rounded-xl bg-indigo-50 flex items-center justify-center border border-indigo-100">
                <Info :size="16" class="sm:w-5 sm:h-5 text-indigo-600" />
              </div>
              <h2 class="text-lg sm:text-xl lg:text-2xl font-bold text-gray-900">Container Info</h2>
            </div>
            
            <div class="space-y-3 sm:space-y-4">
              <!-- Description -->
              <div v-if="selectedContainer.app.description" 
                class="bg-gray-50 rounded-lg sm:rounded-xl p-3 sm:p-4 border border-gray-200 transition-shadow hover:shadow-sm">
                <div class="flex items-center gap-1.5 sm:gap-2 mb-1.5 sm:mb-2">
                  <FileText :size="14" class="sm:w-4 sm:h-4 text-indigo-600" />
                  <div class="text-[10px] sm:text-xs text-gray-700 font-semibold uppercase tracking-wide">Description</div>
                </div>
                <p class="text-xs sm:text-sm text-gray-700 leading-relaxed">{{ selectedContainer.app.description }}</p>
              </div>
              
              <!-- Categories -->
              <div v-if="selectedContainer.app.category && selectedContainer.app.category !== 'uncategorized'" 
                class="bg-gray-50 rounded-xl p-4 border border-gray-200 transition-shadow hover:shadow-sm">
                <div class="flex items-center gap-2 mb-3">
                  <Tags :size="16" class="text-indigo-600" />
                  <div class="text-xs text-gray-700 font-semibold uppercase tracking-wide">Categories</div>
                </div>
                <div class="flex flex-wrap gap-2">
                  <span v-for="cat in selectedContainer.app.category.split(',')" :key="cat"
                    class="px-3 py-1.5 rounded-lg bg-indigo-50 border border-indigo-100 text-indigo-700 text-xs font-semibold shadow-sm hover:shadow-md transition-all">
                    {{ cat.trim() }}
                  </span>
                </div>
              </div>
              
              <!-- Docker Image -->
              <div class="bg-gray-50 rounded-xl p-4 border border-gray-200 transition-shadow hover:shadow-sm">
                <div class="flex items-center gap-2 mb-2">
                  <Box :size="16" class="text-gray-600" />
                  <div class="text-xs text-gray-700 font-semibold uppercase tracking-wide">Docker Image</div>
                </div>
                <div class="text-sm font-mono text-gray-900 bg-white px-3 py-2.5 rounded-lg border border-gray-200 break-all shadow-sm">
                  {{ selectedContainer.image }}
                </div>
              </div>
              
              <!-- Image ID -->
              <div v-if="selectedContainer.imageId" 
                class="bg-gray-50 rounded-xl p-4 border border-gray-200 transition-shadow hover:shadow-sm">
                <div class="flex items-center gap-2 mb-2">
                  <Box :size="16" class="text-gray-600" />
                  <div class="text-xs text-gray-700 font-semibold uppercase tracking-wide">Image SHA256</div>
                </div>
                <div class="text-xs font-mono text-gray-700 bg-white px-3 py-2.5 rounded-lg border border-gray-200 break-all shadow-sm">
                  {{ selectedContainer.imageId.replace('sha256:', '') }}
                </div>
              </div>
              
              <!-- Status -->
              <div class="bg-gray-50 rounded-xl p-4 border border-gray-200 transition-shadow hover:shadow-sm">
                <div class="flex items-center gap-2 mb-2">
                  <Activity :size="16" class="text-gray-700" />
                  <div class="text-xs text-gray-700 font-semibold uppercase tracking-wide">Status</div>
                </div>
                <div class="flex items-center gap-2">
                  <div :class="[
                    'w-3 h-3 rounded-full',
                    selectedContainer.state === 'running' ? 'bg-green-500 motion-safe:animate-pulse' : 'bg-gray-400'
                  ]"></div>
                  <span class="text-sm font-semibold text-gray-900 capitalize">{{ selectedContainer.status }}</span>
                </div>
              </div>
              
              <!-- Website Link -->
              <div v-if="selectedContainer.app.website" 
                class="bg-gray-50 rounded-xl p-4 border border-gray-200 transition-shadow hover:shadow-sm">
                <div class="flex items-center gap-2 mb-3">
                  <Globe :size="16" class="text-indigo-600" />
                  <div class="text-xs text-gray-700 font-semibold uppercase tracking-wide">Documentation</div>
                </div>
                <a :href="selectedContainer.app.website" target="_blank"
                  class="inline-flex items-center gap-2 px-4 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg font-semibold transition-all shadow-sm hover:shadow-md active:scale-95 group">
                  Visit Website
                  <ExternalLink :size="16" class="group-hover:translate-x-1 group-hover:-translate-y-1 transition-transform" />
                </a>
              </div>
            </div>
          </div>

          <!-- Resource Stats Card -->
          <div class="bg-white rounded-xl sm:rounded-2xl shadow-sm border border-gray-200 p-4 sm:p-6 transition-shadow hover:shadow-md">
            <div class="flex items-center gap-3 mb-6">
              <div class="w-10 h-10 rounded-xl bg-indigo-50 flex items-center justify-center border border-indigo-100">
                <TrendingUp :size="20" class="text-indigo-600" />
              </div>
              <h2 class="text-xl sm:text-2xl font-bold text-gray-900">Resource Usage</h2>
            </div>
            
            <div v-if="containerStats" class="space-y-6">
              <!-- CPU Usage -->
              <div class="bg-gray-50 rounded-xl p-4 border border-gray-200">
                <div class="flex items-center justify-between mb-3">
                  <div class="flex items-center gap-2">
                    <Cpu :size="18" class="text-indigo-600" />
                    <span class="text-sm font-semibold text-gray-700 uppercase tracking-wide">CPU Usage</span>
                  </div>
                  <span class="text-lg font-bold text-indigo-600">{{ containerStats.cpu.percent }}%</span>
                </div>
                <div class="w-full bg-gray-200 rounded-full h-3 overflow-hidden shadow-inner">
                  <div class="bg-indigo-600 h-3 rounded-full transition-all duration-500 ease-out shadow-sm" 
                    :style="`width: ${Math.min(containerStats.cpu.percent, 100)}%`"></div>
                </div>
              </div>
              
              <!-- Memory Usage -->
              <div class="bg-gray-50 rounded-xl p-4 border border-gray-200">
                <div class="flex items-center justify-between mb-3">
                  <div class="flex items-center gap-2">
                    <MemoryStick :size="18" class="text-indigo-600" />
                    <span class="text-sm font-semibold text-gray-700 uppercase tracking-wide">Memory</span>
                  </div>
                  <span class="text-lg font-bold text-indigo-600">{{ containerStats.memory.percent }}%</span>
                </div>
                <div class="text-xs text-gray-600 mb-2 font-medium">
                  {{ formatBytes(containerStats.memory.usage) }} / {{ formatBytes(containerStats.memory.limit) }}
                </div>
                <div class="w-full bg-gray-200 rounded-full h-3 overflow-hidden shadow-inner">
                  <div class="bg-indigo-600 h-3 rounded-full transition-all duration-500 ease-out shadow-sm" 
                    :style="`width: ${Math.min(containerStats.memory.percent, 100)}%`"></div>
                </div>
              </div>
              
              <!-- Network I/O -->
              <div class="grid grid-cols-2 gap-3">
                <div class="bg-gray-50 rounded-xl p-4 border border-gray-200 transition-shadow hover:shadow-sm">
                  <div class="flex items-center gap-2 mb-2">
                    <Network :size="16" class="text-indigo-600" />
                    <div class="text-xs text-gray-700 font-semibold uppercase tracking-wide">Net RX</div>
                  </div>
                  <div class="text-xl font-bold text-gray-900">{{ formatBytes(containerStats.network.rx) }}</div>
                </div>
                <div class="bg-gray-50 rounded-xl p-4 border border-gray-200 transition-shadow hover:shadow-sm">
                  <div class="flex items-center gap-2 mb-2">
                    <Network :size="16" class="text-indigo-600" />
                    <div class="text-xs text-gray-700 font-semibold uppercase tracking-wide">Net TX</div>
                  </div>
                  <div class="text-xl font-bold text-gray-900">{{ formatBytes(containerStats.network.tx) }}</div>
                </div>
              </div>
              
              <!-- Block I/O -->
              <div class="grid grid-cols-2 gap-3">
                <div class="bg-gray-50 rounded-xl p-4 border border-gray-200 transition-shadow hover:shadow-sm">
                  <div class="flex items-center gap-2 mb-2">
                    <HardDrive :size="16" class="text-indigo-600" />
                    <div class="text-xs text-gray-700 font-semibold uppercase tracking-wide">Disk Read</div>
                  </div>
                  <div class="text-xl font-bold text-gray-900">{{ formatBytes(containerStats.blockIO.read) }}</div>
                </div>
                <div class="bg-gray-50 rounded-xl p-4 border border-gray-200 transition-shadow hover:shadow-sm">
                  <div class="flex items-center gap-2 mb-2">
                    <HardDrive :size="16" class="text-indigo-600" />
                    <div class="text-xs text-gray-700 font-semibold uppercase tracking-wide">Disk Write</div>
                  </div>
                  <div class="text-xl font-bold text-gray-900">{{ formatBytes(containerStats.blockIO.write) }}</div>
                </div>
              </div>
            </div>
            
            <div v-else class="flex flex-col items-center justify-center py-12">
              <Loader2 :size="32" class="animate-spin text-indigo-600 mb-3" />
              <div class="text-sm text-gray-600 font-medium">Loading statistics...</div>
            </div>
          </div>
        </div>

        <!-- Environment Variables -->
        <div v-if="selectedContainer.env && selectedContainer.env.length > 0" 
          class="bg-white rounded-2xl shadow-sm border border-gray-200 p-6 transition-shadow hover:shadow-md">
          <div class="flex items-center gap-3 mb-4">
            <div class="w-10 h-10 rounded-xl bg-indigo-50 flex items-center justify-center border border-indigo-100">
              <Settings :size="20" class="text-indigo-600" />
            </div>
            <h2 class="text-xl sm:text-2xl font-bold text-gray-900">Environment Variables</h2>
          </div>
          <div class="grid grid-cols-1 lg:grid-cols-2 gap-3 max-h-80 overflow-y-auto custom-scrollbar">
            <div v-for="(envVar, index) in selectedContainer.env" :key="index" 
              class="text-sm font-mono bg-gray-50 px-4 py-3 rounded-lg border border-gray-200 hover:shadow-sm transition-shadow break-all">
              <span class="text-indigo-700 font-semibold">{{ envVar.split('=')[0] }}</span>
              <span class="text-gray-500">=</span>
              <span class="text-gray-700">{{ envVar.split('=').slice(1).join('=') }}</span>
            </div>
          </div>
        </div>

        <!-- Container Logs -->
        <div class="bg-white rounded-2xl shadow-sm border border-gray-200 p-6 transition-shadow hover:shadow-md">
          <div class="flex items-center justify-between mb-4 flex-wrap gap-4">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-xl bg-gray-900 flex items-center justify-center shadow-md">
                <Terminal :size="20" class="text-white" />
              </div>
              <h2 class="text-xl sm:text-2xl font-bold text-gray-900">Container Logs</h2>
            </div>
            <button @click="fetchContainerLogs"
              :disabled="refreshingLogs"
              class="inline-flex items-center gap-2 px-4 py-2.5 bg-gray-100 hover:bg-gray-200 disabled:opacity-50 text-gray-700 rounded-lg font-medium transition-all shadow-sm hover:shadow-md active:scale-95 disabled:transform-none">
              <RefreshCw :size="16" :class="{ 'animate-spin': refreshingLogs }" />
              Refresh
            </button>
          </div>
          <div class="bg-gray-900 rounded-xl overflow-hidden shadow-inner border border-gray-700">
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
  opacity: 0;
  transform: translateY(10px);
  animation: fade-in 0.5s ease-out forwards;
}
</style>
