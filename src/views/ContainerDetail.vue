<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'vue-toastification'
import { ArrowLeft, ExternalLink, RefreshCw, Trash2, Network, FolderOpen, Terminal, Activity, Cpu, HardDrive, ShieldCheck, Share2, Globe, Database, Lock, Folder, Pause, Play, Download, Clock, AlertCircle } from 'lucide-vue-next'

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
const currentTime = ref(Date.now())
const activeTab = ref('resources')

// Backup state
const s3Configured = ref(false)
const volumeBackups = ref({})
const backingUp = ref(false)
const showRestoreMenu = ref({})
const backupJobId = ref(null)

// Update current time every second for live countdown
let timeUpdateInterval = null

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

// Check if container has expiration
const expirationInfo = computed(() => {
  if (!selectedContainer.value?.expireAt) return null
  
  const expireAtTimestamp = parseInt(selectedContainer.value.expireAt, 10)
  if (isNaN(expireAtTimestamp)) return null
  
  const expireAtMs = expireAtTimestamp * 1000
  const timeLeftMs = expireAtMs - currentTime.value
  
  if (timeLeftMs <= 0) {
    return {
      expired: true,
      timeLeft: 'Expired',
      urgency: 'critical'
    }
  }
  
  const seconds = Math.floor(timeLeftMs / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  
  let timeLeft = ''
  let urgency = 'normal'
  
  if (days > 0) {
    timeLeft = `${days}d ${hours % 24}h`
    urgency = days < 1 ? 'warning' : 'normal'
  } else if (hours > 0) {
    timeLeft = `${hours}h ${minutes % 60}m`
    urgency = hours < 2 ? 'critical' : 'warning'
  } else if (minutes > 0) {
    timeLeft = `${minutes}m ${seconds % 60}s`
    urgency = 'critical'
  } else {
    timeLeft = `${seconds}s`
    urgency = 'critical'
  }
  
  return {
    expired: false,
    timeLeft,
    urgency,
    expireAt: new Date(expireAtMs).toLocaleString()
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

// Check S3 configuration
async function checkS3Config() {
  try {
    const response = await fetch(`${apiUrl.value}/api/backup/config`)
    const data = await response.json()
    s3Configured.value = data.configured
  } catch (error) {
    console.error('Failed to check S3 config:', error)
  }
}

// Fetch backups for container volumes
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

// Backup all volumes
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
      toast.success('Backup started for all volumes')
      backupJobId.value = data.jobId
      pollBackupJob(data.jobId)
    } else {
      toast.error(data.error || 'Failed to start backup')
    }
  } catch (error) {
    toast.error('Failed to start backup')
  } finally {
    backingUp.value = false
  }
}

// Poll backup job status
async function pollBackupJob(jobId) {
  const pollInterval = setInterval(async () => {
    try {
      const response = await fetch(`${apiUrl.value}/api/backup/jobs/${jobId}`)
      const data = await response.json()

      if (data.success && data.job) {
        if (data.job.status === 'completed') {
          clearInterval(pollInterval)
          toast.success('Backup completed successfully')
          await fetchVolumeBackups()
        } else if (data.job.status === 'failed') {
          clearInterval(pollInterval)
          toast.error(`Backup failed: ${data.job.error}`)
        }
      }
    } catch (error) {
      clearInterval(pollInterval)
      console.error('Failed to poll backup job:', error)
    }
  }, 2000)
}

// Restore backup
async function restoreBackup(volumeName, backupKey) {
  if (!confirm(`Restore ${volumeName} from backup?\n\nThis will overwrite current data.`)) return

  try {
    const response = await fetch(
      `${apiUrl.value}/api/volumes/${volumeName}/restore`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ backupKey, overwrite: true })
      }
    )
    const data = await response.json()

    if (data.success) {
      toast.success('Restore started')
      pollRestoreJob(data.jobId)
    } else {
      toast.error(data.error || 'Failed to start restore')
    }
  } catch (error) {
    toast.error('Failed to start restore')
  }

  showRestoreMenu.value[volumeName] = false
}

