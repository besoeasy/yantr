<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { usePolling } from '../composables/usePolling'
import { useNotification } from '../composables/useNotification'
import { useYantrAuth } from '../composables/useYantrAuth'
import { useApiUrl } from '../composables/useApiUrl'
import { expectApiSuccess } from '../composables/useApiResponse'
import {
  HardDrive, Trash2, AlertTriangle, AlertCircle, Box,
  Eye, EyeOff, ExternalLink, Loader2, RefreshCw, Download,
} from '@lucide/vue'
import StatCard from '../components/StatCard.vue'
import SizeDistributionChart from '../components/SizeDistributionChart.vue'
import SearchInput from '../components/SearchInput.vue'
import UnderlineTabBar from '../components/UnderlineTabBar.vue'

const { t } = useI18n()
const toast = useNotification()
const { openVolumeBrowser } = useYantrAuth()
const { apiUrl } = useApiUrl()
const route = useRoute()
const router = useRouter()

const COLOR_USED = '#10b981'
const COLOR_UNUSED = '#ef4444'

// One flat tab bar for both resources. These used to be two routes, each with
// its own nested active/unused tabs; merging them removes the nesting so the
// page reads as a single list you filter rather than a list inside a list.
const TABS = ['images-active', 'images-unused', 'volumes-active', 'volumes-unused']

const imagesData = ref({})
const volumesData = ref({})
const loading = ref(false)
const actionLoading = ref({})
const exportingVolume = ref({})
const deletingImage = ref(null)
const deletingVolume = ref(null)
const deletingAllImages = ref(false)
const deletingAllVolumes = ref(false)
const searchQuery = ref('')

const currentTab = ref(TABS.includes(route.query.tab) ? route.query.tab : TABS[0])

// Keep the tab in the URL so widgets can deep-link to the right list, and so a
// reload lands back where the user was.
watch(currentTab, (tab) => {
  if (route.query.tab !== tab) {
    router.replace({ path: route.path, query: { ...route.query, tab } })
  }
})

const isVolumeTab = computed(() => currentTab.value.startsWith('volumes-'))
const isUnusedTab = computed(() => currentTab.value.endsWith('-unused'))

// Filters
function matches(haystack) {
  const q = searchQuery.value.trim().toLowerCase()
  return !q || haystack.toLowerCase().includes(q)
}

const imageText = (img) => [img.shortId, ...(img.tags ?? [])].join(' ')

const filteredUsedImages = computed(() =>
  (imagesData.value.usedImages ?? []).filter((img) => matches(imageText(img)))
)
const filteredUnusedImages = computed(() =>
  (imagesData.value.unusedImages ?? []).filter((img) => matches(imageText(img)))
)
const filteredUsedVolumes = computed(() =>
  (volumesData.value.usedVolumes ?? []).filter((vol) => matches(vol.name))
)
const filteredUnusedVolumes = computed(() =>
  (volumesData.value.unusedVolumes ?? []).filter((vol) => matches(vol.name))
)

const tabs = computed(() => [
  { key: 'images-active', label: t('images.activeImages'), count: filteredUsedImages.value.length },
  { key: 'images-unused', label: t('images.unusedImages'), count: filteredUnusedImages.value.length },
  { key: 'volumes-active', label: t('volumes.activeVolumes'), count: filteredUsedVolumes.value.length },
  { key: 'volumes-unused', label: t('volumes.unusedVolumes'), count: filteredUnusedVolumes.value.length },
])

const browsingVolumes = computed(() =>
  (volumesData.value.volumes ?? []).filter((v) => v.isBrowsing)
)

function imageName(img) {
  const tag = img.tags?.[0]
  return tag && tag !== '<none>:<none>' ? tag.split(':')[0] : img.shortId
}

function shorten(name) {
  return name.length > 20 ? `${name.substring(0, 17)}...` : name
}

