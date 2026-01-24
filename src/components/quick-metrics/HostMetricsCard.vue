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
    type: systemInfo.value.os.type,
    arch: systemInfo.value.os.arch || systemInfo.value.os.architecture, // Fallback for safety
    kernel: systemInfo.value.os.kernel
  }
})

const storageTheme = computed(() => {
  const percent = storageInfo.value.percent
  if (percent >= 90) {
    return {
      text: 'text-rose-600 dark:text-rose-400',
      progress: 'bg-gradient-to-r from-rose-500 to-red-500 dark:from-rose-400 dark:to-red-400'
    }
  } else if (percent >= 75) {
    return {
      text: 'text-amber-600 dark:text-amber-400',
      progress: 'bg-gradient-to-r from-amber-500 to-orange-500 dark:from-amber-400 dark:to-orange-400'
    }
  }
  return {
    text: 'text-emerald-600 dark:text-emerald-400',
    progress: 'bg-gradient-to-r from-emerald-500 to-teal-500 dark:from-emerald-400 dark:to-teal-400'
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
  <div class="group relative bg-gradient-to-br from-white to-slate-50/50 dark:from-slate-900 dark:to-slate-900/80 rounded-3xl shadow-lg border border-slate-200/60 dark:border-slate-800/60 hover:shadow-xl hover:border-indigo-200/50 dark:hover:border-indigo-500/30 transition-all duration-500 overflow-hidden h-full flex flex-col">
    
    <!-- Animated gradient overlay -->
    <div class="absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity duration-500 pointer-events-none">
      <div class="absolute inset-0 bg-gradient-to-br from-indigo-50/40 via-transparent to-blue-50/40 dark:from-indigo-900/10 dark:via-transparent dark:to-blue-900/10"></div>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="p-6 space-y-5">
      <div class="flex justify-between items-center">
         <div class="h-5 w-32 bg-slate-200 dark:bg-slate-800 rounded-lg animate-pulse"></div>
         <div class="h-5 w-20 bg-slate-200 dark:bg-slate-800 rounded-lg animate-pulse"></div>
      </div>
      <div class="grid grid-cols-2 gap-4 pt-2">
        <div class="h-20 bg-gradient-to-br from-slate-100 to-slate-50 dark:from-slate-800/50 dark:to-slate-800/30 rounded-2xl animate-pulse"></div>
        <div class="h-20 bg-gradient-to-br from-slate-100 to-slate-50 dark:from-slate-800/50 dark:to-slate-800/30 rounded-2xl animate-pulse"></div>
      </div>
      <div class="h-16 bg-gradient-to-br from-slate-100 to-slate-50 dark:from-slate-800/50 dark:to-slate-800/30 rounded-2xl animate-pulse mt-2"></div>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="p-8 flex items-center justify-center h-full">
      <div class="flex flex-col items-center gap-3 text-rose-500 dark:text-rose-400 text-center">
        <div class="p-3 rounded-2xl bg-rose-50 dark:bg-rose-900/20 border border-rose-100 dark:border-rose-900/30">
          <Server :size="28" class="opacity-80" />
        </div>
        <span class="text-sm font-semibold">System Info Unavailable</span>
      </div>
    </div>

    <!-- Content -->
    <div v-else class="flex flex-col h-full relative z-10">
      <!-- Decorative gradient blobs -->
      <div class="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-indigo-100/30 to-blue-100/30 dark:from-indigo-900/10 dark:to-blue-900/10 rounded-full blur-3xl pointer-events-none opacity-50 group-hover:opacity-70 transition-opacity"></div>
      <div class="absolute bottom-0 left-0 w-32 h-32 bg-gradient-to-tr from-violet-100/30 to-purple-100/30 dark:from-violet-900/10 dark:to-purple-900/10 rounded-full blur-3xl pointer-events-none opacity-50 group-hover:opacity-70 transition-opacity"></div>

      <!-- Header with OS Info -->
      <div class="px-6 pt-6 pb-4 relative z-10">
        <div class="flex items-start justify-between gap-4 mb-4">
          <div class="flex items-center gap-2.5">
            <div class="p-2 rounded-xl bg-gradient-to-br from-indigo-500 to-indigo-600 dark:from-indigo-500 dark:to-indigo-700 shadow-lg shadow-indigo-500/20 dark:shadow-indigo-500/10 group-hover:scale-110 transition-transform duration-300">
              <Server class="w-4 h-4 text-white" />
            </div>
            <div>
              <h3 class="text-xs font-extrabold uppercase tracking-widest text-indigo-600 dark:text-indigo-400">Host System</h3>
              <p class="text-[10px] text-slate-500 dark:text-slate-400 font-medium mt-0.5">Live Metrics</p>
            </div>
          </div>
        </div>

        <!-- OS Information Panel -->
        <div v-if="osInfo" class="bg-white/60 dark:bg-slate-800/60 backdrop-blur-sm rounded-2xl p-3.5 border border-slate-200/60 dark:border-slate-700/60 shadow-sm">
          <div class="flex items-center justify-between gap-3 mb-2.5">
            <div class="flex-1 min-w-0">
              <div class="text-xs font-semibold text-slate-500 dark:text-slate-400 mb-1">Operating System</div>
              <div class="text-base font-black text-slate-900 dark:text-white truncate" :title="osInfo.name">
                {{ osInfo.name }}
              </div>
            </div>
            <div v-if="osInfo.arch && osInfo.arch !== 'unknown'" class="px-3 py-1.5 rounded-xl bg-gradient-to-r from-indigo-500 to-blue-500 text-white text-xs font-black uppercase shadow-md shadow-indigo-500/20 whitespace-nowrap" :title="`Architecture: ${osInfo.arch}`">
              {{ osInfo.arch }}
            </div>
          </div>
          
          <div class="flex items-center gap-2 flex-wrap">
            <div v-if="osInfo.type && osInfo.type !== 'unknown'" class="text-[10px] font-bold px-2.5 py-1 rounded-lg bg-slate-100 dark:bg-slate-700/50 text-slate-700 dark:text-slate-300 border border-slate-200 dark:border-slate-600 uppercase" :title="`Type: ${osInfo.type}`">
              {{ osInfo.type }}
            </div>
            <div v-if="osInfo.kernel && osInfo.kernel !== 'unknown'" class="text-[10px] font-bold px-2.5 py-1 rounded-lg bg-emerald-50 dark:bg-emerald-900/20 text-emerald-700 dark:text-emerald-400 border border-emerald-200 dark:border-emerald-800 tabular-nums" :title="`Kernel: ${osInfo.kernel}`">
              <span class="opacity-60">v</span>{{ osInfo.kernel }}
            </div>
          </div>
        </div>
      </div>

      <!-- Metrics Grid -->
      <div class="flex-1 px-6 pb-6 flex flex-col gap-3 relative z-10">
        
        <!-- CPU & Memory Row -->
        <div class="grid grid-cols-2 gap-3">
          <!-- CPU -->
          <div class="relative group/cpu bg-gradient-to-br from-blue-50 to-indigo-50/50 dark:from-blue-900/10 dark:to-indigo-900/10 rounded-2xl p-4 border border-blue-100 dark:border-blue-900/30 hover:border-blue-300 dark:hover:border-blue-700/50 transition-all duration-300 hover:shadow-lg hover:shadow-blue-500/10 overflow-hidden">
             <div class="absolute top-0 right-0 p-2 opacity-10 group-hover/cpu:opacity-20 transition-opacity">
                <Cpu class="w-16 h-16 text-blue-500 transform rotate-12" />
             </div>
             <div class="relative z-10">
               <div class="flex items-center gap-2 mb-2 text-blue-600 dark:text-blue-400">
                  <Cpu class="w-4 h-4" />
                  <span class="text-[10px] font-extrabold uppercase tracking-wide">Processor</span>
               </div>
               <div class="text-3xl font-black text-slate-900 dark:text-white tabular-nums leading-none">
                  {{ cpuInfo.cores }}
               </div>
               <div class="text-xs font-bold text-blue-600 dark:text-blue-400 mt-1">vCPU Cores</div>
             </div>
          </div>

          <!-- Memory -->
          <div class="relative group/mem bg-gradient-to-br from-violet-50 to-purple-50/50 dark:from-violet-900/10 dark:to-purple-900/10 rounded-2xl p-4 border border-violet-100 dark:border-violet-900/30 hover:border-violet-300 dark:hover:border-violet-700/50 transition-all duration-300 hover:shadow-lg hover:shadow-violet-500/10 overflow-hidden">
             <div class="absolute top-0 right-0 p-2 opacity-10 group-hover/mem:opacity-20 transition-opacity">
                <MemoryStick class="w-16 h-16 text-violet-500 transform -rotate-12" />
             </div>
             <div class="relative z-10">
               <div class="flex items-center gap-2 mb-2 text-violet-600 dark:text-violet-400">
                  <MemoryStick class="w-4 h-4" />
                  <span class="text-[10px] font-extrabold uppercase tracking-wide">Memory</span>
               </div>
               <div class="text-3xl font-black text-slate-900 dark:text-white tabular-nums leading-none truncate">
                  {{ memoryInfo.totalFormatted.split(' ')[0] }}
               </div>
               <div class="text-xs font-bold text-violet-600 dark:text-violet-400 mt-1">{{ memoryInfo.totalFormatted.split(' ')[1] }} Total RAM</div>
             </div>
          </div>
        </div>

        <!-- Storage -->
        <div v-if="storageInfo.hasData" class="relative bg-gradient-to-br from-emerald-50 to-teal-50/50 dark:from-emerald-900/10 dark:to-teal-900/10 rounded-2xl p-4 border border-emerald-100 dark:border-emerald-900/30 hover:border-emerald-300 dark:hover:border-emerald-700/50 transition-all duration-300 hover:shadow-lg hover:shadow-emerald-500/10 overflow-hidden group/storage">
          
          <div class="absolute top-0 right-0 p-2 opacity-10 group-hover/storage:opacity-20 transition-opacity">
            <HardDrive class="w-16 h-16 text-emerald-500 transform rotate-12" />
          </div>

          <div class="relative z-10">
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-2 text-emerald-600 dark:text-emerald-400">
                <HardDrive class="w-4 h-4" />
                <span class="text-[10px] font-extrabold uppercase tracking-wide">Docker Storage</span>
              </div>
              <div class="text-lg font-black tabular-nums" v-if="storageInfo.total > 0" :class="storageTheme.text">
                 {{ storageInfo.percent }}<span class="text-xs">%</span>
              </div>
            </div>
            
            <!-- Enhanced Progress Bar -->
            <div v-if="storageInfo.total > 0" class="relative h-2.5 w-full bg-white/60 dark:bg-slate-800/60 rounded-full overflow-hidden mb-3 shadow-inner">
              <div 
                class="h-full rounded-full transition-all duration-1000 ease-out shadow-sm relative overflow-hidden" 
                :class="storageTheme.progress"
                :style="{ width: `${Math.min(storageInfo.percent, 100)}%` }">
                <div class="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent animate-shimmer"></div>
              </div>
            </div>

            <div class="flex justify-between items-center">
               <div class="text-sm font-bold text-slate-700 dark:text-slate-200 tabular-nums">
                  {{ storageInfo.usedFormatted }}
               </div>
               <div v-if="storageInfo.total > 0" class="text-xs font-semibold text-slate-500 dark:text-slate-400 tabular-nums">
                  / {{ storageInfo.totalFormatted }}
               </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
@keyframes shimmer {
  0% {
    transform: translateX(-100%);
  }
  100% {
    transform: translateX(100%);
  }
}

.animate-shimmer {
  animation: shimmer 2s infinite;
}
</style>