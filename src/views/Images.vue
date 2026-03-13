<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNotification } from '../composables/useNotification'
import { useApiUrl } from '../composables/useApiUrl'
import { HardDrive, Trash2, Check, AlertTriangle, Box, Database, Layers } from 'lucide-vue-next'
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
  <div class="min-h-screen bg-white dark:bg-[#0A0A0A] font-sans text-gray-900 dark:text-white pb-20">
    <!-- Header -->
    <header class="bg-white/80 dark:bg-[#0A0A0A]/80 backdrop-blur-md border-b border-gray-200 dark:border-zinc-800 sticky top-0 z-30">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div class="flex items-center gap-3">
            <div class="flex items-center justify-center w-10 h-10 rounded-lg border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900/50">
              <Layers class="w-5 h-5 text-gray-700 dark:text-zinc-300" />
            </div>
            <div>
              <h1 class="text-lg font-semibold tracking-tight text-gray-900 dark:text-white">{{ t('images.title') }}</h1>
              <p class="text-xs font-medium text-gray-500 dark:text-zinc-500">{{ t('images.subtitle') }}</p>
            </div>
          </div>
          
          <div class="flex items-center gap-3">
            <SearchInput v-model="searchQuery" :placeholder="t('images.searchPlaceholder')" />
            <button @click="fetchImages" class="p-2 bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 hover:border-gray-300 dark:hover:border-zinc-600 rounded-lg transition-colors shrink-0 group">
              <Database class="w-4 h-4 text-gray-600 dark:text-zinc-400 group-hover:text-blue-500 transition-colors" :class="{ 'animate-spin text-blue-500': loading }" />
            </button>
          </div>
        </div>
      </div>
    </header>

    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8">
      
      <!-- Stats Overview -->
      <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
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
              class="text-[10px] font-bold uppercase tracking-[0.2em] text-red-600 hover:text-red-700 dark:text-red-500 dark:hover:text-red-400 transition-colors flex items-center gap-2 px-3 py-1.5 rounded-md border border-red-200 dark:border-red-900/30 bg-red-50 dark:bg-red-900/10">
              <Trash2 class="w-3 h-3" />
              {{ deletingAllImages ? t('images.cleaning') : t('images.pruneAll') }}
            </button>
          </template>
        </UnderlineTabBar>

        <!-- Active View -->
        <transition name="fade" mode="out-in">
          <div v-if="currentTab === 'active'" class="bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 overflow-hidden">
             <div class="overflow-x-auto">
             <table class="w-full text-left border-collapse">
                <thead>
                   <tr class="text-[10px] font-bold text-gray-500 dark:text-zinc-500 uppercase tracking-[0.2em] border-b border-gray-200 dark:border-zinc-800 bg-gray-50/50 dark:bg-zinc-900/20">
                      <th class="px-6 py-4">{{ t('images.tag') }}</th>
                      <th class="px-6 py-4 w-32">{{ t('images.shortId') }}</th>
                      <th class="px-6 py-4 w-32">{{ t('images.size') }}</th>
                      <th class="px-6 py-4 w-48">{{ t('images.created') }}</th>
                      <th class="px-4 py-4 w-24">{{ t('images.status') }}</th>
                   </tr>
                </thead>
                <tbody class="divide-y divide-gray-200 dark:divide-zinc-800 text-sm font-medium">
                   <tr v-if="filteredUsed.length === 0">
                      <td colspan="5" class="px-6 py-12 text-center text-gray-500 dark:text-zinc-500 text-sm">{{ t('images.noActiveImages') }}</td>
                   </tr>
                   <tr v-for="image in filteredUsed" :key="image.id" class="group hover:bg-gray-50 dark:hover:bg-zinc-900/50 transition-colors">
                      <td class="px-6 py-4 text-gray-900 dark:text-white">
                         <div class="flex flex-col gap-1">
                            <span v-for="tag in image.tags" :key="tag" class="break-all">{{ tag }}</span>
                         </div>
                      </td>
                      <td class="px-6 py-4 font-mono text-gray-500 dark:text-zinc-400 text-xs">{{ image.shortId }}</td>
                      <td class="px-6 py-4 text-gray-600 dark:text-zinc-300 tabular-nums">{{ image.size }} MB</td>
                      <td class="px-6 py-4 text-gray-500 dark:text-zinc-500 text-xs">{{ image.created }}</td>
                      <td class="px-4 py-4 text-right">
                         <div class="flex items-center justify-end gap-1.5">
                            <div class="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse"></div>
                            <span class="text-[10px] font-bold uppercase tracking-wider text-green-600 dark:text-green-500">{{ t('images.inUseStatus') }}</span>
                         </div>
                      </td>
                   </tr>
                 </tbody>
              </table>
              </div>
           </div>
  
           <!-- Unused View -->
          <div v-else-if="currentTab === 'unused'" class="bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 overflow-hidden">
             <div class="overflow-x-auto">
             <table class="w-full text-left border-collapse">
                <thead>
                   <tr class="text-[10px] font-bold text-gray-500 dark:text-zinc-500 uppercase tracking-[0.2em] border-b border-gray-200 dark:border-zinc-800 bg-gray-50/50 dark:bg-zinc-900/20">
                      <th class="px-6 py-4">{{ t('images.tag') }}</th>
                      <th class="px-6 py-4 w-32">{{ t('images.shortId') }}</th>
                      <th class="px-6 py-4 w-32">{{ t('images.size') }}</th>
                      <th class="px-6 py-4 w-48">{{ t('images.created') }}</th>
                      <th class="px-4 py-4 w-24">{{ t('images.action') }}</th>
                   </tr>
                </thead>
                <tbody class="divide-y divide-gray-200 dark:divide-zinc-800 text-sm font-medium">
                   <tr v-if="filteredUnused.length === 0">
                      <td colspan="5" class="px-6 py-12 text-center text-gray-500 dark:text-zinc-500 text-sm">{{ t('images.noUnusedImages') }}</td>
                   </tr>
                   <tr v-for="image in filteredUnused" :key="image.id" class="group hover:bg-gray-50 dark:hover:bg-zinc-900/50 transition-colors">
                      <td class="px-6 py-4 text-gray-900 dark:text-white">
                         <div class="flex flex-col gap-1">
                            <span v-for="tag in image.tags" :key="tag" class="break-all">{{ tag }}</span>
                         </div>
                      </td>
                      <td class="px-6 py-4 font-mono text-gray-500 dark:text-zinc-400 text-xs">{{ image.shortId }}</td>
                      <td class="px-6 py-4 text-gray-600 dark:text-zinc-300 tabular-nums">{{ image.size }} MB</td>
                      <td class="px-6 py-4 text-gray-500 dark:text-zinc-500 text-xs">{{ image.created }}</td>
                      <td class="px-4 py-4 text-right">
                         <button @click="deleteImage(image.id, image.tags[0])" 
                            class="p-2 text-gray-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-500/10 rounded-lg transition-all opacity-0 group-hover:opacity-100 focus:opacity-100"
                            :title="t('images.deleteImage')">
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
