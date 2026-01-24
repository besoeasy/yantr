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
    text: 'text-indigo-600 dark:text-indigo-400',
    progress: 'bg-indigo-500 dark:bg-indigo-400'
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
  <div class="group relative bg-white dark:bg-slate-900 rounded-2xl shadow-sm border border-slate-200 dark:border-slate-800 hover:shadow-md transition-all duration-300 overflow-hidden h-full">
    
    <!-- Loading State -->
    <div v-if="loading" class="p-6 space-y-4">
      <div class="h-4 w-32 bg-slate-200 dark:bg-slate-800 rounded animate-pulse"></div>
      <div class="h-8 w-20 bg-slate-200 dark:bg-slate-800 rounded animate-pulse"></div>
      <div class="h-3 w-full bg-slate-200 dark:bg-slate-800 rounded animate-pulse"></div>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="p-6">
      <div class="flex items-center gap-2 text-rose-600 dark:text-rose-400">
        <Server :size="16" />
        <span class="text-sm font-medium">Failed to load system info</span>
      </div>
    </div>

    <!-- Content -->
    <div v-else class="p-6 flex flex-col h-full">
      <!-- Header -->
      <div class="flex items-center gap-2 mb-6">
        <div class="p-2 rounded-lg bg-indigo-500/10 dark:bg-indigo-500/20">
          <Server class="w-4 h-4 text-indigo-600 dark:text-indigo-400" />
        </div>
        <h3 class="text-sm font-bold uppercase tracking-wider text-slate-500 dark:text-slate-400">Host System</h3>
      </div>

      <!-- Metrics Grid -->
      <div class="space-y-5 flex-1">
        <!-- CPU Cores -->
        <div>
          <div class="flex items-center gap-2 mb-2">
            <Cpu class="w-4 h-4 text-slate-500 dark:text-slate-400" />
            <span class="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wide">CPU Cores</span>
          </div>
          <div class="text-3xl font-black text-slate-900 dark:text-white tabular-nums">
            {{ cpuInfo.cores }}
          </div>
        </div>

        <!-- Memory -->
        <div>
          <div class="flex items-center gap-2 mb-2">
            <MemoryStick class="w-4 h-4 text-slate-500 dark:text-slate-400" />
            <span class="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wide">Total RAM</span>
          </div>
          <div class="text-2xl font-black text-slate-900 dark:text-white tabular-nums">
            {{ memoryInfo.totalFormatted }}
          </div>
        </div>

        <!-- Storage -->
        <div v-if="storageInfo.hasData">
          <div class="flex items-center justify-between mb-2">
            <div class="flex items-center gap-2">
              <HardDrive class="w-4 h-4 text-slate-500 dark:text-slate-400" />
              <span class="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wide">Docker Storage</span>
            </div>
            <span v-if="storageInfo.total > 0" class="text-sm font-bold tabular-nums" :class="storageTheme.text">
              {{ storageInfo.percent }}%
            </span>
          </div>
          
          <div class="text-xs text-slate-600 dark:text-slate-400 mb-2 font-medium">
            <span v-if="storageInfo.totalFormatted">
              {{ storageInfo.usedFormatted }} / {{ storageInfo.totalFormatted }}
            </span>
            <span v-else>
              {{ storageInfo.usedFormatted }} used
            </span>
          </div>
          
          <div v-if="storageInfo.total > 0" class="h-2.5 w-full bg-slate-100 dark:bg-slate-800 rounded-full overflow-hidden border border-slate-200/50 dark:border-slate-700/50">
            <div 
              class="h-full rounded-full transition-all duration-700 ease-out" 
              :class="storageTheme.progress"
              :style="{ width: `${Math.min(storageInfo.percent, 100)}%` }">
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
