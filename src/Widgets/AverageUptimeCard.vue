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
  <div class="relative group h-full flex flex-col rounded-[24px] overflow-hidden transition-all duration-500 hover:-translate-y-2 hover:shadow-[0_20px_40px_-15px_rgba(139,92,246,0.3)] text-left w-full min-h-[320px] border border-gray-200/60 dark:border-white/5 bg-gradient-to-br from-white to-slate-50 dark:from-[#111827] dark:to-[#09090b]">
    
    <!-- Animated Gradient Glow -->
    <div class="absolute -top-[50%] -left-[50%] w-[200%] h-[200%] bg-[radial-gradient(circle_at_50%_50%,_rgba(139,92,246,0.15)_0%,_transparent_40%)] opacity-0 group-hover:opacity-100 transition-opacity duration-1000 pointer-events-none group-hover:animate-[spin_20s_linear_infinite]"></div>

    <!-- Massive Watermark Icon -->
    <Clock class="absolute -bottom-10 -right-10 w-72 h-72 text-violet-500/[0.03] dark:text-violet-400/[0.03] rotate-[-20deg] group-hover:rotate-0 group-hover:scale-110 transition-all duration-700 ease-out pointer-events-none" />

    <div class="relative z-10 p-7 flex flex-col h-full gap-4">
      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Gorgeous Icon Block -->
          <div class="relative w-14 h-14 rounded-2xl flex items-center justify-center shrink-0 bg-gradient-to-br from-violet-400 to-fuchsia-600 shadow-xl shadow-violet-500/40 group-hover:shadow-violet-500/60 transition-all duration-500 group-hover:scale-105">
            <div class="absolute inset-0 bg-white/20 rounded-2xl rounded-bl-none opacity-40 mix-blend-overlay"></div>
            <Clock class="w-7 h-7 text-white drop-shadow-md z-10" />
          </div>
          <div>
            <h3 class="text-xl font-black text-gray-900 dark:text-white tracking-tight leading-none group-hover:text-violet-600 dark:group-hover:text-violet-400 transition-colors">
              {{ t('quickMetrics.averageUptime.title') }}
            </h3>
            <p class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-400 dark:text-zinc-500 mt-2">
              {{ t('quickMetrics.averageUptime.systemStability') }}
            </p>
          </div>
        </div>
      </div>

      <!-- Main Body -->
      <div class="mt-auto pt-6 flex flex-col relative z-10">
        
        <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-400 dark:text-zinc-500 mb-1 drop-shadow-sm">
          {{ t('quickMetrics.averageUptime.timeActive') }}
        </div>
        
        <div class="text-5xl font-black tracking-tighter text-gray-900 dark:text-white mb-8 drop-shadow-sm group-hover:-translate-y-1 transition-transform duration-500">
          {{ stats.formatted }}
        </div>

        <!-- Nested Glass Container for details -->
        <div class="grid grid-cols-5 gap-3">
          
          <div class="col-span-2 p-4 rounded-[16px] bg-white/70 dark:bg-black/40 backdrop-blur-xl border border-white/60 dark:border-white/5 flex flex-col justify-center gap-1.5 group-hover:bg-white/90 dark:group-hover:bg-white/[0.03] transition-all duration-300 shadow-sm">
            <div class="flex items-center gap-1.5 text-gray-500 dark:text-zinc-400">
              <Zap class="w-3.5 h-3.5 text-violet-500" />
              <span class="text-[9px] font-bold uppercase tracking-widest">Active</span>
            </div>
            <div class="text-2xl font-black text-gray-900 dark:text-white tabular-nums tracking-tight leading-none">
               {{ stats.count }} <span class="text-[10px] font-black uppercase text-gray-400 tracking-widest">Apps</span>
            </div>
          </div>
          
          <div class="col-span-3 p-4 rounded-[16px] bg-gradient-to-r from-violet-50 to-fuchsia-50 dark:from-violet-950/40 dark:to-fuchsia-950/40 backdrop-blur-xl border border-violet-100 dark:border-violet-900/50 flex flex-col justify-end items-center group-hover:shadow-md transition-all duration-300 relative overflow-hidden h-[72px]">
             
             <!-- Bar Visual inside the glass panel -->
             <div class="absolute inset-0 flex items-end justify-between px-4 pb-2 opacity-60 group-hover:opacity-100 transition-opacity duration-300">
                <div
                  v-for="(h, i) in bars"
                  :key="i"
                  class="w-2.5 rounded-t-md transition-all duration-500 ease-out bg-gray-300/50 dark:bg-zinc-700/50 group-hover:bg-violet-500/90 group-hover:shadow-[0_0_12px_rgba(139,92,246,0.6)]"
                  :style="{ 
                    height: `${h * 100}%`, 
                    transitionDelay: `${i * 30}ms` 
                  }"
                ></div>
                <!-- Activity Line -->
                <svg class="absolute inset-0 w-full h-full pointer-events-none opacity-30" viewBox="0 0 100 64" preserveAspectRatio="none">
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
      </div>
    </div>
  </div>
</template>
