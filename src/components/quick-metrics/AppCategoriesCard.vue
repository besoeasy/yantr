<script setup>
import { computed } from 'vue'
import { Layers, Trophy } from 'lucide-vue-next'
import TreemapChart from '../charts/TreemapChart.vue'
import { computeCategoryStats, formatCategory } from '../../utils/metrics.js'

const props = defineProps({
  containers: { type: Array, default: () => [] }
})

const categoryStats = computed(() => computeCategoryStats(props.containers))

const categoryTreemap = computed(() => {
  const rows = Array.isArray(categoryStats.value.all) ? categoryStats.value.all : []
  if (rows.length === 0) return { data: [] }

  const sorted = [...rows].sort((a, b) => (Number(b?.[1]) || 0) - (Number(a?.[1]) || 0))
  const topN = 10
  const top = sorted.slice(0, topN)
  const rest = sorted.slice(topN)
  const otherCount = rest.reduce((sum, r) => sum + (Number(r?.[1]) || 0), 0)

  const data = top
    .map((r) => ({ x: formatCategory(r[0]), y: Number(r[1]) || 0 }))
    .filter((d) => d.y > 0)

  if (otherCount > 0) data.push({ x: 'Other', y: otherCount })

  return { data }
})
</script>

<template>
  <div v-if="categoryStats.appsCount > 0" class="group relative rounded-2xl border border-gray-100 bg-white p-4 sm:p-5 transition-all duration-300 hover:shadow-xl hover:shadow-gray-900/5">
    <div class="absolute inset-0 rounded-2xl bg-linear-to-br from-green-50/0 to-emerald-50/0 group-hover:from-green-50/60 group-hover:to-emerald-50/30 transition-all duration-300 pointer-events-none"></div>

    <div class="relative z-10">
      <div class="flex items-start justify-between gap-3 mb-4">
        <div class="flex items-center gap-2">
          <div class="p-2 rounded-xl bg-gray-50 text-gray-700 border border-gray-100">
            <Layers :size="18" />
          </div>
          <div>
            <h3 class="text-sm font-bold text-gray-900">App Categories</h3>
            <p class="text-xs text-gray-500">Usage highlights</p>
          </div>
        </div>
      </div>

      <div class="space-y-4">
        <div class="rounded-2xl border border-gray-100 bg-white/60 p-3">
          <div class="flex items-center justify-between gap-3">
            <div class="text-sm font-semibold text-gray-700">Distribution</div>
            <div class="text-xs font-bold text-gray-900 tabular-nums">{{ categoryStats.appsCount }} apps</div>
          </div>

          <div v-if="categoryTreemap.data.length === 0" class="mt-3 rounded-xl border border-dashed border-gray-200 bg-gray-50/50 px-4 py-6 text-center">
            <div class="text-sm font-semibold text-gray-700">No category data</div>
            <div class="text-xs text-gray-500 mt-1">Apps need categories to chart.</div>
          </div>

          <div v-else class="mt-2">
            <TreemapChart :data="categoryTreemap.data" :height="220" :value-formatter="(v) => `${v} app${v === 1 ? '' : 's'}`" />
            <div class="mt-2 text-[11px] text-gray-400">Top {{ Math.min(10, categoryStats.total) }} categories (plus “Other”)</div>
          </div>
        </div>

        <div v-if="categoryStats.mostUsed" class="rounded-2xl border border-green-100 bg-white/70 p-4">
          <div class="flex items-center justify-between gap-3">
            <div class="flex items-center gap-2 min-w-0">
              <div class="p-1.5 rounded-lg bg-green-50 text-green-700 border border-green-100">
                <Trophy :size="16" />
              </div>
              <div class="min-w-0">
                <div class="text-xs text-gray-500 font-medium">Most used</div>
                <div class="text-sm font-extrabold text-green-700 truncate">
                  {{ formatCategory(categoryStats.mostUsed.category) }}
                </div>
              </div>
            </div>
            <div class="text-sm font-bold text-gray-900 tabular-nums">{{ categoryStats.mostUsed.count }}</div>
          </div>
          <div class="text-xs text-gray-500 mt-2">
            {{ categoryStats.mostUsed.count === 1 ? 'app' : 'apps' }} in this category
          </div>
        </div>

        <div
          v-if="categoryStats.leastUsed && categoryStats.total > 1 && categoryStats.leastUsed.category !== categoryStats.mostUsed?.category"
          class="rounded-2xl border border-blue-100 bg-white/70 p-4"
        >
          <div class="flex items-center justify-between gap-3">
            <div class="min-w-0">
              <div class="text-xs text-gray-500 font-medium">Least used</div>
              <div class="text-sm font-extrabold text-blue-700 truncate">
                {{ formatCategory(categoryStats.leastUsed.category) }}
              </div>
            </div>
            <div class="text-sm font-bold text-gray-900 tabular-nums">{{ categoryStats.leastUsed.count }}</div>
          </div>
          <div class="text-xs text-gray-500 mt-2">
            {{ categoryStats.leastUsed.count === 1 ? 'app' : 'apps' }} in this category
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
