<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { HardDrive, Eye, EyeOff, ExternalLink, Loader2, RefreshCw, Trash2, AlertCircle, Box, Check } from '@lucide/vue'
import StatCard from '../components/StatCard.vue'
import SizeDistributionChart from '../components/SizeDistributionChart.vue'
import SearchInput from '../components/SearchInput.vue'
import UnderlineTabBar from '../components/UnderlineTabBar.vue'
import { useNotification } from '../composables/useNotification'
import { useI18n } from 'vue-i18n'
import { useYantrAuth } from '../composables/useYantrAuth'

const toast = useNotification()
const { t } = useI18n()
const { openVolumeBrowser } = useYantrAuth()

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
  } catch {
  } finally {
    loading.value = false
  }
}

async function startBrowsing(volumeName) {
  actionLoading.value[volumeName] = true
  try {
    const response = await fetch(`/api/volumes/${volumeName}/browse`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ expiryMinutes: 0 }),
    })
    const data = await response.json()
    if (data.success) {
      toast.success(t('volumes.browserStarted'))
      await fetchVolumes()
      
      openVolumeBrowser(volumeName)
    }
  } catch {
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
  } catch {
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
  <div class="min-h-screen bg-white pb-20 font-sans text-zinc-900 dark:bg-[#0A0A0A] dark:text-white">
    <!-- Header -->
    <header class="sticky top-0 z-30 border-b border-zinc-200 bg-white/80 backdrop-blur-md dark:border-zinc-800 dark:bg-[#0A0A0A]/80">
      <div class="mx-auto max-w-7xl px-4 py-4 sm:px-6 lg:px-8">
        <div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
          <div class="flex items-center gap-4">
            <div class="flex h-12 w-12 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-900">
              <HardDrive class="h-5 w-5 text-zinc-900 dark:text-zinc-100" />
            </div>
            <div>
              <h1 class="text-lg font-bold tracking-tight text-zinc-900 dark:text-white">{{ t('volumes.title') }}</h1>
              <p class="text-xs font-semibold text-zinc-500">{{ t('volumes.subtitle') }}</p>
            </div>
          </div>
          
          <div class="flex items-center gap-3">
            <SearchInput v-model="searchQuery" :placeholder="t('volumes.searchPlaceholder')" />
            <button @click="fetchVolumes" class="group flex shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-white p-2.5 transition-colors hover:bg-zinc-50 dark:border-zinc-800 dark:bg-[#0A0A0A] dark:hover:bg-zinc-900/50">
              <RefreshCw class="h-4 w-4 text-zinc-600 transition-colors group-hover:text-zinc-900 dark:text-zinc-400 dark:group-hover:text-white" :class="{ 'animate-spin': loading }" />
            </button>
          </div>
        </div>
      </div>
    </header>

    <main class="mx-auto max-w-7xl space-y-8 px-4 py-8 sm:px-6 lg:px-8">
      
      <!-- Stats Overview -->
      <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
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
               <div class="h-2 w-2 animate-pulse rounded-full bg-blue-500"></div>
               <h3 class="text-sm font-bold uppercase tracking-wider">{{ t('volumes.activeSessions') }}</h3>
           </div>
           <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
              <div v-for="volume in browsingVolumes" :key="volume.name" 
                   class="flex flex-col rounded-2xl border border-blue-200 bg-blue-50/50 p-6 dark:border-blue-900/30 dark:bg-blue-900/10">
                  <div class="mb-4 flex items-start justify-between">
                      <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-blue-600 dark:text-blue-400">{{ t('volumes.browsingLabel') }}</span>
                      <button @click="stopBrowsing(volume.name)"
                          :disabled="actionLoading[volume.name]"
                          class="inline-flex items-center gap-1 rounded-md border border-red-200 px-2.5 py-1 text-[10px] font-bold uppercase tracking-wider text-red-500 transition-colors hover:border-red-400 hover:bg-red-50 hover:text-red-600 dark:border-red-900/40 dark:hover:bg-red-500/10">
                          <Loader2 v-if="actionLoading[volume.name]" class="h-3 w-3 animate-spin" />
                          <EyeOff v-else class="h-3 w-3" />
                          {{ t('volumes.stop') }}
                      </button>
                  </div>
                  <h4 class="mb-5 truncate font-mono text-sm font-medium text-zinc-900 dark:text-white" :title="volume.name">{{ volume.name }}</h4>
                  <div class="mt-auto flex gap-2">
                      <a href="#" @click.prevent="openVolumeBrowser(volume.name)"
                         target="_blank"
                         class="flex flex-1 items-center justify-center gap-2 rounded-xl border border-zinc-900 bg-zinc-900 px-4 py-2.5 text-xs font-bold uppercase tracking-wider text-white transition-colors hover:bg-black dark:border-white dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-100">
                         <ExternalLink class="h-4 w-4" />
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
              class="flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-1.5 text-[10px] font-bold uppercase tracking-[0.2em] text-red-600 transition-colors hover:bg-red-100 dark:border-red-900/30 dark:bg-red-900/10 dark:text-red-500 dark:hover:bg-red-900/20">
              <Trash2 class="h-3 w-3" />
              {{ deletingAllVolumes ? t('volumes.cleaning') : t('volumes.pruneAll') }}
            </button>
          </template>
        </UnderlineTabBar>

        <!-- Active Grid -->
        <transition name="fade" mode="out-in">
          <div v-if="currentTab === 'active'" class="overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-black">
             <div class="overflow-x-auto">
             <table class="w-full border-collapse text-left">
                <thead>
                   <tr class="border-b border-zinc-200 bg-zinc-50 text-[10px] font-bold uppercase tracking-[0.2em] text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50">
                      <th class="w-1/3 px-6 py-4">{{ t('volumes.name') }}</th>
                      <th class="px-6 py-4">{{ t('volumes.driver') }}</th>
                      <th class="px-6 py-4">{{ t('volumes.size') }}</th>
                      <th class="px-6 py-4">{{ t('volumes.created') }}</th>
                      <th class="w-32 px-4 py-4 text-right">{{ t('volumes.actions') }}</th>
                   </tr>
                </thead>
                <tbody class="divide-y divide-zinc-200 text-sm font-medium dark:divide-zinc-800">
                   <tr v-if="filteredUsed.length === 0">
                      <td colspan="5" class="px-6 py-12 text-center text-sm text-zinc-500">{{ t('volumes.noActiveVolumes') }}</td>
                   </tr>
                   <tr v-for="volume in filteredUsed" :key="volume.name" class="group transition-colors hover:bg-zinc-50 dark:hover:bg-zinc-900/50">
                      <td class="px-6 py-4">
                          <div class="max-w-[250px] truncate font-mono text-xs text-zinc-900 dark:text-white" :title="volume.name">{{ volume.name }}</div>
                      </td>
                      <td class="px-6 py-4 text-zinc-500 dark:text-zinc-400">{{ volume.driver }}</td>
                      <td class="tabular-nums px-6 py-4 text-zinc-600 dark:text-zinc-300">{{ volume.size }} {{ t('volumes.mb') }}</td>
                      <td class="px-6 py-4 text-xs text-zinc-500">{{ formatDate(volume.createdAt) }}</td>
                      <td class="px-4 py-4 text-right">
                         <div v-if="volume.isBrowsing" class="inline-flex items-center gap-1.5">
                            <a href="#" @click.prevent="openVolumeBrowser(volume.name)" target="_blank"
                               class="inline-flex items-center gap-1.5 rounded-lg border border-blue-600 bg-blue-600 px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider text-white transition-colors hover:bg-blue-700">
                               <ExternalLink class="h-3 w-3" />
                               {{ t('volumes.open') }}
                            </a>
                            <button @click="stopBrowsing(volume.name)"
                               :disabled="actionLoading[volume.name]"
                               class="inline-flex items-center gap-1.5 rounded-lg border border-red-200 bg-zinc-50 px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider text-red-500 transition-colors hover:border-red-400 hover:text-red-600 dark:border-red-900/40 dark:bg-zinc-900">
                               <Loader2 v-if="actionLoading[volume.name]" class="h-3 w-3 animate-spin" />
                               <EyeOff v-else class="h-3 w-3" />
                               {{ t('volumes.stop') }}
                            </button>
                         </div>
                         <button v-else @click="startBrowsing(volume.name)" 
                            :disabled="actionLoading[volume.name]"
                            class="inline-flex items-center gap-1.5 rounded-lg border border-zinc-200 bg-zinc-50 px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider text-zinc-700 transition-colors hover:border-zinc-300 hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:border-zinc-700 dark:hover:text-white">
                            <Loader2 v-if="actionLoading[volume.name]" class="h-3 w-3 animate-spin text-blue-500" />
                            <Eye v-else class="h-3 w-3" />
                            {{ t('volumes.browse') }}
                         </button>
                      </td>
                   </tr>
                 </tbody>
              </table>
              </div>
           </div>
   
           <!-- Unused Grid -->
          <div v-else-if="currentTab === 'unused'" class="overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-black">
             <div class="overflow-x-auto">
             <table class="w-full border-collapse text-left">
                <thead>
                   <tr class="border-b border-zinc-200 bg-zinc-50 text-[10px] font-bold uppercase tracking-[0.2em] text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50">
                      <th class="w-1/3 px-6 py-4">{{ t('volumes.name') }}</th>
                      <th class="px-6 py-4">{{ t('volumes.driver') }}</th>
                      <th class="px-6 py-4">{{ t('volumes.size') }}</th>
                      <th class="px-6 py-4">{{ t('volumes.created') }}</th>
                      <th class="w-32 px-4 py-4 text-right">{{ t('volumes.actions') }}</th>
                   </tr>
                </thead>
                <tbody class="divide-y divide-zinc-200 text-sm font-medium dark:divide-zinc-800">
                   <tr v-if="filteredUnused.length === 0">
                      <td colspan="5" class="px-6 py-12 text-center text-sm text-zinc-500">{{ t('volumes.noUnusedVolumes') }}</td>
                   </tr>
                   <tr v-for="volume in filteredUnused" :key="volume.name" class="group transition-colors hover:bg-zinc-50 dark:hover:bg-zinc-900/50">
                      <td class="px-6 py-4">
                          <div class="max-w-[250px] truncate font-mono text-xs text-zinc-900 dark:text-white" :title="volume.name">{{ volume.name }}</div>
                      </td>
                      <td class="px-6 py-4 text-zinc-500 dark:text-zinc-400">{{ volume.driver }}</td>
                      <td class="tabular-nums px-6 py-4 text-zinc-600 dark:text-zinc-300">{{ volume.size }} {{ t('volumes.mb') }}</td>
                      <td class="px-6 py-4 text-xs text-zinc-500">{{ formatDate(volume.createdAt) }}</td>
                      <td class="flex items-center justify-end gap-2 px-4 py-4 text-right">
                         <a v-if="volume.isBrowsing" href="#" @click.prevent="openVolumeBrowser(volume.name)" target="_blank"
                            class="rounded-lg p-2 text-blue-500 transition-colors hover:bg-blue-50 dark:hover:bg-blue-500/10"
                            title="Open Browser">
                            <ExternalLink class="h-4 w-4" />
                         </a>
                         <button v-if="volume.isBrowsing" @click="stopBrowsing(volume.name)"
                            :disabled="actionLoading[volume.name]"
                            class="rounded-lg p-2 text-red-400 transition-colors hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-500/10"
                            title="Stop Browser">
                            <Loader2 v-if="actionLoading[volume.name]" class="h-4 w-4 animate-spin" />
                            <EyeOff v-else class="h-4 w-4" />
                         </button>
                         <button v-else @click="startBrowsing(volume.name)" 
                            :disabled="actionLoading[volume.name]"
                            class="rounded-lg p-2 text-zinc-400 transition-colors hover:bg-blue-50 hover:text-blue-500 dark:hover:bg-blue-500/10"
                            title="Browse">
                             <Loader2 v-if="actionLoading[volume.name]" class="h-4 w-4 animate-spin text-blue-500" />
                             <Eye v-else class="h-4 w-4" />
                         </button>
                         <button @click="deleteVolume(volume.name)" 
                            class="rounded-lg p-2 text-zinc-400 opacity-0 transition-colors hover:bg-red-50 hover:text-red-600 group-hover:opacity-100 dark:hover:bg-red-500/10"
                            title="Delete">
                            <Trash2 class="h-4 w-4" />
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