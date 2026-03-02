<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNotification } from '../composables/useNotification'
import { useApiUrl } from '../composables/useApiUrl'
import { HardDrive, Trash2, Check, AlertTriangle, Box, Database, Layers, Search } from 'lucide-vue-next'

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
            <div class="relative flex-1 sm:flex-initial group">
              <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 dark:text-zinc-500 group-focus-within:text-blue-500 transition-colors" />
              <input 
                v-model="searchQuery" 
                type="text" 
                :placeholder="t('images.searchPlaceholder')" 
                class="w-full sm:w-64 pl-9 pr-4 py-2 bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 rounded-lg text-sm text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-zinc-500 focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500 transition-all"
              />
            </div>
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
        <div class="group relative overflow-hidden bg-white dark:bg-[#0A0A0A] p-5 rounded-xl border border-gray-200 dark:border-zinc-800 flex flex-col justify-between h-32 hover:border-gray-300 dark:hover:border-zinc-600 transition-all duration-300">
          <div class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-blue-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
          <div class="flex justify-between items-start z-10">
            <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-500 dark:text-zinc-500">{{ t('images.totalImages') }}</span>
            <Box class="w-4 h-4 text-gray-400 dark:text-zinc-500 group-hover:text-blue-500 transition-colors" />
          </div>
          <span class="text-4xl font-bold tracking-tighter tabular-nums text-gray-900 dark:text-white z-10">{{ imagesData.total || 0 }}</span>
        </div>

        <div class="group relative overflow-hidden bg-white dark:bg-[#0A0A0A] p-5 rounded-xl border border-gray-200 dark:border-zinc-800 flex flex-col justify-between h-32 hover:border-gray-300 dark:hover:border-zinc-600 transition-all duration-300">
          <div class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-green-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
          <div class="flex justify-between items-start z-10">
            <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-500 dark:text-zinc-500">{{ t('images.inUse') }}</span>
            <Check class="w-4 h-4 text-gray-400 dark:text-zinc-500 group-hover:text-green-500 transition-colors" />
          </div>
          <span class="text-4xl font-bold tracking-tighter tabular-nums text-gray-900 dark:text-white z-10">{{ imagesData.used || 0 }}</span>
        </div>

        <div class="group relative overflow-hidden bg-white dark:bg-[#0A0A0A] p-5 rounded-xl border border-gray-200 dark:border-zinc-800 flex flex-col justify-between h-32 hover:border-gray-300 dark:hover:border-zinc-600 transition-all duration-300">
          <div class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-amber-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
          <div class="flex justify-between items-start z-10">
            <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-500 dark:text-zinc-500">{{ t('images.unused') }}</span>
            <AlertTriangle class="w-4 h-4 text-gray-400 dark:text-zinc-500 group-hover:text-amber-500 transition-colors" />
          </div>
          <span class="text-4xl font-bold tracking-tighter tabular-nums text-gray-900 dark:text-white z-10">{{ imagesData.unused || 0 }}</span>
        </div>

        <div class="group relative overflow-hidden bg-white dark:bg-[#0A0A0A] p-5 rounded-xl border border-gray-200 dark:border-zinc-800 flex flex-col justify-between h-32 hover:border-gray-300 dark:hover:border-zinc-600 transition-all duration-300">
          <div class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-red-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
          <div class="flex justify-between items-start z-10">
            <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-500 dark:text-zinc-500">{{ t('images.reclaimable') }}</span>
            <HardDrive class="w-4 h-4 text-gray-400 dark:text-zinc-500 group-hover:text-red-500 transition-colors" />
          </div>
          <div class="flex items-baseline gap-1 z-10">
            <span class="text-4xl font-bold tracking-tighter tabular-nums text-gray-900 dark:text-white">{{ imagesData.unusedSize || 0 }}</span>
            <span class="text-sm font-semibold text-gray-500 dark:text-zinc-500">{{ t('images.mb') }}</span>
          </div>
        </div>
      </div>

      <!-- Storage Distribution -->
      <div v-if="chartItems.length > 0" class="bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 p-6">
        <div class="flex items-center justify-between mb-5">
          <h3 class="text-sm font-semibold tracking-tight text-gray-900 dark:text-white">{{ t('images.storageDistribution') }}</h3>
          <div class="flex items-center gap-4 text-[10px] font-bold uppercase tracking-[0.15em] text-gray-500 dark:text-zinc-500">
            <span class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-emerald-500 inline-block"></span>{{ t('images.inUse') }}</span>
            <span class="flex items-center gap-1.5"><span class="w-2 h-2 rounded-full bg-amber-500 inline-block"></span>{{ t('images.unused') }}</span>
          </div>
        </div>
        <div class="space-y-2.5">
          <div v-for="item in chartItems" :key="item.name" class="flex items-center gap-3">
            <div class="w-36 shrink-0 text-xs font-mono text-gray-500 dark:text-zinc-400 truncate text-right" :title="item.name">{{ item.name }}</div>
            <div class="flex-1 bg-gray-100 dark:bg-zinc-900 rounded-full h-1.5 overflow-hidden">
              <div class="h-full rounded-full transition-all duration-500" :style="{ width: item.pct + '%', backgroundColor: item.color }"></div>
            </div>
            <div class="w-16 shrink-0 text-xs tabular-nums text-gray-500 dark:text-zinc-500 text-right">{{ item.size }} MB</div>
          </div>
        </div>
      </div>

      <!-- Content Tabs -->
      <div class="space-y-4">
        <div class="flex items-center justify-between border-b border-gray-200 dark:border-zinc-800 pb-1">
          <div class="flex gap-6">
            <button 
              @click="currentTab = 'active'"
              :class="currentTab === 'active' ? 'text-gray-900 dark:text-white border-gray-900 dark:border-white' : 'text-gray-500 dark:text-zinc-500 border-transparent hover:text-gray-700 dark:hover:text-zinc-300'"
              class="pb-3 text-sm font-semibold tracking-tight border-b-2 transition-colors flex items-center gap-2">
              {{ t('images.activeImages') }}
              <span class="bg-gray-100 dark:bg-zinc-800 text-gray-600 dark:text-zinc-400 text-xs px-2 py-0.5 rounded-full font-bold tabular-nums">{{ filteredUsed.length }}</span>
            </button>
            <button 
              @click="currentTab = 'unused'"
              :class="currentTab === 'unused' ? 'text-gray-900 dark:text-white border-gray-900 dark:border-white' : 'text-gray-500 dark:text-zinc-500 border-transparent hover:text-gray-700 dark:hover:text-zinc-300'"
              class="pb-3 text-sm font-semibold tracking-tight border-b-2 transition-colors flex items-center gap-2">
              {{ t('images.unusedImages') }}
              <span class="bg-gray-100 dark:bg-zinc-800 text-gray-600 dark:text-zinc-400 text-xs px-2 py-0.5 rounded-full font-bold tabular-nums">{{ filteredUnused.length }}</span>
            </button>
          </div>
          
          <button v-if="currentTab === 'unused' && filteredUnused.length > 0"
            @click="deleteAllUnusedImages"
            :disabled="deletingAllImages"
            class="text-[10px] font-bold uppercase tracking-[0.2em] text-red-600 hover:text-red-700 dark:text-red-500 dark:hover:text-red-400 transition-colors flex items-center gap-2 px-3 py-1.5 rounded-md border border-red-200 dark:border-red-900/30 bg-red-50 dark:bg-red-900/10">
             <Trash2 class="w-3 h-3" />
             {{ deletingAllImages ? t('images.cleaning') : t('images.pruneAll') }}
          </button>
        </div>

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
