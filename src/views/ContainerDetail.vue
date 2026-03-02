<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useNotification } from '../composables/useNotification'
import { useApiUrl } from '../composables/useApiUrl'
import { ArrowLeft, ExternalLink, RefreshCw, Trash2, Network, FolderOpen, Terminal, Activity, Cpu, HardDrive, ShieldCheck, Share2, Globe, Database, Lock, Folder, Pause, Play, Download, Clock, AlertCircle, Package } from 'lucide-vue-next'
import { formatBytes } from '../utils/metrics'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const toast = useNotification()
const { apiUrl } = useApiUrl()

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

const s3Configured = ref(false)
const volumeBackups = ref({})
const backingUp = ref(false)
const showRestoreMenu = ref({})
const backupJobId = ref(null)

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
    const data = await response.json()
    
    if (data.success) {
      selectedContainer.value = data.container
    } else {
      toast.error(t('containerDetail.error.containerNotFound'))
      router.push('/')
    }
  } catch (error) {
    console.error('Failed to fetch container details:', error)
    toast.error(t('containerDetail.error.failedToLoadDetails'))
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
    const data = await response.json()

    if (data.success) {
      let message = t('containerDetail.success.deletedSuccessfully', { name: selectedContainer.value.name })
      if (data.volumesRemoved.length > 0) {
        message += `\n\n${t('containerDetail.success.volumesRemoved', { volumes: data.volumesRemoved.join(', ') })}`
      }
      toast.success(message)
      router.push('/home')
    } else {
      toast.error(t('containerDetail.error.deletionFailed', { error: data.error }))
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
    const data = await response.json()
    if (data.success) {
      const expiryText = expiryMinutes > 0 ? ` (${expiryMinutes}m)` : ' (no expiry)'
      toast.success(t('containerDetail.success.volumeBrowserStarted', { expiry: expiryText }))
      window.open(`/browse/${volumeName}/`, '_blank')
    }
  } catch (error) {
    toast.error(t('containerDetail.error.failedToStartVolumeBrowser'))
    console.error(error)
  } finally {
    delete browsingVolume.value[volumeName]
  }
}

async function checkS3Config() {
  try {
    const response = await fetch(`${apiUrl.value}/api/backup/config`)
    const data = await response.json()
    s3Configured.value = data.configured
  } catch (error) {
    console.error('Failed to check S3 config:', error)
  }
}

async function fetchVolumeBackups() {
  if (!selectedContainer.value) return

  try {
    const response = await fetch(
      `${apiUrl.value}/api/containers/${selectedContainer.value.id}/backups`
    )
    const data = await response.json()

    if (data.success) {
      volumeBackups.value = data.backups
      s3Configured.value = data.configured !== false
    }
  } catch (error) {
    console.error('Failed to fetch backups:', error)
  }
}

async function backupAllVolumes() {
  if (!selectedContainer.value) return

  backingUp.value = true
  try {
    const response = await fetch(
      `${apiUrl.value}/api/containers/${selectedContainer.value.id}/backup`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
      }
    )
    const data = await response.json()

    if (data.success) {
      toast.success(t('containerDetail.success.backupStarted'))
      backupJobId.value = data.jobId
      pollBackupJob(data.jobId)
    } else {
      toast.error(data.error || t('containerDetail.error.failedToStartBackup'))
    }
  } catch (error) {
    toast.error(t('containerDetail.error.failedToStartBackup'))
  } finally {
    backingUp.value = false
  }
}

async function pollBackupJob(jobId) {
  const pollInterval = setInterval(async () => {
    try {
      const response = await fetch(`${apiUrl.value}/api/backup/jobs/${jobId}`)
      const data = await response.json()

      if (data.success && data.job) {
        if (data.job.status === 'completed') {
          clearInterval(pollInterval)
          toast.success(t('containerDetail.success.backupCompleted'))
          await fetchVolumeBackups()
        } else if (data.job.status === 'failed') {
          clearInterval(pollInterval)
          toast.error(t('containerDetail.error.backupFailed', { error: data.job.error }))
        }
      }
    } catch (error) {
      clearInterval(pollInterval)
      console.error('Failed to poll backup job:', error)
    }
  }, 2000)
}

