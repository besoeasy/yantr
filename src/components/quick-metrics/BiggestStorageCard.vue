<script setup>
import { computed } from 'vue'
import { HardDrive } from 'lucide-vue-next'
import TreemapChart from '../charts/TreemapChart.vue'
import { formatBytes, pickImageDisplayName } from '../../utils/metrics.js'

const props = defineProps({
  images: { type: Array, default: () => [] }
})

const biggestImages = computed(() => {
  const rows = Array.isArray(props.images) ? props.images : []
  return [...rows]
    .filter((img) => (Number(img?.sizeBytes) || 0) > 0)
    .sort((a, b) => (Number(b?.sizeBytes) || 0) - (Number(a?.sizeBytes) || 0))
    .slice(0, 10)
})

const treemapData = computed(() => {
  if (biggestImages.value.length === 0) return []
  return biggestImages.value.map((img) => ({
    x: pickImageDisplayName(img),
    y: Math.max(1, Number(img?.sizeBytes) || 0)
  }))
})

const totalTop10Bytes = computed(() => biggestImages.value.reduce((sum, img) => sum + (Number(img?.sizeBytes) || 0), 0))
</script>

<template>
  <div class="group relative rounded-2xl border border-gray-100 bg-white p-4 sm:p-5 transition-all duration-300 hover:shadow-xl hover:shadow-gray-900/5">
    <div class="absolute inset-0 rounded-2xl bg-linear-to-br from-slate-50/0 to-indigo-50/0 group-hover:from-slate-50/50 group-hover:to-indigo-50/30 transition-all duration-300 pointer-events-none"></div>

    <div class="relative z-10">
      <div class="flex items-start justify-between gap-3 mb-4">
        <div class="flex items-center gap-2">
          <div class="p-2 rounded-xl bg-gray-50 text-gray-700 border border-gray-100">
            <HardDrive :size="18" />
          </div>
          <div>
            <h3 class="text-sm font-bold text-gray-900">Biggest Storage</h3>
            <p class="text-xs text-gray-500">Top 10 largest images</p>
          </div>
        </div>

        <div class="text-right">
          <div class="text-[11px] text-gray-500 font-medium">Top 10 total</div>
          <div class="text-sm font-extrabold text-gray-900 tabular-nums">{{ formatBytes(totalTop10Bytes) }}</div>
        </div>
      </div>

      <div v-if="treemapData.length === 0" class="rounded-2xl border border-dashed border-gray-200 bg-gray-50/50 px-4 py-8 text-center">
        <div class="text-sm font-semibold text-gray-700">No image data</div>
        <div class="text-xs text-gray-500 mt-1">Start a container to pull images.</div>
      </div>

      <div v-else class="rounded-2xl border border-gray-100 bg-white/60 p-3">
        <TreemapChart :data="treemapData" :height="240" :value-formatter="formatBytes" />
        <div class="mt-2 text-[11px] text-gray-400">Sized by image bytes</div>
      </div>
    </div>
  </div>
</template>
