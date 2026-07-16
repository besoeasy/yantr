<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useNotification } from '../composables/useNotification'
import { useApiUrl } from '../composables/useApiUrl'
import { expectApiSuccess, getApiErrorMessage, readJsonResponse } from '../composables/useApiResponse'
import { useYantrAuth } from '../composables/useYantrAuth'
import { ExternalLink, Trash2, Network, Terminal, HardDrive, ShieldCheck, Database, Copy, Check } from '@lucide/vue'
import AppLogo from '../components/AppLogo.vue'
import ContainerResources from '../components/ContainerResources.vue'
import ContainerLogs from '../components/ContainerLogs.vue'
import ContainerEnv from '../components/ContainerEnv.vue'
import { formatBytes } from '../utils/metrics'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const toast = useNotification()
const { apiUrl } = useApiUrl()
const { openVolumeBrowser } = useYantrAuth()

const selectedContainer = ref(null)
const containerStats = ref(null)
const containerLogs = ref([])
const deleting = ref(false)
const refreshingLogs = ref(false)
const browsingVolume = ref({})
const showVolumeMenu = ref({})
let statsInterval = null
const autoScrollLogs = ref(true)
const currentTime = ref(Date.now())
const activeTab = ref('resources')
const showOnlyDescribedPorts = ref(true)
const loadErrorState = {
  stats: false,
  logs: false,
}

const copiedCommand = ref(false)
const copyExecCommand = async () => {
  try {
    await navigator.clipboard.writeText(`docker exec -it ${selectedContainer.value.name} /bin/sh`)
    copiedCommand.value = true
    setTimeout(() => copiedCommand.value = false, 2000)
    toast.success('Command copied to clipboard')
  } catch (err) {
    toast.error('Failed to copy command')
  }
}

function notifyLoadErrorOnce(key, message) {
  if (loadErrorState[key]) return
  loadErrorState[key] = true
  toast.error(message)
}

function clearLoadError(key) {
  loadErrorState[key] = false
}

let timeUpdateInterval = null

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

