<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { Database, Cloud, RefreshCw, Trash2, Download, Settings, Plus, Loader2, AlertCircle, CheckCircle, Clock } from 'lucide-vue-next'
import { useToast } from 'vue-toastification'

const toast = useToast()

const s3Config = ref(null)
const s3Configured = ref(false)
const backups = ref([])
const volumes = ref([])
const loading = ref(false)
const configuring = ref(false)
const showConfigModal = ref(false)
const showCreateModal = ref(false)
const showRestoreModal = ref(null)

// S3 Configuration Form
const configForm = ref({
  provider: 'AWS',
  endpoint: '',
  bucket: '',
  accessKey: '',
  secretKey: '',
  region: 'us-east-1'
})

// Create Backup Form
const createForm = ref({
  name: '',
  selectedVolumes: []
})

// Restore Form
const restoreForm = ref({
  selectedVolumes: [],
  overwrite: false
})

// Active jobs
const backupJobs = ref([])
const restoreJobs = ref([])
let pollInterval = null

// Load S3 configuration
async function loadConfig() {
  try {
    const response = await fetch('/api/backup/config')
    const data = await response.json()

    if (data.success) {
      s3Configured.value = data.configured
      if (data.configured) {
        s3Config.value = data.config
      }
    }
  } catch (error) {
    console.error('Failed to load S3 config:', error)
  }
}

// Save S3 configuration
async function saveConfig() {
  configuring.value = true
  try {
    const response = await fetch('/api/backup/config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(configForm.value)
    })

    const data = await response.json()

    if (data.success) {
      toast.success('S3 configuration saved successfully')
      s3Configured.value = true
      showConfigModal.value = false
      await loadConfig()
      await loadBackups()
    } else {
      toast.error(data.error || 'Failed to save configuration')
    }
  } catch (error) {
    toast.error('Failed to save S3 configuration')
  } finally {
    configuring.value = false
  }
}

// Load backups from S3
async function loadBackups() {
  if (!s3Configured.value) return

  loading.value = true
  try {
    const response = await fetch('/api/backup/list')
    const data = await response.json()

    if (data.success) {
      backups.value = data.backups
    } else {
      toast.error(data.error || 'Failed to load backups')
    }
  } catch (error) {
    toast.error('Failed to load backups')
  } finally {
    loading.value = false
  }
}

// Load volumes
async function loadVolumes() {
  try {
    const response = await fetch('/api/volumes')
    const data = await response.json()

    if (data.success) {
      volumes.value = data.volumes
    }
  } catch (error) {
    console.error('Failed to load volumes:', error)
  }
}

// Create backup
async function createBackup() {
  if (createForm.value.selectedVolumes.length === 0) {
    toast.error('Please select at least one volume')
    return
  }

  try {
    const response = await fetch('/api/backup/create', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        volumes: createForm.value.selectedVolumes,
        name: createForm.value.name || undefined
      })
    })

    const data = await response.json()

    if (data.success) {
      toast.success('Backup started! This may take a while...')
      showCreateModal.value = false
      createForm.value = { name: '', selectedVolumes: [] }
      startPollingJobs()
    } else {
      toast.error(data.error || 'Failed to start backup')
    }
  } catch (error) {
    toast.error('Failed to create backup')
  }
}

// Restore backup
async function restoreBackup(backupId) {
  if (restoreForm.value.selectedVolumes.length === 0) {
    toast.error('Please select at least one volume')
    return
  }

  try {
    const response = await fetch(`/api/backup/${backupId}/restore`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        volumes: restoreForm.value.selectedVolumes.length > 0 ? restoreForm.value.selectedVolumes : undefined,
        overwrite: restoreForm.value.overwrite
      })
    })

    const data = await response.json()

    if (data.success) {
      toast.success('Restore started! This may take a while...')
      showRestoreModal.value = null
      restoreForm.value = { selectedVolumes: [], overwrite: false }
      startPollingJobs()
    } else {
      toast.error(data.error || 'Failed to start restore')
    }
  } catch (error) {
    toast.error('Failed to restore backup')
  }
}

// Delete backup
async function deleteBackup(backupId) {
  if (!confirm('Are you sure you want to delete this backup?')) return

  try {
    const response = await fetch(`/api/backup/${backupId}`, {
      method: 'DELETE'
    })

    const data = await response.json()

    if (data.success) {
      toast.success('Backup deleted successfully')
      await loadBackups()
    } else {
      toast.error(data.error || 'Failed to delete backup')
    }
  } catch (error) {
    toast.error('Failed to delete backup')
  }
}

