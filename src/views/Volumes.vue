<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { HardDrive, Eye, EyeOff, ExternalLink, Loader2, RefreshCw, Trash2, AlertCircle, Box, Check } from 'lucide-vue-next'
import StatCard from '../components/StatCard.vue'
import SizeDistributionChart from '../components/SizeDistributionChart.vue'
import SearchInput from '../components/SearchInput.vue'
import UnderlineTabBar from '../components/UnderlineTabBar.vue'
import { useNotification } from '../composables/useNotification'
import { useI18n } from 'vue-i18n'

const toast = useNotification()
const { t } = useI18n()

const volumesData = ref({})
const loading = ref(false)
const actionLoading = ref({})
const deletingVolume = ref(null)
const deletingAllVolumes = ref(false)
const searchQuery = ref('')
const currentTab = ref('active') // 'active', 'unused', 'browsing'

// Chart Data
const chartItems = computed(() => {
  const allVolumes = []

  if (volumesData.value.usedVolumes) {
    volumesData.value.usedVolumes.forEach(vol => {
      const sizeVal = parseFloat(vol.size)
      if (!isNaN(sizeVal) && sizeVal > 0) {
        const name = vol.name.length > 20 ? vol.name.substring(0, 17) + '...' : vol.name
        allVolumes.push({ name, size: sizeVal, color: '#10b981' })
      }
    })
  }

  if (volumesData.value.unusedVolumes) {
    volumesData.value.unusedVolumes.forEach(vol => {
      const sizeVal = parseFloat(vol.size)
      if (!isNaN(sizeVal) && sizeVal > 0) {
        const name = vol.name.length > 20 ? vol.name.substring(0, 17) + '...' : vol.name
        allVolumes.push({ name, size: sizeVal, color: '#ef4444' })
      }
    })
  }

  const sorted = allVolumes.sort((a, b) => b.size - a.size).slice(0, 15)
  const max = sorted[0]?.size || 1
  return sorted.map(item => ({ ...item, pct: Math.round((item.size / max) * 100) }))
})

// Filters
const filteredUsed = computed(() => {
  if (!volumesData.value.usedVolumes) return []
  let vols = volumesData.value.usedVolumes
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
    const res = await fetch('/api/volumes')
    const data = await res.json()
    if (data.success) volumesData.value = data
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
      toast.success(t('volumes.browserStarted'))
      await fetchVolumes()
      window.open(`/browse/${volumeName}/`, '_blank')
    }
  } catch (error) {
    toast.error(t('volumes.failedToStartBrowser'))
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
      toast.success(t('volumes.browserStopped'))
      await fetchVolumes()
    }
  } catch (error) {
    toast.error(t('volumes.failedToStopBrowser'))
  } finally {
    delete actionLoading.value[volumeName]
  }
}

async function deleteVolume(volumeName) {
  if (!confirm(t('volumes.deleteVolume', { name: volumeName }))) return

  deletingVolume.value = volumeName
  try {
    const response = await fetch(`/api/volumes/${volumeName}`, { method: 'DELETE' })
    const data = await response.json()
    if (data.success) {
      toast.success(t('volumes.volumeDeleted'))
      await fetchVolumes()
    } else {
      toast.error(t('volumes.deletionFailed', { message: data.message }))
    }
  } catch (error) {
    toast.error(t('volumes.deletionFailed', { message: error.message }))
  } finally {
    deletingVolume.value = null
  }
}

async function deleteAllUnusedVolumes() {
  const count = volumesData.value.unusedVolumes?.length || 0
  if (!count) return
  if (!confirm(t('volumes.deleteAllUnused', { count }))) return

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
    toast.success(t('volumes.cleanedUp', { count: deleted }))
    await fetchVolumes()
  } catch (error) {
    toast.error(t('volumes.deletionFailed', { message: error.message }))
  } finally {
    deletingAllVolumes.value = false
  }
}

