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
  <div v-if="expiringContainers.count > 0" class="group relative rounded-2xl border border-gray-100 bg-white p-4 sm:p-5 transition-all duration-300 hover:shadow-xl hover:shadow-gray-900/5">
    <div class="absolute inset-0 rounded-2xl bg-linear-to-br from-orange-50/0 to-red-50/0 group-hover:from-orange-50/70 group-hover:to-red-50/40 transition-all duration-300 pointer-events-none"></div>

    <div class="relative z-10">
      <div class="flex items-start justify-between gap-3 mb-4">
        <div class="flex items-center gap-2">
          <div class="p-2 rounded-xl bg-orange-50 text-orange-700 border border-orange-100">
            <Timer :size="18" />
          </div>
          <div>
            <h3 class="text-sm font-bold text-gray-900">Expiring Containers</h3>
            <p class="text-xs text-gray-500">Temporary instances</p>
          </div>
        </div>
        <div class="inline-flex items-center gap-2 rounded-xl bg-orange-50 border border-orange-100 px-3 py-1.5">
          <span class="text-xs font-semibold text-orange-700">Total</span>
          <span class="text-xs font-extrabold text-orange-800 tabular-nums">{{ expiringContainers.count }}</span>
        </div>
      </div>

      <div class="rounded-2xl border border-orange-100 bg-white/70 p-4">
        <div class="flex items-center justify-between gap-3">
          <div>
            <div class="text-xs text-gray-500 font-medium">Next to expire</div>
            <div
              class="text-2xl font-extrabold mt-1 tabular-nums"
              :class="expiringContainers.isExpiringSoon ? 'text-red-600 animate-pulse' : 'text-orange-700'"
            >
              {{ expiringContainers.nextExpiryFormatted }}
            </div>
          </div>
        </div>
        <div class="text-xs text-gray-600 mt-2 truncate" :title="expiringContainers.containerName">
          {{ expiringContainers.containerName }}
        </div>
      </div>

      <div class="mt-4 rounded-2xl border border-gray-100 bg-white/60 p-4" v-if="expiringTop.items.length > 0">
        <div class="flex items-center justify-between gap-3 mb-3">
          <div>
            <div class="text-sm font-bold text-gray-900">Next 4 expiring</div>
            <div class="text-xs text-gray-500">Time remaining (minutes)</div>
          </div>
          <div class="text-xs font-semibold text-gray-500">Hover bars for exact</div>
        </div>

        <HorizontalBarChart
          :values="expiringTop.valuesMinutes"
          :categories="expiringTop.categories"
          :height="150"
          :colors="['#fb923c', '#f97316', '#ef4444', '#f59e0b']"
          :value-formatter="formatMinutesAsDuration"
        />

        <div class="mt-3 space-y-2">
          <div v-for="item in expiringTop.items" :key="item.name" class="flex items-center justify-between gap-3 text-xs">
            <div class="min-w-0">
              <div class="text-gray-700 font-semibold truncate" :title="item.name">{{ item.name }}</div>
            </div>
            <div class="shrink-0 font-bold tabular-nums" :class="item.isSoon ? 'text-red-600' : item.isExpired ? 'text-gray-400' : 'text-orange-700'">
              {{ item.remainingLabel }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
