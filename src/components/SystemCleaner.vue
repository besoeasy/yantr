<script setup>
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { 
  Trash2, 
  RefreshCw, 
  CheckCircle2, 
  Package, 
  Database, 
  Sparkles, 
  ArrowRight,
  ShieldCheck,
  Zap
} from 'lucide-vue-next'
import { useNotification } from '../composables/useNotification'
import { formatBytes } from '../utils/metrics'

const { t } = useI18n()
const toast = useNotification()

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
  return (imageStats.value?.unusedSize || 0) + (volumeStats.value?.unusedSize || 0)
})

const totalReclaimableFormatted = computed(() => {
  return formatBytes(totalReclaimableBytes.value)
})

const hasReclaimable = computed(() => {
  return totalReclaimableBytes.value > 0
})

// Watch for prop changes
watch(() => props.initialImageStats, (newStats) => {
  imageStats.value = newStats || { unusedCount: 0, unusedSize: 0, totalSize: 0 }
}, { deep: true })

watch(() => props.initialVolumeStats, (newStats) => {
  volumeStats.value = newStats || { unusedCount: 0, unusedSize: 0, totalSize: 0 }
}, { deep: true })


// Methods
async function cleanSystem() {
  if (!confirm(`${t('systemCleaner.confirmCleanup', { size: totalReclaimableFormatted.value })}\n\n${t('systemCleaner.willDelete')}\n- ${t('systemCleaner.unusedImages')}: ${imageStats.value.unusedCount}\n- ${t('systemCleaner.danglingVolumes')}: ${volumeStats.value.unusedCount}`)) {
    return
  }

  cleaning.value = true
  error.value = null
  cleaned.value = false

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
      
      const totalReclaimed = (data.results.images?.spaceReclaimed || 0) + (data.results.volumes?.spaceReclaimed || 0)
      toast.success(`${t('systemCleaner.systemCleaned')} ${formatBytes(totalReclaimed)}`)
      
      emit('cleaned')
      
      setTimeout(() => {
        cleaned.value = false
      }, 5000)
    } else {
      throw new Error(data.error || t('systemCleaner.cleanFailed'))
    }
  } catch (err) {
    console.error('Clean failed:', err)
    toast.error(`${t('systemCleaner.cleanFailed')}: ${err.message}`)
    error.value = err.message
  } finally {
    cleaning.value = false
  }
}
</script>

