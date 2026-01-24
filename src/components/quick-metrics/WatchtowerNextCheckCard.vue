<script setup>
import { computed } from 'vue'
import { RefreshCw } from 'lucide-vue-next'
import { formatDuration } from '../../utils/metrics.js'

const props = defineProps({
  containers: { type: Array, default: () => [] },
  currentTime: { type: Number, default: () => Date.now() },
  intervalHours: { type: Number, default: 3 }
})

const intervalMs = computed(() => {
  const hours = Number(props.intervalHours)
  if (!Number.isFinite(hours) || hours <= 0) return 3 * 60 * 60 * 1000
  return hours * 60 * 60 * 1000
})

const watchtowerContainer = computed(() => {
  const list = Array.isArray(props.containers) ? props.containers : []

  const matches = list.filter((c) => {
    const name = (c?.name || '').toString().toLowerCase()
    const names = Array.isArray(c?.Names) ? c.Names : []
    return name.includes('watchtower') || names.some((n) => (n || '').toString().toLowerCase().includes('watchtower'))
  })

  if (matches.length === 0) return null

  const running = matches.find((c) => c?.state === 'running')
  return running || matches[0]
})

const uptimeMs = computed(() => {
  const c = watchtowerContainer.value
  if (!c || c.state !== 'running' || !c.created) return null
  const createdMs = Number(c.created) * 1000
  if (!Number.isFinite(createdMs)) return null
  return Math.max(0, props.currentTime - createdMs)
})

const remainingMs = computed(() => {
  const up = uptimeMs.value
  if (!Number.isFinite(up)) return null
  const interval = intervalMs.value
  const mod = up % interval
  const remaining = interval - mod
  return remaining === 0 ? interval : remaining
})

const progressPct = computed(() => {
  const up = uptimeMs.value
  if (!Number.isFinite(up)) return 0
  const interval = intervalMs.value
  const mod = up % interval
  return Math.min(100, Math.max(0, (mod / interval) * 100))
})

const nextCheckTime = computed(() => {
  const remaining = remainingMs.value
  if (!Number.isFinite(remaining)) return null
  const dt = new Date(props.currentTime + remaining)
  return dt.toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit' })
})

const status = computed(() => {
  const c = watchtowerContainer.value
  if (!c) return { state: 'missing', title: 'Watchtower not detected', subtitle: 'Install Watchtower to enable auto-updates.' }
  if (c.state !== 'running') return { state: 'stopped', title: 'Watchtower paused', subtitle: 'Start Watchtower to resume update checks.' }
  return { state: 'running', title: 'Update checks scheduled', subtitle: `Every ${props.intervalHours} hours (based on Watchtower uptime)` }
})
</script>

<template>
  <div class="relative h-full overflow-hidden group rounded-2xl transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)] hover:-translate-y-1">
    <div class="absolute inset-0 bg-white dark:bg-gray-900">
      <div class="absolute inset-0 bg-linear-to-br from-emerald-200/60 via-cyan-200/30 to-white/80 dark:from-emerald-600/25 dark:via-cyan-600/10 dark:to-gray-900 z-10"></div>
      <div class="absolute top-0 right-0 w-64 h-64 bg-emerald-300/35 dark:bg-emerald-500/20 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2 group-hover:bg-emerald-400/45 dark:group-hover:bg-emerald-500/30 transition-colors duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"></div>
      <div class="absolute bottom-0 left-0 w-48 h-48 bg-cyan-300/30 dark:bg-cyan-600/20 rounded-full blur-3xl translate-y-1/2 -translate-x-1/2 group-hover:bg-cyan-400/40 dark:group-hover:bg-cyan-600/30 transition-colors duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"></div>
    </div>

    <div class="relative z-20 h-full p-6 flex flex-col justify-between border border-slate-200/80 dark:border-slate-700/60 rounded-2xl backdrop-blur-sm group-hover:border-emerald-300/60 dark:group-hover:border-emerald-500/30 transition-colors duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]">
      <div class="flex items-start justify-between gap-4">
        <div class="flex items-center gap-4">
          <div class="relative">
            <div class="absolute inset-0 bg-emerald-400/25 dark:bg-emerald-500/20 rounded-xl blur-lg group-hover:blur-xl transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"></div>
            <div class="relative w-12 h-12 bg-linear-to-br from-emerald-500 to-cyan-600 rounded-xl flex items-center justify-center shadow-lg group-hover:scale-110 transition-transform duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]">
              <RefreshCw class="w-6 h-6 text-white" />
            </div>
          </div>

          <div>
            <h3 class="text-lg font-bold text-slate-900 dark:text-white mb-1 group-hover:text-emerald-700 dark:group-hover:text-emerald-200 transition-colors">Next Update Check</h3>
            <p class="text-sm font-medium text-slate-600 dark:text-gray-400 group-hover:text-slate-700 dark:group-hover:text-gray-300 transition-colors">
              {{ status.title }}
            </p>
          </div>
        </div>
      </div>

      <div class="mt-5">
        <div v-if="status.state !== 'running'" class="rounded-2xl border border-dashed border-slate-200 dark:border-slate-700/60 bg-white/70 dark:bg-white/5 px-4 py-6 text-center">
          <div class="text-sm font-semibold text-slate-700 dark:text-gray-200">{{ status.subtitle }}</div>
          <div class="text-xs text-slate-500 dark:text-gray-400 mt-1">Watchtower checks run on an interval.</div>
        </div>

        <div v-else class="flex flex-col items-center justify-center py-1">
          <div class="text-4xl font-extrabold text-transparent bg-clip-text bg-linear-to-r from-emerald-600 to-cyan-600 dark:from-emerald-200 dark:to-cyan-200 tabular-nums">
            {{ formatDuration(remainingMs) }}
          </div>
          <div class="text-xs text-slate-500 dark:text-gray-400 font-medium mt-1">
            Next run at <span class="font-mono text-slate-700 dark:text-gray-300">{{ nextCheckTime || '—' }}</span>
            <span v-if="uptimeMs" class="text-slate-400 dark:text-gray-500"> • uptime {{ formatDuration(uptimeMs) }}</span>
          </div>

          <div class="mt-5 w-full">
            <div class="h-2 rounded-full bg-slate-200/70 dark:bg-white/5 border border-slate-200/80 dark:border-slate-700/60 overflow-hidden">
              <div
                class="h-full bg-linear-to-r from-emerald-500/80 to-cyan-500/80 transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"
                :style="{ width: `${progressPct}%` }"
              ></div>
            </div>
            <div class="mt-2 flex items-center justify-between text-[10px] text-slate-500 dark:text-gray-500">
              <span class="font-semibold uppercase tracking-wider">{{ status.subtitle }}</span>
              <span class="font-mono">{{ Math.round(progressPct) }}%</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
