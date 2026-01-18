<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { HardDrive, Eye, EyeOff, ExternalLink, Loader2, RefreshCw, Trash2, AlertCircle } from 'lucide-vue-next'
import { useToast } from 'vue-toastification'

const toast = useToast()

const volumesData = ref({})
const loading = ref(false)
const actionLoading = ref({})
const volumePorts = ref({})
const deletingVolume = ref(null)
const deletingAllVolumes = ref(false)

async function fetchVolumes() {
  loading.value = true
  try {
    const response = await fetch('/api/volumes')
    const data = await response.json()
    if (data.success) {
      volumesData.value = data
      
      // Fetch ports for browsing volumes
      const containers = await fetch('/api/containers')
      const containersData = await containers.json()
      if (containersData.success) {
        const newPorts = {}
        containersData.containers.forEach(container => {
          if (container.labels && container.labels['yantra.volume-browser']) {
            const volumeName = container.labels['yantra.volume-browser']
            const port = container.ports?.find(p => p.privatePort === 5000)?.publicPort
            if (port) {
              newPorts[volumeName] = port
            }
          }
        })
        volumePorts.value = newPorts
      }
    }
  } catch (error) {
    toast.error('Failed to fetch volumes')
    console.error(error)
  } finally {
    loading.value = false
  }
}

