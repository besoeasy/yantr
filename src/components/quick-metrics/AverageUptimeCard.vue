<script setup>
import { computed } from 'vue'
import { Clock } from 'lucide-vue-next'

const props = defineProps({
  containers: { type: Array, default: () => [] },
  currentTime: { type: Number, default: () => Date.now() }
})

const averageUptime = computed(() => {
  const runningContainers = props.containers.filter(c => c.state === 'running' && c.created)

  if (runningContainers.length === 0) {
    return { formatted: 'N/A', count: 0 }
  }

  const totalUptime = runningContainers.reduce((sum, container) => {
    const createdTime = container.created * 1000
    const uptime = props.currentTime - createdTime
    return sum + uptime
  }, 0)

  const avgUptime = totalUptime / runningContainers.length

  const days = Math.floor(avgUptime / (1000 * 60 * 60 * 24))
  const hours = Math.floor((avgUptime % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60))
  const minutes = Math.floor((avgUptime % (1000 * 60 * 60)) / (1000 * 60))

  let formatted = ''
  if (days > 0) {
    formatted = `${days}d ${hours}h`
  } else if (hours > 0) {
    formatted = `${hours}h ${minutes}m`
  } else {
    formatted = `${minutes}m`
  }

  return {
    formatted,
    count: runningContainers.length
  }
})
</script>

<template>
  <div class="relative h-full overflow-hidden group rounded-2xl transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)] hover:-translate-y-1">
    <div class="absolute inset-0 bg-white dark:bg-gray-900">
      <div class="absolute inset-0 bg-linear-to-br from-purple-200/60 via-blue-200/30 to-white/80 dark:from-purple-600/25 dark:via-blue-600/10 dark:to-gray-900 z-10"></div>
      <div class="absolute top-0 right-0 w-64 h-64 bg-purple-300/35 dark:bg-purple-500/20 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2 group-hover:bg-purple-400/45 dark:group-hover:bg-purple-500/30 transition-colors duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"></div>
      <div class="absolute bottom-0 left-0 w-48 h-48 bg-blue-300/30 dark:bg-blue-600/20 rounded-full blur-3xl translate-y-1/2 -translate-x-1/2 group-hover:bg-blue-400/40 dark:group-hover:bg-blue-600/30 transition-colors duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"></div>
    </div>

    <div class="relative z-20 h-full p-6 flex flex-col justify-between border border-slate-200/80 dark:border-slate-700/60 rounded-2xl backdrop-blur-sm group-hover:border-purple-300/60 dark:group-hover:border-purple-500/30 transition-none">
      <div class="flex items-start justify-between gap-4">
        <div class="flex items-center gap-4">
          <div class="relative">
            <div class="absolute inset-0 bg-purple-400/25 dark:bg-purple-500/20 rounded-xl blur-lg group-hover:blur-xl transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"></div>
            <div class="relative w-12 h-12 bg-linear-to-br from-purple-500 to-blue-600 rounded-xl flex items-center justify-center shadow-lg group-hover:scale-110 transition-transform duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]">
              <Clock class="w-6 h-6 text-white" />
            </div>
          </div>

          <div>
            <h3 class="text-lg font-bold text-slate-900 dark:text-white mb-1 group-hover:text-purple-700 dark:group-hover:text-purple-200 transition-colors">Average Uptime</h3>
            <p class="text-sm font-medium text-slate-600 dark:text-gray-400 group-hover:text-slate-700 dark:group-hover:text-gray-300 transition-colors">Running containers only</p>
          </div>
        </div>
      </div>

      <div class="mt-5">
        <div v-if="averageUptime.count === 0" class="rounded-2xl border border-dashed border-slate-200 dark:border-slate-700/60 bg-white/70 dark:bg-white/5 px-4 py-6 text-center">
          <div class="text-sm font-semibold text-slate-700 dark:text-gray-200">No running containers</div>
          <div class="text-xs text-slate-500 dark:text-gray-400 mt-1">Start a container to track uptime.</div>
        </div>

        <div v-else class="flex flex-col items-center justify-center py-1">
          <div class="text-4xl font-extrabold text-transparent bg-clip-text bg-linear-to-r from-purple-600 to-blue-600 dark:from-purple-200 dark:to-blue-200 tabular-nums">
            {{ averageUptime.formatted }}
          </div>
          <div class="text-xs text-slate-500 dark:text-gray-400 font-medium mt-1">
            Across {{ averageUptime.count }} running {{ averageUptime.count === 1 ? 'container' : 'containers' }}
          </div>

          <div class="mt-5 w-full">
            <div class="flex items-end justify-center gap-1.5">
              <div
                v-for="i in 10"
                :key="i"
                class="w-2 rounded-full bg-linear-to-t from-purple-400/50 to-purple-200 dark:from-purple-500/30 dark:to-purple-300"
                :class="i <= 4 ? 'animate-pulse' : ''"
                :style="{ height: `${14 + ((i * 7) % 18)}px` }"
              ></div>
            </div>
            <div v-if="averageUptime.count > 10" class="text-center text-xs text-slate-500 dark:text-gray-400 mt-2">
              +{{ averageUptime.count - 10 }} more
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
