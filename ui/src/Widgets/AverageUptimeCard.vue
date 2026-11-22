<script>
// Dashboard engine: one column — the average uptime headline, and nothing else.
// The per-app bar strip that used to sit under it is gone: one bar per running
// app said less than the average directly above it, and on a host running a
// single app it drew an empty box with nothing to compare. The count lives on
// the Host System workload grid. See Home.vue for how meta.cols is consumed.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from 'vue-i18n'
import { Clock } from '@lucide/vue'
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
</script>

<template>
  <WidgetSkeleton v-if="!containersLoaded && !containersLoadFailed" />
  <div v-else class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
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
      
      <div class="text-4xl font-black leading-[1.1] tracking-tighter yantra-text-primary transition-colors duration-200 group-hover:-translate-y-[3px] sm:text-5xl yantra-text-primary">
        {{ stats.formatted }}
      </div>
    </div>
  </div>
</template>
