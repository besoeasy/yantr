<script setup>
import { ref, computed, onMounted } from 'vue'
import { Trash2, HardDrive, AlertTriangle, RefreshCw, CheckCircle2, Package, Database } from 'lucide-vue-next'
import { useToast } from 'vue-toastification'

const toast = useToast()

// Props
const props = defineProps({
  apiUrl: {
    type: String,
    required: true
  }
})

// State
const loading = ref(true)
const cleaning = ref(false)
const cleaned = ref(false)
const error = ref(null)

const imageStats = ref({ unusedCount: 0, unusedSize: 0, totalSize: 0 })
const volumeStats = ref({ unusedCount: 0, unusedSize: 0, totalSize: 0 })
const lastCleanResult = ref(null)

// Computed
const totalReclaimableBytes = computed(() => {
  return imageStats.value.unusedSize + volumeStats.value.unusedSize
})

const totalReclaimableFormatted = computed(() => {
  return formatBytes(totalReclaimableBytes.value)
})

const hasReclaimable = computed(() => {
  return totalReclaimableBytes.value > 0
})

// Methods
function formatBytes(bytes) {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

async function fetchStats() {
  loading.value = true
  cleaned.value = false
  error.value = null
  
  try {
    // parallel fetch
    const [imagesRes, volumesRes] = await Promise.all([
      fetch(`${props.apiUrl}/api/images`),
      fetch(`${props.apiUrl}/api/volumes`)
    ])

    const imagesData = await imagesRes.json()
    const volumesData = await volumesRes.json()

    if (imagesData.success) {
      imageStats.value = {
        unusedCount: imagesData.unused,
        unusedSize: parseFloat(imagesData.unusedSize) * 1024 * 1024, // API returns MB
        totalSize: parseFloat(imagesData.totalSize) * 1024 * 1024
      }
    }

    if (volumesData.success) {
      volumeStats.value = {
        unusedCount: volumesData.unused,
        unusedSize: parseFloat(volumesData.unusedSize) * 1024 * 1024,
        totalSize: parseFloat(volumesData.totalSize) * 1024 * 1024
      }
    }

  } catch (err) {
    console.error('Failed to fetch system stats:', err)
    error.value = 'Failed to load system stats'
  } finally {
    loading.value = false
  }
}

async function cleanSystem() {
  if (!confirm(`Are you sure you want to reclaim ${totalReclaimableFormatted.value}?\n\nThis will permanently delete:\n- ${imageStats.value.unusedCount} unused images\n- ${volumeStats.value.unusedCount} unused volumes`)) {
    return
  }

  cleaning.value = true
  error.value = null

  try {
    const response = await fetch(`${props.apiUrl}/api/system/prune`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        images: true,
        volumes: true
      })
    })

    const data = await response.json()

    if (data.success) {
      lastCleanResult.value = data.results
      cleaned.value = true
      toast.success(`System cleaned! Reclaimed ${formatBytes(data.results.images.spaceReclaimed + data.results.volumes.spaceReclaimed)}`)
      
      // Refresh stats to show 0
      await fetchStats()
      // Keep cleaned state true so we can show summary
      cleaned.value = true 
    } else {
      throw new Error(data.error || 'Clean failed')
    }
  } catch (err) {
    console.error('Clean failed:', err)
    toast.error(`Clean failed: ${err.message}`)
    error.value = err.message
  } finally {
    cleaning.value = false
  }
}

onMounted(() => {
  fetchStats()
})
</script>