// Poll jobs for progress updates
async function pollJobs() {
  try {
    const [backupRes, restoreRes] = await Promise.all([
      fetch('/api/backup/jobs'),
      fetch('/api/restore/jobs')
    ])

    const backupData = await backupRes.json()
    const restoreData = await restoreRes.json()

    if (backupData.success) {
      backupJobs.value = backupData.jobs.filter(j => j.status !== 'completed')

      // Check for newly completed jobs
      backupData.jobs.forEach(job => {
        if (job.status === 'completed' && !backupJobs.value.find(j => j.id === job.id)) {
          toast.success('Backup completed successfully!')
          loadBackups()
        }
      })
    }

    if (restoreData.success) {
      restoreJobs.value = restoreData.jobs.filter(j => j.status !== 'completed')

      // Check for newly completed jobs
      restoreData.jobs.forEach(job => {
        if (job.status === 'completed' && !restoreJobs.value.find(j => j.id === job.id)) {
          toast.success('Restore completed successfully!')
        }
      })
    }

    // Stop polling if no active jobs
    if (backupJobs.value.length === 0 && restoreJobs.value.length === 0) {
      stopPollingJobs()
    }
  } catch (error) {
    console.error('Failed to poll jobs:', error)
  }
}

function startPollingJobs() {
  if (!pollInterval) {
    pollInterval = setInterval(pollJobs, 2000) // Poll every 2 seconds
  }
}

function stopPollingJobs() {
  if (pollInterval) {
    clearInterval(pollInterval)
    pollInterval = null
  }
}

// Format date
function formatDate(dateString) {
  return new Date(dateString).toLocaleString()
}

// Format file size
function formatSize(bytes) {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
}

// Toggle volume selection
function toggleVolume(volumeName, array) {
  const index = array.indexOf(volumeName)
  if (index > -1) {
    array.splice(index, 1)
  } else {
    array.push(volumeName)
  }
}

// Open restore modal
function openRestoreModal(backup) {
  showRestoreModal.value = backup
  restoreForm.value.selectedVolumes = backup.volumes.map(v => v.name)
}

// Lifecycle
onMounted(async () => {
  await loadConfig()
  await loadVolumes()
  if (s3Configured.value) {
    await loadBackups()
  }
  pollJobs() // Initial poll
  startPollingJobs()
})

onUnmounted(() => {
  stopPollingJobs()
})
</script>

