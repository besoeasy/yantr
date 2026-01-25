<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'vue-toastification'
import { ArrowLeft, ExternalLink, RefreshCw, Trash2, Network, FolderOpen, Terminal, Activity, Cpu, HardDrive, ShieldCheck, Share2, Globe, Database, Lock, Folder, Pause, Play, Download } from 'lucide-vue-next'

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
const autoScrollLogs = ref(true)

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
      if (autoScrollLogs.value) {
        scrollToBottom()
      }
    }
  } catch (error) {
    console.error('Failed to fetch container logs:', error)
  } finally {
    setTimeout(() => {
      refreshingLogs.value = false
    }, 300)
  }
}

const scrollToBottom = () => {
    // Implementation via DOM manipulation in template refs would be better in Vue, but simplistic here
    setTimeout(() => {
        const el = document.getElementById('terminal-logs')
        if (el) el.scrollTop = el.scrollHeight
    }, 100)
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
  browsingVolume.value[volumeName] = volumeName
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
  <div class="min-h-screen bg-slate-50 dark:bg-[#09090b] text-slate-900 dark:text-slate-200 font-sans">
    <!-- Header -->
    <header class="bg-white dark:bg-[#0c0c0e] border-b border-slate-200 dark:border-slate-800">
      <div class="max-w-7xl mx-auto px-4 h-14 flex items-center justify-between">
        <div class="flex items-center gap-4">
          <router-link to="/" class="inline-flex items-center justify-center w-8 h-8 rounded-md hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors text-slate-500">
            <ArrowLeft :size="18" />
          </router-link>
          
          <div class="h-4 w-px bg-slate-200 dark:bg-slate-800"></div>

          <div class="flex items-center gap-2 text-sm">
            <span class="text-slate-500">Containers</span>
            <span class="text-slate-300 dark:text-slate-700">/</span>
            <span class="font-medium text-slate-900 dark:text-white" v-if="selectedContainer">{{ selectedContainer.name }}</span>
            <span v-else class="w-24 h-4 bg-slate-200 dark:bg-slate-800 animate-pulse rounded"></span>
          </div>
        </div>

        <div v-if="selectedContainer" class="flex items-center gap-3">
          <div class="flex items-center gap-2 px-3 py-1 rounded-full border text-xs font-semibold uppercase tracking-wide"
            :class="selectedContainer.state === 'running' 
              ? 'bg-emerald-50 dark:bg-emerald-950/30 border-emerald-200 dark:border-emerald-800 text-emerald-700 dark:text-emerald-400' 
              : 'bg-slate-50 dark:bg-slate-900 border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400'">
            <div class="w-1.5 h-1.5 rounded-full" :class="selectedContainer.state === 'running' ? 'bg-emerald-500 animate-pulse' : 'bg-slate-400'"></div>
            <span>{{ selectedContainer.state }}</span>
          </div>
        </div>
      </div>
    </header>

    <div v-if="!selectedContainer" class="max-w-7xl mx-auto p-8 flex justify-center">
       <div class="animate-spin text-slate-300"><RefreshCw :size="32" /></div>
    </div>

    <main v-else class="max-w-7xl mx-auto px-4 py-8 grid grid-cols-1 lg:grid-cols-3 gap-8">
      
      <!-- LEFT COLUMN -->
      <div class="lg:col-span-2 space-y-8">
        
        <!-- Info Card -->
        <div class="bg-white dark:bg-[#0c0c0e] rounded-lg border border-slate-200 dark:border-slate-800 p-6 flex flex-col sm:flex-row gap-6">
           <div class="w-20 h-20 bg-slate-50 dark:bg-slate-900 rounded-lg flex items-center justify-center p-3 shrink-0 border border-slate-100 dark:border-slate-800">
              <img v-if="selectedContainer.app.logo" :src="selectedContainer.app.logo" class="w-full h-full object-contain" />
              <div v-else class="text-3xl">📦</div>
           </div>
           
           <div class="flex-1 space-y-2">
              <h1 class="text-2xl font-bold text-slate-900 dark:text-white">{{ selectedContainer.name }}</h1>
              <p class="text-slate-500 dark:text-slate-400 text-sm leading-relaxed max-w-2xl">
                 {{ selectedContainer.app.description || "Container usage description not available." }}
              </p>
              <div class="pt-2 flex flex-wrap gap-2">
                 <div class="inline-flex items-center gap-1.5 px-2 py-1 bg-slate-50 dark:bg-slate-900 text-xs font-mono text-slate-600 dark:text-slate-400 rounded border border-slate-200 dark:border-slate-800">
                    <Database :size="12" />
                    {{ selectedContainer.image }}
                 </div>
                 <div class="inline-flex items-center gap-1.5 px-2 py-1 bg-slate-50 dark:bg-slate-900 text-xs font-mono text-slate-600 dark:text-slate-400 rounded border border-slate-200 dark:border-slate-800">
                    <span class="text-slate-400 dark:text-slate-500">ID:</span> {{ selectedContainer.id.substring(0, 12) }}
                 </div>
              </div>
           </div>
        </div>

        <!-- Network Access (Moved from Right) -->
        <div v-if="allPortMappings.length > 0" class="space-y-4">
           <h3 class="text-sm font-semibold uppercase tracking-wider text-slate-500 px-1">Network Access</h3>
           <div class="bg-white dark:bg-[#0c0c0e] rounded-lg border border-slate-200 dark:border-slate-800 overflow-hidden">
             <div class="overflow-x-auto">
               <table class="w-full text-left text-sm">
                 <thead>
                   <tr class="bg-slate-50 dark:bg-slate-900/50 border-b border-slate-200 dark:border-slate-800 text-xs font-semibold uppercase text-slate-500">
                     <th class="px-4 py-3 font-medium">Protocol</th>
                     <th class="px-4 py-3 font-medium">Host Port</th>
                     <th class="px-4 py-3 font-medium">Container Port</th>
                     <th class="px-4 py-3 font-medium w-full">Description</th>
                     <th class="px-4 py-3 text-right">Action</th>
                   </tr>
                 </thead>
                 <tbody class="divide-y divide-slate-100 dark:divide-slate-800/50">
                   <tr v-for="(mapping, i) in allPortMappings" :key="i" class="hover:bg-slate-50 dark:hover:bg-slate-900/20">
                     <td class="px-4 py-3">
                       <div class="flex items-center gap-2">
                         <span class="font-mono text-xs font-bold uppercase text-slate-700 dark:text-slate-300">{{ mapping.protocol }}</span>
                         <span v-if="mapping.labeledProtocol" class="text-[10px] px-1.5 py-0.5 bg-slate-100 dark:bg-slate-800 text-slate-500 rounded border border-slate-200 dark:border-slate-700 uppercase">{{ mapping.labeledProtocol }}</span>
                       </div>
                     </td>
                     <td class="px-4 py-3 font-mono">
                       <span v-if="mapping.hostPort" class="text-slate-900 dark:text-white font-medium">{{ mapping.hostPort }}</span>
                       <span v-else class="text-slate-400 text-xs italic">Internal</span>
                     </td>
                     <td class="px-4 py-3 font-mono text-slate-600 dark:text-slate-400">
                       {{ mapping.containerPort }}
                     </td>
                     <td class="px-4 py-3 text-slate-600 dark:text-slate-400 text-xs">
                        {{ mapping.label || '-' }}
                     </td>
                     <td class="px-4 py-3 text-right">
                        <a v-if="mapping.hostPort && mapping.protocol === 'tcp'"
                           :href="appUrl(mapping.hostPort, mapping.labeledProtocol || 'http')"
                           target="_blank"
                           class="inline-flex items-center justify-center p-1.5 text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded transition-colors"
                           title="Open in Browser"
                        >
                           <ExternalLink :size="16" />
                        </a>
                     </td>
                   </tr>
                 </tbody>
               </table>
             </div>
           </div>
        </div>

        <!-- Volumes (Moved Up) -->
        <div v-if="containerVolumes.length > 0" class="space-y-4">
           <h3 class="text-sm font-semibold uppercase tracking-wider text-slate-500 px-1">Attached Storage</h3>
           <div class="grid gap-3">
              <div v-for="volume in containerVolumes" :key="volume.name" 
                   class="group bg-white dark:bg-[#0c0c0e] border border-slate-200 dark:border-slate-800 rounded-lg p-4 flex items-center justify-between hover:border-blue-500/50 transition-colors">
                 
                 <div class="flex items-center gap-4 min-w-0">
                    <div class="w-10 h-10 rounded bg-blue-50 dark:bg-blue-900/20 flex items-center justify-center text-blue-600 dark:text-blue-400 shrink-0">
                       <HardDrive :size="20" />
                    </div>
                    <div class="min-w-0">
                       <div class="font-medium text-slate-900 dark:text-white truncate text-sm" :title="volume.name">{{ volume.name }}</div>
                       <div class="text-xs text-slate-500 font-mono truncate">{{ volume.destination }}</div>
                    </div>
                 </div>

                 <div class="relative min-w-[120px] flex justify-end">
                    <div v-if="browsingVolume[volume.name]" class="text-xs text-blue-500 animate-pulse font-medium">Starting...</div>
                    <button 
                       v-else-if="!showVolumeMenu[volume.name]"
                       @click="showVolumeMenu[volume.name] = true"
                       class="opacity-100 md:opacity-0 md:group-hover:opacity-100 transition-opacity px-3 py-1.5 text-xs font-medium bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 rounded hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/30 dark:hover:text-blue-400"
                    >
                       Browse Files
                    </button>
                    
                    <div v-else class="flex items-center gap-1 animate-in fade-in zoom-in-95 duration-200">
                       <button @click="browseVolume(volume.name, 60)" class="px-2 py-1 text-[10px] font-bold uppercase bg-blue-600 text-white rounded hover:bg-blue-700" title="1 Hour Access">
                          1H
                       </button>
                       <button @click="browseVolume(volume.name, 0)" class="px-2 py-1 text-[10px] font-bold uppercase bg-slate-700 text-white rounded hover:bg-slate-600" title="Permanent Access">
                          Perm
                       </button>
                    </div>
                 </div>

              </div>
           </div>
        </div>

        <!-- Terminal Logs (Moved Down) -->
        <div class="flex flex-col rounded-lg border border-slate-200 dark:border-slate-800 overflow-hidden bg-[#1e1e1e] shadow-sm">
           <div class="flex items-center justify-between px-4 py-2 bg-[#252526] border-b border-[#333]">
              <div class="flex items-center gap-2 text-xs font-medium text-[#cccccc]">
                 <Terminal :size="14" />
                 <span class="uppercase tracking-wider">Output</span>
              </div>
              <div class="flex items-center gap-2">
                 <button @click="autoScrollLogs = !autoScrollLogs" class="p-1 rounded hover:bg-[#3c3c3c] text-[#cccccc]" :title="autoScrollLogs ? 'Pause Auto-Scroll' : 'Enable Auto-Scroll'">
                    <component :is="autoScrollLogs ? Pause : Play" :size="14" />
                 </button>
                 <button @click="fetchContainerLogs" class="p-1 rounded hover:bg-[#3c3c3c] text-[#cccccc]" :title="refreshingLogs ? 'Refreshing...' : 'Refresh'">
                    <RefreshCw :size="14" :class="{ 'animate-spin': refreshingLogs }" />
                 </button>
              </div>
           </div>
           
           <div 
             id="terminal-logs"
             class="h-96 overflow-y-auto p-4 font-mono text-xs leading-5 text-[#d4d4d4] scrollbar-thin scrollbar-thumb-[#424242] scrollbar-track-transparent"
           >
              <div v-if="containerLogs.length === 0" class="flex flex-col items-center justify-center h-full text-[#666]">
                 <div class="mb-2 opacity-50">No output logs found</div>
              </div>
              <div v-else class="space-y-0.5">
                 <div v-for="(log, i) in containerLogs" :key="i" class="break-all whitespace-pre-wrap hover:bg-[#2a2d2e] px-1 -mx-1 rounded-sm">
                    <span class="text-[#569cd6] opacity-50 select-none mr-2 w-6 inline-block text-right">{{ i + 1 }}</span>{{ log }}
                 </div>
              </div>
           </div>
        </div>

      </div>

      <!-- RIGHT COLUMN -->
      <div class="space-y-8">
         
         <!-- Stats Grid -->
         <div v-if="containerStats" class="grid grid-cols-2 gap-4">
            <div class="bg-white dark:bg-[#0c0c0e] p-4 rounded-lg border border-slate-200 dark:border-slate-800">
               <div class="flex items-center gap-2 text-xs uppercase tracking-wider text-slate-500 mb-2">
                  <Cpu :size="14" /> CPU
               </div>
               <div class="text-2xl font-mono font-semibold text-slate-900 dark:text-white">
                  {{ containerStats.cpu.percent }}%
               </div>
            </div>
            
            <div class="bg-white dark:bg-[#0c0c0e] p-4 rounded-lg border border-slate-200 dark:border-slate-800">
               <div class="flex items-center gap-2 text-xs uppercase tracking-wider text-slate-500 mb-2">
                  <Activity :size="14" /> RAM
               </div>
               <div class="text-2xl font-mono font-semibold text-slate-900 dark:text-white">
                  {{ containerStats.memory.percent }}%
               </div>
            </div>
            
            <div class="col-span-2 bg-white dark:bg-[#0c0c0e] p-4 rounded-lg border border-slate-200 dark:border-slate-800 flex justify-between items-center">
               <div>
                  <div class="text-xs uppercase tracking-wider text-slate-500 mb-1">Network I/O</div>
                  <div class="text-sm font-mono font-medium text-slate-900 dark:text-white">
                     <span class="text-emerald-500">↓ {{ formatBytes(containerStats.network.rx) }}</span>
                     <span class="text-slate-300 mx-2">|</span>
                     <span class="text-blue-500">↑ {{ formatBytes(containerStats.network.tx) }}</span>
                  </div>
               </div>
               <div>
                  <div class="text-xs uppercase tracking-wider text-slate-500 mb-1">Block I/O</div>
                  <div class="text-sm font-mono font-medium text-slate-900 dark:text-white text-right">
                     {{ formatBytes(containerStats.blockIO.read) }} / {{ formatBytes(containerStats.blockIO.write) }}
                  </div>
               </div>
            </div>
         </div>

         <!-- Actions -->
         <div class="bg-white dark:bg-[#0c0c0e] rounded-lg border border-slate-200 dark:border-slate-800 p-5 space-y-4 shadow-sm">
            <h3 class="text-xs font-bold uppercase tracking-wider text-slate-500">Control</h3>
            
            <button 
               @click="deleteContainer"
               :disabled="deleting"
               class="w-full flex items-center justify-center gap-2 px-4 py-2.5 bg-red-50 dark:bg-red-900/10 text-red-600 dark:text-red-400 border border-red-200 dark:border-red-900/30 rounded-md hover:bg-red-100 dark:hover:bg-red-900/20 hover:border-red-300 transition-all font-medium text-sm"
            >
               <Trash2 :size="16" />
               {{ deleting ? 'Terminating...' : 'Terminate Container' }}
            </button>
            
            <p class="text-[10px] text-slate-400 text-center px-4 leading-relaxed">
               Warning: This action is irreversible. All associated volumes and data will be permanently destroyed.
            </p>
         </div>

         <!-- Environment Vars (Simplified) -->
         <div v-if="selectedContainer.env && selectedContainer.env.length > 0" class="space-y-4">
             <h3 class="text-sm font-semibold uppercase tracking-wider text-slate-500 px-1">Environment</h3>
             <div class="bg-[#1e1e1e] rounded-lg border border-slate-800 p-4 max-h-60 overflow-y-auto custom-scrollbar">
                <div v-for="(envVar, i) in selectedContainer.env" :key="i" class="font-mono text-[10px] mb-2 last:mb-0 break-all">
                   <div class="text-[#569cd6] mb-0.5">{{ envVar.split('=')[0] }}</div>
                   <div class="text-[#ce9178] pl-2">{{ envVar.split('=').slice(1).join('=') }}</div>
                </div>
             </div>
         </div>

      </div>
      
    </main>
  </div>
</template>

<style scoped>
.scrollbar-thin::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}
.scrollbar-thin::-webkit-scrollbar-track {
  background: transparent;
}
.scrollbar-thin::-webkit-scrollbar-thumb {
  background: #424242;
  border-radius: 3px;
}
.scrollbar-thin::-webkit-scrollbar-thumb:hover {
  background: #4f4f4f;
}

.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 3px;
}
</style>
