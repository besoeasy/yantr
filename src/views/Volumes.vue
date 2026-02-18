<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { HardDrive, Eye, EyeOff, ExternalLink, Loader2, RefreshCw, Trash2, AlertCircle, Box, Check, Search } from 'lucide-vue-next'
import { useToast } from 'vue-toastification'
import VueApexCharts from 'vue3-apexcharts'

const toast = useToast()

const volumesData = ref({})
const loading = ref(false)
const actionLoading = ref({})
const volumePorts = ref({})
const deletingVolume = ref(null)
const deletingAllVolumes = ref(false)
const searchQuery = ref('')
const currentTab = ref('active') // 'active', 'unused', 'browsing'

// Chart Configuration
const treemapOptions = computed(() => ({
  chart: {
    type: 'treemap',
    fontFamily: 'inherit',
    toolbar: { show: false },
    background: 'transparent',
    animations: { enabled: true, speed: 600 }
  },
  colors: ['#8b5cf6', '#ef4444', '#f59e0b', '#3b82f6'],
  states: {
    hover: { filter: { type: 'darken', value: 0.1 } }
  },
  plotOptions: {
    treemap: {
      distributed: true,
      enableShades: true,
      shadeIntensity: 0.5,
      radius: 4,
      useFillColorAsStroke: true,
    }
  },
  dataLabels: {
    enabled: true,
    style: {
      fontSize: '11px',
      fontFamily: 'inherit',
      fontWeight: 600,
    },
    formatter: function(text, op) {
      if (op.value < 1) return [text, '< 1 MB']
      return [text, op.value + ' MB']
    },
    offsetY: -4
  },
  title: { text: undefined },
  tooltip: {
    theme: 'dark',
    style: { fontFamily: 'inherit', fontSize: '12px' },
    y: {
      formatter: function(val) { return val + " MB" }
    },
    marker: { show: false }
  },
  stroke: { show: true, width: 2, colors: ['#ffffff'] }
}))

const treemapSeries = computed(() => {
  if (!volumesData.value.usedVolumes && !volumesData.value.unusedVolumes) return []
  
  const allVolumes = []
  
  // Add used volume
  if (volumesData.value.usedVolumes) {
    volumesData.value.usedVolumes.forEach(vol => {
      let sizeVal = parseFloat(vol.size)
      if (isNaN(sizeVal)) sizeVal = 0
      
      allVolumes.push({
        x: vol.name.length > 20 ? vol.name.substring(0, 17) + '...' : vol.name,
        y: sizeVal > 0 ? sizeVal : 0.01,
        fillColor: '#8b5cf6' // Purple for used
      })
    })
  }

  // Add unused
  if (volumesData.value.unusedVolumes) {
    volumesData.value.unusedVolumes.forEach(vol => {
      let sizeVal = parseFloat(vol.size)
      if (isNaN(sizeVal)) sizeVal = 0
      
      allVolumes.push({
        x: vol.name.length > 20 ? vol.name.substring(0, 17) + '...' : vol.name,
        y: sizeVal > 0 ? sizeVal : 0.01,
        fillColor: '#ef4444' // Red unused
      })
    })
  }
  
  const sortedData = allVolumes.sort((a, b) => b.y - a.y).slice(0, 30) // top 30
  return [{ data: sortedData }]
})

// Filters
const filteredUsed = computed(() => {
  if (!volumesData.value.usedVolumes) return []
  let vols = volumesData.value.usedVolumes.filter(v => !v.isBrowsing)
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    vols = vols.filter(v => v.name.toLowerCase().includes(q))
  }
  return vols
})

const filteredUnused = computed(() => {
  if (!volumesData.value.unusedVolumes) return []
  let vols = volumesData.value.unusedVolumes
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    vols = vols.filter(v => v.name.toLowerCase().includes(q))
  }
  return vols
})

const browsingVolumes = computed(() => {
  if (!volumesData.value.volumes) return []
  return volumesData.value.volumes.filter(v => v.isBrowsing)
})