// One chart across both resources. Images and volumes are both reported in MB
// by the API, so they can share an axis; the legend still separates in-use
// from reclaimable.
const chartItems = computed(() => {
  const rows = []
  for (const img of imagesData.value.usedImages ?? []) {
    const size = parseFloat(img.size)
    if (size > 1) rows.push({ name: imageName(img), size, color: COLOR_USED })
  }
  for (const img of imagesData.value.unusedImages ?? []) {
    const size = parseFloat(img.size)
    if (size > 1) rows.push({ name: imageName(img), size, color: COLOR_UNUSED })
  }
  for (const vol of volumesData.value.usedVolumes ?? []) {
    const size = parseFloat(vol.size)
    if (size > 0) rows.push({ name: shorten(vol.name), size, color: COLOR_USED })
  }
  for (const vol of volumesData.value.unusedVolumes ?? []) {
    const size = parseFloat(vol.size)
    if (size > 0) rows.push({ name: shorten(vol.name), size, color: COLOR_UNUSED })
  }
  const sorted = rows.sort((a, b) => b.size - a.size).slice(0, 15)
  const max = sorted[0]?.size || 1
  return sorted.map((row) => ({ ...row, pct: Math.round((row.size / max) * 100) }))
})

// Data
async function fetchImages() {
  try {
    const response = await fetch(`${apiUrl.value}/api/images`)
    const data = await response.json()
    if (data.success) imagesData.value = data
  } catch {
    // Intentionally swallowed: a failed refresh leaves the previously loaded
    // list on screen rather than blanking the tables.
  }
}

async function fetchVolumes() {
  try {
    const response = await fetch(`${apiUrl.value}/api/volumes`)
    volumesData.value = await expectApiSuccess(response, t('storage.loadFailed'))
  } catch {
    // Intentionally swallowed, for the same reason as images.
  }
}

async function refresh() {
  loading.value = true
  try {
    await Promise.all([fetchImages(), fetchVolumes()])
  } finally {
    loading.value = false
  }
}

// Images
async function deleteImage(imageId, imageName) {
  if (!confirm(t('images.deleteConfirm', { name: imageName }))) return

  deletingImage.value = imageId
  try {
    const response = await fetch(`${apiUrl.value}/api/images/${imageId}`, { method: 'DELETE' })
    const data = await response.json()
    if (data.success) {
      toast.success(t('images.imageDeleted'))
      await fetchImages()
    } else {
      toast.error(t('images.deletionFailed', { message: data.message }))
    }
  } catch (error) {
    toast.error(t('images.deletionFailed', { message: error.message }))
  } finally {
    deletingImage.value = null
  }
}

async function deleteAllUnusedImages() {
  const unused = imagesData.value.unusedImages ?? []
  if (!unused.length) return
  if (!confirm(t('images.deleteAllConfirm', { count: unused.length }))) return

  deletingAllImages.value = true
  let deleted = 0
  try {
    for (const image of unused) {
      try {
        const response = await fetch(`${apiUrl.value}/api/images/${image.id}`, { method: 'DELETE' })
        const data = await response.json()
        if (data.success) deleted++
      } catch {
        // One image failing to delete must not abort the rest of the sweep; the
        // success toast below reports how many actually went.
      }
    }
    await fetchImages()
    toast.success(t('images.cleanedUp', { count: deleted }))
  } catch (error) {
    toast.error(t('images.cleanupInterrupted', { error: error.message }))
  } finally {
    deletingAllImages.value = false
  }
}

// Volumes
async function startBrowsing(volumeName) {
  actionLoading.value[volumeName] = true
  try {
    const response = await fetch(`${apiUrl.value}/api/volumes/${volumeName}/browse`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ expiryMinutes: 30 }),
    })
    await expectApiSuccess(response, t('volumes.failedToStartBrowser'))
    toast.success(t('volumes.browserStarted'))
    await fetchVolumes()
    openVolumeBrowser(volumeName)
  } catch {
    toast.error(t('volumes.failedToStartBrowser'))
  } finally {
    delete actionLoading.value[volumeName]
  }
}

async function stopBrowsing(volumeName) {
  actionLoading.value[volumeName] = true
  try {
    const response = await fetch(`${apiUrl.value}/api/volumes/${volumeName}/browse`, { method: 'DELETE' })
    await expectApiSuccess(response, t('volumes.failedToStopBrowser'))
    toast.success(t('volumes.browserStopped'))
    await fetchVolumes()
  } catch {
    toast.error(t('volumes.failedToStopBrowser'))
  } finally {
    delete actionLoading.value[volumeName]
  }
}

