<script setup>
import { computed } from "vue";
import { Layers, Trophy } from "lucide-vue-next";
import TreemapChart from "../charts/TreemapChart.vue";
import { computeCategoryStats, formatCategory } from "../../utils/metrics.js";

const props = defineProps({
  containers: { type: Array, default: () => [] },
});

const categoryStats = computed(() => computeCategoryStats(props.containers));

const categoryTreemap = computed(() => {
  const rows = Array.isArray(categoryStats.value.all) ? categoryStats.value.all : [];
  if (rows.length === 0) return { data: [] };

  const sorted = [...rows].sort((a, b) => (Number(b?.[1]) || 0) - (Number(a?.[1]) || 0));
  const topN = 10;
  const top = sorted.slice(0, topN);
  const rest = sorted.slice(topN);
  const otherCount = rest.reduce((sum, r) => sum + (Number(r?.[1]) || 0), 0);

  const data = top.map((r) => ({ x: formatCategory(r[0]), y: Number(r[1]) || 0 })).filter((d) => d.y > 0);

  if (otherCount > 0) data.push({ x: "Other", y: otherCount });

  return { data };
});
</script>

<template>
  <div v-if="categoryStats.appsCount > 0" class="relative h-full overflow-hidden group rounded-2xl transition-all duration-500 hover:-translate-y-1">
    <div class="absolute inset-0 bg-gray-900">
      <div class="absolute inset-0 bg-linear-to-br from-emerald-600/25 via-green-600/10 to-gray-900 z-10"></div>
      <div
        class="absolute top-0 right-0 w-64 h-64 bg-emerald-500/20 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2 group-hover:bg-emerald-500/30 transition-colors duration-700"
      ></div>
      <div
        class="absolute bottom-0 left-0 w-48 h-48 bg-green-600/20 rounded-full blur-3xl translate-y-1/2 -translate-x-1/2 group-hover:bg-green-600/30 transition-colors duration-700"
      ></div>
    </div>

    <div
      class="relative z-20 h-full p-6 flex flex-col justify-between border border-white/5 rounded-2xl backdrop-blur-sm group-hover:border-emerald-500/30 transition-colors duration-500"
    >
      <div class="flex items-start justify-between gap-4">
        <div class="flex items-center gap-4">
          <div class="relative">
            <div class="absolute inset-0 bg-emerald-500/20 rounded-xl blur-lg group-hover:blur-xl transition-all duration-500"></div>
            <div
              class="relative w-12 h-12 bg-linear-to-br from-emerald-500 to-green-600 rounded-xl flex items-center justify-center shadow-lg group-hover:scale-110 transition-transform duration-500"
            >
              <Layers class="w-6 h-6 text-white" />
            </div>
          </div>

          <div>
            <h3 class="text-lg font-bold text-white mb-1 group-hover:text-emerald-200 transition-colors">App Categories</h3>
            <p class="text-sm font-medium text-gray-400 group-hover:text-gray-300 transition-colors">Usage highlights</p>
          </div>
        </div>

        <div class="text-right">
          <div class="text-xs font-semibold text-gray-400 uppercase tracking-wider">Apps</div>
          <div class="text-xl font-extrabold text-white tabular-nums">{{ categoryStats.appsCount }}</div>
        </div>
      </div>

      <div class="mt-5 space-y-4">
        <div class="rounded-2xl border border-white/10 bg-white/5 p-4">
          <div class="flex items-center justify-between gap-3">
            <div class="text-sm font-bold text-white">Distribution</div>
            <div class="text-xs font-semibold text-gray-300">Top categories</div>
          </div>

          <div v-if="categoryTreemap.data.length === 0" class="mt-3 rounded-xl border border-dashed border-white/10 bg-white/5 px-4 py-6 text-center">
            <div class="text-sm font-semibold text-gray-200">No category data</div>
            <div class="text-xs text-gray-400 mt-1">Apps need categories to chart.</div>
          </div>

          <div v-else class="mt-3">
            <TreemapChart :data="categoryTreemap.data" :height="220" theme="dark" :value-formatter="(v) => `${v} app${v === 1 ? '' : 's'}`" />
            <div class="mt-2 text-[11px] text-gray-400">Top {{ Math.min(10, categoryStats.total) }} categories (plus “Other”)</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
