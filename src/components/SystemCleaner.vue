<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { Trash2, RefreshCw, CheckCircle2, Package, Database, Sparkles, ArrowRight } from 'lucide-vue-next'
import { useToast } from 'vue-toastification'
import { formatBytes } from '../utils/metrics'

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
  <div class="relative h-full group bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 transition-all duration-300 hover:shadow-xl dark:hover:shadow-slate-900/50 flex flex-col">
    
    <!-- Background Texture -->
    <div class="absolute inset-0 opacity-[0.03] dark:opacity-[0.05] pointer-events-none" 
         style="background-image: radial-gradient(circle at 1rem 1rem, currentColor 1px, transparent 0); background-size: 1rem 1rem;">
    </div>
    
    <div class="absolute inset-0 bg-gradient-to-br from-blue-50/50 via-transparent to-transparent dark:from-blue-900/10 pointer-events-none"></div>

    <!-- Content Container -->
    <div class="relative z-10 flex flex-col h-full p-6">
      
      <!-- Header -->
      <div class="flex items-start justify-between mb-6">
        <div class="flex items-center gap-3">
          <div class="p-2.5 bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400 rounded-lg">
             <Trash2 class="w-5 h-5" />
          </div>
          <div>
            <h3 class="text-sm font-bold text-slate-900 dark:text-white uppercase tracking-wider">System Cleaner</h3>
            <div class="flex items-center gap-1.5 mt-0.5">
               <span class="relative flex h-2 w-2">
                 <span v-if="hasReclaimable" class="animate-ping absolute inline-flex h-full w-full rounded-full bg-amber-400 opacity-75"></span>
                 <span class="relative inline-flex rounded-full h-2 w-2" :class="hasReclaimable ? 'bg-amber-500' : 'bg-emerald-500'"></span>
               </span>
               <span class="text-xs font-medium text-slate-500 dark:text-slate-400">
                 {{ hasReclaimable ? 'Cleanup Available' : 'System Optimized' }}
               </span>
            </div>
          </div>
        </div>

        <button 
          @click="emit('cleaned')" 
          :disabled="loading || cleaning"
          class="text-slate-400 hover:text-blue-600 dark:hover:text-blue-400 transition-colors p-2"
          title="Refresh stats"
        >
          <RefreshCw :size="16" :class="{ 'animate-spin': loading }" />
        </button>
      </div>

      <!-- Content State -->
      <div v-if="loading && !imageStats.totalSize" class="flex-1 flex flex-col items-center justify-center py-4">
        <RefreshCw :size="24" class="animate-spin text-blue-500 mb-3" />
        <span class="text-xs font-bold text-slate-400 uppercase tracking-wider">Scanning...</span>
      </div>

      <div v-else class="flex-1 flex flex-col">
        <!-- Success Banner -->
        <div v-if="cleaned && lastCleanResult" class="mb-6 bg-emerald-50 dark:bg-emerald-900/10 border border-emerald-100 dark:border-emerald-900/20 p-3 flex items-center gap-3 animate-in fade-in slide-in-from-top-2">
           <CheckCircle2 class="w-5 h-5 text-emerald-500" />
           <div>
              <div class="text-xs font-bold text-emerald-700 dark:text-emerald-400 uppercase tracking-wide">Cleaned Successfully</div>
              <div class="text-[10px] text-emerald-600/80 dark:text-emerald-500/80">
                 Recovered {{ formatBytes(lastCleanResult.images.spaceReclaimed + lastCleanResult.volumes.spaceReclaimed) }} of space
              </div>
           </div>
        </div>

        <!-- Main Metric -->
        <div class="flex-1 flex flex-col justify-center text-center py-2 mb-4">
           <div class="text-xs font-bold uppercase tracking-widest text-slate-400 mb-2">Reclaimable Space</div>
           <div class="text-4xl sm:text-5xl font-black text-slate-900 dark:text-white tracking-tight tabular-nums">
              {{ hasReclaimable ? totalReclaimableFormatted : '0 B' }}
           </div>
        </div>

        <!-- Breakdown Grid -->
        <div class="grid grid-cols-2 gap-4 mb-6 border-t border-b border-slate-100 dark:border-slate-800 py-4">
          <!-- Images -->
          <div class="flex flex-col gap-1">
             <div class="flex items-center gap-2 text-slate-500 dark:text-slate-400">
                <Package class="w-3.5 h-3.5" />
                <span class="text-[10px] font-bold uppercase tracking-wider">Images</span>
             </div>
             <div class="font-mono text-lg font-bold text-slate-700 dark:text-slate-300">{{ formatBytes(imageStats.unusedSize) }}</div>
             <div class="text-[10px] text-slate-400">{{ imageStats.unusedCount }} items</div>
          </div>

          <!-- Volumes -->
          <div class="flex flex-col gap-1 border-l border-slate-100 dark:border-slate-800 pl-4">
             <div class="flex items-center gap-2 text-slate-500 dark:text-slate-400">
                <Database class="w-3.5 h-3.5" />
                <span class="text-[10px] font-bold uppercase tracking-wider">Volumes</span>
             </div>
             <div class="font-mono text-lg font-bold text-slate-700 dark:text-slate-300">{{ formatBytes(volumeStats.unusedSize) }}</div>
             <div class="text-[10px] text-slate-400">{{ volumeStats.unusedCount }} items</div>
          </div>
        </div>

        <!-- Action Button -->
        <button
          @click="cleanSystem"
          :disabled="!hasReclaimable || cleaning"
          class="w-full relative group/btn flex items-center justify-center gap-2 py-3 px-4 bg-slate-900 dark:bg-white text-white dark:text-slate-900 font-bold text-sm uppercase tracking-wide disabled:opacity-50 disabled:cursor-not-allowed transition-all hover:bg-slate-800 dark:hover:bg-slate-100"
        >
           <span v-if="cleaning" class="flex items-center gap-2">
              <RefreshCw class="w-4 h-4 animate-spin" />
              Processing...
           </span>
           <span v-else class="flex items-center gap-2">
              <Sparkles class="w-4 h-4 text-blue-400 dark:text-blue-500" />
              {{ hasReclaimable ? 'Start Cleaning' : 'System Optimized' }}
           </span>
        </button>
      </div>
    </div>
  </div>
</template>