async function startBrowsing(volumeName) {
  actionLoading.value[volumeName] = true
  try {
    const response = await fetch(`/api/volumes/${volumeName}/browse`, {
      method: 'POST',
    })
    const data = await response.json()
    if (data.success) {
      toast.success(`Volume browser started on port ${data.port}`)
      
      // Store the port immediately
      if (data.port) {
        volumePorts.value[volumeName] = data.port
      }
      
      await fetchVolumes()
      
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
    delete actionLoading.value[volumeName]
  }
}

async function stopBrowsing(volumeName) {
  actionLoading.value[volumeName] = true
  try {
    const response = await fetch(`/api/volumes/${volumeName}/browse`, {
      method: 'DELETE',
    })
    const data = await response.json()
    if (data.success) {
      toast.success('Volume browser stopped')
      
      // Remove the port from storage
      delete volumePorts.value[volumeName]
      
      await fetchVolumes()
    }
  } catch (error) {
    toast.error('Failed to stop volume browser')
    console.error(error)
  } finally {
    delete actionLoading.value[volumeName]
  }
}

async function deleteVolume(volumeName) {
  if (!confirm(`Delete volume ${volumeName}?\n\nThis will permanently remove all data stored in this volume.\n\nThis action cannot be undone.`)) return

  deletingVolume.value = volumeName
  try {
    const response = await fetch(`/api/volumes/${volumeName}`, {
      method: 'DELETE'
    })
    const data = await response.json()

    if (data.success) {
      toast.success(`Volume deleted successfully!`)
      await fetchVolumes()
    } else {
      toast.error(`Deletion failed: ${data.error}\n${data.message}`)
    }
  } catch (error) {
    toast.error(`Deletion failed: ${error.message}`)
  } finally {
    deletingVolume.value = null
  }
}

async function deleteAllUnusedVolumes() {
  const count = volumesData.value.unusedVolumes?.length || 0
  if (!count) return
  
  if (!confirm(`Delete all ${count} unused volumes?\n\nThis will permanently remove all data stored in these volumes.\n\nThis action cannot be undone.`)) return

  deletingAllVolumes.value = true
  let deleted = 0
  let failed = 0

  try {
    for (const volume of volumesData.value.unusedVolumes) {
      try {
        const response = await fetch(`/api/volumes/${volume.name}`, {
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
      toast.success(`Successfully deleted ${deleted} unused volume${deleted > 1 ? 's' : ''}!${failed > 0 ? `\n${failed} failed.` : ''}`)
      await fetchVolumes()
    } else {
      toast.error(`Failed to delete volumes`)
    }
  } catch (error) {
    toast.error(`Deletion failed: ${error.message}`)
  } finally {
    deletingAllVolumes.value = false
  }
}

function formatDate(dateString) {
  if (!dateString) return 'N/A'
  return new Date(dateString).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

// Auto-refresh every 10 seconds
let refreshInterval = null
onMounted(() => {
  fetchVolumes()
  refreshInterval = setInterval(fetchVolumes, 10000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>

<template>
  <div class="p-4 sm:p-6 md:p-10 lg:p-12">
    <div class="mb-6 md:mb-8">
      <div class="flex items-center gap-3 mb-2">
        <HardDrive class="w-8 h-8 text-blue-600" />
        <h2 class="text-3xl sm:text-4xl font-bold text-gray-900">Docker Volumes</h2>
      </div>
      <p class="text-sm sm:text-base text-gray-600">Browse and manage your Docker volumes</p>
    </div>
    
    <div v-if="loading && (!volumesData.volumes || volumesData.volumes.length === 0)" class="text-center py-16">
      <div class="w-8 h-8 border-3 border-blue-600 border-t-transparent rounded-full animate-spin mx-auto mb-3"></div>
      <div class="text-gray-500 font-medium">Loading volumes...</div>
    </div>
    <div v-else>
      <!-- Summary Stats -->
      <div class="grid grid-cols-2 md:grid-cols-4 gap-3 sm:gap-4 md:gap-6 mb-8 md:mb-10">
        <div class="bg-gradient-to-br from-blue-50 to-indigo-50 rounded-2xl p-5 sm:p-6 smooth-shadow border border-blue-100 transition-all hover:shadow-lg hover:-translate-y-0.5">
          <div class="flex items-center gap-2 mb-3">
            <HardDrive class="w-4 h-4 text-blue-600" />
            <div class="text-blue-700 text-xs font-bold uppercase tracking-wider">Total</div>
          </div>
          <div class="text-3xl sm:text-4xl font-bold text-blue-900">{{ volumesData.total || 0 }}</div>
        </div>
        <div class="bg-gradient-to-br from-green-50 to-emerald-50 rounded-2xl p-5 sm:p-6 smooth-shadow border border-green-100 transition-all hover:shadow-lg hover:-translate-y-0.5">
          <div class="flex items-center gap-2 mb-3">
            <HardDrive class="w-4 h-4 text-green-600" />
            <div class="text-green-700 text-xs font-bold uppercase tracking-wider">In Use</div>
          </div>
          <div class="text-3xl sm:text-4xl font-bold text-green-900">{{ volumesData.used || 0 }}</div>
        </div>
        <div class="bg-gradient-to-br from-orange-50 to-amber-50 rounded-2xl p-5 sm:p-6 smooth-shadow border border-orange-100 transition-all hover:shadow-lg hover:-translate-y-0.5">
          <div class="flex items-center gap-2 mb-3">
            <AlertCircle class="w-4 h-4 text-orange-600" />
            <div class="text-orange-700 text-xs font-bold uppercase tracking-wider">Unused</div>
          </div>
          <div class="text-3xl sm:text-4xl font-bold text-orange-900">{{ volumesData.unused || 0 }}</div>
        </div>
        <div class="bg-gradient-to-br from-purple-50 to-pink-50 rounded-2xl p-5 sm:p-6 smooth-shadow border border-purple-100 transition-all hover:shadow-lg hover:-translate-y-0.5">
          <div class="flex items-center gap-2 mb-3">
            <Eye class="w-4 h-4 text-purple-600" />
            <div class="text-purple-700 text-xs font-bold uppercase tracking-wider">Browsing</div>
          </div>
          <div class="text-3xl sm:text-4xl font-bold text-purple-900">{{ volumesData.volumes?.filter(v => v.isBrowsing).length || 0 }}</div>
        </div>
      </div>

      <!-- Unused Volumes Section -->
      <div v-if="volumesData.unusedVolumes && volumesData.unusedVolumes.length > 0" class="mb-8 md:mb-10">
        <div class="bg-gradient-to-r from-orange-50 to-red-50 rounded-2xl p-4 sm:p-5 mb-5 border border-orange-200">
          <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
            <div>
              <div class="flex items-center gap-2.5 mb-1">
                <AlertCircle class="w-5 h-5 text-orange-600" />
                <h3 class="text-lg sm:text-xl font-bold text-orange-900">Unused Volumes</h3>
              </div>
              <p class="text-xs sm:text-sm text-orange-700">These volumes are not mounted by any containers and can be safely deleted</p>
              <div v-if="volumesData.unusedSize && volumesData.unusedSize > 0" class="text-xs sm:text-sm text-orange-600 font-semibold mt-1">Free up {{ volumesData.unusedSize }} MB</div>
            </div>
            <button @click="deleteAllUnusedVolumes"
              :disabled="deletingAllVolumes"
              class="w-full sm:w-auto px-4 py-2.5 bg-gradient-to-r from-red-500 to-red-600 hover:from-red-600 hover:to-red-700 disabled:from-gray-400 disabled:to-gray-400 disabled:cursor-not-allowed text-white rounded-xl text-sm font-semibold transition-all smooth-shadow active:scale-95 touch-manipulation flex items-center justify-center gap-2 whitespace-nowrap">
              <Trash2 class="w-4 h-4" />
              <span>{{ deletingAllVolumes ? 'Deleting...' : 'Delete All' }}</span>
            </button>
          </div>
        </div>
        <div class="space-y-3">
          <div v-for="volume in volumesData.unusedVolumes" :key="volume.name"
            class="bg-white rounded-2xl p-4 sm:p-5 border border-gray-200 hover:border-orange-300 transition-all smooth-shadow hover:shadow-lg">
            <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
              <div class="flex-1 w-full sm:w-auto">
                <div class="flex items-center gap-3 mb-2">
                  <div class="p-2 bg-orange-100 rounded-lg">
                    <HardDrive class="w-5 h-5 text-orange-600" />
                  </div>
                  <div class="flex-1 min-w-0">
                    <div class="font-bold text-gray-900 text-sm sm:text-base break-all">{{ volume.name }}</div>
                    <div class="text-xs text-gray-500 mt-0.5">{{ volume.driver }} • {{ formatDate(volume.createdAt) }} • {{ volume.size }} MB</div>
                  </div>
                </div>
              </div>
              <div class="flex items-center gap-3 sm:gap-4 w-full sm:w-auto">
                <button @click="startBrowsing(volume.name)"
                  :disabled="actionLoading[volume.name]"
                  class="px-4 py-2.5 bg-gradient-to-r from-purple-500 to-pink-600 hover:from-purple-600 hover:to-pink-700 disabled:from-gray-400 disabled:to-gray-400 disabled:cursor-not-allowed text-white rounded-xl text-sm font-semibold transition-all smooth-shadow active:scale-95 touch-manipulation flex items-center gap-2 whitespace-nowrap">
                  <Loader2 v-if="actionLoading[volume.name]" class="w-4 h-4 animate-spin" />
                  <Eye v-else class="w-4 h-4" />
                  <span>{{ actionLoading[volume.name] ? 'Starting...' : 'Browse' }}</span>
                </button>
                <button @click="deleteVolume(volume.name)"
                  :disabled="deletingVolume === volume.name"
                  class="px-4 py-2.5 bg-gradient-to-r from-red-500 to-red-600 hover:from-red-600 hover:to-red-700 disabled:from-gray-400 disabled:to-gray-400 disabled:cursor-not-allowed text-white rounded-xl text-sm font-semibold transition-all smooth-shadow active:scale-95 touch-manipulation flex items-center gap-2 whitespace-nowrap"
                  title="Delete this volume">
                  <Trash2 class="w-4 h-4" />
                  <span>{{ deletingVolume === volume.name ? 'Deleting...' : 'Delete' }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Browsing Volumes Section -->
      <div v-if="volumesData.volumes && volumesData.volumes.filter(v => v.isBrowsing).length > 0" class="mb-8 md:mb-10">
        <div class="bg-gradient-to-r from-green-50 to-emerald-50 rounded-2xl p-4 sm:p-5 mb-5 border border-green-200">
          <div class="flex items-center gap-2.5">
            <Eye class="w-5 h-5 text-green-600" />
            <h3 class="text-lg sm:text-xl font-bold text-green-900">Currently Browsing</h3>
          </div>
          <p class="text-xs sm:text-sm text-green-700 mt-1 ml-7">Active volume browsers that are currently open</p>
        </div>
        <div class="space-y-3">
          <div v-for="volume in volumesData.volumes.filter(v => v.isBrowsing)" :key="volume.name"
            class="bg-white rounded-2xl p-4 sm:p-5 border border-gray-200 hover:border-green-300 transition-all smooth-shadow hover:shadow-lg">
            <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
              <div class="flex-1 w-full sm:w-auto">
                <div class="flex items-center gap-3 mb-2">
                  <div class="p-2 bg-green-100 rounded-lg">
                    <HardDrive class="w-5 h-5 text-green-600" />
                  </div>
                  <div class="flex-1 min-w-0">
                    <div class="font-bold text-gray-900 text-sm sm:text-base break-all">{{ volume.name }}</div>
                    <div class="text-xs text-gray-500 mt-0.5">{{ volume.driver }} • {{ formatDate(volume.createdAt) }} • {{ volume.size }} MB</div>
                  </div>
                </div>
              </div>
              <div class="flex items-center gap-3 sm:gap-4 w-full sm:w-auto">
                <a
                  v-if="volumePorts[volume.name]"
                  :href="`http://${window.location.hostname || 'localhost'}:${volumePorts[volume.name]}`"
                  target="_blank"
                  class="px-4 py-2.5 bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 text-white rounded-xl text-sm font-semibold transition-all smooth-shadow active:scale-95 touch-manipulation flex items-center gap-2 whitespace-nowrap">
                  <ExternalLink class="w-4 h-4" />
                  <span>Open :{{ volumePorts[volume.name] }}</span>
                </a>
                <button @click="stopBrowsing(volume.name)"
                  :disabled="actionLoading[volume.name]"
                  class="px-4 py-2.5 bg-gradient-to-r from-red-500 to-red-600 hover:from-red-600 hover:to-red-700 disabled:from-gray-400 disabled:to-gray-400 disabled:cursor-not-allowed text-white rounded-xl text-sm font-semibold transition-all smooth-shadow active:scale-95 touch-manipulation flex items-center gap-2 whitespace-nowrap">
                  <Loader2 v-if="actionLoading[volume.name]" class="w-4 h-4 animate-spin" />
                  <EyeOff v-else class="w-4 h-4" />
                  <span>{{ actionLoading[volume.name] ? 'Stopping...' : 'Stop' }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Available Volumes Section (Used, not browsing) -->
      <div v-if="volumesData.usedVolumes && volumesData.usedVolumes.filter(v => !v.isBrowsing).length > 0">
        <div class="bg-gradient-to-r from-purple-50 to-pink-50 rounded-2xl p-4 sm:p-5 mb-5 border border-purple-200">
          <div class="flex items-center gap-2.5">
            <HardDrive class="w-5 h-5 text-purple-600" />
            <h3 class="text-lg sm:text-xl font-bold text-purple-900">Volumes in Use</h3>
          </div>
          <p class="text-xs sm:text-sm text-purple-700 mt-1 ml-7">These volumes are mounted by active containers</p>
        </div>
        <div class="space-y-3">
          <div v-for="volume in volumesData.usedVolumes.filter(v => !v.isBrowsing)" :key="volume.name"
            class="bg-white rounded-2xl p-4 sm:p-5 border border-gray-200 hover:border-purple-300 transition-all smooth-shadow hover:shadow-lg">
            <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
              <div class="flex-1 w-full sm:w-auto">
                <div class="flex items-center gap-3 mb-2">
                  <div class="p-2 bg-purple-100 rounded-lg">
                    <HardDrive class="w-5 h-5 text-purple-600" />
                  </div>
                  <div class="flex-1 min-w-0">
                    <div class="font-bold text-gray-900 text-sm sm:text-base break-all">{{ volume.name }}</div>
                    <div class="text-xs text-gray-500 mt-0.5">{{ volume.driver }} • {{ formatDate(volume.createdAt) }} • {{ volume.size }} MB</div>
                  </div>
                </div>
              </div>
              <div class="flex items-center gap-3 sm:gap-4 w-full sm:w-auto">
                <button @click="startBrowsing(volume.name)"
                  :disabled="actionLoading[volume.name]"
                  class="px-4 py-2.5 bg-gradient-to-r from-purple-500 to-pink-600 hover:from-purple-600 hover:to-pink-700 disabled:from-gray-400 disabled:to-gray-400 disabled:cursor-not-allowed text-white rounded-xl text-sm font-semibold transition-all smooth-shadow active:scale-95 touch-manipulation flex items-center gap-2 whitespace-nowrap">
                  <Loader2 v-if="actionLoading[volume.name]" class="w-4 h-4 animate-spin" />
                  <Eye v-else class="w-4 h-4" />
                  <span>{{ actionLoading[volume.name] ? 'Starting...' : 'Browse' }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="!volumesData.volumes || volumesData.volumes.length === 0"
        class="text-center py-16 sm:py-20">
        <div class="w-16 h-16 bg-gray-100 rounded-2xl flex items-center justify-center mx-auto mb-4">
          <HardDrive class="w-8 h-8 text-gray-400" />
        </div>
        <div class="text-gray-900 font-bold text-xl mb-2">No Volumes Found</div>
        <div class="text-gray-500 text-sm">Docker volumes will appear here once you install apps</div>
      </div>
    </div>
  </div>
</template>