async function exportVolume(volumeName) {
  exportingVolume.value[volumeName] = true
  toast.info(t('volumes.exporting'))
  try {
    const res = await fetch(`${apiUrl.value}/api/volumes/${encodeURIComponent(volumeName)}/export`)
    if (!res.ok) {
      throw new Error(`HTTP ${res.status}`)
    }
    const blob = await res.blob()
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${volumeName}-backup-${new Date().toISOString().slice(0, 10)}.tar.gz`
    document.body.appendChild(a)
    a.click()
    window.URL.revokeObjectURL(url)
    document.body.removeChild(a)
    toast.success(t('volumes.exportSuccess', { name: volumeName }))
  } catch (error) {
    toast.error(t('volumes.exportFailed', { message: error.message }))
  } finally {
    delete exportingVolume.value[volumeName]
  }
}

async function deleteVolume(volumeName) {
  if (!confirm(t('volumes.deleteVolume', { name: volumeName }))) return

  deletingVolume.value = volumeName
  try {
    const response = await fetch(`${apiUrl.value}/api/volumes/${volumeName}`, { method: 'DELETE' })
    await expectApiSuccess(response, t('volumes.deletionFailed', { message: '' }))
    toast.success(t('volumes.volumeDeleted'))
    await fetchVolumes()
  } catch (error) {
    toast.error(t('volumes.deletionFailed', { message: error.message }))
  } finally {
    deletingVolume.value = null
  }
}

async function deleteAllUnusedVolumes() {
  const unused = volumesData.value.unusedVolumes ?? []
  if (!unused.length) return
  if (!confirm(t('volumes.deleteAllUnused', { count: unused.length }))) return

  deletingAllVolumes.value = true
  let deleted = 0
  try {
    for (const volume of unused) {
      try {
        const response = await fetch(`${apiUrl.value}/api/volumes/${volume.name}`, { method: 'DELETE' })
        await expectApiSuccess(response)
        deleted++
      } catch {
        // One volume failing to delete must not abort the rest of the sweep.
      }
    }
    toast.success(t('volumes.cleanedUp', { count: deleted }))
    await fetchVolumes()
  } catch (error) {
    toast.error(t('volumes.deletionFailed', { message: error.message }))
  } finally {
    deletingAllVolumes.value = false
  }
}

const pruning = computed(() => deletingAllImages.value || deletingAllVolumes.value)

function pruneUnused() {
  return isVolumeTab.value ? deleteAllUnusedVolumes() : deleteAllUnusedImages()
}

function formatDate(dateString) {
  if (!dateString) return t('volumes.notAvailable')
  return new Date(dateString).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

const { start: startPolling } = usePolling(refresh, 10000)

onMounted(startPolling)
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
              <h1 class="text-lg font-bold tracking-tight text-zinc-900 dark:text-white">{{ t('storage.title') }}</h1>
              <p class="text-xs font-semibold text-zinc-500">{{ t('storage.subtitle') }}</p>
            </div>
          </div>

          <div class="flex items-center gap-3">
            <SearchInput v-model="searchQuery" :placeholder="t('storage.searchPlaceholder')" />
            <button @click="refresh" :aria-label="t('storage.refresh')" class="group flex shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-white p-2.5 transition-colors hover:bg-zinc-50 dark:border-zinc-800 dark:bg-[#0A0A0A] dark:hover:bg-zinc-900/50">
              <RefreshCw class="h-4 w-4 text-zinc-600 transition-colors group-hover:text-zinc-900 dark:text-zinc-400 dark:group-hover:text-white" :class="{ 'animate-spin': loading }" />
            </button>
          </div>
        </div>
      </div>
    </header>

    <main class="mx-auto max-w-7xl space-y-8 px-4 py-8 sm:px-6 lg:px-8">

      <!-- Stats Overview -->
      <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <StatCard :label="t('images.totalImages')" :value="imagesData.total || 0" :icon="Box" accent="blue" />
        <StatCard :label="t('images.unused')" :value="imagesData.unused || 0" :icon="AlertTriangle" accent="amber" />
        <StatCard :label="t('volumes.totalVolumes')" :value="volumesData.total || 0" :icon="HardDrive" accent="green" />
        <StatCard :label="t('volumes.unused')" :value="volumesData.unused || 0" :icon="AlertCircle" accent="red" />
      </div>

      <!-- Combined size distribution -->
      <SizeDistributionChart
        :items="chartItems"
        :title="t('images.storageDistribution')"
        :legend="[
          { color: COLOR_USED, label: t('images.inUse') },
          { color: COLOR_UNUSED, label: t('images.unused') }
        ]"
        :unit="t('images.mb')"
      />

      <!-- Active browser sessions, independent of the selected list -->
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
                <button type="button" @click="openVolumeBrowser(volume.name)"
                   class="flex flex-1 items-center justify-center gap-2 rounded-xl border border-zinc-900 bg-zinc-900 px-4 py-2.5 text-xs font-bold uppercase tracking-wider text-white transition-colors hover:bg-black dark:border-white dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-100">
                   <ExternalLink class="h-4 w-4" />
                   {{ t('volumes.openFinder') }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </transition>

      <!-- Lists -->
      <div class="space-y-4">
        <UnderlineTabBar v-model="currentTab" :tabs="tabs">
          <template #action>
            <button v-if="isUnusedTab"
              @click="pruneUnused"
              :disabled="pruning"
              class="flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-1.5 text-[10px] font-bold uppercase tracking-[0.2em] text-red-600 transition-colors hover:bg-red-100 dark:border-red-900/30 dark:bg-red-900/10 dark:text-red-500 dark:hover:bg-red-900/20">
              <Trash2 class="h-3 w-3" />
              {{ pruning ? t('images.cleaning') : t('images.pruneAll') }}
            </button>
          </template>
        </UnderlineTabBar>

        <transition name="fade" mode="out-in">
          <!-- Active images -->
          <div v-if="currentTab === 'images-active'" class="overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-black">
            <div class="overflow-x-auto">
            <table class="w-full border-collapse text-left">
              <thead>
                <tr class="border-b border-zinc-200 bg-zinc-50 text-[10px] font-bold uppercase tracking-[0.2em] text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50">
                  <th class="px-6 py-4">{{ t('images.tag') }}</th>
                  <th class="w-32 px-6 py-4">{{ t('images.shortId') }}</th>
                  <th class="w-32 px-6 py-4">{{ t('images.size') }}</th>
                  <th class="w-48 px-6 py-4">{{ t('images.created') }}</th>
                  <th class="w-24 px-4 py-4">{{ t('images.status') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-zinc-200 text-sm font-medium dark:divide-zinc-800">
                <tr v-if="filteredUsedImages.length === 0">
                  <td colspan="5" class="px-6 py-12 text-center text-sm text-zinc-500">{{ t('images.noActiveImages') }}</td>
                </tr>
                <tr v-for="image in filteredUsedImages" :key="image.id" class="group transition-colors hover:bg-zinc-50 dark:hover:bg-zinc-900/50">
                  <td class="px-6 py-4 text-zinc-900 dark:text-white">
                    <div class="flex flex-col gap-1">
                      <span v-for="tag in image.tags" :key="tag" class="break-all">{{ tag }}</span>
                    </div>
                  </td>
                  <td class="px-6 py-4 font-mono text-xs text-zinc-500 dark:text-zinc-400">{{ image.shortId }}</td>
                  <td class="tabular-nums px-6 py-4 text-zinc-600 dark:text-zinc-300">{{ image.size }} MB</td>
                  <td class="px-6 py-4 text-xs text-zinc-500">{{ image.created }}</td>
                  <td class="px-4 py-4 text-right">
                    <div class="flex items-center justify-end gap-1.5">
                      <div class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500"></div>
                      <span class="text-[10px] font-bold uppercase tracking-wider text-emerald-600 dark:text-emerald-500">{{ t('images.inUseStatus') }}</span>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
            </div>
          </div>

          <!-- Unused images -->
          <div v-else-if="currentTab === 'images-unused'" class="overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-black">
            <div class="overflow-x-auto">
            <table class="w-full border-collapse text-left">
              <thead>
                <tr class="border-b border-zinc-200 bg-zinc-50 text-[10px] font-bold uppercase tracking-[0.2em] text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50">
                  <th class="px-6 py-4">{{ t('images.tag') }}</th>
                  <th class="w-32 px-6 py-4">{{ t('images.shortId') }}</th>
                  <th class="w-32 px-6 py-4">{{ t('images.size') }}</th>
                  <th class="w-48 px-6 py-4">{{ t('images.created') }}</th>
                  <th class="w-24 px-4 py-4">{{ t('images.action') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-zinc-200 text-sm font-medium dark:divide-zinc-800">
                <tr v-if="filteredUnusedImages.length === 0">
                  <td colspan="5" class="px-6 py-12 text-center text-sm text-zinc-500">{{ t('images.noUnusedImages') }}</td>
                </tr>
                <tr v-for="image in filteredUnusedImages" :key="image.id" class="group transition-colors hover:bg-zinc-50 dark:hover:bg-zinc-900/50">
                  <td class="px-6 py-4 text-zinc-900 dark:text-white">
                    <div class="flex flex-col gap-1">
                      <span v-for="tag in image.tags" :key="tag" class="break-all">{{ tag }}</span>
                    </div>
                  </td>
                  <td class="px-6 py-4 font-mono text-xs text-zinc-500 dark:text-zinc-400">{{ image.shortId }}</td>
                  <td class="tabular-nums px-6 py-4 text-zinc-600 dark:text-zinc-300">{{ image.size }} MB</td>
                  <td class="px-6 py-4 text-xs text-zinc-500">{{ image.created }}</td>
                  <td class="px-4 py-4 text-right">
                    <button @click="deleteImage(image.id, image.tags[0])"
                       class="rounded-lg p-2 text-zinc-400 opacity-0 transition-all hover:bg-red-50 hover:text-red-600 focus:opacity-100 group-hover:opacity-100 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                       :title="t('images.deleteImage')">
                      <Trash2 class="h-4 w-4" />
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
            </div>
          </div>

          <!-- Active volumes -->
          <div v-else-if="currentTab === 'volumes-active'" class="overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-black">
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
                <tr v-if="filteredUsedVolumes.length === 0">
                  <td colspan="5" class="px-6 py-12 text-center text-sm text-zinc-500">{{ t('volumes.noActiveVolumes') }}</td>
                </tr>
                <tr v-for="volume in filteredUsedVolumes" :key="volume.name" class="group transition-colors hover:bg-zinc-50 dark:hover:bg-zinc-900/50">
                  <td class="px-6 py-4">
                    <div class="max-w-[250px] truncate font-mono text-xs text-zinc-900 dark:text-white" :title="volume.name">{{ volume.name }}</div>
                  </td>
                  <td class="px-6 py-4 text-zinc-500 dark:text-zinc-400">{{ volume.driver }}</td>
                  <td class="tabular-nums px-6 py-4 text-zinc-600 dark:text-zinc-300">{{ volume.size }} {{ t('volumes.mb') }}</td>
                  <td class="px-6 py-4 text-xs text-zinc-500">{{ formatDate(volume.createdAt) }}</td>
                  <td class="px-4 py-4 text-right">
                    <div class="inline-flex items-center gap-1.5">
                      <button @click="exportVolume(volume.name)"
                             :disabled="exportingVolume[volume.name]"
                             class="inline-flex items-center gap-1.5 rounded-lg border border-zinc-200 bg-zinc-50 px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider text-zinc-700 transition-colors hover:border-zinc-300 hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:border-zinc-700 dark:hover:text-white"
                             :title="t('volumes.export')">
                        <Loader2 v-if="exportingVolume[volume.name]" class="h-3 w-3 animate-spin text-amber-500" />
                        <Download v-else class="h-3 w-3" />
                        {{ t('volumes.export') }}
                      </button>
                      <div v-if="volume.isBrowsing" class="inline-flex items-center gap-1.5">
                        <button type="button" @click="openVolumeBrowser(volume.name)"
                               class="inline-flex items-center gap-1.5 rounded-lg border border-blue-600 bg-blue-600 px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider text-white transition-colors hover:border-blue-700">
                          <ExternalLink class="h-3 w-3" />
                          {{ t('volumes.open') }}
                        </button>
                        <button @click="stopBrowsing(volume.name)"
                               :disabled="actionLoading[volume.name]"
                               class="inline-flex items-center gap-1.5 rounded-lg border border-red-200 bg-red-50 px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider text-red-500 transition-colors hover:border-red-400 hover:text-red-600 dark:border-red-900/40 dark:bg-zinc-900">
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
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
            </div>
          </div>

          <!-- Unused volumes -->
          <div v-else class="overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-black">
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
                <tr v-if="filteredUnusedVolumes.length === 0">
                  <td colspan="5" class="px-6 py-12 text-center text-sm text-zinc-500">{{ t('volumes.noUnusedVolumes') }}</td>
                </tr>
                <tr v-for="volume in filteredUnusedVolumes" :key="volume.name" class="group transition-colors hover:bg-zinc-50 dark:hover:bg-zinc-900/50">
                  <td class="px-6 py-4">
                    <div class="max-w-[250px] truncate font-mono text-xs text-zinc-900 dark:text-white" :title="volume.name">{{ volume.name }}</div>
                  </td>
                  <td class="px-6 py-4 text-zinc-500 dark:text-zinc-400">{{ volume.driver }}</td>
                  <td class="tabular-nums px-6 py-4 text-zinc-600 dark:text-zinc-300">{{ volume.size }} {{ t('volumes.mb') }}</td>
                  <td class="px-6 py-4 text-xs text-zinc-500">{{ formatDate(volume.createdAt) }}</td>
                  <td class="px-4 py-4 text-right">
                    <div class="inline-flex items-center gap-2">
                      <button @click="exportVolume(volume.name)"
                             :disabled="exportingVolume[volume.name]"
                             class="rounded-lg p-2 text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800 dark:hover:text-white"
                             :title="t('volumes.export')">
                        <Loader2 v-if="exportingVolume[volume.name]" class="h-4 w-4 animate-spin text-amber-500" />
                        <Download v-else class="h-4 w-4" />
                      </button>
                      <button v-if="volume.isBrowsing" type="button" @click="openVolumeBrowser(volume.name)"
                             class="rounded-lg p-2 text-blue-500 transition-colors hover:bg-blue-50 dark:hover:bg-blue-500/10"
                             :title="t('volumes.openFinder')"
                             :aria-label="t('volumes.openFinder')">
                        <ExternalLink class="h-4 w-4" />
                      </button>
                      <button v-if="volume.isBrowsing" @click="stopBrowsing(volume.name)"
                             :disabled="actionLoading[volume.name]"
                             class="rounded-lg p-2 text-red-400 transition-colors hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-500/10"
                             :title="t('volumes.stop')">
                        <Loader2 v-if="actionLoading[volume.name]" class="h-4 w-4 animate-spin" />
                        <EyeOff v-else class="h-4 w-4" />
                      </button>
                      <button v-else @click="startBrowsing(volume.name)"
                             :disabled="actionLoading[volume.name]"
                             class="rounded-lg p-2 text-zinc-400 transition-colors hover:bg-blue-50 hover:text-blue-500 dark:hover:bg-blue-500/10"
                             :title="t('volumes.browse')">
                        <Loader2 v-if="actionLoading[volume.name]" class="h-4 w-4 animate-spin text-blue-500" />
                        <Eye v-else class="h-4 w-4" />
                      </button>
                      <button @click="deleteVolume(volume.name)"
                             class="rounded-lg p-2 text-zinc-400 opacity-0 transition-colors hover:bg-red-50 hover:text-red-600 focus:opacity-100 group-hover:opacity-100 dark:hover:bg-red-500/10"
                             :title="t('images.deleteImage')">
                        <Trash2 class="h-4 w-4" />
                      </button>
                    </div>
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