// Poll restore job status
async function pollRestoreJob(jobId) {
  const pollInterval = setInterval(async () => {
    try {
      const response = await fetch(`${apiUrl.value}/api/restore/jobs/${jobId}`)
      const data = await response.json()

      if (data.success && data.job) {
        if (data.job.status === 'completed') {
          clearInterval(pollInterval)
          toast.success('Restore completed successfully')
        } else if (data.job.status === 'failed') {
          clearInterval(pollInterval)
          toast.error(`Restore failed: ${data.job.error}`)
        }
      }
    } catch (error) {
      clearInterval(pollInterval)
      console.error('Failed to poll restore job:', error)
    }
  }, 2000)
}

// Delete backup
async function deleteBackupFile(volumeName, backupKey) {
  const timestamp = backupKey.split('/')[1].replace('.tar', '')

  if (!confirm('Delete this backup?')) return

  try {
    const response = await fetch(
      `${apiUrl.value}/api/volumes/${volumeName}/backup/${timestamp}`,
      { method: 'DELETE' }
    )
    const data = await response.json()

    if (data.success) {
      toast.success('Backup deleted')
      await fetchVolumeBackups()
    } else {
      toast.error(data.error || 'Failed to delete backup')
    }
  } catch (error) {
    toast.error('Failed to delete backup')
  }
}

// Toggle restore menu
function toggleRestoreMenu(volumeName) {
  showRestoreMenu.value[volumeName] = !showRestoreMenu.value[volumeName]
}

// Check if volume has backups
function hasBackups(volumeName) {
  return volumeBackups.value[volumeName]?.length > 0
}

// Get latest backup age
function getLatestBackupAge(volumeName) {
  const backups = volumeBackups.value[volumeName]
  if (!backups || backups.length === 0) return 'Never'

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
  return 'Just now'
}

