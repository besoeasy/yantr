<script setup>
import { computed } from 'vue'
import { HardDrive } from 'lucide-vue-next'
import DonutChart from '../charts/DonutChart.vue'
import { formatBytes } from '../../utils/metrics.js'

const props = defineProps({
  volumes: { type: Array, default: () => [] }
})

const metrics = computed(() => {
  const total = props.volumes.reduce((sum, vol) => sum + (vol.sizeBytes || 0), 0)
  const unused = props.volumes.filter((vol) => !vol.isUsed).reduce((sum, vol) => sum + (vol.sizeBytes || 0), 0)
  const used = total - unused

  return { total, used, unused }
})

const donut = computed(() => {
  const value = metrics.value

  return {
    total: value.total,
    used: value.used,
    unused: value.unused,
    series: [value.used, value.unused],
    labels: ['Used', 'Unused'],
    colors: ['#6366f1', '#fdba74']
  }
})

const usedPercent = computed(() => {
  const total = Number(donut.value.total) || 0
  const used = Number(donut.value.used) || 0
  if (!total) return 0
  return Math.max(0, Math.min(100, Math.round((used / total) * 100)))
})

const unusedPercent = computed(() => {
  const total = Number(donut.value.total) || 0
  const unused = Number(donut.value.unused) || 0
  if (!total) return 0
  return Math.max(0, Math.min(100, Math.round((unused / total) * 100)))
})
</script>

<template>
  <div class="relative h-full overflow-hidden group rounded-2xl transition-all duration-500 hover:-translate-y-1">
    <div class="absolute inset-0 bg-gray-900">
      <div class="absolute inset-0 bg-linear-to-br from-indigo-600/25 via-purple-600/10 to-gray-900 z-10"></div>
      <div
        class="absolute top-0 right-0 w-64 h-64 bg-indigo-500/20 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2 group-hover:bg-indigo-500/30 transition-colors duration-700"
      ></div>
      <div
        class="absolute bottom-0 left-0 w-48 h-48 bg-purple-600/20 rounded-full blur-3xl translate-y-1/2 -translate-x-1/2 group-hover:bg-purple-600/30 transition-colors duration-700"
      ></div>
    </div>

    <div
      class="relative z-20 h-full p-6 flex flex-col justify-between border border-white/5 rounded-2xl backdrop-blur-sm group-hover:border-indigo-500/30 transition-colors duration-500"
    >
      <div class="flex items-start justify-between gap-4">
        <div class="flex items-center gap-4">
          <div class="relative">
            <div class="absolute inset-0 bg-indigo-500/20 rounded-xl blur-lg group-hover:blur-xl transition-all duration-500"></div>
            <div
              class="relative w-12 h-12 bg-linear-to-br from-indigo-500 to-purple-600 rounded-xl flex items-center justify-center shadow-lg group-hover:scale-110 transition-transform duration-500"
            >
              <HardDrive class="w-6 h-6 text-white" />
            </div>
          </div>

          <div>
            <h3 class="text-lg font-bold text-white mb-1 group-hover:text-indigo-200 transition-colors">Volume Usage</h3>
            <p class="text-sm font-medium text-gray-400 group-hover:text-gray-300 transition-colors">Used vs unused storage</p>
          </div>
        </div>

        <div class="inline-flex items-center gap-2 rounded-xl bg-white/5 border border-white/10 px-3 py-1.5">
          <span class="text-xs font-semibold text-gray-300">Total</span>
          <span class="text-xs font-extrabold text-white tabular-nums">{{ formatBytes(donut.total) }}</span>
        </div>
      </div>

      <div class="mt-5">
        <div v-if="donut.total === 0" class="rounded-2xl border border-dashed border-white/10 bg-white/5 px-4 py-6 text-center">
          <div class="text-sm font-semibold text-gray-200">No volume data</div>
          <div class="text-xs text-gray-400 mt-1">Nothing to visualize yet.</div>
        </div>

        <div v-else class="rounded-2xl border border-white/10 bg-white/5 p-4">
          <div class="flex items-center justify-between gap-3 mb-3">
            <div>
              <div class="text-sm font-bold text-white">Breakdown</div>
              <div class="text-xs text-gray-400">Storage used by volumes</div>
            </div>
            <div class="text-xs font-semibold text-gray-400">Hover chart for exact</div>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-5 gap-4 items-center">
            <div class="sm:col-span-2">
              <DonutChart
                :series="donut.series"
                :labels="donut.labels"
                :colors="donut.colors"
                :height="180"
                donut-label="Volumes"
                theme="dark"
                :value-formatter="formatBytes"
                :total-formatter="() => formatBytes(donut.total)"
              />
            </div>

            <div class="sm:col-span-3 space-y-3">
              <div class="rounded-xl border border-white/10 bg-white/5 px-3 py-2">
                <div class="flex items-center justify-between gap-3 text-xs">
                  <div class="inline-flex items-center gap-2 min-w-0">
                    <span class="w-2 h-2 rounded-full" :style="{ backgroundColor: donut.colors[0] }"></span>
                    <span class="text-gray-300 font-semibold">Used</span>
                  </div>
                  <div class="shrink-0 text-gray-200 font-bold tabular-nums">
                    {{ formatBytes(donut.used) }}
                    <span class="text-gray-400 font-semibold">({{ usedPercent }}%)</span>
                  </div>
                </div>
                <div class="mt-2 h-2 rounded-full bg-black/20 overflow-hidden">
                  <div class="h-full rounded-full" :style="{ width: `${usedPercent}%`, backgroundColor: donut.colors[0] }"></div>
                </div>
              </div>

              <div class="rounded-xl border border-white/10 bg-white/5 px-3 py-2">
                <div class="flex items-center justify-between gap-3 text-xs">
                  <div class="inline-flex items-center gap-2 min-w-0">
                    <span class="w-2 h-2 rounded-full" :style="{ backgroundColor: donut.colors[1] }"></span>
                    <span class="text-gray-300 font-semibold">Unused</span>
                  </div>
                  <div class="shrink-0 text-gray-200 font-bold tabular-nums">
                    {{ formatBytes(donut.unused) }}
                    <span class="text-gray-400 font-semibold">({{ unusedPercent }}%)</span>
                  </div>
                </div>
                <div class="mt-2 h-2 rounded-full bg-black/20 overflow-hidden">
                  <div class="h-full rounded-full" :style="{ width: `${unusedPercent}%`, backgroundColor: donut.colors[1] }"></div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
