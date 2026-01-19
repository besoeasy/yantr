<script setup>
import { computed } from 'vue'
import { Timer } from 'lucide-vue-next'
import HorizontalBarChart from '../charts/HorizontalBarChart.vue'
import { formatDuration, formatMinutesAsDuration } from '../../utils/metrics.js'

const props = defineProps({
  containers: { type: Array, default: () => [] },
  currentTime: { type: Number, default: () => Date.now() }
})

const expiringContainers = computed(() => {
  const tempContainers = props.containers.filter(c => c?.labels && c.labels['yantra.expireAt'])

  if (tempContainers.length === 0) {
    return { count: 0, nextExpiryFormatted: null, containerName: null, isExpiringSoon: false }
  }

  let soonest = null
  let soonestContainer = null

  tempContainers.forEach(container => {
    const expireAt = parseInt(container.labels['yantra.expireAt'], 10) * 1000
    if (!soonest || expireAt < soonest) {
      soonest = expireAt
      soonestContainer = container
    }
  })

  const remaining = soonest - props.currentTime

  return {
    count: tempContainers.length,
    nextExpiryFormatted: formatDuration(remaining),
    containerName: soonestContainer?.app?.name || soonestContainer?.name || 'Unknown',
    isExpiringSoon: remaining > 0 && remaining < (60 * 60 * 1000)
  }
})

const expiringTop = computed(() => {
  const tempContainers = props.containers
    .filter(c => c?.labels && c.labels['yantra.expireAt'])
    .map((c) => {
      const expireAtSec = parseInt(c.labels['yantra.expireAt'], 10)
      const expireAt = Number.isFinite(expireAtSec) ? expireAtSec * 1000 : null
      if (!expireAt) return null
      const remainingMs = expireAt - props.currentTime
      return {
        name: c?.app?.name || c?.name || 'Unknown',
        expireAt,
        remainingMs,
        remainingLabel: formatDuration(remainingMs),
        isExpired: remainingMs <= 0,
        isSoon: remainingMs > 0 && remainingMs < (60 * 60 * 1000)
      }
    })
    .filter(Boolean)
    .sort((a, b) => a.expireAt - b.expireAt)

  const top = tempContainers.slice(0, 4)
  const valuesMinutes = top.map((t) => Math.max(0, Math.round((t.remainingMs || 0) / 60000)))
  const categories = top.map((t) => t.name)

  return {
    items: top,
    valuesMinutes,
    categories
  }
})
</script>

<template>
  <div v-if="expiringContainers.count > 0" class="relative h-full overflow-hidden group rounded-2xl transition-all duration-500 hover:-translate-y-1">
    <div class="absolute inset-0 bg-gray-900">
      <div class="absolute inset-0 bg-linear-to-br from-orange-600/25 via-red-600/10 to-gray-900 z-10"></div>
      <div class="absolute top-0 right-0 w-64 h-64 bg-orange-500/20 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2 group-hover:bg-orange-500/30 transition-colors duration-700"></div>
      <div class="absolute bottom-0 left-0 w-48 h-48 bg-red-600/20 rounded-full blur-3xl translate-y-1/2 -translate-x-1/2 group-hover:bg-red-600/30 transition-colors duration-700"></div>
    </div>

    <div class="relative z-20 h-full p-6 flex flex-col justify-between border border-white/5 rounded-2xl backdrop-blur-sm group-hover:border-orange-500/30 transition-colors duration-500">
      <div class="flex items-start justify-between gap-4">
        <div class="flex items-center gap-4">
          <div class="relative">
            <div class="absolute inset-0 bg-orange-500/20 rounded-xl blur-lg group-hover:blur-xl transition-all duration-500"></div>
            <div class="relative w-12 h-12 bg-linear-to-br from-orange-500 to-red-600 rounded-xl flex items-center justify-center shadow-lg group-hover:scale-110 transition-transform duration-500">
              <Timer class="w-6 h-6 text-white" />
            </div>
          </div>

          <div>
            <h3 class="text-lg font-bold text-white mb-1 group-hover:text-orange-200 transition-colors">Expiring Containers</h3>
            <p class="text-sm font-medium text-gray-400 group-hover:text-gray-300 transition-colors">Temporary instances</p>
          </div>
        </div>

        <div class="inline-flex items-center gap-2 rounded-xl bg-white/5 border border-white/10 px-3 py-1.5">
          <span class="text-xs font-semibold text-gray-300">Total</span>
          <span class="text-xs font-extrabold text-white tabular-nums">{{ expiringContainers.count }}</span>
        </div>
      </div>

      <div class="mt-5 rounded-2xl border border-white/10 bg-white/5 p-4" v-if="expiringTop.items.length > 0">
        <div class="flex items-center justify-between gap-3 mb-3">
          <div>
            <div class="text-sm font-bold text-white">Next 4 expiring</div>
            <div class="text-xs text-gray-400">Time remaining (minutes)</div>
          </div>
          <div class="text-xs font-semibold text-gray-400">Hover bars for exact</div>
        </div>

        <HorizontalBarChart
          :values="expiringTop.valuesMinutes"
          :categories="expiringTop.categories"
          :height="150"
          theme="dark"
          :colors="['#fb923c', '#f97316', '#ef4444', '#f59e0b']"
          :value-formatter="formatMinutesAsDuration"
        />

        <div class="mt-3 space-y-2">
          <div v-for="item in expiringTop.items" :key="item.name" class="flex items-center justify-between gap-3 text-xs">
            <div class="min-w-0">
              <div class="text-gray-200 font-semibold truncate" :title="item.name">{{ item.name }}</div>
            </div>
            <div class="shrink-0 font-bold tabular-nums" :class="item.isSoon ? 'text-red-300' : item.isExpired ? 'text-gray-500' : 'text-orange-200'">
              {{ item.remainingLabel }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
