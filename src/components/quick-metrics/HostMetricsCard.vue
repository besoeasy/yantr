<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Cpu, MemoryStick, HardDrive, Server } from 'lucide-vue-next'

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
  
  // Show storage even if we only have 'used' data
  if (used && used > 0) {
    if (total && total > 0) {
      // We have both used and total
      const percent = Math.round((used / total) * 100)
      return {
        used,
        total,
        percent,
        usedFormatted: formatBytes(used),
        totalFormatted: formatBytes(total),
        hasData: true
      }
    } else {
      // We only have used, show it without percentage
      return {
        used,
        total: 0,
        percent: 0,
        usedFormatted: formatBytes(used),
        totalFormatted: null,
        hasData: true
      }
    }
  }
  
  return { used: 0, total: 0, percent: 0, usedFormatted: '0 B', totalFormatted: '0 B', hasData: false }
})

const osInfo = computed(() => {
  if (!systemInfo.value?.os) return null
  return {
    name: systemInfo.value.os.name.replace('Debian GNU/Linux', 'Debian').replace('Ubuntu', 'Ubuntu'),
    type: systemInfo.value.os.type
  }
})

const storageTheme = computed(() => {
  const percent = storageInfo.value.percent
  if (percent >= 90) {
    return {
      text: 'text-rose-600 dark:text-rose-400',
      progress: 'bg-rose-500 dark:bg-rose-400'
    }
  } else if (percent >= 75) {
    return {
      text: 'text-amber-600 dark:text-amber-400',
      progress: 'bg-amber-500 dark:bg-amber-400'
    }
  }
  return {
    text: 'text-emerald-600 dark:text-emerald-400',
    progress: 'bg-emerald-500 dark:bg-emerald-400'
  }
})

function formatBytes(bytes) {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
}

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
  // Refresh every 30 seconds
  refreshInterval = setInterval(fetchSystemInfo, 30000)
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>