<template>
  <div class="bg-white rounded-2xl shadow-sm border border-gray-100 p-6 transition-all hover:shadow-md h-full flex flex-col">
    <div class="flex items-center justify-between mb-6">
      <div class="flex items-center gap-3">
        <div class="w-10 h-10 rounded-xl bg-orange-50 flex items-center justify-center border border-orange-100">
          <Trash2 :size="20" class="text-orange-600" />
        </div>
        <div>
          <h2 class="text-lg font-bold text-gray-900 leading-tight">System Cleaner</h2>
          <p class="text-xs text-gray-500 font-medium mt-0.5">Reclaim disk space</p>
        </div>
      </div>
      
      <button 
        @click="fetchStats" 
        :disabled="loading || cleaning"
        class="text-gray-400 hover:text-gray-600 p-2 rounded-lg hover:bg-gray-50 transition-colors"
        title="Refresh stats"
      >
        <RefreshCw :size="16" :class="{ 'animate-spin': loading }" />
      </button>
    </div>

    <div v-if="loading && !imageStats.totalSize" class="flex-1 flex flex-col items-center justify-center py-8">
      <RefreshCw :size="24" class="animate-spin text-gray-300 mb-2" />
      <span class="text-xs text-gray-400 font-medium">Analyzing...</span>
    </div>

    <div v-else class="flex-1 flex flex-col">
      <!-- Success State -->
      <div v-if="cleaned && lastCleanResult" class="mb-6 bg-green-50 rounded-xl p-4 border border-green-100 animate-in fade-in slide-in-from-top-2">
        <div class="flex items-start gap-3">
          <CheckCircle2 :size="18" class="text-green-600 mt-0.5" />
          <div>
            <h3 class="text-sm font-bold text-green-800 mb-1">Cleanup Successful!</h3>
            <p class="text-xs text-green-700 leading-relaxed">
              Removed {{ lastCleanResult.images.count }} images and {{ lastCleanResult.volumes.count }} volumes.
            </p>
            <p class="text-xs font-bold text-green-800 mt-1.5">
              Reclaimed: {{ formatBytes(lastCleanResult.images.spaceReclaimed + lastCleanResult.volumes.spaceReclaimed) }}
            </p>
          </div>
        </div>
      </div>

      <!-- Stats Grid -->
      <div class="grid grid-cols-2 gap-3 mb-6">
        <!-- Images -->
        <div class="bg-gray-50 rounded-xl p-3 border border-gray-100">
          <div class="flex items-center gap-2 mb-2">
            <Package :size="14" class="text-gray-500" />
            <span class="text-xs font-semibold text-gray-600 uppercase">Images</span>
          </div>
          <div class="flex items-baseline gap-1">
            <span class="text-lg font-bold text-gray-900">{{ formatBytes(imageStats.unusedSize) }}</span>
          </div>
          <div class="text-[10px] text-gray-500 font-medium">
            {{ imageStats.unusedCount }} unused
          </div>
        </div>

        <!-- Volumes -->
        <div class="bg-gray-50 rounded-xl p-3 border border-gray-100">
          <div class="flex items-center gap-2 mb-2">
            <Database :size="14" class="text-gray-500" />
            <span class="text-xs font-semibold text-gray-600 uppercase">Volumes</span>
          </div>
          <div class="flex items-baseline gap-1">
            <span class="text-lg font-bold text-gray-900">{{ formatBytes(volumeStats.unusedSize) }}</span>
          </div>
          <div class="text-[10px] text-gray-500 font-medium">
            {{ volumeStats.unusedCount }} unused
          </div>
        </div>
      </div>

      <!-- Action Button -->
      <div class="mt-auto">
        <button
          @click="cleanSystem"
          :disabled="!hasReclaimable || cleaning"
          :class="[
            'w-full flex items-center justify-center gap-2 py-3 px-4 rounded-xl font-bold transition-all active:scale-95 shadow-sm',
            hasReclaimable 
              ? 'bg-gradient-to-r from-orange-500 to-red-500 hover:from-orange-600 hover:to-red-600 text-white shadow-orange-200' 
              : 'bg-gray-100 text-gray-400 cursor-not-allowed'
          ]"
        >
          <span v-if="cleaning" class="flex items-center gap-2">
            <RefreshCw :size="18" class="animate-spin" />
            <span>Cleaning...</span>
          </span>
          <span v-else-if="!hasReclaimable">
            System is Clean ✨
          </span>
          <span v-else>
            Free Up {{ totalReclaimableFormatted }}
          </span>
        </button>
      </div>
    </div>
  </div>
</template>
