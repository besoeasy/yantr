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
    'http': { icon: Globe, color: 'text-blue-600 dark:text-blue-300', bg: 'bg-blue-100 dark:bg-blue-500/15', label: 'HTTP' },
    'https': { icon: Lock, color: 'text-green-600 dark:text-emerald-300', bg: 'bg-green-100 dark:bg-emerald-500/15', label: 'HTTPS' },
    'smb': { icon: Share2, color: 'text-purple-600 dark:text-purple-300', bg: 'bg-purple-100 dark:bg-purple-500/15', label: 'SMB' },
    'ftp': { icon: FolderOpen, color: 'text-orange-600 dark:text-orange-300', bg: 'bg-orange-100 dark:bg-orange-500/15', label: 'FTP' },
    'sftp': { icon: ShieldCheck, color: 'text-teal-600 dark:text-teal-300', bg: 'bg-teal-100 dark:bg-teal-500/15', label: 'SFTP' },
    'webdav': { icon: Database, color: 'text-indigo-600 dark:text-indigo-300', bg: 'bg-indigo-100 dark:bg-indigo-500/15', label: 'WebDAV' },
    'ssh': { icon: Terminal, color: 'text-gray-700 dark:text-slate-300', bg: 'bg-gray-100 dark:bg-slate-800', label: 'SSH' },
    'tcp': { icon: Network, color: 'text-blue-600 dark:text-blue-300', bg: 'bg-blue-100 dark:bg-blue-500/15', label: 'TCP' },
    'udp': { icon: Network, color: 'text-cyan-600 dark:text-cyan-300', bg: 'bg-cyan-100 dark:bg-cyan-500/15', label: 'UDP' },
  }
  
  return protocolMap[proto] || { icon: Network, color: 'text-gray-600 dark:text-slate-300', bg: 'bg-gray-100 dark:bg-slate-800', label: proto.toUpperCase() }
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
  <div class="min-h-screen bg-slate-50 dark:bg-[#09090b] text-slate-900 dark:text-slate-200 font-sans pb-32">
    
    <!-- Top Navigation -->
    <nav class="sticky top-0 z-40 bg-white/80 dark:bg-[#09090b]/90 backdrop-blur-md border-b border-slate-200 dark:border-slate-800">
      <div class="max-w-7xl mx-auto px-4 lg:px-8 h-16 flex items-center justify-between">
        <router-link to="/apps"
          class="flex items-center gap-2 text-sm font-semibold tracking-wide uppercase text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-white transition-colors">
          <ArrowLeft :size="18" />
          <span>Dashboard</span>
        </router-link>
        
        <div v-if="selectedContainer" class="flex items-center gap-3">
          <span class="text-xs font-mono text-slate-400 dark:text-slate-600">{{ selectedContainer.id.substring(0, 12) }}</span>
          <div :class="[
            'flex items-center gap-1.5 px-2 py-0.5 border text-xs font-bold uppercase tracking-wider',
            selectedContainer.state === 'running' 
              ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-600 dark:text-emerald-400' 
              : 'bg-slate-500/10 border-slate-500/20 text-slate-600 dark:text-slate-400'
          ]">
            <div :class="['w-1.5 h-1.5 rounded-full', selectedContainer.state === 'running' ? 'bg-emerald-500 animate-pulse' : 'bg-slate-500']"></div>
            {{ selectedContainer.state }}
          </div>
        </div>
      </div>
    </nav>

    <!-- Loading State -->
    <div v-if="!selectedContainer" class="flex flex-col items-center justify-center min-h-[60vh]">
      <div class="relative">
        <div class="w-16 h-16 border-2 border-slate-200 dark:border-slate-800 rounded-none transform rotate-45"></div>
        <div class="absolute inset-0 w-16 h-16 border-2 border-sky-500 border-t-transparent border-l-transparent rounded-none animate-spin"></div>
      </div>
      <div class="mt-8 font-mono text-sm tracking-widest text-slate-400 uppercase">Connecting to Host...</div>
    </div>

    <!-- Main Content -->
    <div v-else class="max-w-7xl mx-auto px-4 lg:px-8 py-8 lg:py-12">
      <div class="grid grid-cols-1 lg:grid-cols-12 gap-10">
        
        <!-- Left Column: Metrics & Logs -->
        <div class="lg:col-span-8 space-y-10">
          
          <!-- Identity Card -->
          <div class="flex flex-col sm:flex-row gap-6 sm:gap-8 border-b border-dashed border-slate-300 dark:border-slate-800 pb-10">
            <div class="w-24 h-24 sm:w-32 sm:h-32 bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 p-4 flex items-center justify-center shrink-0">
              <img v-if="selectedContainer.app.logo" :src="selectedContainer.app.logo" :alt="selectedContainer.name" class="w-full h-full object-contain" />
              <div v-else class="text-4xl">📦</div>
            </div>
            
            <div class="flex-1 space-y-4">
              <div>
                <h1 class="text-3xl sm:text-4xl font-extrabold tracking-tight text-slate-900 dark:text-white mb-2">{{ selectedContainer.name }}</h1>
                <div class="flex flex-wrap gap-2">
                  <span class="px-2 py-0.5 bg-slate-100 dark:bg-slate-800/50 text-slate-600 dark:text-slate-400 text-xs font-mono border border-slate-200 dark:border-slate-700 uppercase"
                    :title="selectedContainer.image">
                    {{ selectedContainer.image }}
                  </span>
                </div>
              </div>
              
              <p class="text-base sm:text-lg leading-relaxed text-slate-600 dark:text-slate-300 max-w-3xl">
                {{ selectedContainer.app.description || "Container running without specific application metadata." }}
              </p>
            </div>
          </div>

          <!-- Network Config -->
          <div v-if="allPortMappings.length > 0" class="space-y-6">
             <div class="flex items-center justify-between border-l-2 border-indigo-500 pl-3">
               <h3 class="text-sm font-mono uppercase tracking-widest text-slate-500 dark:text-slate-500">Network Bindings</h3>
             </div>
             
             <div class="border border-slate-200 dark:border-slate-800 bg-white dark:bg-[#0c0c0e] overflow-hidden">
                <div class="overflow-x-auto">
                   <table class="w-full text-left border-collapse">
                      <thead>
                         <tr class="bg-slate-50 dark:bg-slate-900/50 border-b border-slate-200 dark:border-slate-800 text-[10px] uppercase text-slate-500 font-bold tracking-wider">
                            <th class="py-3 px-4 font-mono whitespace-nowrap">Protocol</th>
                            <th class="py-3 px-4 font-mono whitespace-nowrap">Host Port</th>
                            <th class="py-3 px-4 font-mono whitespace-nowrap">Container Port</th>
                            <th class="py-3 px-4 font-mono w-full">Description</th>
                            <th class="py-3 px-4 text-right font-mono whitespace-nowrap">Access</th>
                         </tr>
                      </thead>
                      <tbody class="divide-y divide-slate-100 dark:divide-slate-800/50">
                         <tr v-for="(mapping, i) in allPortMappings" :key="i" class="hover:bg-slate-50 dark:hover:bg-slate-900/30 transition-colors group">
                            <td class="py-3 px-4">
                               <div class="flex items-center gap-2">
                                  <Network :size="14" :class="mapping.protocol === 'tcp' ? 'text-indigo-500' : 'text-slate-500'" />
                                  <span class="font-mono font-bold uppercase text-slate-700 dark:text-slate-300 text-xs">{{ mapping.protocol }}</span>
                                  <span v-if="mapping.labeledProtocol" class="text-[9px] px-1.5 py-0.5 bg-slate-100 dark:bg-slate-800 text-slate-500 rounded uppercase font-bold border border-slate-200 dark:border-slate-700">{{ mapping.labeledProtocol }}</span>
                               </div>
                            </td>
                            <td class="py-3 px-4 font-mono">
                               <span v-if="mapping.hostPort" class="text-slate-900 dark:text-white font-bold text-sm">{{ mapping.hostPort }}</span>
                               <span v-else class="text-slate-400 italic text-xs">unbound</span>
                            </td>
                            <td class="py-3 px-4 font-mono text-slate-600 dark:text-slate-400 text-sm">
                               {{ mapping.containerPort }}
                            </td>
                            <td class="py-3 px-4">
                               <span v-if="mapping.label" class="text-slate-600 dark:text-slate-400 font-mono text-xs">{{ mapping.label }}</span>
                               <span v-else class="text-slate-400 italic text-xs opacity-50">--</span>
                            </td>
                            <td class="py-3 px-4 text-right">
                               <a v-if="mapping.hostPort && mapping.protocol === 'tcp'"
                                  :href="appUrl(mapping.hostPort, mapping.labeledProtocol || 'http')"
                                  target="_blank"
                                  class="inline-flex items-center gap-1.5 px-3 py-1 bg-indigo-50 dark:bg-indigo-500/10 hover:bg-indigo-100 dark:hover:bg-indigo-500/20 text-indigo-600 dark:text-indigo-400 border border-indigo-200 dark:border-indigo-500/30 rounded-sm text-[10px] font-bold uppercase tracking-wide transition-colors"
                               >
                                  <span>Open</span>
                                  <ExternalLink :size="12" />
                               </a>
                               <span v-else class="text-[10px] text-slate-400 uppercase tracking-wide opacity-50 cursor-not-allowed">Local</span>
                            </td>
                         </tr>
                      </tbody>
                   </table>
                </div>
             </div>
          </div>

          <!-- Volumes -->
          <div v-if="containerVolumes.length > 0" class="space-y-6">
             <div class="flex items-center justify-between border-l-2 border-sky-500 pl-3">
               <h3 class="text-sm font-mono uppercase tracking-widest text-slate-500 dark:text-slate-500">Storage & File Browser</h3>
             </div>
             
             <div class="grid gap-4">
                <div v-for="(volume, idx) in containerVolumes" :key="volume.name" 
                     class="group flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-4 border border-slate-200 dark:border-slate-800 bg-white dark:bg-[#0c0c0e] hover:border-sky-500/30 transition-all duration-300 hover:shadow-lg hover:shadow-sky-500/5 hover:-translate-y-1 animate-in"
                     :style="{ animationDelay: `${idx * 100}ms` }">
                   <div class="min-w-0 flex-1">
                      <div class="flex items-center gap-2 mb-1">
                         <HardDrive :size="16" class="text-sky-500 group-hover:scale-110 transition-transform duration-300" />
                         <span class="font-mono text-sm font-bold truncate group-hover:text-sky-500 transition-colors" :title="volume.name">{{ volume.name }}</span>
                      </div>
                      <div class="text-xs font-mono text-slate-500 truncate pl-6">{{ volume.destination }}</div>
                   </div>
                   
                   <div class="flex items-center gap-3 shrink-0">
                      <span :class="['text-[10px] font-bold uppercase px-2 py-0.5 border', volume.rw ? 'border-emerald-500/30 text-emerald-600 dark:text-emerald-400' : 'border-slate-500/30 text-slate-500']">
                         {{ volume.rw ? 'RW' : 'RO' }}
                      </span>
                      
                      <div class="flex items-center gap-2">
                        <button v-if="!showVolumeMenu[volume.name]" @click="showVolumeMenu[volume.name] = true" class="group/btn flex items-center gap-2 px-3 py-1.5 bg-slate-50 dark:bg-slate-900 hover:bg-sky-50 dark:hover:bg-sky-900/20 border border-slate-200 dark:border-slate-800 hover:border-sky-500/50 dark:hover:border-sky-500/50 text-slate-600 dark:text-slate-400 hover:text-sky-600 dark:hover:text-sky-400 transition-all cursor-pointer rounded-sm">
                           <FolderOpen :size="14" class="group-hover/btn:scale-110 transition-transform duration-300" />
                           <span class="text-[10px] font-bold uppercase tracking-wider">Browse Files</span>
                        </button>
                        
                        <div v-else class="flex items-center gap-2 animate-in slide-in-from-right-2 fade-in duration-200 bg-slate-100 dark:bg-slate-900 border border-slate-200 dark:border-slate-800 p-1 rounded-sm">
                           <button @click="browseVolume(volume.name, 60)" class="text-[10px] font-bold uppercase px-3 py-1 bg-sky-500 hover:bg-sky-600 text-white transition-all shadow-sm hover:shadow-sky-500/20 hover:scale-105 active:scale-95 rounded-sm" title="Mount for 1 hour">Temp (1h)</button>
                           <button @click="browseVolume(volume.name, 0)" class="text-[10px] font-bold uppercase px-3 py-1 bg-slate-700 hover:bg-slate-600 text-white transition-all shadow-sm hover:scale-105 active:scale-95 rounded-sm" title="Mount permanently">Perm</button>
                        </div>
                      </div>
                   </div>
                </div>
             </div>
          </div>

          <!-- Logs Console -->
          <div class="space-y-4">
            <div class="flex items-center justify-between">
              <h3 class="text-sm font-mono uppercase tracking-widest text-slate-500 dark:text-slate-500 border-l-2 border-slate-500 pl-3">System Output</h3>
              <button @click="fetchContainerLogs" class="text-xs font-mono uppercase text-sky-500 hover:text-sky-400 flex items-center gap-2">
                 <RefreshCw :size="12" :class="{ 'animate-spin': refreshingLogs }" />
                 Refresh Logs
              </button>
            </div>
            
            <div class="bg-[#0c0c0e] border border-slate-200 dark:border-slate-800 p-4 font-mono text-xs h-96 overflow-y-auto custom-scrollbar text-slate-300">
               <div v-if="containerLogs.length === 0" class="h-full flex flex-col items-center justify-center text-slate-600">
                  <Terminal :size="32" class="mb-2 opacity-50" />
                  <span>No output detected</span>
               </div>
               <div v-else class="space-y-1">
                  <div v-for="(log, i) in containerLogs" :key="i" class="break-all whitespace-pre-wrap font-mono">{{ log }}</div>
               </div>
            </div>
          </div>
          
        </div>

        <!-- Right Column: Network & Actions -->
        <div class="lg:col-span-4 space-y-8">
           
           <!-- Resource Usage -->
           <div v-if="containerStats">
              <h3 class="text-sm font-mono uppercase tracking-widest text-slate-500 dark:text-slate-500 border-l-2 border-pink-500 pl-3 mb-4">Resource Usage</h3>
              <div class="grid grid-cols-2 gap-3">
                <div class="bg-white dark:bg-[#0c0c0e] border border-slate-200 dark:border-slate-800 p-3">
                   <div class="text-[10px] uppercase tracking-wider text-slate-500 mb-1">CPU</div>
                   <div class="text-lg font-mono font-bold text-slate-900 dark:text-white">{{ containerStats.cpu.percent }}%</div>
                </div>
                <div class="bg-white dark:bg-[#0c0c0e] border border-slate-200 dark:border-slate-800 p-3">
                   <div class="text-[10px] uppercase tracking-wider text-slate-500 mb-1">Memory</div>
                   <div class="text-lg font-mono font-bold text-slate-900 dark:text-white">{{ containerStats.memory.percent }}%</div>
                </div>
                <div class="bg-white dark:bg-[#0c0c0e] border border-slate-200 dark:border-slate-800 p-3">
                   <div class="text-[10px] uppercase tracking-wider text-slate-500 mb-1">Net I/O</div>
                   <div class="text-lg font-mono font-bold text-slate-900 dark:text-white text-xs truncate">{{ formatBytes(containerStats.network.rx) }} / {{ formatBytes(containerStats.network.tx) }}</div>
                </div>
                <div class="bg-white dark:bg-[#0c0c0e] border border-slate-200 dark:border-slate-800 p-3">
                   <div class="text-[10px] uppercase tracking-wider text-slate-500 mb-1">Block I/O</div>
                   <div class="text-lg font-mono font-bold text-slate-900 dark:text-white text-xs truncate">{{ formatBytes(containerStats.blockIO.read) }} / {{ formatBytes(containerStats.blockIO.write) }}</div>
                </div>
              </div>
           </div>

           <!-- Environment Vars Small View -->
           <div v-if="selectedContainer.env && selectedContainer.env.length > 0">
              <h3 class="text-sm font-mono uppercase tracking-widest text-slate-500 dark:text-slate-500 border-l-2 border-emerald-500 pl-3 mb-4">Runtime Env</h3>
              <div class="bg-[#0c0c0e] border border-slate-200 dark:border-slate-800 p-4 max-h-64 overflow-y-auto custom-scrollbar">
                 <div v-for="(envVar, i) in selectedContainer.env" :key="i" class="font-mono text-[10px] mb-2 break-all border-b border-slate-800 pb-2 last:border-0 last:mb-0 last:pb-0">
                    <span class="text-emerald-500 block mb-0.5">{{ envVar.split('=')[0] }}</span>
                    <span class="text-slate-300 block pl-2">{{ envVar.split('=').slice(1).join('=') }}</span>
                 </div>
              </div>
           </div>

           <!-- Action Panel -->
           <div class="bg-white dark:bg-[#0c0c0e] border border-slate-200 dark:border-slate-800 p-6 shadow-xl shadow-slate-200/50 dark:shadow-black/50">
              <h2 class="text-xl font-bold text-slate-900 dark:text-white mb-6 uppercase tracking-tight flex items-center gap-3">
                <span class="w-2 h-8 bg-red-500 block"></span>
                Control
              </h2>

              <button 
                @click="deleteContainer"
                :disabled="deleting"
                class="w-full relative group overflow-hidden bg-red-500 hover:bg-red-600 text-white p-4 font-bold uppercase tracking-widest text-sm transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-3"
              >
                  <Trash2 :size="16" />
                  <span>{{ deleting ? 'Terminating...' : 'Terminate Container' }}</span>
              </button>
              
              <p class="text-[10px] text-slate-500 mt-4 text-center leading-relaxed">
                 Warning: This action is irreversible. All associated volumes and data will be permanently destroyed.
              </p>
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

.dark .custom-scrollbar::-webkit-scrollbar-track {
  background: rgba(148, 163, 184, 0.15);
}

.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.3);
  border-radius: 4px;
}

.dark .custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(148, 163, 184, 0.45);
}

.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.5);
}

.dark .custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: rgba(148, 163, 184, 0.6);
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
