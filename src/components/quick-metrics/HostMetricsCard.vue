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
  <div class="group relative h-full overflow-hidden rounded-3xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 transition-all duration-300 hover:shadow-xl dark:hover:shadow-slate-900/50 flex flex-col">
    
    <!-- Animated Background Mesh -->
    <div class="absolute inset-0 overflow-hidden pointer-events-none">
       <div class="absolute top-[-50%] left-[-20%] w-[80%] h-[200%] bg-gradient-to-r from-transparent via-sky-100/20 dark:via-sky-900/10 to-transparent transform -rotate-12 translate-x-[-100%] animate-shine"></div>
       <div class="absolute inset-0 opacity-[0.03] dark:opacity-[0.05]" 
         style="background-image: radial-gradient(circle at 1rem 1rem, currentColor 1px, transparent 0); background-size: 1rem 1rem;">
       </div>
    </div>

    <!-- Decorative Background Icon -->
    <div class="absolute -right-6 -bottom-6 opacity-5 dark:opacity-10 transform rotate-12 group-hover:scale-110 group-hover:rotate-6 transition-transform duration-700 ease-in-out pointer-events-none">
       <Server class="w-48 h-48 text-indigo-900 dark:text-indigo-100" />
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="relative z-10 p-6 flex-1 flex flex-col gap-4">
      <div class="space-y-2">
         <div class="h-8 w-3/4 bg-slate-100 dark:bg-slate-800 rounded-lg animate-pulse"></div>
         <div class="h-4 w-1/2 bg-slate-100 dark:bg-slate-800 rounded animate-pulse"></div>
      </div>
      <div class="flex-1 flex flex-col gap-3">
        <div class="h-24 bg-slate-50 dark:bg-slate-800/50 rounded-2xl animate-pulse"></div>
        <div class="h-24 bg-slate-50 dark:bg-slate-800/50 rounded-2xl animate-pulse"></div>
        <div class="h-24 bg-slate-50 dark:bg-slate-800/50 rounded-2xl animate-pulse"></div>
      </div>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="relative z-10 p-6 flex flex-col items-center justify-center h-full text-center">
      <div class="p-3 rounded-full bg-rose-50 dark:bg-rose-900/20 text-rose-500 mb-3">
        <Server :size="24" />
      </div>
      <span class="text-sm font-semibold text-slate-600 dark:text-slate-300">System Info Unavailable</span>
    </div>

    <!-- Content -->
    <div v-else class="relative z-10 p-6 flex flex-col h-full gap-6">
      
      <!-- Header Section -->
      <div class="flex flex-col gap-2">
        <div class="flex items-center gap-2 mb-1">
           <div class="relative flex h-2.5 w-2.5">
             <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
             <span class="relative inline-flex rounded-full h-2.5 w-2.5 bg-emerald-500"></span>
           </div>
           <span class="text-[10px] font-bold text-emerald-600 dark:text-emerald-400 uppercase tracking-widest">Host Active</span>
        </div>
        
        <div>
          <h2 class="text-2xl font-black text-slate-900 dark:text-white tracking-tight leading-tight truncate drop-shadow-sm" :title="osInfo.name">
            {{ osInfo.name }}
          </h2>
          <div class="flex flex-wrap items-center gap-2 mt-2">
             <span v-if="osInfo.type" class="px-2.5 py-1 rounded-lg bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 text-[10px] text-slate-600 dark:text-slate-300 uppercase tracking-wider font-extrabold shadow-sm">
               {{ osInfo.type }}
             </span>
             <span v-if="osInfo.kernel" class="px-2.5 py-1 rounded-lg bg-indigo-50 dark:bg-indigo-900/20 border border-indigo-100 dark:border-indigo-800 text-[10px] text-indigo-600 dark:text-indigo-300 font-bold tabular-nums shadow-sm">
               v{{ osInfo.kernel }}
             </span>
          </div>
        </div>
      </div>

      <!-- Metrics Section -->
      <div class="grid grid-cols-2 gap-3">
        
        <!-- CPU -->
        <div class="relative overflow-hidden p-3 rounded-2xl bg-slate-50 dark:bg-slate-800/40 border border-slate-100 dark:border-slate-700/60 group/stat hover:bg-white dark:hover:bg-slate-800 transition-colors duration-200 hover:shadow-sm">
           <div class="absolute right-0 top-0 p-2 opacity-10 group-hover/stat:opacity-20 transition-opacity">
              <Cpu class="w-10 h-10 text-blue-500 transform rotate-12" />
           </div>
           <div class="relative z-10 flex flex-col justify-between gap-1.5">
             <div class="flex items-center gap-1.5 text-slate-500 dark:text-slate-400">
               <span class="p-1 rounded-md bg-white dark:bg-slate-900 shadow-sm border border-slate-100 dark:border-slate-700 text-blue-500">
                 <Cpu class="w-3 h-3" />
               </span>
               <span class="text-[10px] font-bold uppercase tracking-wider">CPU</span>
             </div>
             <div>
                <div class="text-xl font-black text-slate-900 dark:text-white tabular-nums tracking-tight">{{ cpuInfo.cores }} <span class="text-xs font-bold text-slate-400">Cores</span></div>
                <div class="h-1 w-8 bg-blue-500 rounded-full mt-1.5 opacity-80"></div>
             </div>
           </div>
        </div>

        <!-- RAM -->
        <div class="relative overflow-hidden p-3 rounded-2xl bg-slate-50 dark:bg-slate-800/40 border border-slate-100 dark:border-slate-700/60 group/stat hover:bg-white dark:hover:bg-slate-800 transition-colors duration-200 hover:shadow-sm">
           <div class="absolute right-0 top-0 p-2 opacity-10 group-hover/stat:opacity-20 transition-opacity">
              <MemoryStick class="w-10 h-10 text-violet-500 transform -rotate-12" />
           </div>
           <div class="relative z-10 flex flex-col justify-between gap-1.5">
             <div class="flex items-center gap-1.5 text-slate-500 dark:text-slate-400">
               <span class="p-1 rounded-md bg-white dark:bg-slate-900 shadow-sm border border-slate-100 dark:border-slate-700 text-violet-500">
                 <MemoryStick class="w-3 h-3" />
               </span>
               <span class="text-[10px] font-bold uppercase tracking-wider">RAM</span>
             </div>
             <div>
                <div class="text-xl font-black text-slate-900 dark:text-white tabular-nums tracking-tight truncate">
                  {{ memoryInfo.totalFormatted.split(' ')[0] }}
                  <span class="text-xs font-bold text-slate-400">{{ memoryInfo.totalFormatted.split(' ')[1] }}</span>
                </div>
                <div class="h-1 w-8 bg-violet-500 rounded-full mt-1.5 opacity-80"></div>
             </div>
           </div>
        </div>

        <!-- Storage -->
        <div v-if="storageInfo.hasData" class="col-span-2 relative overflow-hidden p-3 rounded-2xl bg-slate-50 dark:bg-slate-800/40 border border-slate-100 dark:border-slate-700/60 group/stat hover:bg-white dark:hover:bg-slate-800 transition-colors duration-200 hover:shadow-sm">
           <div class="absolute right-0 top-0 p-2 opacity-10 group-hover/stat:opacity-20 transition-opacity">
              <HardDrive class="w-10 h-10 text-emerald-500 transform rotate-6" />
           </div>
           <div class="relative z-10 flex flex-col justify-between gap-1.5">
             <div class="flex items-center gap-1.5 text-slate-500 dark:text-slate-400">
               <span class="p-1 rounded-md bg-white dark:bg-slate-900 shadow-sm border border-slate-100 dark:border-slate-700 text-emerald-500">
                 <HardDrive class="w-3 h-3" />
               </span>
               <span class="text-[10px] font-bold uppercase tracking-wider">Storage</span>
             </div>
             <div class="flex items-baseline justify-between">
                <div class="text-xl font-black text-slate-900 dark:text-white tabular-nums tracking-tight">
                  {{ storageInfo.usedFormatted }}
                </div>
                <div class="text-[10px] font-bold text-slate-400 tabular-nums">
                   <span class="opacity-50 mr-1">of</span> {{ storageInfo.totalFormatted }}
                </div>
             </div>
           </div>
        </div>

      </div>
    </div>
  </div>
</template>

<style scoped>
@keyframes shine {
  0% { transform: translateX(-100%) rotate(-12deg); }
  20%, 100% { transform: translateX(200%) rotate(-12deg); }
}

.animate-shine {
  animation: shine 8s infinite ease-in-out;
}
</style>