// Format backup date
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
  
  // Start polling stats every 2 seconds
  statsInterval = setInterval(() => {
    fetchContainerStats()
  }, 2000)
  
  // Update current time every second for expiration countdown
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
  <div class="min-h-screen bg-slate-50 dark:bg-[#09090b] text-slate-900 dark:text-slate-200 font-sans">
    <!-- Header -->
    <header class="bg-white dark:bg-[#18181b] border-b border-slate-200 dark:border-slate-800/50">
      <div class="max-w-7xl mx-auto px-6 h-16 flex items-center justify-between">
        <div class="flex items-center gap-4">
          <router-link to="/" class="inline-flex items-center justify-center w-9 h-9 rounded-xl hover:bg-slate-100 dark:hover:bg-slate-800/50 transition-all text-slate-600 dark:text-slate-400">
            <ArrowLeft :size="18" />
          </router-link>
          
          <div class="h-5 w-px bg-slate-200 dark:bg-slate-800"></div>

          <div class="flex items-center gap-2.5 text-sm">
            <span class="text-slate-500 dark:text-slate-400">Containers</span>
            <span class="text-slate-300 dark:text-slate-700">/</span>
            <span class="font-semibold text-slate-900 dark:text-white" v-if="selectedContainer">{{ selectedContainer.name }}</span>
            <span v-else class="w-32 h-5 bg-slate-200 dark:bg-slate-800 animate-pulse rounded-lg"></span>
          </div>
        </div>

        <div v-if="selectedContainer" class="flex items-center gap-2">
          <!-- Expiration Badge -->
          <div v-if="expirationInfo" 
               class="flex items-center gap-2 px-3 py-1.5 rounded-full text-[11px] font-bold uppercase tracking-wide"
               :class="{
                 'bg-red-100 dark:bg-red-950/50 text-red-700 dark:text-red-400': expirationInfo.urgency === 'critical',
                 'bg-orange-100 dark:bg-orange-950/50 text-orange-700 dark:text-orange-400': expirationInfo.urgency === 'warning',
                 'bg-blue-100 dark:bg-blue-950/50 text-blue-700 dark:text-blue-400': expirationInfo.urgency === 'normal'
               }"
               :title="`Expires at: ${expirationInfo.expireAt}`">
            <Clock :size="13" :class="expirationInfo.urgency === 'critical' ? 'animate-pulse' : ''" />
            <span>{{ expirationInfo.timeLeft }}</span>
          </div>
          
          <!-- Running State Badge -->
          <div class="flex items-center gap-2 px-3 py-1.5 rounded-full text-[11px] font-bold uppercase tracking-wide"
            :class="selectedContainer.state === 'running' 
              ? 'bg-emerald-100 dark:bg-emerald-950/50 text-emerald-700 dark:text-emerald-400' 
              : 'bg-slate-100 dark:bg-slate-900/50 text-slate-600 dark:text-slate-400'">
            <div class="w-1.5 h-1.5 rounded-full" :class="selectedContainer.state === 'running' ? 'bg-emerald-500 animate-pulse' : 'bg-slate-400'"></div>
            <span>{{ selectedContainer.state }}</span>
          </div>
        </div>
      </div>
    </header>

    <div v-if="!selectedContainer" class="max-w-7xl mx-auto p-8 flex justify-center">
       <div class="animate-spin text-slate-300"><RefreshCw :size="32" /></div>
    </div>

    <main v-else class="max-w-7xl mx-auto px-6 py-6 space-y-5">
        
        <!-- Info Card -->
        <div class="bg-white dark:bg-[#1c1c1e] rounded-2xl border border-slate-200/50 dark:border-slate-800/50 p-6 flex flex-col sm:flex-row gap-6 shadow-sm">
           <div class="w-20 h-20 bg-slate-100/50 dark:bg-slate-900/50 rounded-xl flex items-center justify-center p-3 shrink-0">
              <img v-if="selectedContainer.app.logo" :src="selectedContainer.app.logo" class="w-full h-full object-contain" />
              <div v-else class="text-3xl">📦</div>
           </div>
           
           <div class="flex-1 space-y-3">
              <h1 class="text-2xl font-bold text-slate-900 dark:text-white">{{ selectedContainer.name }}</h1>
              <p class="text-slate-600 dark:text-slate-300 text-sm leading-relaxed max-w-2xl">
                 {{ selectedContainer.app.description || "Container usage description not available." }}
              </p>
              <div class="pt-1 flex flex-wrap gap-2">
                 <div class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-slate-100 dark:bg-slate-900/50 text-xs font-mono text-slate-700 dark:text-slate-300 rounded-lg">
                    <Database :size="12" />
                    {{ selectedContainer.image }}
                 </div>
                 <div class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-slate-100 dark:bg-slate-900/50 text-xs font-mono text-slate-700 dark:text-slate-300 rounded-lg">
                    <span class="text-slate-500">ID:</span> {{ selectedContainer.id.substring(0, 12) }}
                 </div>
              </div>
           </div>
        </div>

        <!-- Expiration Warning Banner -->
        <div v-if="expirationInfo" 
             class="rounded-2xl p-4 flex items-start gap-3"
             :class="{
               'bg-red-100/50 dark:bg-red-950/30': expirationInfo.urgency === 'critical',
               'bg-orange-100/50 dark:bg-orange-950/30': expirationInfo.urgency === 'warning',
               'bg-blue-100/50 dark:bg-blue-950/30': expirationInfo.urgency === 'normal'
             }">
          <div class="shrink-0 mt-0.5">
            <AlertCircle :size="20" 
                        :class="{
                          'text-red-600 dark:text-red-400': expirationInfo.urgency === 'critical',
                          'text-orange-600 dark:text-orange-400': expirationInfo.urgency === 'warning',
                          'text-blue-600 dark:text-blue-400': expirationInfo.urgency === 'normal'
                        }" />
          </div>
          <div class="flex-1">
            <div class="font-semibold text-sm mb-1"
                 :class="{
                   'text-red-900 dark:text-red-200': expirationInfo.urgency === 'critical',
                   'text-orange-900 dark:text-orange-200': expirationInfo.urgency === 'warning',
                   'text-blue-900 dark:text-blue-200': expirationInfo.urgency === 'normal'
                 }">
              {{ expirationInfo.expired ? 'Container Expired' : 'Temporary Container' }}
            </div>
            <div class="text-xs"
                 :class="{
                   'text-red-700 dark:text-red-300': expirationInfo.urgency === 'critical',
                   'text-orange-700 dark:text-orange-300': expirationInfo.urgency === 'warning',
                   'text-blue-700 dark:text-blue-300': expirationInfo.urgency === 'normal'
                 }">
              <span v-if="!expirationInfo.expired">
                This container will be automatically removed in <span class="font-mono font-bold">{{ expirationInfo.timeLeft }}</span>
              </span>
              <span v-else>
                This container has expired and will be removed shortly.
              </span>
              <span class="block mt-1 opacity-75">Scheduled removal: {{ expirationInfo.expireAt }}</span>
            </div>
          </div>
        </div>

        <!-- Network Access (Moved from Right) -->
        <div v-if="allPortMappings.length > 0" class="space-y-3">
           <h3 class="text-xs font-bold uppercase tracking-wider text-slate-500">Network Access</h3>
           <div class="bg-white dark:bg-[#1c1c1e] rounded-2xl border border-slate-200/50 dark:border-slate-800/50 overflow-hidden shadow-sm">
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

        <!-- Storage (Attached + Backups) -->
        <div v-if="containerVolumes.length > 0" class="space-y-3">
           <h3 class="text-xs font-bold uppercase tracking-wider text-slate-500">Storage</h3>
           <div class="bg-white dark:bg-[#1c1c1e] rounded-2xl border border-slate-200/50 dark:border-slate-800/50 p-5 space-y-6 shadow-sm">
             <div class="space-y-2">
               <div class="text-xs font-bold uppercase tracking-wider text-slate-500">Attached Storage</div>
               <div class="flex items-start gap-2 px-3 py-2 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800/30 rounded-lg">
                 <div class="shrink-0 mt-0.5">
                   <Share2 :size="16" class="text-blue-600 dark:text-blue-400" />
                 </div>
                 <div class="text-xs text-blue-900 dark:text-blue-200">
                   <span class="font-semibold">Access via WebDAV:</span> Browse volumes on any device using WebDAV protocol. Click <span class="font-mono bg-blue-100 dark:bg-blue-950/50 px-1 py-0.5 rounded">Browse Files</span> to start a temporary WebDAV server and mount volumes as network drives.
                 </div>
               </div>
             </div>

             <div class="grid gap-3">
               <div v-for="volume in containerVolumes" :key="volume.name" 
                  class="group bg-slate-50/50 dark:bg-slate-900/30 border border-slate-200 dark:border-slate-800/50 rounded-xl p-4 space-y-3 hover:border-blue-500/50 transition-all">
                   
                 <div class="flex items-center justify-between gap-4">
                  <div class="flex items-center gap-4 min-w-0">
                    <div class="w-10 h-10 rounded-xl bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center text-blue-600 dark:text-blue-400 shrink-0">
                      <HardDrive :size="20" />
                    </div>
                    <div class="min-w-0">
                      <div class="font-medium text-slate-900 dark:text-white truncate text-sm" :title="volume.name">{{ volume.name }}</div>
                      <div class="text-xs text-slate-500 font-mono truncate">{{ volume.destination }}</div>
                    </div>
                  </div>

                  <div class="flex items-center gap-2">
                    <div class="relative min-w-[120px] flex justify-end">
                      <div v-if="browsingVolume[volume.name]" class="text-xs text-blue-500 animate-pulse font-medium">Starting...</div>
                      <button 
                        v-else-if="!showVolumeMenu[volume.name]"
                        @click="showVolumeMenu[volume.name] = true"
                        class="px-3 py-1.5 text-xs font-medium bg-slate-200 dark:bg-slate-800 text-slate-700 dark:text-slate-300 rounded-lg hover:bg-blue-100 hover:text-blue-600 dark:hover:bg-blue-900/30 dark:hover:text-blue-400 transition-all"
                      >
                        Browse
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

                    <button
                      @click="backupAllVolumes"
                      :disabled="backingUp || !s3Configured"
                      class="text-xs px-3 py-1.5 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed transition-all font-medium"
                      title="Backup this container's volumes"
                    >
                      Backup
                    </button>
                    <button
                      @click="toggleRestoreMenu(volume.name)"
                      :disabled="!hasBackups(volume.name) || !s3Configured"
                      class="text-xs px-3 py-1.5 bg-slate-200 dark:bg-slate-700 rounded-lg hover:bg-slate-300 dark:hover:bg-slate-600 disabled:opacity-30 disabled:cursor-not-allowed transition-all font-medium"
                    >
                      Restore
                    </button>
                  </div>
                 </div>

                 <div class="text-xs text-slate-500">
                  Latest backup: {{ getLatestBackupAge(volume.name) }}
                 </div>

                 <div
                  v-if="showRestoreMenu[volume.name] && hasBackups(volume.name)"
                  class="bg-white dark:bg-slate-800 rounded border border-slate-200 dark:border-slate-600 p-2 max-h-40 overflow-y-auto space-y-1"
                 >
                  <div
                    v-for="backup in volumeBackups[volume.name]"
                    :key="backup.key"
                    class="flex items-center justify-between py-1.5 px-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded text-xs"
                  >
                    <div>
                     <div class="font-mono text-xs">{{ formatBackupDate(backup.timestamp) }}</div>
                     <div class="text-slate-500 text-[10px]">{{ formatBytes(backup.size) }}</div>
                    </div>
                    <div class="flex gap-1">
                     <button
                      @click="restoreBackup(volume.name, backup.key)"
                      class="px-2 py-1 bg-green-600 text-white rounded hover:bg-green-700 transition-all text-[10px]"
                     >
                      Restore
                     </button>
                     <button
                      @click="deleteBackupFile(volume.name, backup.key)"
                      class="px-2 py-1 bg-red-600 text-white rounded hover:bg-red-700 transition-all text-[10px]"
                     >
                      Delete
                     </button>
                    </div>
                  </div>
                 </div>

                </div>
             </div>

             <div class="border-t border-slate-200 dark:border-slate-800 pt-5 space-y-3">
               <div class="flex items-center justify-between">
                 <div class="text-xs font-bold uppercase tracking-wider text-slate-500">Volume Backups</div>
                 <button
                   v-if="s3Configured"
                   @click="backupAllVolumes"
                   :disabled="backingUp"
                   class="text-xs px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-all font-medium"
                 >
                   {{ backingUp ? 'Backing up...' : 'Backup All' }}
                 </button>
               </div>

               <div v-if="!s3Configured" class="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-3">
                 <p class="text-xs text-yellow-800 dark:text-yellow-200">
                   Configure S3 storage to enable backups.
                   <router-link to="/minioconfig" class="underline hover:text-yellow-900 dark:hover:text-yellow-100 font-semibold">Configure now</router-link>
                 </p>
               </div>
             </div>
           </div>
        </div>

        <!-- System Panel (Tabs) -->
        <div class="bg-white dark:bg-[#1c1c1e] rounded-2xl border border-slate-200/50 dark:border-slate-800/50 overflow-hidden shadow-sm">
          <div class="flex items-center justify-between px-5 py-4 border-b border-slate-200/50 dark:border-slate-800/50">
            <div class="flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-slate-500">
              <ShieldCheck :size="14" />
              System
            </div>
            <div class="flex items-center gap-1 rounded-full bg-slate-100/80 dark:bg-slate-900/50 p-1">
              <button
                @click="activeTab = 'resources'"
                :class="activeTab === 'resources' ? 'bg-white dark:bg-slate-800 text-slate-900 dark:text-white shadow-sm' : 'text-slate-500 hover:text-slate-700 dark:hover:text-slate-300'"
                class="px-3 py-1.5 text-xs rounded-full transition-all font-medium"
              >
                Resources
              </button>
              <button
                @click="activeTab = 'output'"
                :class="activeTab === 'output' ? 'bg-white dark:bg-slate-800 text-slate-900 dark:text-white shadow-sm' : 'text-slate-500 hover:text-slate-700 dark:hover:text-slate-300'"
                class="px-3 py-1.5 text-xs rounded-full transition-all font-medium"
              >
                Output
              </button>
              <button
                @click="activeTab = 'env'"
                :class="activeTab === 'env' ? 'bg-white dark:bg-slate-800 text-slate-900 dark:text-white shadow-sm' : 'text-slate-500 hover:text-slate-700 dark:hover:text-slate-300'"
                class="px-3 py-1.5 text-xs rounded-full transition-all font-medium"
              >
                Environment
              </button>
            </div>
          </div>

          <div class="p-5">
            <!-- Resources Tab -->
            <div v-if="activeTab === 'resources'">
              <div v-if="containerStats" class="grid grid-cols-1 md:grid-cols-2 gap-3">
                <div class="bg-slate-100/50 dark:bg-slate-900/30 p-4 rounded-xl">
                  <div class="flex items-center gap-2 text-xs uppercase tracking-wider text-slate-500 mb-2">
                    <Cpu :size="14" /> CPU
                  </div>
                  <div class="text-2xl font-mono font-semibold text-slate-900 dark:text-white">
                    {{ containerStats.cpu.percent }}%
                  </div>
                </div>
                
                <div class="bg-slate-100/50 dark:bg-slate-900/30 p-4 rounded-xl">
                  <div class="flex items-center gap-2 text-xs uppercase tracking-wider text-slate-500 mb-2">
                    <Activity :size="14" /> RAM
                  </div>
                  <div class="text-2xl font-mono font-semibold text-slate-900 dark:text-white">
                    {{ containerStats.memory.percent }}%
                  </div>
                </div>
                
                <div class="md:col-span-2 bg-slate-100/50 dark:bg-slate-900/30 p-4 rounded-xl flex flex-col sm:flex-row sm:justify-between sm:items-center gap-3">
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
              <div v-else class="text-sm text-slate-500">No resource data available.</div>
            </div>

            <!-- Output Tab -->
            <div v-else-if="activeTab === 'output'" class="space-y-3">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2 text-xs font-medium text-slate-500 uppercase tracking-wider">
                  <Terminal :size="14" /> Output Console
                </div>
                <div class="flex items-center gap-2">
                  <button @click="autoScrollLogs = !autoScrollLogs" class="p-1.5 rounded hover:bg-slate-100 dark:hover:bg-slate-900 text-slate-500" :title="autoScrollLogs ? 'Pause Auto-Scroll' : 'Enable Auto-Scroll'">
                    <component :is="autoScrollLogs ? Pause : Play" :size="14" />
                  </button>
                  <button @click="fetchContainerLogs" class="p-1.5 rounded hover:bg-slate-100 dark:hover:bg-slate-900 text-slate-500" :title="refreshingLogs ? 'Refreshing...' : 'Refresh'">
                    <RefreshCw :size="14" :class="{ 'animate-spin': refreshingLogs }" />
                  </button>
                </div>
              </div>
              
              <div 
                id="terminal-logs"
                class="h-96 overflow-y-auto p-4 font-mono text-xs leading-5 text-slate-700 dark:text-slate-200 bg-slate-100/50 dark:bg-[#1e1e1e] rounded-xl scrollbar-thin scrollbar-thumb-[#424242] scrollbar-track-transparent"
              >
                <div v-if="containerLogs.length === 0" class="flex flex-col items-center justify-center h-full text-slate-400">
                  <div class="mb-2 opacity-70">No output logs found</div>
                </div>
                <div v-else class="space-y-0.5">
                  <div v-for="(log, i) in containerLogs" :key="i" class="break-all whitespace-pre-wrap hover:bg-slate-100 dark:hover:bg-[#2a2d2e] px-1 -mx-1 rounded-sm">
                    <span class="text-slate-400 select-none mr-2 w-6 inline-block text-right">{{ i + 1 }}</span>{{ log }}
                  </div>
                </div>
              </div>
            </div>

            <!-- Environment Tab -->
            <div v-else class="space-y-3">
              <div class="flex items-center gap-2 text-xs font-medium text-slate-500 uppercase tracking-wider">
                <Lock :size="14" /> Environment
              </div>
              <div v-if="selectedContainer.env && selectedContainer.env.length > 0" class="bg-slate-100/50 dark:bg-[#1e1e1e] rounded-xl p-4 max-h-80 overflow-y-auto custom-scrollbar">
                <div v-for="(envVar, i) in selectedContainer.env" :key="i" class="font-mono text-[10px] mb-2 last:mb-0 break-all">
                  <div class="text-slate-500 mb-0.5">{{ envVar.split('=')[0] }}</div>
                  <div class="text-slate-800 dark:text-slate-200 pl-2">{{ envVar.split('=').slice(1).join('=') }}</div>
                </div>
              </div>
              <div v-else class="text-sm text-slate-500">No environment variables available.</div>
            </div>
          </div>
        </div>

      <!-- RIGHT COLUMN (now stacked) -->
         


         <!-- Actions -->
         <div class="bg-white dark:bg-[#1c1c1e] rounded-2xl border border-slate-200/50 dark:border-slate-800/50 p-5 space-y-4 shadow-sm">
            <h3 class="text-xs font-bold uppercase tracking-wider text-slate-500">Control</h3>
            
            <button 
               @click="deleteContainer"
               :disabled="deleting"
               class="w-full flex items-center justify-center gap-2 px-4 py-3 bg-red-100 dark:bg-red-900/20 text-red-600 dark:text-red-400 rounded-xl hover:bg-red-200 dark:hover:bg-red-900/30 transition-all font-semibold text-sm"
            >
               <Trash2 :size="16" />
               {{ deleting ? 'Terminating...' : 'Terminate Container' }}
            </button>
            
            <p class="text-[10px] text-slate-400 text-center px-4 leading-relaxed">
               Warning: This action is irreversible. All associated volumes and data will be permanently destroyed.
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
