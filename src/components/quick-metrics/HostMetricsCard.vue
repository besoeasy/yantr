<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Cpu, MemoryStick, HardDrive, Server } from 'lucide-vue-next'
import { formatBytes } from '../../utils/metrics'

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
  <div class="group relative h-full overflow-hidden rounded-none bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 transition-all duration-300 hover:shadow-xl dark:hover:shadow-slate-900/50 flex flex-col font-mono text-xs">
    
    <!-- Hacker Background Texture -->
    <div class="absolute inset-0 opacity-[0.03] dark:opacity-[0.05]" 
         style="background-image: linear-gradient(0deg, transparent 24%, rgba(0, 0, 0, .3) 25%, rgba(0, 0, 0, .3) 26%, transparent 27%, transparent 74%, rgba(0, 0, 0, .3) 75%, rgba(0, 0, 0, .3) 76%, transparent 77%, transparent), linear-gradient(90deg, transparent 24%, rgba(0, 0, 0, .3) 25%, rgba(0, 0, 0, .3) 26%, transparent 27%, transparent 74%, rgba(0, 0, 0, .3) 75%, rgba(0, 0, 0, .3) 76%, transparent 77%, transparent); background-size: 30px 30px;">
    </div>
    
    <!-- Background Icon Glitch Effect -->
    <div class="absolute -right-6 -bottom-6 opacity-5 dark:opacity-10 transform rotate-12 group-hover:scale-110 group-hover:rotate-6 transition-transform duration-700 ease-in-out pointer-events-none">
       <Server class="w-48 h-48 text-indigo-900 dark:text-indigo-100" />
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="relative z-10 p-5 flex-1 flex flex-col items-center justify-center">
      <div class="flex items-center gap-2 text-slate-400 animate-pulse">
         <span class="inline-block w-2 h-4 bg-slate-400"></span>
         <span class="uppercase tracking-widest">INITIALIZING_SCAN...</span>
      </div>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="relative z-10 p-5 flex flex-col items-center justify-center h-full text-center">
        <div class="border border-rose-500/30 p-4 bg-rose-500/5 text-rose-600 dark:text-rose-400 max-w-full">
            <span class="block font-bold mb-1 uppercase tracking-wider">FATAL_ERROR</span>
            <span class="opacity-75 break-words">{{ error }}</span>
        </div>
    </div>

    <!-- Content -->
    <div v-else class="relative z-10 p-5 flex flex-col h-full gap-4">
      
      <!-- Header -->
      <div class="flex items-start justify-between border-b border-slate-100 dark:border-slate-800 pb-3">
         <div class="min-w-0 pr-2">
            <div class="text-[10px] text-slate-400 mb-0.5 tracking-widest">HOST__ID</div>
            <div class="text-sm font-bold text-slate-800 dark:text-slate-200 uppercase tracking-tight truncate" :title="osInfo.name">{{ osInfo.name }}</div>
         </div>
         <div class="text-right shrink-0">
             <div class="flex items-center justify-end gap-1.5 mb-0.5">
                  <span class="relative flex h-1.5 w-1.5">
                    <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                    <span class="relative inline-flex rounded-full h-1.5 w-1.5 bg-emerald-500"></span>
                  </span>
                  <span class="text-[10px] text-emerald-600 dark:text-emerald-400 font-bold uppercase tracking-wider">ONLINE</span>
             </div>
             <div class="text-[10px] text-slate-400 tabular-nums font-bold">KRNL: {{ osInfo.kernel }}</div>
         </div>
      </div>

      <!-- Specs Grid -->
      <div class="grid grid-cols-2 gap-x-4 gap-y-4 flex-1">
         
         <!-- CPU section -->
         <div class="flex flex-col gap-1">
             <div class="flex justify-between items-end border-b border-slate-100 dark:border-slate-800/50 pb-1 mb-1">
                <span class="font-bold text-slate-500 tracking-wider">CPU_CORE</span>
                <Cpu class="w-3 h-3 text-slate-400" />
             </div>
             <div class="flex items-baseline gap-1 mt-auto">
                 <span class="text-2xl font-bold text-slate-800 dark:text-slate-200 tabular-nums">{{ cpuInfo.cores }}</span>
                 <span class="text-[10px] text-slate-400 font-bold">THREADS</span>
             </div>
             <!-- Fake activity bar -->
             <div class="flex gap-0.5 mt-1 h-1">
                <div v-for="i in 5" :key="i" class="flex-1 bg-blue-500/10 dark:bg-blue-400/10 rounded-sm overflow-hidden">
                    <div class="h-full bg-blue-500 dark:bg-blue-400 animate-pulse" :style="{ width: `${30 + Math.random() * 70}%`, animationDuration: `${0.5 + Math.random()}s` }"></div>
                </div>
             </div>
         </div>

         <!-- RAM section -->
         <div class="flex flex-col gap-1">
            <div class="flex justify-between items-end border-b border-slate-100 dark:border-slate-800/50 pb-1 mb-1">
                <span class="font-bold text-slate-500 tracking-wider">MEM_ALLOC</span>
                <MemoryStick class="w-3 h-3 text-slate-400" />
             </div>
             <div class="flex items-baseline gap-1 mt-auto">
                 <span class="text-xl font-bold text-slate-800 dark:text-slate-200 tabular-nums">{{ memoryInfo.totalFormatted.split(' ')[0] }}</span>
                 <span class="text-[10px] text-slate-400 font-bold">{{ memoryInfo.totalFormatted.split(' ')[1] }}</span>
             </div>
             <div class="h-1 w-full bg-violet-500/10 dark:bg-violet-400/10 rounded-sm">
                <div class="h-full bg-violet-500 dark:bg-violet-400 w-full rounded-sm opacity-50"></div>
             </div>
         </div>
         
         <!-- Storage Section (Full Width) -->
         <div v-if="storageInfo.hasData" class="col-span-2 mt-auto pt-2">
             <div class="flex justify-between items-center mb-1.5">
                 <span class="font-bold text-slate-500 tracking-wider">DOCKER_USED</span>
                 <div class="text-xl font-bold text-slate-800 dark:text-slate-200 tabular-nums">{{ storageInfo.usedFormatted }}</div>
             </div>
             
             <!-- Static Visual Bar -->
             <div class="flex gap-0.5 h-1.5 w-full">
                 <div v-for="i in 20" :key="i" 
                      class="flex-1 rounded-[1px] bg-emerald-500/40 dark:bg-emerald-400/40">
                 </div>
             </div>
         </div>

      </div>

      <!-- Footer / Deco -->
      <div class="text-[9px] text-slate-300 dark:text-slate-700 flex justify-between uppercase tracking-widest pt-2 border-t border-slate-50 dark:border-slate-800/50">
          <span>SYS.MON.V1</span>
          <span>// READY</span>
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