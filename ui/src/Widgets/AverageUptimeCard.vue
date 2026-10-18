<script>
// Dashboard engine: one column — the uptime number, the active-apps stat and
// the mini bar chart were arranged for a single cell. See Home.vue for how
// meta.cols is consumed.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from 'vue-i18n'
import { Clock, Activity, Zap } from '@lucide/vue'
import { formatDuration } from '../utils/metrics'
import { useCurrentTime } from '../composables/useCurrentTime'
import { useContainers } from '../composables/useDashboardData'
import WidgetSkeleton from '../components/WidgetSkeleton.vue'

const { t } = useI18n()
const { currentTime } = useCurrentTime()

// The container list is shared with the other three cards that want it — one
// request at mount and one 15s ticker for the dashboard, not four.
const { containers, containersLoaded, containersLoadFailed } = useContainers({ live: true })


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
  <WidgetSkeleton v-if="!containersLoaded && !containersLoadFailed" />
  <div v-else class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden rounded-2xl p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <div class="flex w-full items-start justify-between">
      <div class="flex items-center gap-4">
        <!-- Minimal Icon Block -->
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-emerald-200 bg-emerald-50 transition-transform duration-200 group-hover:scale-105 dark:border-emerald-900/50 dark:bg-emerald-900/20">
          <Clock class="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
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
      <div class="mb-2 text-[10px] font-semibold uppercase tracking-widest yantra-text-secondary">
        {{ t('quickMetrics.averageUptime.timeActive') }}
      </div>
      
      <div class="mb-6 text-4xl font-black leading-[1.1] tracking-tighter yantra-text-primary transition-colors duration-200 group-hover:-translate-y-[3px] sm:text-5xl yantra-text-primary">
        {{ stats.formatted }}
      </div>

      <!-- Nested Minimal Container for details -->
      <div class="grid grid-cols-5 gap-3">
        <!-- Stats Card -->
        <div class="col-span-2 flex flex-col justify-center gap-2 rounded-xl bg-zinc-50 p-4 transition-[border-color,box-shadow,transform] duration-200 dark:border-zinc-800 dark:bg-zinc-900/50">
          <div class="flex items-center gap-1.5 yantra-text-secondary">
            <Zap class="h-3.5 w-3.5" />
            <span class="text-[9px] font-semibold uppercase tracking-widest">{{ t('quickMetrics.averageUptime.active') }}</span>
          </div>
          <div class="text-2xl font-black leading-none tracking-tight yantra-text-primary tabular-nums yantra-text-primary">
             {{ stats.count }} <span class="text-[10px] font-semibold uppercase tracking-widest yantra-text-secondary">{{ t('quickMetrics.averageUptime.apps') }}</span>
          </div>
        </div>
        
        <!-- Mini Bar Chart -->
        <div class="col-span-3 relative flex h-[72px] flex-col items-center justify-end overflow-hidden rounded-xl bg-zinc-50 transition-[border-color,box-shadow,transform] duration-200 dark:border-zinc-800 dark:bg-zinc-900/50">
           <div class="absolute inset-0 flex items-end justify-between px-4 pb-2">
              <div
                v-for="(h, i) in bars"
                :key="i"
                class="w-2.5 rounded-t-md bg-emerald-200 transition-[border-color,box-shadow,transform,opacity] duration-200 ease-out group-hover:bg-emerald-500 dark:bg-emerald-900/60 dark:group-hover:bg-emerald-400"
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
                   class="yantra-text-primary"
                 />
              </svg>
           </div>
        </div>
      </div>
    </div>
  </div>
</template>
