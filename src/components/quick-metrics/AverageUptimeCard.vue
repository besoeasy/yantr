<script setup>
import { computed } from 'vue'
import { Clock, Activity, Zap } from 'lucide-vue-next'

const props = defineProps({
  containers: { type: Array, default: () => [] },
  currentTime: { type: Number, default: () => Date.now() }
})

const stats = computed(() => {
  const runningContainers = props.containers.filter(c => c.state === 'running' && c.created)
  const count = runningContainers.length

  if (count === 0) {
    return { formatted: '0m', count: 0, rawAvg: 0 }
  }

  const totalUptime = runningContainers.reduce((sum, container) => {
    const createdTime = container.created * 1000
    const uptime = props.currentTime - createdTime
    return sum + uptime
  }, 0)

  const avgUptime = totalUptime / count
  const days = Math.floor(avgUptime / (1000 * 60 * 60 * 24))
  const hours = Math.floor((avgUptime % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60))
  const minutes = Math.floor((avgUptime % (1000 * 60 * 60)) / (1000 * 60))

  let formatted = ''
  if (days > 0) formatted = `${days}d ${hours}h`
  else if (hours > 0) formatted = `${hours}h ${minutes}m`
  else formatted = `${minutes}m`

  return { formatted, count, rawAvg: avgUptime }
})

const theme = computed(() => {
  if (stats.value.count > 0) {
    return {
      text: 'text-violet-600 dark:text-violet-400',
      bg: 'bg-violet-500/10 dark:bg-violet-500/20',
      border: 'group-hover:border-violet-500/30 dark:group-hover:border-violet-400/30',
      ring: 'text-violet-500 dark:text-violet-400',
      bar: 'bg-violet-500 dark:bg-violet-400'
    }
  }
  return {
    text: 'text-slate-500 dark:text-slate-400',
    bg: 'bg-slate-500/10 dark:bg-slate-500/20',
    border: 'group-hover:border-slate-500/30 dark:group-hover:border-slate-400/30',
    ring: 'text-slate-300 dark:text-slate-600',
    bar: 'bg-slate-300 dark:bg-slate-600'
  }
})

// Generate random bar heights for visualization
const bars = [0.4, 0.7, 0.5, 0.9, 0.6, 0.8, 0.3, 0.7, 0.5, 0.8]
</script>

<template>
  <div
    class="relative h-full overflow-hidden group rounded-3xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 transition-all duration-300 hover:shadow-lg dark:hover:shadow-slate-900/50"
    :class="theme.border"
  >
    <!-- Background Texture -->
    <div class="absolute inset-0 opacity-[0.03] dark:opacity-[0.05] pointer-events-none" 
         style="background-image: radial-gradient(circle at 1rem 1rem, currentColor 1px, transparent 0); background-size: 1rem 1rem;">
    </div>

    <div class="relative z-10 h-full p-6 flex flex-col justify-between">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="p-2.5 rounded-xl transition-colors duration-300" :class="theme.bg">
            <Clock class="w-5 h-5 transition-transform duration-700 ease-in-out group-hover:rotate-[360deg]" :class="theme.text" />
          </div>
          <div>
            <h3 class="text-sm font-bold text-slate-900 dark:text-white uppercase tracking-wider">Avg. Uptime</h3>
            <div class="flex items-center gap-1.5 mt-0.5">
              <Activity class="w-3 h-3 text-slate-400" />
              <span class="text-xs font-medium text-slate-500 dark:text-slate-400">System Stability</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Main Content & Chart -->
      <div class="flex items-end justify-between gap-4 mt-6">
        <!-- Big Metric -->
        <div class="flex-1 min-w-0">
          <div v-if="stats.count > 0">
             <div class="text-xs font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-wide mb-1">Time Active</div>
             <div class="text-3xl sm:text-4xl font-black tabular-nums tracking-tight text-slate-900 dark:text-white leading-none">
               {{ stats.formatted }}
             </div>
             <div class="mt-2 text-xs font-medium text-slate-500 dark:text-slate-400 flex items-center gap-1.5">
               <Zap class="w-3.5 h-3.5 text-violet-500" />
               <span>Across {{ stats.count }} active containers</span>
             </div>
           </div>
           
           <div v-else>
              <div class="text-sm text-slate-500 dark:text-slate-400 leading-relaxed max-w-[14ch]">
                 No running containers
              </div>
              <div class="mt-2 text-[10px] font-medium px-2 py-0.5 rounded bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400 inline-block">
                 System Idle
              </div>
           </div>
        </div>

        <!-- Bar Visual -->
        <div class="relative shrink-0 w-24 h-16 flex items-end justify-between gap-1 pb-1">
          <div
            v-for="(h, i) in bars"
            :key="i"
            class="w-1.5 rounded-t-sm transition-all duration-500 ease-out group-hover:opacity-100 opacity-60"
            :class="theme.bar"
            :style="{ 
              height: `${h * 100}%`, 
              transitionDelay: `${i * 30}ms` 
            }"
          ></div>
          
          <!-- Activity Line (Fake Trend) -->
          <svg class="absolute inset-0 w-full h-full pointer-events-none opacity-30" viewBox="0 0 100 64" preserveAspectRatio="none">
             <path 
               d="M0 50 C 20 50, 20 10, 40 30 S 60 50, 80 20 L 100 40" 
               fill="none" 
               stroke="currentColor" 
               stroke-width="2" 
               class="text-slate-900 dark:text-white"
             />
          </svg>
        </div>
      </div>
    </div>
  </div>
</template>
