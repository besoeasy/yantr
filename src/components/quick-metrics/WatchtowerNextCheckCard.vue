<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { RefreshCw, Clock, PauseCircle, AlertCircle, Zap } from 'lucide-vue-next'
import { formatDuration } from '../../utils/metrics.js'

const { t } = useI18n()
const props = defineProps({
  containers: { type: Array, default: () => [] },
  currentTime: { type: Number, default: () => Date.now() }
})

// Calculate next run based on cron schedule: "0 */3 * * *" (every 3 hours at :00)
// Schedule runs at: 00:00, 03:00, 06:00, 09:00, 12:00, 15:00, 18:00, 21:00
const getNextCronRun = (fromTime) => {
  const now = new Date(fromTime)
  const currentHour = now.getHours()
  const currentMinute = now.getMinutes()
  
  // Calculate next 3-hour interval
  const nextHour = Math.ceil((currentHour + 1) / 3) * 3
  
  if (nextHour > 23) {
    // Next run is tomorrow at 00:00
    const tomorrow = new Date(now)
    tomorrow.setDate(tomorrow.getDate() + 1)
    tomorrow.setHours(0, 0, 0, 0)
    return tomorrow
  }
  
  // If we're at or past the hour mark, go to next interval
  if (currentHour % 3 === 0 && currentMinute === 0) {
    // Exactly at the hour, next run is in 3 hours
    const nextRun = new Date(now)
    nextRun.setHours(currentHour + 3, 0, 0, 0)
    if (nextRun.getHours() < currentHour) {
      // Wrapped to next day
      nextRun.setDate(nextRun.getDate() + 1)
    }
    return nextRun
  }
  
  // Next run is at the next 3-hour mark
  const nextRun = new Date(now)
  nextRun.setHours(nextHour, 0, 0, 0)
  return nextRun
}

const INTERVAL_HOURS = 3

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

  // Calculate time until next cron scheduled run
  const nextRun = getNextCronRun(props.currentTime)
  return nextRun.getTime() - props.currentTime
})

const progressPct = computed(() => {
  const up = uptimeMs.value
  if (!Number.isFinite(up)) return 0
  
  // Calculate progress in current 3-hour interval
  const now = new Date(props.currentTime)
  const currentHour = now.getHours()
  const currentMinute = now.getMinutes()
  const currentSecond = now.getSeconds()
  
  // Time since last 3-hour mark
  const hoursSinceLastInterval = currentHour % 3
  const totalSecondsSinceInterval = (hoursSinceLastInterval * 3600) + (currentMinute * 60) + currentSecond
  const totalSecondsInInterval = INTERVAL_HOURS * 3600
  
  return Math.min(100, Math.max(0, (totalSecondsSinceInterval / totalSecondsInInterval) * 100))
})

const nextCheckTime = computed(() => {
  const remaining = remainingMs.value
  if (!Number.isFinite(remaining)) return null
  const dt = new Date(props.currentTime + remaining)
  return dt.toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit' })
})

const intervalHours = computed(() => INTERVAL_HOURS)

const status = computed(() => {
  const c = watchtowerContainer.value
  if (!c) return { state: 'missing', label: t('quickMetrics.watchtowerNextCheck.missing'), icon: AlertCircle }
  if (c.state !== 'running') return { state: 'stopped', label: t('quickMetrics.watchtowerNextCheck.paused'), icon: PauseCircle }
  return { state: 'running', label: t('quickMetrics.watchtowerNextCheck.active'), icon: Clock }
})
</script>