async function restoreBackup(volumeName, snapshotId) {
  if (!confirm(`${t('backupVolumes.confirmRestore', { volume: volumeName })}`)) return

  try {
    const response = await fetch(
      `${apiUrl.value}/api/volumes/${volumeName}/restore`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ snapshotId, overwrite: true })
      }
    )
    const data = await response.json()

    if (data.success) {
      toast.success(t('containerDetail.success.restoreStarted'))
      pollRestoreJob(data.jobId)
    } else {
      toast.error(data.error || t('containerDetail.error.failedToStartRestore'))
    }
  } catch (error) {
    toast.error(t('containerDetail.error.failedToStartRestore'))
  }

  showRestoreMenu.value[volumeName] = false
}

async function pollRestoreJob(jobId) {
  const pollInterval = setInterval(async () => {
    try {
      const response = await fetch(`${apiUrl.value}/api/restore/jobs/${jobId}`)
      const data = await response.json()

      if (data.success && data.job) {
        if (data.job.status === 'completed') {
          clearInterval(pollInterval)
          toast.success(t('containerDetail.success.restoreCompleted'))
        } else if (data.job.status === 'failed') {
          clearInterval(pollInterval)
          toast.error(t('containerDetail.error.restoreFailed', { error: data.job.error }))
        }
      }
    } catch (error) {
      clearInterval(pollInterval)
      console.error('Failed to poll restore job:', error)
    }
  }, 2000)
}

async function deleteBackupFile(volumeName, snapshotId) {
  if (!confirm(t('backupVolumes.confirmDelete'))) return

  try {
    const response = await fetch(
      `${apiUrl.value}/api/volumes/${volumeName}/backup/${snapshotId}`,
      { method: 'DELETE' }
    )
    const data = await response.json()

    if (data.success) {
      toast.success(t('containerDetail.success.backupDeleted'))
      await fetchVolumeBackups()
    } else {
      toast.error(data.error || t('containerDetail.error.failedToDeleteBackup'))
    }
  } catch (error) {
    toast.error(t('containerDetail.error.failedToDeleteBackup'))
  }
}

function toggleRestoreMenu(volumeName) {
  showRestoreMenu.value[volumeName] = !showRestoreMenu.value[volumeName]
}

function hasBackups(volumeName) {
  return volumeBackups.value[volumeName]?.length > 0
}

function getLatestBackupAge(volumeName) {
  const backups = volumeBackups.value[volumeName]
  if (!backups || backups.length === 0) return t('containerDetail.never')

  const latest = backups[0]
  const date = new Date(latest.timestamp)
  const now = new Date()
  const diffMs = now - date
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMins / 60)
  const diffDays = Math.floor(diffHours / 24)

  if (diffDays > 0) return `${diffDays}d ago`
  if (diffHours > 0) return `${diffHours}h ago`
  if (diffMins > 0) return `${diffMins}m ago`
  return t('containerDetail.justNow')
}

function formatBackupDate(timestamp) {
  return new Date(timestamp).toLocaleString()
}