async function fetchVolumes() {
  loading.value = true
  try {
    const response = await fetch('/api/volumes')
    const data = await response.json()
    if (data.success) {
      volumesData.value = data
      
      const containers = await fetch('/api/containers')
      const containersData = await containers.json()
      if (containersData.success) {
        const newPorts = {}
        containersData.containers.forEach(container => {
          if (container.labels && container.labels['yantr.volume-browser']) {
            const volumeName = container.labels['yantr.volume-browser']
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
    console.error(error)
  } finally {
    loading.value = false
  }
}

async function startBrowsing(volumeName) {
  actionLoading.value[volumeName] = true
  try {
    const response = await fetch(`/api/volumes/${volumeName}/browse`, { method: 'POST' })
    const data = await response.json()
    if (data.success) {
      toast.success(`Browser started`)
      if (data.port) volumePorts.value[volumeName] = data.port
      await fetchVolumes()
      if (data.port) {
        const url = `http://${window.location.hostname || 'localhost'}:${data.port}`
        window.open(url, '_blank')
      }
    }
  } catch (error) {
    toast.error('Failed to start browser')
  } finally {
    delete actionLoading.value[volumeName]
  }
}

async function stopBrowsing(volumeName) {
  actionLoading.value[volumeName] = true
  try {
    const response = await fetch(`/api/volumes/${volumeName}/browse`, { method: 'DELETE' })
    const data = await response.json()
    if (data.success) {
      toast.success('Browser stopped')
      delete volumePorts.value[volumeName]
      await fetchVolumes()
    }
  } catch (error) {
    toast.error('Failed to stop browser')
  } finally {
    delete actionLoading.value[volumeName]
  }
}

async function deleteVolume(volumeName) {
  if (!confirm(`Delete volume ${volumeName}?`)) return

  deletingVolume.value = volumeName
  try {
    const response = await fetch(`/api/volumes/${volumeName}`, { method: 'DELETE' })
    const data = await response.json()
    if (data.success) {
      toast.success(`Volume deleted`)
      await fetchVolumes()
    } else {
      toast.error(`Deletion failed: ${data.message}`)
    }
  } catch (error) {
    toast.error(error.message)
  } finally {
    deletingVolume.value = null
  }
}

async function deleteAllUnusedVolumes() {
  const count = volumesData.value.unusedVolumes?.length || 0
  if (!count) return
  if (!confirm(`Delete all ${count} unused volumes?`)) return

  deletingAllVolumes.value = true
  let deleted = 0
  try {
    for (const volume of volumesData.value.unusedVolumes) {
      try {
        const response = await fetch(`/api/volumes/${volume.name}`, { method: 'DELETE' })
        const data = await response.json()
        if (data.success) deleted++
      } catch (error) {}
    }
    toast.success(`Cleaned up ${deleted} volumes`)
    await fetchVolumes()
  } catch (error) {
    toast.error(error.message)
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

let refreshInterval = null
onMounted(() => {
  fetchVolumes()
  refreshInterval = setInterval(fetchVolumes, 10000)
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
})
</script>

<template>
  <div class="min-h-screen bg-neutral-50 dark:bg-[#0f1117] font-sans text-slate-800 dark:text-slate-200 pb-20">
    <!-- Header -->
    <header class="bg-white dark:bg-[#181b21] border-b border-gray-200 dark:border-gray-800 sticky top-0 z-30">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 bg-purple-50 dark:bg-purple-500/10 rounded-lg flex items-center justify-center text-purple-600 dark:text-purple-400">
              <HardDrive class="w-5 h-5" />
            </div>
            <h1 class="text-lg font-bold text-gray-900 dark:text-white">Volumes</h1>
          </div>
          
          <div class="flex items-center gap-3">
            <div class="relative flex-1 sm:flex-initial">
              <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input 
                v-model="searchQuery" 
                type="text" 
                placeholder="Search volumes..." 
                class="w-full sm:w-64 pl-10 pr-4 py-2 bg-gray-100 dark:bg-gray-800 border-none rounded-lg text-sm focus:ring-2 focus:ring-purple-500 focus:bg-white dark:focus:bg-[#181b21] transition-all"
              />
            </div>
            <button @click="fetchVolumes" class="p-2 bg-white dark:bg-[#181b21] border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800 rounded-lg transition-colors shrink-0">
              <RefreshCw class="w-4 h-4 text-gray-500" :class="{ 'animate-spin': loading }" />
            </button>
          </div>
        </div>
      </div>
    </header>

    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8">
      
      <!-- Stats Overview -->
      <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div class="bg-white dark:bg-[#181b21] p-5 rounded-xl border border-gray-100 dark:border-gray-800 shadow-sm flex flex-col justify-between h-32">
          <div class="flex justify-between items-start">
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Total Volumes</span>
            <Box class="w-5 h-5 text-purple-500 bg-purple-50 dark:bg-purple-500/10 p-1 rounded" />
          </div>
          <span class="text-3xl font-bold text-gray-900 dark:text-white">{{ volumesData.total || 0 }}</span>
        </div>

        <div class="bg-white dark:bg-[#181b21] p-5 rounded-xl border border-gray-100 dark:border-gray-800 shadow-sm flex flex-col justify-between h-32">
          <div class="flex justify-between items-start">
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">In Use</span>
            <Check class="w-5 h-5 text-emerald-500 bg-emerald-50 dark:bg-emerald-500/10 p-1 rounded" />
          </div>
          <span class="text-3xl font-bold text-gray-900 dark:text-white">{{ volumesData.used || 0 }}</span>
        </div>

        <div class="bg-white dark:bg-[#181b21] p-5 rounded-xl border border-gray-100 dark:border-gray-800 shadow-sm flex flex-col justify-between h-32">
          <div class="flex justify-between items-start">
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Unused</span>
            <AlertCircle class="w-5 h-5 text-amber-500 bg-amber-50 dark:bg-amber-500/10 p-1 rounded" />
          </div>
          <span class="text-3xl font-bold text-gray-900 dark:text-white">{{ volumesData.unused || 0 }}</span>
        </div>

        <div class="bg-white dark:bg-[#181b21] p-5 rounded-xl border border-gray-100 dark:border-gray-800 shadow-sm flex flex-col justify-between h-32">
          <div class="flex justify-between items-start">
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Browsing</span>
            <Eye class="w-5 h-5 text-blue-500 bg-blue-50 dark:bg-blue-500/10 p-1 rounded" />
          </div>
          <span class="text-3xl font-bold text-gray-900 dark:text-white">{{ browsingVolumes.length || 0 }}</span>
        </div>
      </div>

      <!-- Treemap -->
      <div v-if="treemapSeries.length > 0" class="bg-white dark:bg-[#181b21] rounded-xl border border-gray-100 dark:border-gray-800 shadow-sm p-6">
         <h3 class="text-base font-semibold text-gray-900 dark:text-white mb-4">Volume Size</h3>
         <div class="rounded-lg overflow-hidden bg-gray-50 dark:bg-gray-900/50 p-1">
            <VueApexCharts :options="treemapOptions" :series="treemapSeries" height="280" />
         </div>
      </div>

       <!-- Browsing Section - Show prominent if active -->
       <div v-if="browsingVolumes.length > 0" class="space-y-4">
         <div class="flex items-center gap-2 text-blue-600 dark:text-blue-400">
             <Eye class="w-5 h-5" />
             <h3 class="font-bold text-lg">Active Sessions</h3>
         </div>
         <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            <div v-for="volume in browsingVolumes" :key="volume.name" 
                 class="bg-blue-50 dark:bg-blue-900/10 border border-blue-100 dark:border-blue-800 rounded-xl p-5 flex flex-col">
                <div class="flex justify-between items-start mb-3">
                    <span class="text-xs font-bold uppercase tracking-wider text-blue-600 dark:text-blue-400 bg-white dark:bg-blue-900/50 px-2 py-1 rounded">Browsing</span>
                    <button @click="stopBrowsing(volume.name)" class="text-gray-400 hover:text-red-500 transition-colors">
                        <EyeOff class="w-4 h-4" />
                    </button>
                </div>
                <h4 class="font-mono font-medium text-sm text-gray-900 dark:text-white truncate mb-4" :title="volume.name">{{ volume.name }}</h4>
                <div class="mt-auto flex gap-2">
                    <a v-if="volumePorts[volume.name]"
                       :href="`http://${window.location.hostname || 'localhost'}:${volumePorts[volume.name]}`"
                       target="_blank"
                       class="flex-1 flex items-center justify-center gap-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg px-4 py-2 text-sm font-medium transition-colors shadow-sm">
                       <ExternalLink class="w-4 h-4" />
                       Open Finder
                    </a>
                </div>
            </div>
         </div>
      </div>

      <!-- Main Tabs -->
      <div class="space-y-4">
        <div class="flex items-center justify-between border-b border-gray-200 dark:border-gray-800 pb-1">
          <div class="flex gap-6">
            <button 
              @click="currentTab = 'active'"
              :class="currentTab === 'active' ? 'text-purple-600 dark:text-purple-400 border-purple-600 dark:border-purple-400' : 'text-gray-500 dark:text-gray-400 border-transparent hover:text-gray-700 dark:hover:text-gray-300'"
              class="pb-3 text-sm font-medium border-b-2 transition-colors flex items-center gap-2">
              <Check class="w-4 h-4" />
              Active Volumes
              <span class="bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 text-xs px-2 py-0.5 rounded-full">{{ filteredUsed.length }}</span>
            </button>
            <button 
              @click="currentTab = 'unused'"
              :class="currentTab === 'unused' ? 'text-amber-600 dark:text-amber-400 border-amber-600 dark:border-amber-400' : 'text-gray-500 dark:text-gray-400 border-transparent hover:text-gray-700 dark:hover:text-gray-300'"
              class="pb-3 text-sm font-medium border-b-2 transition-colors flex items-center gap-2">
              <AlertCircle class="w-4 h-4" />
              Unused Volumes
              <span class="bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 text-xs px-2 py-0.5 rounded-full">{{ filteredUnused.length }}</span>
            </button>
          </div>
          
           <button v-if="currentTab === 'unused' && filteredUnused.length > 0"
            @click="deleteAllUnusedVolumes"
            :disabled="deletingAllVolumes"
            class="text-xs font-semibold uppercase text-amber-600 hover:text-amber-700 dark:text-amber-500 dark:hover:text-amber-400 transition-colors flex items-center gap-1">
             <Trash2 class="w-3 h-3" />
             {{ deletingAllVolumes ? 'Cleaning...' : 'Prune All' }}
           </button>
        </div>

        <!-- Active Grid -->
         <div v-if="currentTab === 'active'" class="bg-white dark:bg-[#181b21] rounded-xl border border-gray-100 dark:border-gray-800 shadow-sm overflow-hidden">
           <table class="w-full text-left border-collapse">
              <thead>
                 <tr class="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider border-b border-gray-100 dark:border-gray-800">
                    <th class="px-6 py-4 font-medium w-1/3">Name</th>
                    <th class="px-6 py-4 font-medium">Driver</th>
                    <th class="px-6 py-4 font-medium">Size</th>
                    <th class="px-6 py-4 font-medium">Created</th>
                    <th class="px-4 py-4 w-32 text-right">Actions</th>
                 </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-gray-800 text-sm">
                 <tr v-if="filteredUsed.length === 0" class="bg-gray-50/50 dark:bg-gray-900/20">
                    <td colspan="5" class="px-6 py-12 text-center text-gray-400">No active/mounted volumes found</td>
                 </tr>
                 <tr v-for="volume in filteredUsed" :key="volume.name" class="group hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors">
                    <td class="px-6 py-4">
                        <div class="font-medium text-gray-900 dark:text-white truncate max-w-[250px]" :title="volume.name">{{ volume.name }}</div>
                    </td>
                    <td class="px-6 py-4 text-gray-500">{{ volume.driver }}</td>
                    <td class="px-6 py-4 text-gray-600 dark:text-gray-300">{{ volume.size }} MB</td>
                    <td class="px-6 py-4 text-gray-500">{{ formatDate(volume.createdAt) }}</td>
                    <td class="px-4 py-4 text-right">
                       <button @click="startBrowsing(volume.name)" 
                          :disabled="actionLoading[volume.name]"
                          class="inline-flex items-center gap-1 px-3 py-1.5 bg-gray-100 dark:bg-gray-800 hover:bg-purple-50 dark:hover:bg-purple-900/20 text-gray-600 dark:text-gray-300 hover:text-purple-600 dark:hover:text-purple-400 rounded-lg text-xs font-medium transition-colors">
                          <Loader2 v-if="actionLoading[volume.name]" class="w-3 h-3 animate-spin" />
                          <Eye v-else class="w-3 h-3" />
                          Browse
                       </button>
                    </td>
                 </tr>
              </tbody>
           </table>
        </div>

        <!-- Unused Grid -->
        <div v-if="currentTab === 'unused'" class="bg-white dark:bg-[#181b21] rounded-xl border border-gray-100 dark:border-gray-800 shadow-sm overflow-hidden">
           <div class="overflow-x-auto">
           <table class="w-full text-left border-collapse">
              <thead>
                 <tr class="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider border-b border-gray-100 dark:border-gray-800">
                    <th class="px-6 py-4 font-medium w-1/3">Name</th>
                    <th class="px-6 py-4 font-medium">Driver</th>
                    <th class="px-6 py-4 font-medium">Size</th>
                    <th class="px-6 py-4 font-medium">Created</th>
                    <th class="px-4 py-4 w-32 text-right">Actions</th>
                 </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-gray-800 text-sm">
                 <tr v-if="filteredUnused.length === 0" class="bg-gray-50/50 dark:bg-gray-900/20">
                    <td colspan="5" class="px-6 py-12 text-center text-gray-400">No unused volumes found</td>
                 </tr>
                 <tr v-for="volume in filteredUnused" :key="volume.name" class="group hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors">
                    <td class="px-6 py-4">
                        <div class="font-medium text-gray-900 dark:text-white truncate max-w-[250px]" :title="volume.name">{{ volume.name }}</div>
                    </td>
                    <td class="px-6 py-4 text-gray-500">{{ volume.driver }}</td>
                    <td class="px-6 py-4 text-gray-600 dark:text-gray-300">{{ volume.size }} MB</td>
                    <td class="px-6 py-4 text-gray-500">{{ formatDate(volume.createdAt) }}</td>
                    <td class="px-4 py-4 text-right flex items-center justify-end gap-2">
                       <button @click="startBrowsing(volume.name)" 
                          :disabled="actionLoading[volume.name]"
                          class="p-2 text-gray-400 hover:text-purple-500 rounded-lg transition-colors"
                          title="Browse">
                           <Loader2 v-if="actionLoading[volume.name]" class="w-4 h-4 animate-spin" />
                           <Eye v-else class="w-4 h-4" />
                       </button>
                       <button @click="deleteVolume(volume.name)" 
                          class="p-2 text-gray-400 hover:text-rose-500 rounded-lg transition-colors opacity-0 group-hover:opacity-100"
                          title="Delete">
                          <Trash2 class="w-4 h-4" />
                       </button>
                    </td>
                 </tr>
              </tbody>
           </table>
           </div>
        </div>

      </div>
    </main>
  </div>
</template>