const allPortMappings = computed(() => {
  if (!selectedContainer.value || !selectedContainer.value.ports) {
    return []
  }
  
  const portLabels = {}
  for (const p of (selectedContainer.value.app?.ports || [])) {
    if (p.port != null) {
      portLabels[String(p.port)] = {
        protocol: (p.protocol || '').toLowerCase(),
        label: p.label || null,
      }
    }
  }
  
  const mappings = []
  const portKeys = Object.keys(selectedContainer.value.ports)
  
  portKeys.forEach(key => {
    const [privatePort, type] = key.split('/')
    const bindings = selectedContainer.value.ports[key]
    
    if (bindings && bindings.length > 0) {
      const seenHostPorts = new Set()
      bindings.forEach(binding => {
        if (binding.HostPort && !seenHostPorts.has(binding.HostPort)) {
          seenHostPorts.add(binding.HostPort)
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
  
  return mappings.sort((a, b) => {
    if (a.hostPort && b.hostPort) {
      return parseInt(a.hostPort) - parseInt(b.hostPort)
    }
    if (a.hostPort && !b.hostPort) return -1
    if (!a.hostPort && b.hostPort) return 1
    return parseInt(a.containerPort) - parseInt(b.containerPort)
  })
})

const hasDescribedPorts = computed(() => allPortMappings.value.some(m => m.label))

const filteredPortMappings = computed(() => {
  if (!hasDescribedPorts.value) {
    return allPortMappings.value
  }
  if (!showOnlyDescribedPorts.value) {
    return allPortMappings.value
  }
  return allPortMappings.value.filter(mapping => mapping.label)
})

const expirationInfo = computed(() => {
  if (!selectedContainer.value?.expireAt) return null
  
  const expireAtTimestamp = parseInt(selectedContainer.value.expireAt, 10)
  if (isNaN(expireAtTimestamp)) return null
  
  const expireAtMs = expireAtTimestamp * 1000
  const timeLeftMs = expireAtMs - currentTime.value
  
  if (timeLeftMs <= 0) {
    return {
      expired: true,
      timeLeft: t('containerDetail.expired'),
      urgency: 'critical',
      percentage: 0
    }
  }
  
  const totalMinutes = Math.floor(timeLeftMs / 60000)
  const hours = Math.floor(totalMinutes / 60)
  const days = Math.floor(hours / 24)
  const minutes = totalMinutes % 60
  
  const oneDayMs = 86400000
  const percentage = Math.min(100, Math.max(0, (timeLeftMs / oneDayMs) * 100))
  
  let timeLeft = ''
  let urgency = 'normal'
  
  if (days > 0) {
    timeLeft = `${days} ${days === 1 ? t('containerDetail.day') : t('containerDetail.days')}${hours % 24 > 0 ? `, ${hours % 24} ${hours % 24 === 1 ? t('containerDetail.hour') : t('containerDetail.hours')}` : ''}`
    urgency = days < 1 ? 'warning' : 'normal'
  } else if (hours > 0) {
    timeLeft = `${hours} ${hours === 1 ? t('containerDetail.hour') : t('containerDetail.hours')}${minutes > 0 ? `, ${minutes} ${minutes === 1 ? t('containerDetail.minute') : t('containerDetail.minutes')}` : ''}`
    urgency = hours < 2 ? 'critical' : 'warning'
  } else if (totalMinutes > 0) {
    timeLeft = `${totalMinutes} ${totalMinutes === 1 ? t('containerDetail.minute') : t('containerDetail.minutes')}`
    urgency = 'critical'
  } else {
    timeLeft = t('containerDetail.lessThanMinute')
    urgency = 'critical'
  }
  
  return {
    expired: false,
    timeLeft,
    urgency,
    percentage,
    expireAt: new Date(expireAtMs).toLocaleString(),
    totalMinutes
  }
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

async function fetchContainerDetail() {
  try {
    const response = await fetch(`${apiUrl.value}/api/containers/${route.params.id}`)
    const data = await expectApiSuccess(response, t('containerDetail.error.containerNotFound'))
    selectedContainer.value = data.container
  } catch (error) {
    toast.error(error.message || t('containerDetail.error.failedToLoadDetails'))
    router.push('/')
  }
}

async function fetchContainerStats() {
  if (!selectedContainer.value) return
  
  try {
    const response = await fetch(`${apiUrl.value}/api/containers/${selectedContainer.value.id}/stats`)
    const data = await expectApiSuccess(response, 'Failed to load container stats')
    containerStats.value = data.stats
    clearLoadError('stats')
  } catch (error) {
    notifyLoadErrorOnce('stats', error.message || 'Failed to load container stats')
  }
}

async function fetchContainerLogs() {
  if (!selectedContainer.value) return
  
  refreshingLogs.value = true
  try {
    const response = await fetch(`${apiUrl.value}/api/containers/${selectedContainer.value.id}/logs?tail=200`)
    const data = await expectApiSuccess(response, 'Failed to load container logs')
    containerLogs.value = Array.isArray(data.logs) ? data.logs : []
    clearLoadError('logs')
    if (autoScrollLogs.value) {
      scrollToBottom()
    }
  } catch (error) {
    notifyLoadErrorOnce('logs', error.message || 'Failed to load container logs')
  } finally {
    setTimeout(() => {
      refreshingLogs.value = false
    }, 300)
  }
}

const scrollToBottom = () => {
    setTimeout(() => {
        const el = document.getElementById('terminal-logs')
        if (el) el.scrollTop = el.scrollHeight
    }, 100)
}

async function deleteContainer() {
  if (!confirm(`${t('containerDetail.terminateContainer')} ${selectedContainer.value.name}?\n\n${t('containerDetail.warning')}`)) return

  deleting.value = true
  try {
    const response = await fetch(`${apiUrl.value}/api/containers/${selectedContainer.value.id}`, {
      method: 'DELETE'
    })
    const data = await readJsonResponse(response)

    if (response.ok && data.success) {
      let message = t('containerDetail.success.deletedSuccessfully', { name: selectedContainer.value.name })
      const removedVolumes = Array.isArray(data.volumesRemoved) ? data.volumesRemoved : []
      if (removedVolumes.length > 0) {
        message += `\n\n${t('containerDetail.success.volumesRemoved', { volumes: data.volumesRemoved.join(', ') })}`
      }
      toast.success(message)
      router.push('/home')
    } else {
      throw new Error(getApiErrorMessage(data, t('containerDetail.error.deletionFailed', { error: t('common.error') })))
    }
  } catch (error) {
    toast.error(t('containerDetail.error.deletionFailed', { error: error.message }))
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
    await expectApiSuccess(response, t('containerDetail.error.failedToStartVolumeBrowser'))
    const expiryText = expiryMinutes > 0 ? ` (${expiryMinutes}m)` : ' (no expiry)'
    toast.success(t('containerDetail.success.volumeBrowserStarted', { expiry: expiryText }))
    openVolumeBrowser(volumeName)
  } catch (error) {
    toast.error(error.message || t('containerDetail.error.failedToStartVolumeBrowser'))
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
  
  statsInterval = setInterval(() => {
    fetchContainerStats()
  }, 2000)
  
  timeUpdateInterval = setInterval(() => {
    currentTime.value = Date.now()
  }, 1000)
})

onUnmounted(() => {
  if (statsInterval) {
    clearInterval(statsInterval)
  }
  if (timeUpdateInterval) {
    clearInterval(timeUpdateInterval)
  }
})
</script>

<template>
  <div class="min-h-screen bg-white dark:bg-[#0A0A0A] text-gray-900 dark:text-zinc-100 font-sans selection:bg-blue-500/30">
    <div v-if="!selectedContainer" class="max-w-7xl mx-auto p-8 flex justify-center py-32">
       <div class="w-8 h-8 border-[3px] border-gray-200 dark:border-zinc-800 border-t-blue-500 dark:border-t-blue-500 rounded-full animate-spin"></div>
    </div>

    <main v-else class="max-w-7xl mx-auto px-6 py-8 space-y-6">
        
        <div class="group relative bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 p-6 flex flex-col sm:flex-row gap-6 hover:border-gray-300 dark:hover:border-zinc-700 transition-all duration-300">
           <div class="absolute top-0 left-0 w-full h-0.5 bg-linear-to-r from-transparent via-blue-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>

           <div class="w-20 h-20 bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 rounded-xl flex items-center justify-center p-4 shrink-0 shadow-sm transition-transform group-hover:scale-105 duration-500">
          <AppLogo
           :logo="selectedContainer.app.logo"
           :name="selectedContainer.app?.name || selectedContainer.name"
           :seed="selectedContainer.app?.id || selectedContainer.name"
           img-class="w-full h-full object-contain filter dark:brightness-90 group-hover:brightness-100 transition-all"
           icon-class="w-full h-full text-gray-400 dark:text-zinc-600"
          />
           </div>
           
           <div class="flex-1 space-y-3">
              <h1 class="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">{{ selectedContainer.name }}</h1>
              <p class="text-gray-500 dark:text-zinc-400 text-sm leading-relaxed max-w-2xl">
                 {{ selectedContainer.app.description || t('containerDetail.descriptionNotAvailable') }}
              </p>
              <div class="pt-2 flex flex-wrap gap-2">
                 <div class="inline-flex items-center gap-1.5 px-2.5 py-1 border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900/50 text-[10px] font-bold tracking-widest text-gray-600 dark:text-zinc-400 rounded-md uppercase">
                    <Database :size="10" />
                    {{ selectedContainer.image }}
                 </div>
                 <div class="inline-flex items-center gap-1.5 px-2.5 py-1 border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900/50 text-[10px] font-bold tracking-widest text-gray-600 dark:text-zinc-400 rounded-md uppercase">
                    <span class="opacity-60">{{ t('containerDetail.id') }}:</span> {{ selectedContainer.id.substring(0, 12) }}
                 </div>
              </div>
           </div>
        </div>

        <div v-if="allPortMappings.length > 0" class="space-y-4">
           <div class="flex items-center justify-between">
             <h3 class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t('containerDetail.networkAccess') }}</h3>
             <div v-if="hasDescribedPorts" class="flex items-center gap-1 rounded-lg bg-gray-100 dark:bg-zinc-900 p-1">
               <button
                 @click="showOnlyDescribedPorts = false"
                 :class="!showOnlyDescribedPorts ? 'bg-white dark:bg-zinc-800 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 hover:text-gray-700 dark:hover:text-zinc-300'"
                 class="px-3 py-1 text-[10px] font-bold uppercase tracking-wider rounded-md transition-all"
               >
                 {{ t('containerDetail.allPorts') }}
               </button>
               <button
                 @click="showOnlyDescribedPorts = true"
                 :class="showOnlyDescribedPorts ? 'bg-white dark:bg-zinc-800 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 hover:text-gray-700 dark:hover:text-zinc-300'"
                 class="px-3 py-1 text-[10px] font-bold uppercase tracking-wider rounded-md transition-all"
               >
                 {{ t('containerDetail.described') }}
               </button>
             </div>
           </div>
           <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
             <div v-for="(mapping, i) in filteredPortMappings" :key="i" 
                  class="group bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl p-5 hover:border-gray-300 dark:hover:border-zinc-600 transition-all duration-300">
               
               <div class="flex items-start justify-between mb-4">
                 <div class="flex items-start gap-3.5 flex-1 min-w-0">
                   <div class="w-10 h-10 rounded-lg bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center text-gray-600 dark:text-zinc-400 shrink-0 shadow-sm group-hover:text-blue-500 transition-colors">
                     <Network :size="18" />
                   </div>
                   <div class="min-w-0 flex-1">
                     <div class="flex items-center gap-2 mb-1.5">
                       <span class="font-mono text-[10px] font-bold uppercase text-gray-900 dark:text-white">{{ mapping.protocol }}</span>
                       <span v-if="mapping.labeledProtocol" class="text-[9px] px-1.5 py-0.5 bg-gray-100 dark:bg-zinc-800 text-gray-500 dark:text-zinc-400 rounded-md uppercase font-bold tracking-widest border border-gray-200 dark:border-zinc-700">{{ mapping.labeledProtocol }}</span>
                     </div>
                     <div class="text-[11px] text-gray-500 dark:text-zinc-400 truncate" :title="mapping.label">
                       {{ mapping.label || t('containerDetail.networkPort') }}
                     </div>
                   </div>
                 </div>
               </div>

               <div class="space-y-2 mb-5">
                 <div class="flex items-center justify-between text-[11px]">
                   <span class="text-gray-500 dark:text-zinc-500 uppercase font-bold tracking-wider">{{ t('containerDetail.hostPort') }}</span>
                   <span v-if="mapping.hostPort" class="font-mono font-bold text-gray-900 dark:text-white">{{ mapping.hostPort }}</span>
                   <span v-else class="text-gray-400 italic">{{ t('containerDetail.internal') }}</span>
                 </div>
                 <div class="flex items-center justify-between text-[11px]">
                   <span class="text-gray-500 dark:text-zinc-500 uppercase font-bold tracking-wider">{{ t('containerDetail.containerPort') }}</span>
                   <span class="font-mono font-medium text-gray-700 dark:text-zinc-300">{{ mapping.containerPort }}</span>
                 </div>
               </div>

               <a v-if="mapping.hostPort && mapping.protocol === 'tcp'"
                  :href="appUrl(mapping.hostPort, mapping.labeledProtocol || 'http')"
                  target="_blank"
                  class="w-full flex items-center justify-center gap-2 px-3 py-2 bg-black dark:bg-white text-white dark:text-black rounded-lg hover:bg-gray-800 dark:hover:bg-gray-200 transition-all text-[11px] font-bold uppercase tracking-wider"
               >
                  <ExternalLink :size="12" />
                  {{ t('containerDetail.open') }}
               </a>
               <div v-else class="w-full flex items-center justify-center px-3 py-2 bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 text-gray-400 dark:text-zinc-500 rounded-lg text-[11px] font-bold uppercase tracking-wider">
                 {{ t('containerDetail.internalOnly') }}
               </div>
             </div>
           </div>
        </div>

        <div v-if="containerVolumes.length > 0" class="space-y-4">
           <div class="flex items-center justify-between">
             <h3 class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t('containerDetail.storageVolumes') }}</h3>
           </div>

           <div class="grid gap-4">
               <div v-for="volume in containerVolumes" :key="volume.name" 
                  class="group bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl p-5 hover:border-gray-300 dark:hover:border-zinc-600 transition-all duration-300">
                   
                 <div class="flex items-start justify-between gap-4 mb-5">
                  <div class="flex items-start gap-4 min-w-0 flex-1">
                    <div class="w-10 h-10 rounded-lg bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center text-gray-500 dark:text-zinc-400 shrink-0 shadow-sm group-hover:text-blue-500 transition-colors">
                      <HardDrive :size="18" />
                    </div>
                    <div class="min-w-0 flex-1">
                      <div class="font-bold text-sm text-gray-900 dark:text-white truncate tracking-tight" :title="volume.name">{{ volume.name }}</div>
                      <div class="text-[11px] text-gray-500 dark:text-zinc-400 font-mono truncate mt-1">{{ volume.destination }}</div>
                    </div>
                  </div>
                </div>

                <div class="flex items-center gap-2 flex-wrap pt-4 border-t border-gray-100 dark:border-zinc-800">
                  <div v-if="browsingVolume[volume.name]" class="text-[10px] font-bold uppercase tracking-wider text-blue-600 dark:text-blue-400 animate-pulse px-3 py-2 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800/50 rounded-lg">
                    {{ t('containerDetail.startingWebDAV') }}
                  </div>
                  
                  <button 
                    v-else-if="!showVolumeMenu[volume.name]"
                    @click="showVolumeMenu[volume.name] = true"
                    class="px-3.5 py-2 text-[10px] font-bold uppercase tracking-wider border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900 text-gray-700 dark:text-zinc-300 rounded-lg hover:bg-gray-100 dark:hover:bg-zinc-800 transition-all"
                  >
                    {{ t('containerDetail.browseFiles') }}
                  </button>
                       
                  <div v-else class="flex items-center gap-1.5 animate-in fade-in zoom-in-95 duration-200">
                    <button @click="browseVolume(volume.name, 60)" class="px-3 py-2 text-[10px] font-bold uppercase tracking-wider bg-black dark:bg-white text-white dark:text-black rounded-lg hover:bg-gray-800 dark:hover:bg-gray-200 transition-all" :title="t('containerDetail.oneHourAccess')">
                      1H
                    </button>
                    <button @click="browseVolume(volume.name, 0)" class="px-3 py-2 text-[10px] font-bold uppercase tracking-wider bg-gray-200 dark:bg-zinc-800 text-gray-800 dark:text-zinc-200 rounded-lg hover:bg-gray-300 dark:hover:bg-zinc-700 transition-all" :title="t('containerDetail.permanentAccess')">
                      Perm
                    </button>
                  </div>
                </div>

              </div>
             </div>
        </div>

        <div class="bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 overflow-hidden shadow-sm">
          <div class="flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-zinc-800">
            <div class="flex items-center gap-2 text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">
              <ShieldCheck :size="14" />
              {{ t('containerDetail.systemDiagnostics') }}
            </div>
            <div class="flex items-center gap-1 rounded-lg bg-gray-100 dark:bg-zinc-900 p-1">
              <button
                @click="activeTab = 'resources'"
                :class="activeTab === 'resources' ? 'bg-white dark:bg-zinc-800 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 hover:text-gray-700 dark:hover:text-zinc-300'"
                class="px-3 py-1 text-[10px] font-bold uppercase tracking-wider rounded-md transition-all"
              >
                {{ t('containerDetail.resources') }}
              </button>
              <button
                @click="activeTab = 'output'"
                :class="activeTab === 'output' ? 'bg-white dark:bg-zinc-800 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 hover:text-gray-700 dark:hover:text-zinc-300'"
                class="px-3 py-1 text-[10px] font-bold uppercase tracking-wider rounded-md transition-all"
              >
                {{ t('containerDetail.output') }}
              </button>
              <button
                @click="activeTab = 'terminal'"
                :class="activeTab === 'terminal' ? 'bg-white dark:bg-zinc-800 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 hover:text-gray-700 dark:hover:text-zinc-300'"
                class="px-3 py-1 text-[10px] font-bold uppercase tracking-wider rounded-md transition-all"
              >
                Terminal
              </button>
              <button
                @click="activeTab = 'env'"
                :class="activeTab === 'env' ? 'bg-white dark:bg-zinc-800 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 hover:text-gray-700 dark:hover:text-zinc-300'"
                class="px-3 py-1 text-[10px] font-bold uppercase tracking-wider rounded-md transition-all"
              >
                {{ t('containerDetail.env') }}
              </button>
            </div>
          </div>

          <div class="p-6">
            <div v-if="activeTab === 'resources'">
              <ContainerResources :container-stats="containerStats" />
            </div>

            <div v-else-if="activeTab === 'output'" class="space-y-4">
              <ContainerLogs 
                :logs="containerLogs" 
                v-model:autoScroll="autoScrollLogs" 
                :refreshing="refreshingLogs" 
                @refresh="fetchContainerLogs" 
              />
            </div>

            <div v-else-if="activeTab === 'terminal'" class="space-y-4">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2 text-[10px] font-bold text-gray-500 dark:text-zinc-500 uppercase tracking-widest">
                  <Terminal :size="12" /> Container Shell Access
                </div>
              </div>
              <div class="bg-gray-50 dark:bg-[#111] border border-gray-200 dark:border-zinc-800 rounded-xl p-5">
                <p class="text-[11px] text-gray-500 dark:text-zinc-400 mb-4 font-medium leading-relaxed">
                  To access the interactive shell for this container, open your host terminal and run the following command:
                </p>
                <div class="flex items-center gap-3 bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-700 rounded-lg p-3">
                  <div class="flex-1 font-mono text-[11px] text-gray-900 dark:text-zinc-300 select-all overflow-x-auto whitespace-nowrap scrollbar-thin">
                    docker exec -it {{ selectedContainer.name }} /bin/sh
                  </div>
                  <button
                    @click="copyExecCommand"
                    class="p-2 rounded-md transition-colors shrink-0"
                    :class="copiedCommand ? 'text-green-600 bg-green-50 dark:bg-green-500/10' : 'text-gray-500 hover:text-gray-900 dark:text-zinc-400 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-zinc-800'"
                    :title="copiedCommand ? 'Copied!' : 'Copy command'"
                  >
                    <component :is="copiedCommand ? Check : Copy" :size="14" />
                  </button>
                </div>
              </div>
            </div>

            <div v-else class="space-y-4">
              <ContainerEnv :env="selectedContainer.env" />
            </div>
          </div>
        </div>

        <div class="bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 p-6 space-y-4 shadow-sm">
           <h3 class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t('containerDetail.control') }}</h3>
           
           <button 
              @click="deleteContainer"
              :disabled="deleting"
              class="w-full flex items-center justify-center gap-2 px-4 py-3 bg-white dark:bg-[#0A0A0A] border border-red-200 dark:border-red-500/20 text-red-600 dark:text-red-500 rounded-xl hover:bg-red-50 dark:hover:bg-red-500/10 transition-all font-bold text-[11px] uppercase tracking-wider"
           >
              <Trash2 :size="14" />
              {{ deleting ? t('containerDetail.terminating') : t('containerDetail.terminateContainer') }}
           </button>
           
           <p class="text-[10px] text-gray-500 dark:text-zinc-500 text-center px-4 leading-relaxed font-medium">
              {{ t('containerDetail.warning') }}
           </p>
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