onMounted(async () => {
  await fetchContainerDetail()
  await Promise.all([
    fetchContainerStats(),
    fetchContainerLogs(),
    checkS3Config(),
    fetchVolumeBackups()
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
     <header class="bg-white/80 dark:bg-[#0A0A0A]/80 backdrop-blur-md border-b border-gray-200 dark:border-zinc-800 sticky top-0 z-30">
       <div class="max-w-7xl mx-auto px-4 sm:px-6 h-14 sm:h-16 flex items-center justify-between">
         <div class="flex items-center gap-2 sm:gap-4 min-w-0">
           <router-link to="/" class="inline-flex items-center justify-center w-10 h-10 rounded-lg hover:bg-gray-100 dark:hover:bg-zinc-900 transition-all text-gray-500 dark:text-zinc-400 group shrink-0">
             <ArrowLeft :size="16" class="group-hover:-translate-x-0.5 transition-transform" />
           </router-link>
           
           <div class="h-4 w-px bg-gray-300 dark:bg-zinc-800 shrink-0"></div>

           <div class="flex items-center gap-1.5 sm:gap-2.5 text-sm min-w-0">
             <span class="hidden sm:inline text-[10px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500">{{ t('containerDetail.containers') }}</span>
             <span class="hidden sm:inline text-gray-300 dark:text-zinc-700">/</span>
             <span class="font-semibold tracking-tight text-gray-900 dark:text-white truncate" v-if="selectedContainer">{{ selectedContainer.name }}</span>
             <span v-else class="w-32 h-4 bg-gray-200 dark:bg-zinc-800 animate-pulse rounded"></span>
           </div>
         </div>

        <div v-if="selectedContainer" class="flex items-center gap-2">
          <div v-if="expirationInfo" 
               class="flex items-center gap-1.5 px-2.5 py-1 rounded-md border text-[10px] font-bold uppercase tracking-wider"
               :class="{
                 'bg-red-50 dark:bg-red-500/10 border-red-200 dark:border-red-500/20 text-red-600 dark:text-red-400': expirationInfo.urgency === 'critical',
                 'bg-orange-50 dark:bg-orange-500/10 border-orange-200 dark:border-orange-500/20 text-orange-600 dark:text-orange-400': expirationInfo.urgency === 'warning',
                 'bg-blue-50 dark:bg-blue-500/10 border-blue-200 dark:border-blue-500/20 text-blue-600 dark:text-blue-400': expirationInfo.urgency === 'normal'
               }"
               :title="`${t('containerDetail.expiresAt')}: ${expirationInfo.expireAt}`">
            <Clock :size="12" :class="expirationInfo.urgency === 'critical' ? 'animate-pulse' : ''" />
            <span>{{ expirationInfo.timeLeft }}</span>
          </div>
          
          <div class="flex items-center gap-1.5 px-2.5 py-1 rounded-md border text-[10px] font-bold uppercase tracking-wider"
            :class="selectedContainer.state === 'running' 
              ? 'bg-green-50 dark:bg-green-500/10 border-green-200 dark:border-green-500/20 text-green-600 dark:text-green-400' 
              : 'bg-gray-50 dark:bg-zinc-900 border-gray-200 dark:border-zinc-800 text-gray-600 dark:text-zinc-400'">
            <div class="w-1.5 h-1.5 rounded-full" :class="selectedContainer.state === 'running' ? 'bg-green-500 animate-pulse' : 'bg-gray-400'"></div>
            <span>{{ selectedContainer.state }}</span>
          </div>
        </div>
      </div>
    </header>

    <div v-if="!selectedContainer" class="max-w-7xl mx-auto p-8 flex justify-center py-32">
       <div class="w-8 h-8 border-[3px] border-gray-200 dark:border-zinc-800 border-t-blue-500 dark:border-t-blue-500 rounded-full animate-spin"></div>
    </div>

    <main v-else class="max-w-7xl mx-auto px-6 py-8 space-y-6">
        
        <div class="group relative bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 p-6 flex flex-col sm:flex-row gap-6 hover:border-gray-300 dark:hover:border-zinc-700 transition-all duration-300">
           <div class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-blue-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>

           <div class="w-20 h-20 bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 rounded-xl flex items-center justify-center p-4 shrink-0 shadow-sm transition-transform group-hover:scale-105 duration-500">
              <img v-if="selectedContainer.app.logo" :src="selectedContainer.app.logo" loading="lazy" class="w-full h-full object-contain filter dark:brightness-90 group-hover:brightness-100 transition-all" />
              <Package v-else :size="32" class="text-gray-400 dark:text-zinc-600" />
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
             <button
               v-if="s3Configured"
               @click="backupAllVolumes"
               :disabled="backingUp"
               class="text-[10px] uppercase tracking-wider px-3 py-1.5 bg-black dark:bg-white text-white dark:text-black rounded-lg hover:bg-gray-800 dark:hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed transition-all font-bold"
             >
               {{ backingUp ? t('containerDetail.backingUp') : t('containerDetail.backupAll') }}
             </button>
           </div>

           <div v-if="!s3Configured" class="bg-amber-50 dark:bg-amber-500/10 border border-amber-200 dark:border-amber-500/20 rounded-xl p-4 flex items-start gap-3">
             <div class="text-amber-600 dark:text-amber-500 shrink-0 mt-0.5">
               <AlertCircle :size="14" />
             </div>
             <p class="text-xs text-amber-900 dark:text-amber-200">
               <span class="font-bold">{{ t('containerDetail.s3NotConfigured') }}</span>
               <router-link to="/backup-config" class="underline hover:text-amber-700 font-semibold ml-1">{{ t('containerDetail.configureNow') }}</router-link> {{ t('containerDetail.toEnableBackups') }}
             </p>
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
                      <div class="text-[10px] text-gray-400 dark:text-zinc-500 mt-2 font-bold uppercase tracking-wider">
                        {{ t('containerDetail.latestBackup') }}: <span class="text-gray-600 dark:text-zinc-300">{{ getLatestBackupAge(volume.name) }}</span>
                      </div>
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

                  <button
                    @click="backupAllVolumes"
                    :disabled="backingUp || !s3Configured"
                    class="px-3.5 py-2 text-[10px] font-bold uppercase tracking-wider bg-black dark:bg-white text-white dark:text-black rounded-lg hover:bg-gray-800 dark:hover:bg-gray-200 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
                    :title="t('containerDetail.backup')"
                  >
                    {{ t('containerDetail.backup') }}
                  </button>
                  <button
                    @click="toggleRestoreMenu(volume.name)"
                    :disabled="!hasBackups(volume.name) || !s3Configured"
                    class="px-3.5 py-2 text-[10px] font-bold uppercase tracking-wider border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900 text-gray-700 dark:text-zinc-300 rounded-lg hover:bg-gray-100 dark:hover:bg-zinc-800 disabled:opacity-30 disabled:cursor-not-allowed transition-all"
                  >
                    {{ t('containerDetail.restore') }}
                  </button>
                </div>

                <div
                  v-if="showRestoreMenu[volume.name] && hasBackups(volume.name)"
                  class="mt-4 pt-4 border-t border-gray-200 dark:border-zinc-800"
                >
                  <div class="text-[10px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500 mb-3">{{ t('containerDetail.availableBackups') }}</div>
                  <div class="space-y-2 max-h-40 overflow-y-auto scrollbar-thin">
                    <div
                      v-for="backup in volumeBackups[volume.name]"
                      :key="backup.snapshotId"
                      class="flex items-center justify-between py-2.5 px-3 bg-gray-50 dark:bg-zinc-900/50 hover:bg-gray-100 dark:hover:bg-zinc-900 rounded-lg border border-gray-200 dark:border-zinc-800 transition-all"
                    >
                      <div class="flex-1 min-w-0">
                        <div class="font-mono text-[11px] font-medium text-gray-900 dark:text-white">{{ formatBackupDate(backup.timestamp) }}</div>
                        <div class="text-gray-500 dark:text-zinc-400 text-[10px] mt-0.5 font-bold uppercase tracking-wider">
                           <span v-if="backup.sizeMB != null">{{ backup.sizeMB }} MB</span>
                           <span class="font-mono ml-2 text-gray-400 dark:text-zinc-500">{{ backup.snapshotId }}</span>
                         </div>
                      </div>
                      <div class="flex gap-2 ml-3">
                        <button
                          @click="restoreBackup(volume.name, backup.snapshotId)"
                          class="px-2.5 py-1.5 border border-gray-300 dark:border-zinc-700 text-gray-700 dark:text-zinc-300 bg-white dark:bg-[#0A0A0A] rounded-md hover:bg-gray-50 dark:hover:bg-zinc-800 transition-all text-[10px] font-bold uppercase tracking-wider"
                        >
                          {{ t('containerDetail.restore') }}
                        </button>
                        <button
                          @click="deleteBackupFile(volume.name, backup.snapshotId)"
                          class="px-2.5 py-1.5 border border-red-200 dark:border-red-500/20 text-red-600 dark:text-red-500 bg-red-50 dark:bg-red-500/10 rounded-md hover:bg-red-100 dark:hover:bg-red-500/20 transition-all text-[10px] font-bold uppercase tracking-wider"
                        >
                          {{ t('containerDetail.delete') }}
                        </button>
                      </div>
                    </div>
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
              <div v-if="containerStats" class="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div class="bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 p-5 rounded-xl">
                  <div class="flex items-center gap-2 text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 mb-3">
                    <Cpu :size="12" /> {{ t('containerDetail.cpu') }}
                  </div>
                  <div class="text-3xl font-mono font-bold tracking-tighter text-gray-900 dark:text-white">
                    {{ containerStats.cpu.percent }}%
                  </div>
                </div>
                
                <div class="bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 p-5 rounded-xl">
                  <div class="flex items-center gap-2 text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 mb-3">
                    <Activity :size="12" /> {{ t('containerDetail.ram') }}
                  </div>
                  <div class="text-3xl font-mono font-bold tracking-tighter text-gray-900 dark:text-white">
                    {{ containerStats.memory.percent }}%
                  </div>
                </div>
                
                <div class="md:col-span-2 bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 p-5 rounded-xl flex flex-col sm:flex-row sm:justify-between sm:items-center gap-4">
                  <div>
                    <div class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 mb-1.5">{{ t('containerDetail.networkIO') }}</div>
                    <div class="text-sm font-mono font-semibold text-gray-900 dark:text-white">
                      <span class="text-green-600 dark:text-green-500">↓ {{ formatBytes(containerStats.network.rx) }}</span>
                      <span class="text-gray-300 dark:text-zinc-700 mx-3">|</span>
                      <span class="text-blue-600 dark:text-blue-500">↑ {{ formatBytes(containerStats.network.tx) }}</span>
                    </div>
                  </div>
                  <div class="sm:text-right">
                    <div class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 mb-1.5">{{ t('containerDetail.blockIO') }}</div>
                    <div class="text-sm font-mono font-semibold text-gray-900 dark:text-white">
                      {{ formatBytes(containerStats.blockIO.read) }} / {{ formatBytes(containerStats.blockIO.write) }}
                    </div>
                  </div>
                </div>
              </div>
              <div v-else class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t('containerDetail.noResourceData') }}</div>
            </div>

            <div v-else-if="activeTab === 'output'" class="space-y-4">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2 text-[10px] font-bold text-gray-500 dark:text-zinc-500 uppercase tracking-widest">
                  <Terminal :size="12" /> {{ t('containerDetail.outputConsole') }}
                </div>
                <div class="flex items-center gap-2">
                  <button @click="autoScrollLogs = !autoScrollLogs" class="p-1.5 rounded-md hover:bg-gray-100 dark:hover:bg-zinc-800 text-gray-500 dark:text-zinc-400 transition-colors" :title="autoScrollLogs ? t('containerDetail.pauseAutoScroll') : t('containerDetail.enableAutoScroll')">
                    <component :is="autoScrollLogs ? Pause : Play" :size="14" />
                  </button>
                  <button @click="fetchContainerLogs" class="p-1.5 rounded-md hover:bg-gray-100 dark:hover:bg-zinc-800 text-gray-500 dark:text-zinc-400 transition-colors" :title="refreshingLogs ? t('containerDetail.refreshing') : t('common.refresh')">
                    <RefreshCw :size="14" :class="{ 'animate-spin': refreshingLogs }" />
                  </button>
                </div>
              </div>
              
              <div 
                id="terminal-logs"
                class="h-96 overflow-y-auto p-4 font-mono text-[11px] leading-5 text-gray-800 dark:text-gray-300 bg-gray-50 dark:bg-[#111] border border-gray-200 dark:border-zinc-800 rounded-xl scrollbar-thin scrollbar-thumb-gray-300 dark:scrollbar-thumb-zinc-700 scrollbar-track-transparent"
              >
                <div v-if="containerLogs.length === 0" class="flex flex-col items-center justify-center h-full text-gray-400 dark:text-zinc-600">
                  <div class="mb-2 text-[10px] font-bold uppercase tracking-widest">{{ t('containerDetail.noOutputLogs') }}</div>
                </div>
                <div v-else class="space-y-0.5">
                  <div v-for="(log, i) in containerLogs" :key="i" class="break-all whitespace-pre-wrap hover:bg-gray-100 dark:hover:bg-zinc-900 px-1 -mx-1 rounded-sm">
                    <span class="text-gray-400 dark:text-zinc-600 select-none mr-3 w-6 inline-block text-right">{{ i + 1 }}</span>{{ log }}
                  </div>
                </div>
              </div>
            </div>

            <div v-else class="space-y-4">
              <div class="flex items-center gap-2 text-[10px] font-bold text-gray-500 dark:text-zinc-500 uppercase tracking-widest">
                <Lock :size="12" /> {{ t('containerDetail.environmentVariables') }}
              </div>
              <div v-if="selectedContainer.env && selectedContainer.env.length > 0" class="bg-gray-50 dark:bg-[#111] border border-gray-200 dark:border-zinc-800 rounded-xl p-5 max-h-80 overflow-y-auto custom-scrollbar">
                <div v-for="(envVar, i) in selectedContainer.env" :key="i" class="font-mono text-[11px] mb-3 last:mb-0 break-all flex flex-col sm:flex-row gap-1 sm:gap-4">
                  <div class="text-gray-500 dark:text-zinc-500 font-bold shrink-0 sm:w-1/3">{{ envVar.split('=')[0] }}</div>
                  <div class="text-gray-900 dark:text-zinc-300 flex-1">{{ envVar.split('=').slice(1).join('=') }}</div>
                </div>
              </div>
              <div v-else class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t('containerDetail.noEnvVars') }}</div>
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