<template>
  <div class="group relative bg-white dark:bg-slate-900 rounded-2xl shadow-sm border border-slate-200 dark:border-slate-800 hover:shadow-md transition-all duration-300 overflow-hidden h-full flex flex-col">
    
    <!-- Loading State -->
    <div v-if="loading" class="p-5 space-y-4">
      <div class="flex justify-between">
         <div class="h-4 w-24 bg-slate-200 dark:bg-slate-800 rounded animate-pulse"></div>
         <div class="h-4 w-12 bg-slate-200 dark:bg-slate-800 rounded animate-pulse"></div>
      </div>
      <div class="grid grid-cols-2 gap-4 pt-2">
        <div class="h-16 bg-slate-100 dark:bg-slate-800/50 rounded-xl animate-pulse"></div>
        <div class="h-16 bg-slate-100 dark:bg-slate-800/50 rounded-xl animate-pulse"></div>
      </div>
      <div class="h-12 bg-slate-100 dark:bg-slate-800/50 rounded-xl animate-pulse mt-2"></div>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="p-6 flex items-center justify-center h-full">
      <div class="flex flex-col items-center gap-2 text-rose-500 dark:text-rose-400 text-center">
        <Server :size="24" class="opacity-50" />
        <span class="text-xs font-medium">System Info Unavailable</span>
      </div>
    </div>

    <!-- Content -->
    <div v-else class="flex flex-col h-full relative z-10">
      <!-- Background decoration -->
      <div class="absolute top-0 right-0 p-4 opacity-[0.03] dark:opacity-[0.05] pointer-events-none transition-opacity group-hover:opacity-[0.08]">
        <Server class="w-24 h-24 rotate-12" />
      </div>

      <!-- Header -->
      <div class="px-5 pt-5 pb-2 flex items-center justify-between z-10">
        <div class="flex items-center gap-2">
          <div class="p-1.5 rounded-md bg-white dark:bg-slate-800 shadow-sm border border-slate-100 dark:border-slate-700 text-indigo-500 dark:text-indigo-400">
            <Server class="w-3.5 h-3.5" />
          </div>
          <h3 class="text-xs font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">System Resources</h3>
        </div>
        
        <div v-if="osInfo" class="text-[10px] font-bold px-2 py-0.5 rounded-full bg-slate-100 dark:bg-slate-800 text-slate-500 dark:text-slate-400 border border-slate-200 dark:border-slate-700 truncate max-w-[100px]">
          {{ osInfo.name }}
        </div>
      </div>

      <!-- Metrics Grid -->
      <div class="flex-1 px-5 pb-5 flex flex-col gap-4">
        
        <!-- Top Row: CPU & RAM -->
        <div class="grid grid-cols-2 gap-3">
          <!-- CPU -->
          <div class="bg-slate-50 dark:bg-slate-800/50 rounded-xl p-3 border border-slate-100 dark:border-slate-800 group/cpu transition-colors hover:border-indigo-200 dark:hover:border-indigo-500/30">
             <div class="flex items-center gap-1.5 mb-1.5 text-slate-400 dark:text-slate-500">
                <Cpu class="w-3.5 h-3.5" />
                <span class="text-[10px] font-bold uppercase tracking-wide">CPU Cores</span>
             </div>
             <div class="text-xl sm:text-2xl font-black text-slate-700 dark:text-slate-200 tabular-nums leading-none">
                {{ cpuInfo.cores }} <span class="text-xs font-medium text-slate-400 dark:text-slate-500 ml-0.5">vCPU</span>
             </div>
          </div>

          <!-- Memory -->
          <div class="bg-slate-50 dark:bg-slate-800/50 rounded-xl p-3 border border-slate-100 dark:border-slate-800 group/mem transition-colors hover:border-indigo-200 dark:hover:border-indigo-500/30">
             <div class="flex items-center gap-1.5 mb-1.5 text-slate-400 dark:text-slate-500">
                <MemoryStick class="w-3.5 h-3.5" />
                <span class="text-[10px] font-bold uppercase tracking-wide">Memory</span>
             </div>
             <div class="text-xl sm:text-2xl font-black text-slate-700 dark:text-slate-200 tabular-nums leading-none truncate">
                {{ memoryInfo.totalFormatted.split(' ')[0] }}<span class="text-sm text-slate-400 dark:text-slate-500 ml-0.5">{{ memoryInfo.totalFormatted.split(' ')[1] }}</span>
             </div>
          </div>
        </div>

        <!-- Bottom: Storage -->
        <div v-if="storageInfo.hasData" class="mt-auto bg-slate-50 dark:bg-slate-800/50 rounded-xl p-3 border border-slate-100 dark:border-slate-800 group/storage transition-colors hover:border-indigo-200 dark:hover:border-indigo-500/30 relative overflow-hidden">
          
          <div class="flex items-center justify-between mb-2 relative z-10">
            <div class="flex items-center gap-1.5 text-slate-500 dark:text-slate-400">
              <HardDrive class="w-3.5 h-3.5" />
              <span class="text-[10px] font-bold uppercase tracking-wide">Docker Storage</span>
            </div>
            <div class="text-xs font-semibold tabular-nums" v-if="storageInfo.total > 0" :class="storageTheme.text">
               {{ storageInfo.percent }}%
            </div>
          </div>
          
          <!-- Progress Bar -->
          <div v-if="storageInfo.total > 0" class="h-1.5 w-full bg-slate-200 dark:bg-slate-700 rounded-full overflow-hidden mb-2">
            <div 
              class="h-full rounded-full transition-all duration-1000 ease-out" 
              :class="storageTheme.progress"
              :style="{ width: `${Math.min(storageInfo.percent, 100)}%` }">
            </div>
          </div>

          <div class="flex justify-between items-end relative z-10">
             <div class="text-xs font-medium text-slate-600 dark:text-slate-300 tabular-nums">
                {{ storageInfo.usedFormatted }} <span class="text-slate-400 dark:text-slate-500">used</span>
             </div>
             <div v-if="storageInfo.total > 0" class="text-[10px] font-medium text-slate-400 dark:text-slate-500 tabular-nums text-right">
                of {{ storageInfo.totalFormatted }}
             </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>