<template>
  <div class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl p-6 overflow-hidden transition-all duration-400 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:border-gray-300 dark:hover:border-zinc-600">
    
    <!-- Hover Accents -->
    <div class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-blue-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
    <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none [mask-image:linear-gradient(to_bottom,white,transparent)]"></div>

    <!-- Header Section -->
    <div class="relative z-10 flex items-start justify-between mb-8">
      <div class="flex items-center gap-4">
        <div class="w-10 h-10 rounded-lg bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center shrink-0 group-hover:scale-105 group-hover:border-zinc-700 transition-all duration-500">
           <Trash2 class="w-5 h-5 text-gray-400 dark:text-zinc-500 group-hover:text-blue-500 transition-colors" />
        </div>
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">{{ t('systemCleaner.title') }}</h3>
          <div class="flex items-center gap-2 mt-1">
             <div class="w-1.5 h-1.5 rounded-full" :class="hasReclaimable ? 'bg-amber-500 animate-pulse' : 'bg-green-500'"></div>
             <span class="text-[11px] font-medium text-gray-500 dark:text-zinc-400 uppercase tracking-wider">
               {{ hasReclaimable ? t('systemCleaner.optimizationNeeded') : t('systemCleaner.systemHealthy') }}
             </span>
          </div>
        </div>
      </div>

      <button 
        @click="emit('cleaned')" 
        :disabled="loading || cleaning"
        class="text-gray-400 dark:text-zinc-500 hover:text-blue-600 dark:hover:text-blue-400 transition-colors p-2 rounded-md hover:bg-gray-50 dark:hover:bg-zinc-900/50"
        title="Refresh metrics"
      >
        <RefreshCw :size="16" :class="{ 'animate-spin text-blue-500': loading || cleaning }" />
      </button>
    </div>

    <!-- Main Content -->
    <div class="relative z-10 flex-1 flex flex-col">
      
      <!-- Success State Alert -->
      <transition
        enter-active-class="transition duration-300 ease-out"
        enter-from-class="transform -translate-y-4 opacity-0"
        enter-to-class="transform translate-y-0 opacity-100"
        leave-active-class="transition duration-200 ease-in"
        leave-from-class="transform translate-y-0 opacity-100"
        leave-to-class="transform -translate-y-4 opacity-0"
      >
        <div v-if="cleaned" class="mb-6 flex items-start gap-3 p-3 bg-green-50/50 dark:bg-green-500/10 border border-green-100 dark:border-green-500/20 rounded-lg">
           <CheckCircle2 class="w-4 h-4 text-green-600 dark:text-green-500 mt-0.5 shrink-0" />
           <div>
              <div class="text-[11px] font-bold text-green-800 dark:text-green-400 uppercase tracking-wider mb-0.5">{{ t('systemCleaner.cleanupComplete') }}</div>
              <div class="text-[11px] text-green-700/80 dark:text-green-500/80 font-medium">
                 {{ t('systemCleaner.cleanupCompleteDesc') }}
              </div>
           </div>
        </div>
      </transition>

      <!-- Primary Metric Display -->
      <div class="flex-1 flex flex-col justify-center py-4 mb-6 relative">
         <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-400 dark:text-zinc-500 mb-3">{{ t('systemCleaner.reclaimableStorage') }}</div>
         
         <div class="flex items-baseline gap-2">
           <span class="text-4xl sm:text-5xl font-bold text-gray-900 dark:text-white tracking-tighter tabular-nums" :class="{'opacity-50': !hasReclaimable}">
              {{ hasReclaimable ? totalReclaimableFormatted.split(' ')[0] : '0' }}
           </span>
           <span class="text-xl font-medium text-gray-400 dark:text-zinc-500">
             {{ hasReclaimable ? totalReclaimableFormatted.split(' ')[1] : 'B' }}
           </span>
         </div>
         
         <!-- Progress/Capacity visualization line -->
         <div class="w-full h-1 bg-gray-100 dark:bg-zinc-800/80 rounded-full mt-6 overflow-hidden flex">
            <div v-if="hasReclaimable" class="h-full bg-amber-500/80 w-1/3 transition-all duration-1000"></div>
            <div v-else class="h-full bg-green-500/80 w-full transition-all duration-1000"></div>
         </div>
      </div>

      <!-- Detail Grid -->
      <div class="grid grid-cols-2 gap-4 mb-8">
        <!-- Images Stat -->
        <div class="flex flex-col p-3 rounded-lg bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50 transition-colors group-hover:border-gray-200 dark:group-hover:border-zinc-700/50">
           <div class="flex items-center gap-2 mb-2">
              <Package class="w-3.5 h-3.5 text-gray-400 dark:text-zinc-500" />
              <span class="text-[10px] font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wider">{{ t('systemCleaner.unusedImages') }}</span>
           </div>
           <div class="flex items-baseline gap-1.5">
             <span class="text-lg font-semibold text-gray-700 dark:text-gray-200">{{ formatBytes(imageStats?.unusedSize || 0) }}</span>
           </div>
           <span class="text-[10px] text-gray-400 dark:text-zinc-500 mt-0.5">{{ t('systemCleaner.detectedItems', { count: imageStats?.unusedCount || 0 }) }}</span>
        </div>

        <!-- Volumes Stat -->
        <div class="flex flex-col p-3 rounded-lg bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50 transition-colors group-hover:border-gray-200 dark:group-hover:border-zinc-700/50">
           <div class="flex items-center gap-2 mb-2">
              <Database class="w-3.5 h-3.5 text-gray-400 dark:text-zinc-500" />
              <span class="text-[10px] font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wider">{{ t('systemCleaner.danglingVolumes') }}</span>
           </div>
           <div class="flex items-baseline gap-1.5">
             <span class="text-lg font-semibold text-gray-700 dark:text-gray-200">{{ formatBytes(volumeStats?.unusedSize || 0) }}</span>
           </div>
           <span class="text-[10px] text-gray-400 dark:text-zinc-500 mt-0.5">{{ t('systemCleaner.detectedItems', { count: volumeStats?.unusedCount || 0 }) }}</span>
        </div>
      </div>

      <!-- Action Button -->
      <button
        @click="cleanSystem"
        :disabled="!hasReclaimable || cleaning"
        class="relative w-full overflow-hidden flex items-center justify-center gap-2 py-3.5 px-4 rounded-lg font-semibold text-xs uppercase tracking-widest transition-all duration-300 disabled:opacity-50 disabled:cursor-not-allowed"
        :class="hasReclaimable 
          ? 'bg-gray-900 dark:bg-white text-white dark:text-black hover:bg-gray-800 dark:hover:bg-gray-100 hover:shadow-lg hover:-translate-y-0.5' 
          : 'bg-gray-100 dark:bg-zinc-900 text-gray-400 dark:text-zinc-500 border border-gray-200 dark:border-zinc-800'"
      >
         <span v-if="cleaning" class="flex items-center gap-2 relative z-10">
            <RefreshCw class="w-4 h-4 animate-spin" />
            {{ t('systemCleaner.executingPurge') }}
         </span>
         <span v-else-if="!hasReclaimable" class="flex items-center gap-2 relative z-10">
            <ShieldCheck class="w-4 h-4" />
            {{ t('systemCleaner.systemOptimized') }}
         </span>
         <span v-else class="flex items-center gap-2 relative z-10">
            <Zap class="w-4 h-4" :class="{'text-amber-400': hasReclaimable}" />
            {{ t('systemCleaner.executeCleanup') }}
         </span>
      </button>
    </div>
  </div>
</template>
