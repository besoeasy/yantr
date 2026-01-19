<script setup>
import { computed } from 'vue'
import { HardDrive } from 'lucide-vue-next'
import DonutChart from '../charts/DonutChart.vue'
import { formatBytes } from '../../utils/metrics.js'

const props = defineProps({
  images: { type: Array, default: () => [] },
  volumes: { type: Array, default: () => [] }
})

const diskMetrics = computed(() => {
  const totalImagesSize = props.images.reduce((sum, img) => sum + (img.sizeBytes || 0), 0)
  const unusedImagesSize = props.images
    .filter(img => !img.isUsed)
    .reduce((sum, img) => sum + (img.sizeBytes || 0), 0)
  const usedImagesSize = totalImagesSize - unusedImagesSize

  const totalVolumesSize = props.volumes.reduce((sum, vol) => sum + (vol.sizeBytes || 0), 0)
  const unusedVolumesSize = props.volumes
    .filter(vol => !vol.isUsed)
    .reduce((sum, vol) => sum + (vol.sizeBytes || 0), 0)
  const usedVolumesSize = totalVolumesSize - unusedVolumesSize

  return {
    images: {
      total: totalImagesSize,
      used: usedImagesSize,
      unused: unusedImagesSize
    },
    volumes: {
      total: totalVolumesSize,
      used: usedVolumesSize,
      unused: unusedVolumesSize
    }
  }
})

const donuts = computed(() => {
  const images = diskMetrics.value.images
  const volumes = diskMetrics.value.volumes

  return {
    images: {
      total: images.total,
      used: images.used,
      unused: images.unused,
      series: [images.used, images.unused],
      labels: ['Used', 'Unused'],
      colors: ['#3b82f6', '#fdba74']
    },
    volumes: {
      total: volumes.total,
      used: volumes.used,
      unused: volumes.unused,
      series: [volumes.used, volumes.unused],
      labels: ['Used', 'Unused'],
      colors: ['#6366f1', '#fdba74']
    }
  }
})

const totalAll = computed(() => donuts.value.images.total + donuts.value.volumes.total)
</script>

<template>
  <div class="group relative rounded-2xl border border-gray-100 bg-white p-4 sm:p-5 transition-all duration-300 hover:shadow-xl hover:shadow-gray-900/5">
    <div class="absolute inset-0 rounded-2xl bg-linear-to-br from-gray-50/0 to-gray-50/0 group-hover:from-gray-50/60 group-hover:to-gray-50/20 transition-all duration-300 pointer-events-none"></div>

    <div class="relative z-10">
      <div class="flex items-start justify-between gap-3 mb-4">
        <div class="flex items-center gap-2">
          <div class="p-2 rounded-xl bg-gray-50 text-gray-700 border border-gray-100">
            <HardDrive :size="18" />
          </div>
          <div>
            <h3 class="text-sm font-bold text-gray-900">Disk Usage</h3>
            <p class="text-xs text-gray-500">Used vs unused</p>
          </div>
        </div>
        <div class="text-right">
          <div class="text-xs text-gray-500">Total</div>
          <div class="text-sm font-bold text-gray-900">{{ formatBytes(totalAll) }}</div>
        </div>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div class="rounded-2xl border border-gray-100 bg-white/60 p-3">
          <div class="flex items-center justify-between gap-3">
            <div class="text-sm font-semibold text-gray-700">Images</div>
            <div class="text-xs font-bold text-gray-900 tabular-nums">{{ formatBytes(donuts.images.total) }}</div>
          </div>

          <div v-if="donuts.images.total === 0" class="mt-3 rounded-xl border border-dashed border-gray-200 bg-gray-50/50 px-4 py-6 text-center">
            <div class="text-sm font-semibold text-gray-700">No image data</div>
            <div class="text-xs text-gray-500 mt-1">Nothing to visualize yet.</div>
          </div>

          <div v-else class="mt-2">
            <DonutChart
              :series="donuts.images.series"
              :labels="donuts.images.labels"
              :colors="donuts.images.colors"
              :height="185"
              donut-label="Images"
              :value-formatter="formatBytes"
              :total-formatter="() => formatBytes(donuts.images.total)"
            />

            <div class="mt-2 flex items-center justify-between text-xs text-gray-600">
              <span class="inline-flex items-center gap-1.5">
                <span class="w-2 h-2 rounded-full" :style="{ backgroundColor: donuts.images.colors[0] }"></span>
                Used <span class="font-semibold tabular-nums">{{ formatBytes(donuts.images.used) }}</span>
              </span>
              <span class="inline-flex items-center gap-1.5">
                <span class="w-2 h-2 rounded-full" :style="{ backgroundColor: donuts.images.colors[1] }"></span>
                Unused <span class="font-semibold tabular-nums">{{ formatBytes(donuts.images.unused) }}</span>
              </span>
            </div>
          </div>
        </div>

        <div class="rounded-2xl border border-gray-100 bg-white/60 p-3">
          <div class="flex items-center justify-between gap-3">
            <div class="text-sm font-semibold text-gray-700">Volumes</div>
            <div class="text-xs font-bold text-gray-900 tabular-nums">{{ formatBytes(donuts.volumes.total) }}</div>
          </div>

          <div v-if="donuts.volumes.total === 0" class="mt-3 rounded-xl border border-dashed border-gray-200 bg-gray-50/50 px-4 py-6 text-center">
            <div class="text-sm font-semibold text-gray-700">No volume data</div>
            <div class="text-xs text-gray-500 mt-1">Nothing to visualize yet.</div>
          </div>

          <div v-else class="mt-2">
            <DonutChart
              :series="donuts.volumes.series"
              :labels="donuts.volumes.labels"
              :colors="donuts.volumes.colors"
              :height="185"
              donut-label="Volumes"
              :value-formatter="formatBytes"
              :total-formatter="() => formatBytes(donuts.volumes.total)"
            />

            <div class="mt-2 flex items-center justify-between text-xs text-gray-600">
              <span class="inline-flex items-center gap-1.5">
                <span class="w-2 h-2 rounded-full" :style="{ backgroundColor: donuts.volumes.colors[0] }"></span>
                Used <span class="font-semibold tabular-nums">{{ formatBytes(donuts.volumes.used) }}</span>
              </span>
              <span class="inline-flex items-center gap-1.5">
                <span class="w-2 h-2 rounded-full" :style="{ backgroundColor: donuts.volumes.colors[1] }"></span>
                Unused <span class="font-semibold tabular-nums">{{ formatBytes(donuts.volumes.unused) }}</span>
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
