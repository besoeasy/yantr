<script>
// Dashboard engine: one column — the average uptime headline and one bar per
// running app, height proportional to how long each has been up. The count
// lives on the Host System workload grid (which was rendering it twice here);
// the decorative overview curve was removed with it — bars a browser invented
// are not a fact. See Home.vue for how meta.cols is consumed.
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

// One bar per running app, height proportional to its uptime relative to the
// oldest: a just-restarted stack reads as short bars beside tall ones, which
// is the story behind the average and is real — derived from the same
// createdAt ticks the average itself uses. Ten bounds the paint so a busy box
// stays legible.
const liveBars = computed(() => {
  const running = containers.value.filter(c => c.state === 'running' && c.created)
  if (!running.length) return []
  const uptimes = running.map(c => ({
    id: c.id,
    up: Math.max(1, currentTime.value - c.created * 1000),
  }))
  const max = Math.max(...uptimes.map(u => u.up))
  return uptimes
    .sort((a, b) => b.up - a.up)
    .slice(0, 10)
    .map(u => ({ id: u.id, height: Math.max(4, Math.round((u.up / max) * 100)) }))
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
      
      <div class="mb-6 text-4xl font-black leading-[1.1] tracking-tighter yantra-text-primary transition-colors duration-200 group-hover:-translate-y-[3px] sm:text-5xl yantra-text-primary">
        {{ stats.formatted }}
      </div>

      <!-- The distribution behind the average: real bars, one per running
           app, longest up on the left. A machine whose apps were all restarted
           five minutes ago reads very differently from one that has been
           undisturbed, and this is where that story shows. -->
      <div
        v-if="liveBars.length"
        class="relative flex h-[72px] flex-col items-end justify-end overflow-hidden rounded-xl bg-zinc-50 px-4 pb-2 transition-[border-color,box-shadow,transform] duration-200 dark:border-zinc-800 dark:bg-zinc-900/50"
      >
        <div class="flex w-full items-end justify-between gap-1">
          <div
            v-for="bar in liveBars"
            :key="bar.id"
            class="flex-1 rounded-t-md bg-emerald-200 transition-[border-color,box-shadow,transform,opacity] duration-200 ease-out group-hover:bg-emerald-500 dark:bg-emerald-900/60 dark:group-hover:bg-emerald-400"
            :style="{ height: `${bar.height}%` }"
          ></div>
        </div>
      </div>
      <div
        v-else
        class="flex h-[72px] items-center justify-center rounded-xl bg-zinc-50 text-[10px] font-semibold uppercase tracking-widest yantra-text-secondary dark:bg-zinc-900/50"
      >
        {{ t('quickMetrics.averageUptime.systemIdle') }}
      </div>
    </div>
  </div>
</template>
