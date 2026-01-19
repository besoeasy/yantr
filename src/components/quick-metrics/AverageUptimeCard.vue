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
  <div class="group relative rounded-2xl border border-gray-100 bg-white p-4 sm:p-5 transition-all duration-300 hover:shadow-xl hover:shadow-gray-900/5">
    <div class="absolute inset-0 rounded-2xl bg-linear-to-br from-gray-50/0 to-gray-50/0 group-hover:from-gray-50/60 group-hover:to-gray-50/20 transition-all duration-300 pointer-events-none"></div>

    <div class="relative z-10">
      <div class="flex items-start justify-between gap-3 mb-4">
        <div class="flex items-center gap-2">
          <div class="p-2 rounded-xl bg-gray-50 text-gray-700 border border-gray-100">
            <Clock :size="18" />
          </div>
          <div>
            <h3 class="text-sm font-bold text-gray-900">Average Uptime</h3>
            <p class="text-xs text-gray-500">Running containers only</p>
          </div>
        </div>
      </div>

      <div v-if="averageUptime.count === 0" class="rounded-xl border border-dashed border-gray-200 bg-gray-50/50 px-4 py-6 text-center">
        <div class="text-sm font-semibold text-gray-700">No running containers</div>
        <div class="text-xs text-gray-500 mt-1">Start a container to track uptime.</div>
      </div>

      <div v-else class="flex flex-col items-center justify-center py-2">
        <div class="text-4xl font-extrabold text-transparent bg-clip-text bg-linear-to-r from-purple-600 to-blue-600 tabular-nums">
          {{ averageUptime.formatted }}
        </div>
        <div class="text-xs text-gray-500 font-medium mt-1">
          Across {{ averageUptime.count }} running {{ averageUptime.count === 1 ? 'container' : 'containers' }}
        </div>

        <div class="mt-5 w-full">
          <div class="flex items-end justify-center gap-1.5">
            <div
              v-for="i in 10"
              :key="i"
              class="w-2 rounded-full bg-linear-to-t from-purple-200 to-purple-500"
              :class="i <= 4 ? 'animate-pulse' : ''"
              :style="{ height: `${14 + ((i * 7) % 18)}px` }"
            ></div>
          </div>
          <div v-if="averageUptime.count > 10" class="text-center text-xs text-gray-400 mt-2">
            +{{ averageUptime.count - 10 }} more
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
