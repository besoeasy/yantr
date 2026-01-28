<script setup>
import { ref, onMounted, computed } from 'vue'
import { useToast } from 'vue-toastification'
import { HardDrive, Trash2, Check, AlertTriangle, Box, Database, Layers, Search, Filter } from 'lucide-vue-next'
import VueApexCharts from 'vue3-apexcharts'

const toast = useToast()

const imagesData = ref({})
const loading = ref(false)
const deletingImage = ref(null)
const deletingAllImages = ref(false)
const apiUrl = ref('')
const searchQuery = ref('')
const currentTab = ref('active') // 'active', 'unused'

// Chart Data Configuration
const treemapOptions = computed(() => ({
  chart: {
    type: 'treemap',
    fontFamily: 'inherit',
    toolbar: { show: false },
    background: 'transparent',
    animations: { enabled: true, speed: 600 }
  },
  colors: ['#3b82f6', '#10b981', '#f59e0b', '#ef4444'],
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
  stroke: { show: true, width: 2, colors: ['#ffffff'] } // White stroke for separation
}))

const treemapSeries = computed(() => {
  if (!imagesData.value.usedImages && !imagesData.value.unusedImages) return []
  
  const allImages = []
  
  // Add used images
  if (imagesData.value.usedImages) {
    imagesData.value.usedImages.forEach(img => {
      const sizeVal = parseFloat(img.size)
      if (sizeVal > 1) {
        const tagName = img.tags?.[0] && img.tags[0] !== '<none>:<none>' ? img.tags[0].split(':')[0] : img.shortId
        allImages.push({
          x: tagName,
          y: sizeVal,
          fillColor: '#10b981'
        })
      }
    })
  }
  
  // Add unused images
  if (imagesData.value.unusedImages) {
    imagesData.value.unusedImages.forEach(img => {
      const sizeVal = parseFloat(img.size)
      if (sizeVal > 1) {
        const tagName = img.tags?.[0] && img.tags[0] !== '<none>:<none>' ? img.tags[0].split(':')[0] : img.shortId
        allImages.push({
          x: tagName,
          y: sizeVal,
          fillColor: '#f59e0b' // Amber/Orange for unused
        })
      }
    })
  }

  const sortedData = allImages.sort((a, b) => b.y - a.y).slice(0, 30) // Top 30
  return [{ data: sortedData }]
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
  if (!confirm(`Delete image ${imageName}?`)) return

  deletingImage.value = imageId
  try {
    const response = await fetch(`${apiUrl.value}/api/images/${imageId}`, { method: 'DELETE' })
    const data = await response.json()

    if (data.success) {
      toast.success(`Image deleted successfully`)
      await fetchImages()
    } else {
      toast.error(`Deletion failed: ${data.message}`)
    }
  } catch (error) {
    toast.error(`Deletion failed: ${error.message}`)
  } finally {
    deletingImage.value = null
  }
}

async function deleteAllUnusedImages() {
  const count = imagesData.value.unusedImages?.length || 0
  if (!count) return
  if (!confirm(`Delete all ${count} unused images?`)) return

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
    toast.success(`Cleaned up ${deleted} images`)
  } catch (error) {
    toast.error(`Cleanup interrupted: ${error.message}`)
  } finally {
    deletingAllImages.value = false
  }
}

onMounted(() => {
  fetchImages()
})
</script>

