<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Cpu, MemoryStick, Server, ShieldCheck, Zap } from 'lucide-vue-next'
import { formatBytes } from '../../utils/metrics'

const { t } = useI18n()
const props = defineProps({
  apiUrl: { type: String, required: true }
})

const systemInfo = ref(null)
const loading = ref(true)
const error = ref(null)
let refreshInterval = null

const cpuInfo = computed(() => {
  if (!systemInfo.value) return { cores: 0 }
  return {
    cores: systemInfo.value.cpu.cores
  }
})

const memoryInfo = computed(() => {
  if (!systemInfo.value) return { total: 0, totalFormatted: '0 B' }
  const total = systemInfo.value.memory.total
  return {
    total,
    totalFormatted: formatBytes(total)
  }
})

const storageInfo = computed(() => {
  if (!systemInfo.value?.storage) return { used: 0, total: 0, percent: 0, usedFormatted: '0 B', totalFormatted: '0 B', hasData: false }
  
  const { used, total } = systemInfo.value.storage
  
  if (used && used > 0) {
    if (total && total > 0) {
      const percent = Math.round((used / total) * 100)
      return { used, total, percent, usedFormatted: formatBytes(used), totalFormatted: formatBytes(total), hasData: true }
    } else {
      return { used, total: 0, percent: 0, usedFormatted: formatBytes(used), totalFormatted: null, hasData: true }
    }
  }
  
  return { used: 0, total: 0, percent: 0, usedFormatted: '0 B', totalFormatted: '0 B', hasData: false }
})

const osInfo = computed(() => {
  if (!systemInfo.value?.os) return null
  return {
    name: systemInfo.value.os.name.replace('Debian GNU/Linux', 'Debian').replace('Ubuntu', 'Ubuntu'),
    type: systemInfo.value.os.type,
    arch: systemInfo.value.os.arch || systemInfo.value.os.architecture,
    kernel: systemInfo.value.os.kernel
  }
})

async function fetchSystemInfo() {
  try {
    const response = await fetch(`${props.apiUrl}/api/system/info`)
    const data = await response.json()
    
    if (data.success) {
      systemInfo.value = data.info
      error.value = null
    } else {
      error.value = data.error || 'Failed to fetch system info'
    }
  } catch (err) {
    error.value = err.message
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchSystemInfo()
  refreshInterval = setInterval(fetchSystemInfo, 30000)
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
})
</script>

<template>
  <div class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl overflow-hidden transition-all duration-400 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:border-gray-300 dark:hover:border-zinc-600">
    
    <!-- Hover Accents -->
    <div class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-blue-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
    <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none [mask-image:linear-gradient(to_bottom,white,transparent)]"></div>

    <!-- Loading State -->
    <div v-if="loading" class="relative z-10 p-6 flex-1 flex flex-col items-center justify-center">
      <div class="w-1.5 h-1.5 bg-blue-500 rounded-full animate-ping mb-3"></div>
      <span class="text-[10px] font-semibold uppercase tracking-widest text-gray-400">{{ t('quickMetrics.hostMetrics.scanningHost') }}</span>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="relative z-10 p-6 flex flex-col items-center justify-center h-full text-center">
        <div class="border border-red-200 dark:border-red-900/50 p-4 rounded-lg bg-red-50 dark:bg-red-500/10 text-red-600 dark:text-red-400 w-full">
            <span class="block text-[10px] font-bold mb-1 uppercase tracking-widest">{{ t('quickMetrics.hostMetrics.connectionFailed') }}</span>
            <span class="text-xs opacity-80 break-words line-clamp-2">{{ error }}</span>
        </div>
    </div>

    <!-- Content -->
    <div v-else class="relative z-10 p-6 flex flex-col h-full gap-5">
      
      <!-- Header -->
      <div class="flex items-start justify-between">
         <div class="min-w-0 pr-2">
            <div class="flex items-center gap-2 mb-1">
               <Server class="w-3.5 h-3.5 text-blue-500" />
               <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-400 dark:text-zinc-500">{{ t('quickMetrics.hostMetrics.hostSystem') }}</span>
            </div>
            <div class="text-base font-semibold text-gray-900 dark:text-white truncate tracking-tight group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors" :title="osInfo.name">
               {{ osInfo.name }}
            </div>
         </div>
         <div class="text-right shrink-0">
             <div class="flex items-center justify-end gap-1.5 mb-1.5">
                  <div class="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse"></div>
                  <span class="text-[10px] text-green-600 dark:text-green-500 font-bold uppercase tracking-wider">{{ t('quickMetrics.hostMetrics.online') }}</span>
             </div>
             <div class="text-[10px] text-gray-400 dark:text-zinc-500 font-mono">
               {{ osInfo.arch }}
             </div>
         </div>
      </div>

      <!-- Specs Grid -->
      <div class="grid grid-cols-2 gap-4 flex-1">
         
         <!-- CPU section -->
         <div class="flex flex-col p-3 rounded-lg bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50 group-hover:border-gray-200 dark:group-hover:border-zinc-700/50 transition-colors">
             <div class="flex items-center gap-1.5 mb-2 text-gray-500 dark:text-zinc-400">
                <Cpu class="w-3 h-3" />
                <span class="text-[9px] font-bold uppercase tracking-widest">{{ t('quickMetrics.hostMetrics.processors') }}</span>
             </div>
             <div class="flex items-baseline gap-1 mt-auto">
                 <span class="text-2xl font-bold text-gray-900 dark:text-white tabular-nums tracking-tighter">{{ cpuInfo.cores }}</span>
                 <span class="text-[10px] text-gray-400 dark:text-zinc-500 font-medium">{{ t('quickMetrics.hostMetrics.cores') }}</span>
             </div>
         </div>

         <!-- RAM section -->
         <div class="flex flex-col p-3 rounded-lg bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50 group-hover:border-gray-200 dark:group-hover:border-zinc-700/50 transition-colors">
            <div class="flex items-center gap-1.5 mb-2 text-gray-500 dark:text-zinc-400">
                <MemoryStick class="w-3 h-3" />
                <span class="text-[9px] font-bold uppercase tracking-widest">{{ t('quickMetrics.hostMetrics.memory') }}</span>
             </div>
             <div class="flex items-baseline gap-1 mt-auto">
                 <span class="text-xl font-bold text-gray-900 dark:text-white tabular-nums tracking-tighter">{{ memoryInfo.totalFormatted.split(' ')[0] }}</span>
                 <span class="text-[10px] text-gray-400 dark:text-zinc-500 font-medium">{{ memoryInfo.totalFormatted.split(' ')[1] }}</span>
             </div>
         </div>
      </div>

      <!-- Footer / Storage -->
      <div class="flex items-center justify-between pt-4 border-t border-gray-100 dark:border-zinc-800/80">
         <div class="flex items-center gap-1.5 text-gray-500 dark:text-zinc-500">
            <ShieldCheck class="w-3.5 h-3.5" />
            <span class="text-[10px] font-bold uppercase tracking-widest">{{ t('quickMetrics.hostMetrics.kernel', { kernel: osInfo.kernel }) }}</span>
         </div>
         
         <div v-if="storageInfo.hasData" class="flex items-center gap-2">
            <span class="text-[10px] font-medium text-gray-400 dark:text-zinc-500">{{ t('quickMetrics.hostMetrics.dockerVol') }}</span>
            <span class="text-xs font-bold text-gray-700 dark:text-gray-300 tabular-nums">{{ storageInfo.usedFormatted }}</span>
         </div>
      </div>
    </div>
  </div>
</template>
