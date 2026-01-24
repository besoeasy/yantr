<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { Trash2, RefreshCw, CheckCircle2, Package, Database, Sparkles, ArrowRight } from 'lucide-vue-next'
import { useToast } from 'vue-toastification'

const toast = useToast()

// Props
const props = defineProps({
  apiUrl: {
    type: String,
    required: true
  },
  initialImageStats: {
    type: Object,
    default: () => ({ unusedCount: 0, unusedSize: 0, totalSize: 0 })
  },
  initialVolumeStats: {
    type: Object,
    default: () => ({ unusedCount: 0, unusedSize: 0, totalSize: 0 })
  }
})

const emit = defineEmits(['cleaned'])

// State
const loading = ref(false)
const cleaning = ref(false)
const cleaned = ref(false)
const error = ref(null)

const imageStats = ref(props.initialImageStats)
const volumeStats = ref(props.initialVolumeStats)
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

// Watch for prop changes
watch(() => props.initialImageStats, (newStats) => {
  imageStats.value = newStats
}, { deep: true })

watch(() => props.initialVolumeStats, (newStats) => {
  volumeStats.value = newStats
}, { deep: true })


// Methods
function formatBytes(bytes) {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
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
      
      // Emit event so parent can refresh stats
      emit('cleaned')
      
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
</script>

<template>
  <div class="relative h-full overflow-hidden group rounded-2xl transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)] hover:-translate-y-1">
    <!-- Animated Background Mesh -->
    <div class="absolute inset-0 bg-white dark:bg-gray-900">
      <div class="absolute inset-0 bg-gradient-to-br from-blue-200/60 via-cyan-200/30 to-white/80 dark:from-blue-600/20 dark:via-cyan-600/10 dark:to-gray-900 z-10"></div>
      
      <!-- Animated Orbs -->
      <div class="absolute top-0 right-0 w-64 h-64 bg-blue-300/35 dark:bg-blue-500/20 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2 group-hover:bg-blue-400/45 dark:group-hover:bg-blue-500/30 transition-colors duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"></div>
      <div class="absolute bottom-0 left-0 w-48 h-48 bg-cyan-300/30 dark:bg-cyan-600/20 rounded-full blur-3xl translate-y-1/2 -translate-x-1/2 group-hover:bg-cyan-400/40 dark:group-hover:bg-cyan-600/30 transition-colors duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"></div>
    </div>

    <!-- Content Container -->
    <div class="relative z-20 h-full p-6 flex flex-col justify-between border border-slate-200/80 dark:border-slate-700/60 rounded-2xl backdrop-blur-sm group-hover:border-blue-300/70 dark:group-hover:border-blue-500/30 transition-none">
      
      <!-- Header -->
      <div class="flex items-start justify-between mb-6">
        <div class="flex items-center gap-4">
          <!-- Icon Container -->
          <div class="relative">
            <div class="absolute inset-0 bg-blue-400/25 dark:bg-blue-500/20 rounded-xl blur-lg group-hover:blur-xl transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"></div>
            <div class="relative w-12 h-12 bg-gradient-to-br from-blue-500 to-cyan-600 rounded-xl flex items-center justify-center shadow-lg group-hover:scale-110 transition-transform duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]">
              <Trash2 class="w-6 h-6 text-white" />
            </div>
          </div>
          
          <div>
            <h3 class="text-lg font-bold text-slate-900 dark:text-white mb-1 group-hover:text-blue-700 dark:group-hover:text-blue-200 transition-colors">
              System Cleaner
            </h3>
            <p class="text-sm font-medium text-slate-600 dark:text-gray-400 group-hover:text-slate-700 dark:group-hover:text-gray-300 transition-colors">
              Reclaim disk space
            </p>
          </div>
        </div>

        <button 
          @click="emit('cleaned')" 
          :disabled="loading || cleaning"
          class="text-slate-500 dark:text-gray-500 hover:text-slate-900 dark:hover:text-white transition-colors p-2 rounded-lg hover:bg-white/60 dark:hover:bg-white/10"
          title="Refresh stats"
        >
          <RefreshCw :size="16" :class="{ 'animate-spin': loading }" />
        </button>
      </div>

      <!-- Content State -->
      <div v-if="loading && !imageStats.totalSize" class="flex-1 flex flex-col items-center justify-center py-4">
        <RefreshCw :size="24" class="animate-spin text-blue-600 dark:text-blue-400 mb-2" />
        <span class="text-xs text-blue-600 dark:text-blue-300 font-medium">Analyzing...</span>
      </div>

      <div v-else class="flex-1 flex flex-col">
        <!-- Success State -->
        <div v-if="cleaned && lastCleanResult" class="mb-4 bg-emerald-50 dark:bg-green-500/10 rounded-xl p-4 border border-emerald-200 dark:border-green-500/20 animate-in fade-in slide-in-from-top-2">
          <div class="flex items-start gap-3">
            <CheckCircle2 :size="18" class="text-emerald-500 dark:text-green-400 mt-0.5" />
            <div>
              <h3 class="text-sm font-bold text-emerald-600 dark:text-green-400 mb-1">Cleanup Successful!</h3>
              <p class="text-xs text-emerald-600/80 dark:text-green-300/80 leading-relaxed">
                Reclaimed {{ formatBytes(lastCleanResult.images.spaceReclaimed + lastCleanResult.volumes.spaceReclaimed) }}
              </p>
            </div>
          </div>
        </div>

        <!-- Stats Grid -->
        <div class="grid grid-cols-2 gap-3 mb-6">
          <!-- Images -->
          <div class="bg-white/70 dark:bg-white/5 rounded-xl p-3 border border-slate-200/70 dark:border-slate-700/60 group-hover:border-blue-200/70 dark:group-hover:border-slate-600/70 transition-colors">
            <div class="flex items-center gap-2 mb-2">
              <Package :size="14" class="text-blue-500 dark:text-blue-400" />
              <span class="text-xs font-semibold text-slate-500 dark:text-gray-400 uppercase">Images</span>
            </div>
            <div class="flex items-baseline gap-1">
              <span class="text-lg font-bold text-slate-900 dark:text-white">{{ formatBytes(imageStats.unusedSize) }}</span>
            </div>
            <div class="text-[10px] text-slate-500 dark:text-gray-500 font-medium">
              {{ imageStats.unusedCount }} unused
            </div>
          </div>

          <!-- Volumes -->
          <div class="bg-white/70 dark:bg-white/5 rounded-xl p-3 border border-slate-200/70 dark:border-slate-700/60 group-hover:border-blue-200/70 dark:group-hover:border-slate-600/70 transition-colors">
            <div class="flex items-center gap-2 mb-2">
              <Database :size="14" class="text-cyan-500 dark:text-cyan-400" />
              <span class="text-xs font-semibold text-slate-500 dark:text-gray-400 uppercase">Volumes</span>
            </div>
            <div class="flex items-baseline gap-1">
              <span class="text-lg font-bold text-slate-900 dark:text-white">{{ formatBytes(volumeStats.unusedSize) }}</span>
            </div>
            <div class="text-[10px] text-slate-500 dark:text-gray-500 font-medium">
              {{ volumeStats.unusedCount }} unused
            </div>
          </div>
        </div>

        <!-- Action Button -->
        <div class="mt-auto pt-4 border-t border-slate-200/70 dark:border-slate-700/60">
          <button
            @click="cleanSystem"
            :disabled="!hasReclaimable || cleaning"
            class="w-full group/btn relative flex items-center justify-between py-3 px-4 rounded-xl font-bold transition-all disabled:opacity-50 disabled:cursor-not-allowed overflow-hidden"
            :class="hasReclaimable ? 'bg-gradient-to-r from-blue-600 to-cyan-600 hover:from-blue-500 hover:to-cyan-500 text-white shadow-lg shadow-blue-900/20' : 'bg-white/70 text-slate-500 dark:bg-white/5 dark:text-gray-500'"
          >
            <div class="flex items-center gap-2">
              <Sparkles v-if="!cleaning" class="w-4 h-4 text-blue-200" />
              <RefreshCw v-else class="w-4 h-4 animate-spin" />
              <span v-if="cleaning">Cleaning...</span>
              <span v-else-if="!hasReclaimable">System Clean</span>
              <span v-else>Free Up {{ totalReclaimableFormatted }}</span>
            </div>
            
            <ArrowRight v-if="hasReclaimable && !cleaning" class="w-4 h-4 group-hover/btn:translate-x-1 transition-transform" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