<template>
  <div class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl p-6 overflow-hidden transition-all duration-400 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:border-gray-300 dark:hover:border-zinc-600">
    
    <!-- Hover Accents -->
    <div class="absolute top-0 left-0 w-full h-[2px] opacity-0 group-hover:opacity-100 transition-opacity duration-500"
         :class="status.state === 'running' ? 'bg-gradient-to-r from-transparent via-green-500 to-transparent' : 'bg-gradient-to-r from-transparent via-amber-500 to-transparent'">
    </div>
    <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none [mask-image:linear-gradient(to_bottom,white,transparent)]"></div>

    <div class="relative z-10 flex items-start justify-between mb-6">
      <div class="flex items-center gap-4">
        <!-- Icon Container -->
        <div class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0 group-hover:scale-105 group-hover:border-zinc-700 transition-all duration-500"
             :class="status.state === 'running' ? 'bg-green-50/50 dark:bg-green-500/10 border border-green-100 dark:border-green-500/20' : 'bg-amber-50/50 dark:bg-amber-500/10 border border-amber-100 dark:border-amber-500/20'">
           <RefreshCw class="w-5 h-5 transition-transform duration-700 ease-in-out group-hover:rotate-180" 
                      :class="status.state === 'running' ? 'text-green-600 dark:text-green-500' : 'text-amber-600 dark:text-amber-500'" />
        </div>
        
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight transition-colors"
              :class="status.state === 'running' ? 'group-hover:text-green-600 dark:group-hover:text-green-400' : 'group-hover:text-amber-600 dark:group-hover:text-amber-400'">
            {{ t('quickMetrics.watchtowerNextCheck.title') }}
          </h3>
          <div class="flex items-center gap-2 mt-1">
             <div class="w-1.5 h-1.5 rounded-full"
                  :class="status.state === 'running' ? 'bg-green-500 animate-pulse' : 'bg-amber-500'">
             </div>
             <span class="text-[11px] font-medium text-gray-500 dark:text-zinc-400 uppercase tracking-wider">
               {{ status.label }}
             </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Main Content & Gauge -->
    <div class="relative z-10 flex-1 flex items-end justify-between gap-4 mt-2">
      <!-- Big Timer -->
      <div class="flex-1 min-w-0">
         <div v-if="status.state === 'running'">
           <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-400 dark:text-zinc-500 mb-2">{{ t('quickMetrics.watchtowerNextCheck.nextRunIn') }}</div>
           <div class="text-4xl font-bold tabular-nums tracking-tighter text-gray-900 dark:text-white leading-none">
             {{ formatDuration(remainingMs) }}
           </div>
           <div class="mt-3 flex items-center gap-1.5 text-[11px] font-medium text-gray-500 dark:text-zinc-400">
             <Clock class="w-3.5 h-3.5" />
             <span>{{ t('quickMetrics.watchtowerNextCheck.at') }} {{ nextCheckTime }}</span>
           </div>
         </div>
         
         <div v-else>
            <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-amber-600/80 dark:text-amber-500/60 mb-2">{{ t('quickMetrics.watchtowerNextCheck.actionRequired') }}</div>
            <div class="text-2xl font-bold tracking-tighter text-gray-900 dark:text-white leading-none">
              {{ t('quickMetrics.watchtowerNextCheck.setupUpdates') }}
            </div>
            <div class="mt-3 inline-block text-[10px] font-semibold px-2 py-1 rounded bg-amber-50/50 dark:bg-amber-500/10 text-amber-600 dark:text-amber-400 uppercase tracking-wider">
               {{ t('quickMetrics.watchtowerNextCheck.updatesPaused') }}
            </div>
         </div>
      </div>

      <!-- Linear Progress (replaces circular) -->
      <div v-if="status.state === 'running'" class="shrink-0 w-24 flex flex-col items-end gap-1.5 opacity-50 group-hover:opacity-100 transition-opacity duration-300">
         <div class="text-[10px] font-bold text-gray-900 dark:text-white tabular-nums tracking-wider">{{ Math.round(progressPct) }}%</div>
         <div class="h-1.5 w-full bg-gray-100 dark:bg-zinc-800 rounded-full overflow-hidden flex">
            <div class="h-full bg-green-500 transition-all duration-1000 ease-out" :style="{ width: `${progressPct}%` }"></div>
         </div>
         <div class="text-[9px] text-gray-400 dark:text-zinc-500 uppercase tracking-widest mt-0.5">{{ t('quickMetrics.watchtowerNextCheck.interval') }} {{ intervalHours }}h</div>
      </div>
      <div v-else class="shrink-0 w-12 h-12 flex items-center justify-center rounded-lg bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800">
         <component :is="status.icon" class="w-6 h-6 text-gray-400 dark:text-zinc-500" />
      </div>
    </div>
  </div>
</template>