<template>
  <div class="container mx-auto px-4 py-6 max-w-7xl">
    <!-- Header -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-3xl font-bold flex items-center gap-2">
          <Cloud :size="32" />
          Backup & Restore
        </h1>
        <p class="text-gray-600 dark:text-gray-400 mt-1">
          Backup your volumes to S3-compatible storage
        </p>
      </div>
      <div class="flex gap-2">
        <button
          @click="showConfigModal = true"
          class="flex items-center gap-2 px-4 py-2 bg-gray-700 text-white rounded-lg hover:bg-gray-600 transition"
        >
          <Settings :size="18" />
          Configure S3
        </button>
        <button
          v-if="s3Configured"
          @click="loadBackups"
          :disabled="loading"
          class="flex items-center gap-2 px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition disabled:opacity-50"
        >
          <RefreshCw :size="18" :class="{ 'animate-spin': loading }" />
          Refresh
        </button>
      </div>
    </div>

    <!-- Active Jobs -->
    <div v-if="backupJobs.length > 0 || restoreJobs.length > 0" class="mb-6 space-y-2">
      <div v-for="job in backupJobs" :key="job.id" class="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <Loader2 :size="20" class="animate-spin text-blue-600" />
            <div>
              <p class="font-semibold text-gray-900 dark:text-white">Creating Backup</p>
              <p class="text-sm text-gray-600 dark:text-gray-400">
                {{ job.currentVolume || `${job.volumes.length} volume(s)` }}
              </p>
            </div>
          </div>
          <div class="text-right">
            <p class="text-lg font-bold text-blue-600">{{ job.progress }}%</p>
          </div>
        </div>
      </div>

      <div v-for="job in restoreJobs" :key="job.id" class="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <Loader2 :size="20" class="animate-spin text-green-600" />
            <div>
              <p class="font-semibold text-gray-900 dark:text-white">Restoring Backup</p>
              <p class="text-sm text-gray-600 dark:text-gray-400">
                {{ job.currentVolume || 'Processing...' }}
              </p>
            </div>
          </div>
          <div class="text-right">
            <p class="text-lg font-bold text-green-600">{{ job.progress }}%</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Not Configured State -->
    <div v-if="!s3Configured" class="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-8 text-center">
      <AlertCircle :size="48" class="mx-auto text-yellow-600 mb-4" />
      <h2 class="text-xl font-bold mb-2">S3 Not Configured</h2>
      <p class="text-gray-600 dark:text-gray-400 mb-4">
        Configure your S3 storage to start backing up your volumes
      </p>
      <button
        @click="showConfigModal = true"
        class="px-6 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition"
      >
        Configure S3 Now
      </button>
    </div>

    <!-- Backups List -->
    <div v-else>
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-xl font-bold">Backups ({{ backups.length }})</h2>
        <button
          @click="showCreateModal = true"
          class="flex items-center gap-2 px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition"
        >
          <Plus :size="18" />
          Create Backup
        </button>
      </div>

      <div v-if="loading" class="text-center py-12">
        <Loader2 :size="48" class="animate-spin mx-auto text-purple-600 mb-4" />
        <p class="text-gray-600 dark:text-gray-400">Loading backups...</p>
      </div>

      <div v-else-if="backups.length === 0" class="text-center py-12 bg-gray-50 dark:bg-gray-800 rounded-lg">
        <Database :size="48" class="mx-auto text-gray-400 mb-4" />
        <p class="text-gray-600 dark:text-gray-400">No backups found</p>
        <p class="text-sm text-gray-500 dark:text-gray-500 mt-2">Create your first backup to get started</p>
      </div>

      <div v-else class="grid gap-4">
        <div
          v-for="backup in backups"
          :key="backup.id"
          class="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-6 hover:border-purple-500 dark:hover:border-purple-500 transition"
        >
          <div class="flex items-start justify-between">
            <div class="flex-1">
              <div class="flex items-center gap-3 mb-2">
                <Database :size="24" class="text-purple-600" />
                <div>
                  <h3 class="text-lg font-bold">{{ backup.name }}</h3>
                  <p class="text-sm text-gray-500 dark:text-gray-400">{{ formatDate(backup.timestamp) }}</p>
                </div>
              </div>

              <div class="mt-4 space-y-2">
                <div class="flex items-center gap-2 text-sm">
                  <span class="text-gray-600 dark:text-gray-400">Volumes:</span>
                  <span class="font-semibold">{{ backup.volumes.length }}</span>
                </div>
                <div class="flex items-center gap-2 text-sm">
                  <span class="text-gray-600 dark:text-gray-400">Total Size:</span>
                  <span class="font-semibold">{{ formatSize(backup.totalSize) }}</span>
                </div>
                <div class="flex flex-wrap gap-2 mt-2">
                  <span
                    v-for="vol in backup.volumes"
                    :key="vol.name"
                    class="px-2 py-1 bg-gray-100 dark:bg-gray-700 rounded text-xs"
                  >
                    {{ vol.name }}
                  </span>
                </div>
              </div>
            </div>

            <div class="flex gap-2">
              <button
                @click="openRestoreModal(backup)"
                class="flex items-center gap-2 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition"
              >
                <Download :size="18" />
                Restore
              </button>
              <button
                @click="deleteBackup(backup.id)"
                class="p-2 text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition"
              >
                <Trash2 :size="18" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Configure S3 Modal -->
    <div v-if="showConfigModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white dark:bg-gray-800 rounded-lg max-w-md w-full p-6">
        <h2 class="text-xl font-bold mb-4">Configure S3 Storage</h2>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-1">Provider</label>
            <select v-model="configForm.provider" class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700">
              <option value="AWS">AWS S3</option>
              <option value="Other">MinIO / Other S3</option>
            </select>
          </div>

          <div v-if="configForm.provider === 'Other'">
            <label class="block text-sm font-medium mb-1">Endpoint URL</label>
            <input
              v-model="configForm.endpoint"
              type="text"
              placeholder="https://s3.example.com"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700"
            />
          </div>

          <div>
            <label class="block text-sm font-medium mb-1">Bucket Name</label>
            <input
              v-model="configForm.bucket"
              type="text"
              placeholder="my-backups"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700"
            />
          </div>

          <div>
            <label class="block text-sm font-medium mb-1">Access Key</label>
            <input
              v-model="configForm.accessKey"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700"
            />
          </div>

          <div>
            <label class="block text-sm font-medium mb-1">Secret Key</label>
            <input
              v-model="configForm.secretKey"
              type="password"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700"
            />
          </div>

          <div>
            <label class="block text-sm font-medium mb-1">Region</label>
            <input
              v-model="configForm.region"
              type="text"
              placeholder="us-east-1"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700"
            />
          </div>
        </div>

        <div class="flex gap-2 mt-6">
          <button
            @click="saveConfig"
            :disabled="configuring"
            class="flex-1 px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition disabled:opacity-50"
          >
            {{ configuring ? 'Saving...' : 'Save Configuration' }}
          </button>
          <button
            @click="showConfigModal = false"
            class="px-4 py-2 bg-gray-300 dark:bg-gray-600 rounded-lg hover:bg-gray-400 dark:hover:bg-gray-500 transition"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>

    <!-- Create Backup Modal -->
    <div v-if="showCreateModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white dark:bg-gray-800 rounded-lg max-w-md w-full p-6">
        <h2 class="text-xl font-bold mb-4">Create Backup</h2>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium mb-1">Backup Name (Optional)</label>
            <input
              v-model="createForm.name"
              type="text"
              placeholder="My Backup"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700"
            />
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">Select Volumes</label>
            <div class="max-h-60 overflow-y-auto border border-gray-300 dark:border-gray-600 rounded-lg p-2 space-y-1">
              <label
                v-for="volume in volumes"
                :key="volume.name"
                class="flex items-center gap-2 p-2 hover:bg-gray-50 dark:hover:bg-gray-700 rounded cursor-pointer"
              >
                <input
                  type="checkbox"
                  :checked="createForm.selectedVolumes.includes(volume.name)"
                  @change="toggleVolume(volume.name, createForm.selectedVolumes)"
                  class="w-4 h-4"
                />
                <div class="flex-1">
                  <p class="text-sm font-medium">{{ volume.name }}</p>
                  <p class="text-xs text-gray-500">{{ volume.size }} MB</p>
                </div>
              </label>
            </div>
          </div>
        </div>

        <div class="flex gap-2 mt-6">
          <button
            @click="createBackup"
            class="flex-1 px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition"
          >
            Create Backup
          </button>
          <button
            @click="showCreateModal = false"
            class="px-4 py-2 bg-gray-300 dark:bg-gray-600 rounded-lg hover:bg-gray-400 dark:hover:bg-gray-500 transition"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>

    <!-- Restore Backup Modal -->
    <div v-if="showRestoreModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white dark:bg-gray-800 rounded-lg max-w-md w-full p-6">
        <h2 class="text-xl font-bold mb-4">Restore Backup</h2>

        <div class="space-y-4">
          <div class="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-3">
            <p class="text-sm text-yellow-800 dark:text-yellow-200">
              Warning: Restoring will stop containers using these volumes and replace their data.
            </p>
          </div>

          <div>
            <label class="block text-sm font-medium mb-2">Select Volumes to Restore</label>
            <div class="max-h-60 overflow-y-auto border border-gray-300 dark:border-gray-600 rounded-lg p-2 space-y-1">
              <label
                v-for="vol in showRestoreModal.volumes"
                :key="vol.name"
                class="flex items-center gap-2 p-2 hover:bg-gray-50 dark:hover:bg-gray-700 rounded cursor-pointer"
              >
                <input
                  type="checkbox"
                  :checked="restoreForm.selectedVolumes.includes(vol.name)"
                  @change="toggleVolume(vol.name, restoreForm.selectedVolumes)"
                  class="w-4 h-4"
                />
                <div class="flex-1">
                  <p class="text-sm font-medium">{{ vol.name }}</p>
                  <p class="text-xs text-gray-500">{{ formatSize(vol.size) }}</p>
                </div>
              </label>
            </div>
          </div>

          <label class="flex items-center gap-2 cursor-pointer">
            <input
              v-model="restoreForm.overwrite"
              type="checkbox"
              class="w-4 h-4"
            />
            <span class="text-sm">Overwrite existing volumes</span>
          </label>
        </div>

        <div class="flex gap-2 mt-6">
          <button
            @click="restoreBackup(showRestoreModal.id)"
            class="flex-1 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition"
          >
            Restore
          </button>
          <button
            @click="showRestoreModal = null"
            class="px-4 py-2 bg-gray-300 dark:bg-gray-600 rounded-lg hover:bg-gray-400 dark:hover:bg-gray-500 transition"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