<template>
  <div class="min-h-screen bg-neutral-50 dark:bg-[#0f1117] font-sans text-slate-800 dark:text-slate-200 pb-20">
    <!-- Header -->
    <header class="bg-white dark:bg-[#181b21] border-b border-gray-200 dark:border-gray-800 sticky top-0 z-30">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 bg-indigo-50 dark:bg-indigo-500/10 rounded-lg flex items-center justify-center text-indigo-600 dark:text-indigo-400">
              <Layers class="w-5 h-5" />
            </div>
            <h1 class="text-lg font-bold text-gray-900 dark:text-white">Images</h1>
          </div>
          
          <div class="flex items-center gap-3">
            <div class="relative flex-1 sm:flex-initial">
              <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input 
                v-model="searchQuery" 
                type="text" 
                placeholder="Search images..." 
                class="w-full sm:w-64 pl-10 pr-4 py-2 bg-gray-100 dark:bg-gray-800 border-none rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 focus:bg-white dark:focus:bg-[#181b21] transition-all"
              />
            </div>
            <button @click="fetchImages" class="p-2 bg-white dark:bg-[#181b21] border border-gray-200 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800 rounded-lg transition-colors shrink-0">
              <Database class="w-4 h-4 text-gray-500" :class="{ 'animate-spin': loading }" />
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
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Total Images</span>
            <Box class="w-5 h-5 text-indigo-500 bg-indigo-50 dark:bg-indigo-500/10 p-1 rounded" />
          </div>
          <span class="text-3xl font-bold text-gray-900 dark:text-white">{{ imagesData.total || 0 }}</span>
        </div>

        <div class="bg-white dark:bg-[#181b21] p-5 rounded-xl border border-gray-100 dark:border-gray-800 shadow-sm flex flex-col justify-between h-32">
          <div class="flex justify-between items-start">
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">In Use</span>
            <Check class="w-5 h-5 text-emerald-500 bg-emerald-50 dark:bg-emerald-500/10 p-1 rounded" />
          </div>
          <span class="text-3xl font-bold text-gray-900 dark:text-white">{{ imagesData.used || 0 }}</span>
        </div>

        <div class="bg-white dark:bg-[#181b21] p-5 rounded-xl border border-gray-100 dark:border-gray-800 shadow-sm flex flex-col justify-between h-32">
          <div class="flex justify-between items-start">
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Unused</span>
            <AlertTriangle class="w-5 h-5 text-amber-500 bg-amber-50 dark:bg-amber-500/10 p-1 rounded" />
          </div>
          <span class="text-3xl font-bold text-gray-900 dark:text-white">{{ imagesData.unused || 0 }}</span>
        </div>

        <div class="bg-white dark:bg-[#181b21] p-5 rounded-xl border border-gray-100 dark:border-gray-800 shadow-sm flex flex-col justify-between h-32">
          <div class="flex justify-between items-start">
            <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Reclaimable</span>
            <HardDrive class="w-5 h-5 text-rose-500 bg-rose-50 dark:bg-rose-500/10 p-1 rounded" />
          </div>
          <div class="flex items-baseline gap-1">
            <span class="text-3xl font-bold text-gray-900 dark:text-white">{{ imagesData.unusedSize || 0 }}</span>
            <span class="text-sm text-gray-400">MB</span>
          </div>
        </div>
      </div>

      <!-- Treemap Chart -->
      <div v-if="treemapSeries.length > 0" class="bg-white dark:bg-[#181b21] rounded-xl border border-gray-100 dark:border-gray-800 shadow-sm p-6">
         <h3 class="text-base font-semibold text-gray-900 dark:text-white mb-4">Storage Distribution</h3>
         <div class="rounded-lg overflow-hidden bg-gray-50 dark:bg-gray-900/50 p-1">
            <VueApexCharts :options="treemapOptions" :series="treemapSeries" height="280" />
         </div>
      </div>

      <!-- Content Tabs -->
      <div class="space-y-4">
        <div class="flex items-center justify-between border-b border-gray-200 dark:border-gray-800 pb-1">
          <div class="flex gap-6">
            <button 
              @click="currentTab = 'active'"
              :class="currentTab === 'active' ? 'text-indigo-600 dark:text-indigo-400 border-indigo-600 dark:border-indigo-400' : 'text-gray-500 dark:text-gray-400 border-transparent hover:text-gray-700 dark:hover:text-gray-300'"
              class="pb-3 text-sm font-medium border-b-2 transition-colors flex items-center gap-2">
              <Check class="w-4 h-4" />
              Active Images
              <span class="bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 text-xs px-2 py-0.5 rounded-full">{{ filteredUsed.length }}</span>
            </button>
            <button 
              @click="currentTab = 'unused'"
              :class="currentTab === 'unused' ? 'text-amber-600 dark:text-amber-400 border-amber-600 dark:border-amber-400' : 'text-gray-500 dark:text-gray-400 border-transparent hover:text-gray-700 dark:hover:text-gray-300'"
              class="pb-3 text-sm font-medium border-b-2 transition-colors flex items-center gap-2">
              <AlertTriangle class="w-4 h-4" />
              Unused Images
              <span class="bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 text-xs px-2 py-0.5 rounded-full">{{ filteredUnused.length }}</span>
            </button>
          </div>
          
          <button v-if="currentTab === 'unused' && filteredUnused.length > 0"
            @click="deleteAllUnusedImages"
            :disabled="deletingAllImages"
            class="text-xs font-semibold uppercase text-amber-600 hover:text-amber-700 dark:text-amber-500 dark:hover:text-amber-400 transition-colors flex items-center gap-1">
             <Trash2 class="w-3 h-3" />
             {{ deletingAllImages ? 'Cleaning...' : 'Prune All' }}
          </button>
        </div>

        <!-- Active View -->
        <div v-if="currentTab === 'active'" class="bg-white dark:bg-[#181b21] rounded-xl border border-gray-100 dark:border-gray-800 shadow-sm overflow-hidden">
           <table class="w-full text-left border-collapse">
              <thead>
                 <tr class="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider border-b border-gray-100 dark:border-gray-800">
                    <th class="px-6 py-4 font-medium">Tag</th>
                    <th class="px-6 py-4 font-medium w-32">Short ID</th>
                    <th class="px-6 py-4 font-medium w-32">Size</th>
                    <th class="px-6 py-4 font-medium w-48">Created</th>
                    <th class="px-4 py-4 w-24"></th>
                 </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-gray-800 text-sm">
                 <tr v-if="filteredUsed.length === 0" class="bg-gray-50/50 dark:bg-gray-900/20">
                    <td colspan="5" class="px-6 py-12 text-center text-gray-400">No active images found</td>
                 </tr>
                 <tr v-for="image in filteredUsed" :key="image.id" class="group hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors">
                    <td class="px-6 py-4 text-gray-900 dark:text-white font-medium">
                       <div class="flex flex-col">
                          <span v-for="tag in image.tags" :key="tag">{{ tag }}</span>
                       </div>
                    </td>
                    <td class="px-6 py-4 font-mono text-gray-500 text-xs">{{ image.shortId }}</td>
                    <td class="px-6 py-4 text-gray-600 dark:text-gray-300">{{ image.size }} MB</td>
                    <td class="px-6 py-4 text-gray-500">{{ image.created }}</td>
                    <td class="px-4 py-4 text-right">
                       <span class="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-400 border border-emerald-100 dark:border-emerald-500/20">In Use</span>
                    </td>
                 </tr>
              </tbody>
           </table>
        </div>

        <!-- Unused View -->
        <div v-if="currentTab === 'unused'" class="bg-white dark:bg-[#181b21] rounded-xl border border-gray-100 dark:border-gray-800 shadow-sm overflow-hidden">
           <table class="w-full text-left border-collapse">
              <thead>
                 <tr class="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider border-b border-gray-100 dark:border-gray-800">
                    <th class="px-6 py-4 font-medium">Tag</th>
                    <th class="px-6 py-4 font-medium w-32">Short ID</th>
                    <th class="px-6 py-4 font-medium w-32">Size</th>
                    <th class="px-6 py-4 font-medium w-48">Created</th>
                    <th class="px-4 py-4 w-24"></th>
                 </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-gray-800 text-sm">
                 <tr v-if="filteredUnused.length === 0" class="bg-gray-50/50 dark:bg-gray-900/20">
                    <td colspan="5" class="px-6 py-12 text-center text-gray-400">No unused images found on disk</td>
                 </tr>
                 <tr v-for="image in filteredUnused" :key="image.id" class="group hover:bg-gray-50 dark:hover:bg-gray-800/50 transition-colors">
                    <td class="px-6 py-4 text-gray-900 dark:text-white font-medium">
                       <div class="flex flex-col">
                          <span v-for="tag in image.tags" :key="tag">{{ tag }}</span>
                       </div>
                    </td>
                    <td class="px-6 py-4 font-mono text-gray-500 text-xs">{{ image.shortId }}</td>
                    <td class="px-6 py-4 text-gray-600 dark:text-gray-300">{{ image.size }} MB</td>
                    <td class="px-6 py-4 text-gray-500">{{ image.created }}</td>
                    <td class="px-4 py-4 text-right">
                       <button @click="deleteImage(image.id, image.tags[0])" 
                          class="p-2 text-gray-400 hover:text-rose-500 hover:bg-rose-50 dark:hover:bg-rose-500/10 rounded-lg transition-all opacity-0 group-hover:opacity-100 focus:opacity-100">
                          <Trash2 class="w-4 h-4" />
                       </button>
                    </td>
                 </tr>
              </tbody>
           </table>
        </div>

      </div>
    </main>
  </div>
</template>
