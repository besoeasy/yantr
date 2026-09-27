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
  <div class="group relative flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-all duration-300 hover:-translate-y-1 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <div class="flex w-full items-start justify-between">
      <div class="flex items-center gap-4">
        <!-- Minimal Icon Block -->
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-emerald-200 bg-emerald-50 transition-transform duration-300 group-hover:scale-105 dark:border-emerald-900/50 dark:bg-emerald-900/20">
          <Clock class="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors group-hover:text-black dark:text-white dark:group-hover:text-white">
            {{ t('quickMetrics.averageUptime.title') }}
          </h3>
          <div class="mt-2 flex items-center gap-1.5">
            <div class="h-1.5 w-1.5 rounded-full bg-emerald-500"></div>
            <span class="text-[10px] font-semibold uppercase tracking-widest text-emerald-600 dark:text-emerald-500">
              {{ t('quickMetrics.averageUptime.systemStability') }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Main Body -->
    <div class="relative z-10 mt-auto flex flex-col pt-6">
      <div class="mb-2 text-[10px] font-semibold uppercase tracking-widest text-zinc-500 dark:text-zinc-400">
        {{ t('quickMetrics.averageUptime.timeActive') }}
      </div>
      
      <div class="mb-6 text-4xl font-black leading-[1.1] tracking-tighter text-zinc-900 transition-colors duration-500 group-hover:-translate-y-1 sm:text-5xl dark:text-white">
        {{ stats.formatted }}
      </div>

      <!-- Nested Minimal Container for details -->
      <div class="grid grid-cols-5 gap-3">
        <!-- Stats Card -->
        <div class="col-span-2 flex flex-col justify-center gap-2 rounded-xl border border-zinc-200 bg-zinc-50 p-4 transition-all duration-300 dark:border-zinc-800 dark:bg-zinc-900/50">
          <div class="flex items-center gap-1.5 text-zinc-500 dark:text-zinc-400">
            <Zap class="h-3.5 w-3.5" />
            <span class="text-[9px] font-semibold uppercase tracking-widest">Active</span>
          </div>
          <div class="text-2xl font-black leading-none tracking-tight text-zinc-900 tabular-nums dark:text-white">
             {{ stats.count }} <span class="text-[10px] font-semibold uppercase tracking-widest text-zinc-500 dark:text-zinc-400">Apps</span>
          </div>
        </div>
        
        <!-- Mini Bar Chart -->
        <div class="col-span-3 relative flex h-[72px] flex-col items-center justify-end overflow-hidden rounded-xl border border-zinc-200 bg-zinc-50 transition-all duration-300 dark:border-zinc-800 dark:bg-zinc-900/50">
           <div class="absolute inset-0 flex items-end justify-between px-4 pb-2">
              <div
                v-for="(h, i) in bars"
                :key="i"
                class="w-2.5 rounded-t-md bg-emerald-200 transition-all duration-500 ease-out group-hover:bg-emerald-500 dark:bg-emerald-900/60 dark:group-hover:bg-emerald-400"
                :style="{ 
                  height: `${h * 100}%`, 
                  transitionDelay: `${i * 30}ms` 
                }"
              ></div>
              <svg class="absolute inset-0 h-full w-full pointer-events-none opacity-10 dark:opacity-20" viewBox="0 0 100 64" preserveAspectRatio="none">
                 <path 
                   d="M0 50 C 20 50, 20 10, 40 30 S 60 50, 80 20 L 100 40" 
                   fill="none" 
                   stroke="currentColor" 
                   stroke-width="2" 
                   class="text-zinc-900 dark:text-white"
                 />
              </svg>
           </div>
        </div>
      </div>
    </div>
  </div>
</template>
