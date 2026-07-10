<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNotification } from '../composables/useNotification'
import { useApiUrl } from '../composables/useApiUrl'
import { HardDrive, Trash2, Check, AlertTriangle, Box, Database, Layers } from '@lucide/vue'
import StatCard from '../components/StatCard.vue'
import SizeDistributionChart from '../components/SizeDistributionChart.vue'
import SearchInput from '../components/SearchInput.vue'
import UnderlineTabBar from '../components/UnderlineTabBar.vue'

const { t } = useI18n()
const toast = useNotification()
const { apiUrl } = useApiUrl()

const imagesData = ref({})
const loading = ref(false)
const deletingImage = ref(null)
const deletingAllImages = ref(false)
const searchQuery = ref('')
const currentTab = ref('active') // 'active', 'unused'

// Chart Data
const chartItems = computed(() => {
  const allImages = []

  if (imagesData.value.usedImages) {
    imagesData.value.usedImages.forEach(img => {
      const sizeVal = parseFloat(img.size)
      if (sizeVal > 1) {
        const name = img.tags?.[0] && img.tags[0] !== '<none>:<none>' ? img.tags[0].split(':')[0] : img.shortId
        allImages.push({ name, size: sizeVal, color: '#10b981' })
      }
    })
  }

  if (imagesData.value.unusedImages) {
    imagesData.value.unusedImages.forEach(img => {
      const sizeVal = parseFloat(img.size)
      if (sizeVal > 1) {
        const name = img.tags?.[0] && img.tags[0] !== '<none>:<none>' ? img.tags[0].split(':')[0] : img.shortId
        allImages.push({ name, size: sizeVal, color: '#f59e0b' })
      }
    })
  }

  const sorted = allImages.sort((a, b) => b.size - a.size).slice(0, 15)
  const max = sorted[0]?.size || 1
  return sorted.map(item => ({ ...item, pct: Math.round((item.size / max) * 100) }))
})

// Filtered Lists
const filteredUnused = computed(() => {
  if (!imagesData.value.unusedImages) return []
  let imgs = imagesData.value.unusedImages
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    imgs = imgs.filter(img => 
      img.shortId.toLowerCase().includes(q) || 
      img.tags.some(t => t.toLowerCase().includes(q))
    )
  }
  return imgs
})

const filteredUsed = computed(() => {
  if (!imagesData.value.usedImages) return []
  let imgs = imagesData.value.usedImages
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    imgs = imgs.filter(img => 
      img.shortId.toLowerCase().includes(q) || 
      img.tags.some(t => t.toLowerCase().includes(q))
    )
  }
  return imgs
})

async function fetchImages() {
  loading.value = true
  try {
    const response = await fetch(`${apiUrl.value}/api/images`)
    const data = await response.json()
    if (data.success) {
      imagesData.value = data
    }
  } catch (error) {
    console.error('Failed to fetch images:', error)
  } finally {
    loading.value = false
  }
}

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
  const count = imagesData.value.unusedImages?.length || 0
  if (!count) return
  if (!confirm(t('images.deleteAllConfirm', { count }))) return

  deletingAllImages.value = true
  let deleted = 0
  
  try {
    for (const image of imagesData.value.unusedImages) {
      try {
        const response = await fetch(`${apiUrl.value}/api/images/${image.id}`, { method: 'DELETE' })
        const data = await response.json()
        if (data.success) deleted++
      } catch (error) {}
    }
    await fetchImages()
    toast.success(t('images.cleanedUp', { count: deleted }))
  } catch (error) {
    toast.error(t('images.cleanupInterrupted', { error: error.message }))
  } finally {
    deletingAllImages.value = false
  }
}

onMounted(() => {
  fetchImages()
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
              <Layers class="h-5 w-5 text-zinc-900 dark:text-zinc-100" />
            </div>
            <div>
              <h1 class="text-lg font-bold tracking-tight text-zinc-900 dark:text-white">{{ t('images.title') }}</h1>
              <p class="text-xs font-semibold text-zinc-500">{{ t('images.subtitle') }}</p>
            </div>
          </div>
          
          <div class="flex items-center gap-3">
            <SearchInput v-model="searchQuery" :placeholder="t('images.searchPlaceholder')" />
            <button @click="fetchImages" class="group flex shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-white p-2.5 transition-colors hover:bg-zinc-50 dark:border-zinc-800 dark:bg-[#0A0A0A] dark:hover:bg-zinc-900/50">
              <Database class="h-4 w-4 text-zinc-600 transition-colors group-hover:text-zinc-900 dark:text-zinc-400 dark:group-hover:text-white" :class="{ 'animate-spin': loading }" />
            </button>
          </div>
        </div>
      </div>
    </header>

    <main class="mx-auto max-w-7xl space-y-8 px-4 py-8 sm:px-6 lg:px-8">
      
      <!-- Stats Overview -->
      <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <StatCard :label="t('images.totalImages')" :value="imagesData.total || 0" :icon="Box" accent="blue" />
        <StatCard :label="t('images.inUse')" :value="imagesData.used || 0" :icon="Check" accent="green" />
        <StatCard :label="t('images.unused')" :value="imagesData.unused || 0" :icon="AlertTriangle" accent="amber" />
        <StatCard :label="t('images.reclaimable')" :value="imagesData.unusedSize || 0" :unit="t('images.mb')" :icon="HardDrive" accent="red" />
      </div>

      <!-- Storage Distribution -->
      <SizeDistributionChart
        :items="chartItems"
        :title="t('images.storageDistribution')"
        :legend="[{ color: '#10b981', label: t('images.inUse') }, { color: '#f59e0b', label: t('images.unused') }]"
      />

      <!-- Content Tabs -->
      <div class="space-y-4">
        <UnderlineTabBar
          v-model="currentTab"
          :tabs="[
            { key: 'active', label: t('images.activeImages'), count: filteredUsed.length },
            { key: 'unused', label: t('images.unusedImages'), count: filteredUnused.length }
          ]"
        >
          <template #action>
            <button v-if="currentTab === 'unused' && filteredUnused.length > 0"
              @click="deleteAllUnusedImages"
              :disabled="deletingAllImages"
              class="flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-1.5 text-[10px] font-bold uppercase tracking-[0.2em] text-red-600 transition-colors hover:bg-red-100 dark:border-red-900/30 dark:bg-red-900/10 dark:text-red-500 dark:hover:bg-red-900/20">
              <Trash2 class="h-3 w-3" />
              {{ deletingAllImages ? t('images.cleaning') : t('images.pruneAll') }}
            </button>
          </template>
        </UnderlineTabBar>

        <!-- Active View -->
        <transition name="fade" mode="out-in">
          <div v-if="currentTab === 'active'" class="overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-black">
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
                   <tr v-if="filteredUsed.length === 0">
                      <td colspan="5" class="px-6 py-12 text-center text-sm text-zinc-500">{{ t('images.noActiveImages') }}</td>
                   </tr>
                   <tr v-for="image in filteredUsed" :key="image.id" class="group transition-colors hover:bg-zinc-50 dark:hover:bg-zinc-900/50">
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
  
           <!-- Unused View -->
          <div v-else-if="currentTab === 'unused'" class="overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-black">
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
                   <tr v-if="filteredUnused.length === 0">
                      <td colspan="5" class="px-6 py-12 text-center text-sm text-zinc-500">{{ t('images.noUnusedImages') }}</td>
                   </tr>
                   <tr v-for="image in filteredUnused" :key="image.id" class="group transition-colors hover:bg-zinc-50 dark:hover:bg-zinc-900/50">
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
