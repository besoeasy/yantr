<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Clock, Activity, Zap } from '@lucide/vue'
import { formatDuration } from '../utils/metrics'
import { useApiUrl } from '../composables/useApiUrl'
import { expectApiSuccess } from '../composables/useApiResponse'
import { useCurrentTime } from '../composables/useCurrentTime'

const { t } = useI18n()
const { apiUrl } = useApiUrl()
const { currentTime } = useCurrentTime()

const containers = ref([])
let refreshInterval = null

async function fetchContainers() {
  try {
    const response = await fetch(`${apiUrl.value}/api/containers`)
    const data = await expectApiSuccess(response, 'Failed to load containers')
    containers.value = Array.isArray(data.containers) ? data.containers : []
  } catch {}
}

onMounted(() => {
  fetchContainers()
  refreshInterval = setInterval(fetchContainers, 15000)
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
})

const stats = computed(() => {
  const runningContainers = containers.value.filter(c => c.state === 'running' && c.created)
  const count = runningContainers.length

  if (count === 0) {
    return { formatted: '0m', count: 0, rawAvg: 0 }
  }

  const totalUptime = runningContainers.reduce((sum, container) => {
    const createdTime = container.created * 1000
    const uptime = currentTime.value - createdTime
    return sum + uptime
  }, 0)

  const avgUptime = totalUptime / count
  const formatted = formatDuration(avgUptime)

  return { formatted, count, rawAvg: avgUptime }
})

// Generate random bar heights for visualization
const bars = [0.4, 0.7, 0.5, 0.9, 0.6, 0.8, 0.3, 0.7, 0.5, 0.8]
</script>

<template>
  <div class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] rounded-xl p-6 overflow-hidden transition-all duration-400 hover:shadow-2xl hover:shadow-violet-500/20 dark:hover:shadow-violet-500/20">
    
    <!-- Hover Accents -->
    <div class="absolute top-0 left-0 w-full h-0.5 bg-violet-500 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>

    <div class="relative z-10 flex items-start justify-between mb-6">
      <div class="flex items-center gap-4">
        <!-- Icon Container -->
        <div class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0 group-hover:scale-105 transition-all duration-500">
           <Clock class="w-5 h-5 text-gray-400 dark:text-zinc-500 group-hover:text-violet-500 transition-colors" />
        </div>
        
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight group-hover:text-violet-600 dark:group-hover:text-violet-400 transition-colors">
            {{ t('quickMetrics.averageUptime.title') }}
          </h3>
          <div class="flex items-center gap-2 mt-1 text-gray-500 dark:text-zinc-400">
             <Activity class="w-3 h-3" />
             <span class="text-[11px] font-medium uppercase tracking-widest">
               {{ t('quickMetrics.averageUptime.systemStability') }}
             </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Main Content & Chart -->
    <div class="relative z-10 flex-1 flex items-end justify-between gap-4 mt-2">
      <!-- Big Metric -->
      <div class="flex-1 min-w-0">
        <div v-if="stats.count > 0">
           <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-400 dark:text-zinc-500 mb-2">{{ t('quickMetrics.averageUptime.timeActive') }}</div>
           <div class="text-4xl font-bold text-gray-900 dark:text-white tabular-nums tracking-tighter leading-none">
             {{ stats.formatted }}
           </div>
           <div class="mt-3 flex items-center gap-1.5 text-[11px] font-medium text-gray-500 dark:text-zinc-400">
             <Zap class="w-3.5 h-3.5 text-violet-500" />
             <span>{{ t('quickMetrics.averageUptime.acrossContainers', { count: stats.count }) }}</span>
           </div>
         </div>
         
         <div v-else>
            <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-400 dark:text-zinc-500 mb-2">{{ t('quickMetrics.averageUptime.timeActive') }}</div>
            <div class="text-3xl font-bold text-gray-400 dark:text-zinc-600 tracking-tighter leading-none">
              0m
            </div>
            <div class="mt-3 inline-block text-[10px] font-semibold px-2 py-1 rounded text-gray-500 dark:text-zinc-400 uppercase tracking-widest">
               {{ t('quickMetrics.averageUptime.systemIdle') }}
            </div>
         </div>
      </div>

      <!-- Bar Visual -->
      <div class="relative shrink-0 w-24 h-16 flex items-end justify-between gap-1 pb-1 opacity-50 group-hover:opacity-100 transition-opacity duration-300">
        <div
          v-for="(h, i) in bars"
          :key="i"
          class="w-1.5 rounded-t-sm transition-all duration-500 ease-out bg-gray-300 dark:bg-zinc-700 group-hover:bg-violet-500/80"
          :style="{ 
            height: `${h * 100}%`, 
            transitionDelay: `${i * 30}ms` 
          }"
        ></div>
        
        <!-- Activity Line -->
        <svg class="absolute inset-0 w-full h-full pointer-events-none opacity-20" viewBox="0 0 100 64" preserveAspectRatio="none">
           <path 
             d="M0 50 C 20 50, 20 10, 40 30 S 60 50, 80 20 L 100 40" 
             fill="none" 
             stroke="currentColor" 
             stroke-width="2" 
             class="text-violet-500"
           />
        </svg>
      </div>
    </div>
  </div>
</template>