function formatDate(dateString) {
  if (!dateString) return t('volumes.notAvailable')
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
  <div class="min-h-screen bg-white dark:bg-[#0A0A0A] font-sans text-gray-900 dark:text-white pb-20">
    <!-- Header -->
    <header class="bg-white/80 dark:bg-[#0A0A0A]/80 backdrop-blur-md border-b border-gray-200 dark:border-zinc-800 sticky top-0 z-30">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div class="flex items-center gap-3">
            <div class="flex items-center justify-center w-10 h-10 rounded-lg border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900/50">
              <HardDrive class="w-5 h-5 text-gray-700 dark:text-zinc-300" />
            </div>
            <div>
              <h1 class="text-lg font-semibold tracking-tight text-gray-900 dark:text-white">{{ t('volumes.title') }}</h1>
              <p class="text-xs font-medium text-gray-500 dark:text-zinc-500">{{ t('volumes.subtitle') }}</p>
            </div>
          </div>
          
          <div class="flex items-center gap-3">
            <SearchInput v-model="searchQuery" :placeholder="t('volumes.searchPlaceholder')" />
            <button @click="fetchVolumes" class="p-2 bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 hover:border-gray-300 dark:hover:border-zinc-600 rounded-lg transition-colors shrink-0 group">
              <RefreshCw class="w-4 h-4 text-gray-600 dark:text-zinc-400 group-hover:text-blue-500 transition-colors" :class="{ 'animate-spin text-blue-500': loading }" />
            </button>
          </div>
        </div>
      </div>
    </header>

    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8">
      
      <!-- Stats Overview -->
      <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard :label="t('volumes.totalVolumes')" :value="volumesData.total || 0" :icon="Box" accent="blue" />
        <StatCard :label="t('volumes.inUse')" :value="volumesData.used || 0" :icon="Check" accent="green" />
        <StatCard :label="t('volumes.unused')" :value="volumesData.unused || 0" :icon="AlertCircle" accent="amber" />
        <StatCard :label="t('volumes.browsing')" :value="browsingVolumes.length || 0" :icon="Eye" accent="blue-light" />
      </div>

      <!-- Volume Size Distribution -->
      <SizeDistributionChart
        :items="chartItems"
        :title="t('volumes.volumeSizeDistribution')"
        :legend="[{ color: '#10b981', label: t('volumes.inUse') }, { color: '#ef4444', label: t('volumes.unused') }]"
        :unit="t('volumes.mb')"
      />

       <!-- Browsing Section - Show prominent if active -->
       <transition name="fade">
         <div v-if="browsingVolumes.length > 0" class="space-y-4">
           <div class="flex items-center gap-2 text-blue-600 dark:text-blue-500">
               <div class="w-2 h-2 rounded-full bg-blue-500 animate-pulse"></div>
               <h3 class="text-sm font-bold uppercase tracking-wider">{{ t('volumes.activeSessions') }}</h3>
           </div>
           <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              <div v-for="volume in browsingVolumes" :key="volume.name" 
                   class="bg-blue-50/50 dark:bg-blue-900/10 border border-blue-200 dark:border-blue-900/30 rounded-xl p-5 flex flex-col">
                  <div class="flex justify-between items-start mb-3">
                      <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-blue-600 dark:text-blue-400">{{ t('volumes.browsingLabel') }}</span>
                      <button @click="stopBrowsing(volume.name)"
                          :disabled="actionLoading[volume.name]"
                          class="inline-flex items-center gap-1 px-2.5 py-1 rounded-md border border-red-200 dark:border-red-900/40 text-red-500 hover:text-red-600 hover:border-red-400 hover:bg-red-50 dark:hover:bg-red-500/10 text-[10px] font-bold uppercase tracking-wider transition-colors">
                          <Loader2 v-if="actionLoading[volume.name]" class="w-3 h-3 animate-spin" />
                          <EyeOff v-else class="w-3 h-3" />
                          {{ t('volumes.stop') }}
                      </button>
                  </div>
                  <h4 class="font-mono font-medium text-sm text-gray-900 dark:text-white truncate mb-4" :title="volume.name">{{ volume.name }}</h4>
                  <div class="mt-auto flex gap-2">
                      <a :href="`/browse/${volume.name}/`"
                         target="_blank"
                         class="flex-1 flex items-center justify-center gap-2 bg-gray-900 hover:bg-gray-800 text-white dark:bg-white dark:hover:bg-gray-100 dark:text-gray-900 rounded-lg px-4 py-2.5 text-xs font-bold uppercase tracking-wider transition-colors shadow-sm">
                         <ExternalLink class="w-4 h-4" />
                         {{ t('volumes.openFinder') }}
                      </a>
                  </div>
              </div>
           </div>
        </div>
      </transition>

      <!-- Main Tabs -->
      <div class="space-y-4">
        <UnderlineTabBar
          v-model="currentTab"
          :tabs="[
            { key: 'active', label: t('volumes.activeVolumes'), count: filteredUsed.length },
            { key: 'unused', label: t('volumes.unusedVolumes'), count: filteredUnused.length }
          ]"
        >
          <template #action>
            <button v-if="currentTab === 'unused' && filteredUnused.length > 0"
              @click="deleteAllUnusedVolumes"
              :disabled="deletingAllVolumes"
              class="text-[10px] font-bold uppercase tracking-[0.2em] text-red-600 hover:text-red-700 dark:text-red-500 dark:hover:text-red-400 transition-colors flex items-center gap-2 px-3 py-1.5 rounded-md border border-red-200 dark:border-red-900/30 bg-red-50 dark:bg-red-900/10">
              <Trash2 class="w-3 h-3" />
              {{ deletingAllVolumes ? t('volumes.cleaning') : t('volumes.pruneAll') }}
            </button>
          </template>
        </UnderlineTabBar>

        <!-- Active Grid -->
        <transition name="fade" mode="out-in">
          <div v-if="currentTab === 'active'" class="bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 overflow-hidden">
             <div class="overflow-x-auto">
             <table class="w-full text-left border-collapse">
                <thead>
                   <tr class="text-[10px] font-bold text-gray-500 dark:text-zinc-500 uppercase tracking-[0.2em] border-b border-gray-200 dark:border-zinc-800 bg-gray-50/50 dark:bg-zinc-900/20">
                      <th class="px-6 py-4 w-1/3">{{ t('volumes.name') }}</th>
                      <th class="px-6 py-4">{{ t('volumes.driver') }}</th>
                      <th class="px-6 py-4">{{ t('volumes.size') }}</th>
                      <th class="px-6 py-4">{{ t('volumes.created') }}</th>
                      <th class="px-4 py-4 w-32 text-right">{{ t('volumes.actions') }}</th>
                   </tr>
                </thead>
                <tbody class="divide-y divide-gray-200 dark:divide-zinc-800 text-sm font-medium">
                   <tr v-if="filteredUsed.length === 0">
                      <td colspan="5" class="px-6 py-12 text-center text-gray-500 dark:text-zinc-500 text-sm">{{ t('volumes.noActiveVolumes') }}</td>
                   </tr>
                   <tr v-for="volume in filteredUsed" :key="volume.name" class="group hover:bg-gray-50 dark:hover:bg-zinc-900/50 transition-colors">
                      <td class="px-6 py-4">
                          <div class="font-mono text-gray-900 dark:text-white truncate max-w-62.5 text-xs" :title="volume.name">{{ volume.name }}</div>
                      </td>
                      <td class="px-6 py-4 text-gray-500 dark:text-zinc-400">{{ volume.driver }}</td>
                      <td class="px-6 py-4 text-gray-600 dark:text-zinc-300 tabular-nums">{{ volume.size }} {{ t('volumes.mb') }}</td>
                      <td class="px-6 py-4 text-gray-500 dark:text-zinc-500 text-xs">{{ formatDate(volume.createdAt) }}</td>
                      <td class="px-4 py-4 text-right">
                         <div v-if="volume.isBrowsing" class="inline-flex items-center gap-1.5">
                            <a :href="`/browse/${volume.name}/`" target="_blank"
                               class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-md text-[10px] font-bold uppercase tracking-wider transition-colors shadow-sm">
                               <ExternalLink class="w-3 h-3" />
                               {{ t('volumes.open') }}
                            </a>
                            <button @click="stopBrowsing(volume.name)"
                               :disabled="actionLoading[volume.name]"
                               class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-gray-50 dark:bg-zinc-900 border border-red-200 dark:border-red-900/40 hover:border-red-400 text-red-500 hover:text-red-600 rounded-md text-[10px] font-bold uppercase tracking-wider transition-colors shadow-sm">
                               <Loader2 v-if="actionLoading[volume.name]" class="w-3 h-3 animate-spin" />
                               <EyeOff v-else class="w-3 h-3" />
                               {{ t('volumes.stop') }}
                            </button>
                         </div>
                         <button v-else @click="startBrowsing(volume.name)" 
                            :disabled="actionLoading[volume.name]"
                            class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 hover:border-gray-300 dark:hover:border-zinc-700 text-gray-700 dark:text-zinc-300 hover:text-gray-900 dark:hover:text-white rounded-md text-[10px] font-bold uppercase tracking-wider transition-colors shadow-sm">
                            <Loader2 v-if="actionLoading[volume.name]" class="w-3 h-3 animate-spin text-blue-500" />
                            <Eye v-else class="w-3 h-3" />
                            {{ t('volumes.browse') }}
                         </button>
                      </td>
                   </tr>
                 </tbody>
              </table>
              </div>
           </div>
   
           <!-- Unused Grid -->
          <div v-else-if="currentTab === 'unused'" class="bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 overflow-hidden">
             <div class="overflow-x-auto">
             <table class="w-full text-left border-collapse">
                <thead>
                   <tr class="text-[10px] font-bold text-gray-500 dark:text-zinc-500 uppercase tracking-[0.2em] border-b border-gray-200 dark:border-zinc-800 bg-gray-50/50 dark:bg-zinc-900/20">
                      <th class="px-6 py-4 w-1/3">{{ t('volumes.name') }}</th>
                      <th class="px-6 py-4">{{ t('volumes.driver') }}</th>
                      <th class="px-6 py-4">{{ t('volumes.size') }}</th>
                      <th class="px-6 py-4">{{ t('volumes.created') }}</th>
                      <th class="px-4 py-4 w-32 text-right">{{ t('volumes.actions') }}</th>
                   </tr>
                </thead>
                <tbody class="divide-y divide-gray-200 dark:divide-zinc-800 text-sm font-medium">
                   <tr v-if="filteredUnused.length === 0">
                      <td colspan="5" class="px-6 py-12 text-center text-gray-500 dark:text-zinc-500 text-sm">{{ t('volumes.noUnusedVolumes') }}</td>
                   </tr>
                   <tr v-for="volume in filteredUnused" :key="volume.name" class="group hover:bg-gray-50 dark:hover:bg-zinc-900/50 transition-colors">
                      <td class="px-6 py-4">
                          <div class="font-mono text-gray-900 dark:text-white truncate max-w-62.5 text-xs" :title="volume.name">{{ volume.name }}</div>
                      </td>
                      <td class="px-6 py-4 text-gray-500 dark:text-zinc-400">{{ volume.driver }}</td>
                      <td class="px-6 py-4 text-gray-600 dark:text-zinc-300 tabular-nums">{{ volume.size }} {{ t('volumes.mb') }}</td>
                      <td class="px-6 py-4 text-gray-500 dark:text-zinc-500 text-xs">{{ formatDate(volume.createdAt) }}</td>
                      <td class="px-4 py-4 text-right flex items-center justify-end gap-2">
                         <a v-if="volume.isBrowsing" :href="`/browse/${volume.name}/`" target="_blank"
                            class="p-2 text-blue-500 hover:bg-blue-50 dark:hover:bg-blue-500/10 rounded-lg transition-colors"
                            title="Open Browser">
                            <ExternalLink class="w-4 h-4" />
                         </a>
                         <button v-if="volume.isBrowsing" @click="stopBrowsing(volume.name)"
                            :disabled="actionLoading[volume.name]"
                            class="p-2 text-red-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-500/10 rounded-lg transition-colors"
                            title="Stop Browser">
                            <Loader2 v-if="actionLoading[volume.name]" class="w-4 h-4 animate-spin" />
                            <EyeOff v-else class="w-4 h-4" />
                         </button>
                         <button v-else @click="startBrowsing(volume.name)" 
                            :disabled="actionLoading[volume.name]"
                            class="p-2 text-gray-400 hover:text-blue-500 hover:bg-blue-50 dark:hover:bg-blue-500/10 rounded-lg transition-colors"
                            title="Browse">
                             <Loader2 v-if="actionLoading[volume.name]" class="w-4 h-4 animate-spin text-blue-500" />
                             <Eye v-else class="w-4 h-4" />
                         </button>
                         <button @click="deleteVolume(volume.name)" 
                            class="p-2 text-gray-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-500/10 rounded-lg transition-colors opacity-0 group-hover:opacity-100"
                            title="Delete">
                            <Trash2 class="w-4 h-4" />
                         </button>
                      </td>
                   </tr>
                </tbody>
             </table>
             </div>
          </div>
        </transition>
      </div>
    </main>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(4px);
}
</style>