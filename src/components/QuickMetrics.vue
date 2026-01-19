<script setup>
import { computed } from 'vue'
import { Activity, Package, HardDrive, Clock, Timer, Layers, Trophy } from 'lucide-vue-next'
import DonutChart from './charts/DonutChart.vue'

const props = defineProps({
  containers: {
    type: Array,
    default: () => []
  },
  volumes: {
    type: Array,
    default: () => []
  },
  images: {
    type: Array,
    default: () => []
  },
  currentTime: {
    type: Number,
    default: () => Date.now()
  }
})

// Container State Breakdown
const containersByState = computed(() => {
  const states = {
    running: 0,
    stopped: 0,
    paused: 0,
    exited: 0,
    other: 0
  }
  
  props.containers.forEach(container => {
    const state = container.state?.toLowerCase() || 'other'
    if (states.hasOwnProperty(state)) {
      states[state]++
    } else if (state === 'created' || state === 'restarting') {
      states.stopped++
    } else {
      states.other++
    }
  })
  
  return states
})

const totalContainers = computed(() => props.containers.length)

const hasContainers = computed(() => totalContainers.value > 0)

const containerStatePercentages = computed(() => {
  if (totalContainers.value === 0) return {}
  
  return {
    running: (containersByState.value.running / totalContainers.value) * 100,
    stopped: (containersByState.value.stopped / totalContainers.value) * 100,
    paused: (containersByState.value.paused / totalContainers.value) * 100,
    exited: (containersByState.value.exited / totalContainers.value) * 100
  }
})

const containerStateRows = computed(() => {
  const pct = containerStatePercentages.value

  return [
    {
      key: 'running',
      label: 'Running',
      count: containersByState.value.running,
      percent: pct.running || 0,
      dotClass: 'bg-green-500',
      barClass: 'bg-linear-to-r from-green-500 to-green-400'
    },
    {
      key: 'stopped',
      label: 'Stopped',
      count: containersByState.value.stopped,
      percent: pct.stopped || 0,
      dotClass: 'bg-gray-400',
      barClass: 'bg-gray-400'
    },
    {
      key: 'exited',
      label: 'Exited',
      count: containersByState.value.exited,
      percent: pct.exited || 0,
      dotClass: 'bg-red-500',
      barClass: 'bg-red-500'
    },
    {
      key: 'paused',
      label: 'Paused',
      count: containersByState.value.paused,
      percent: pct.paused || 0,
      dotClass: 'bg-orange-500',
      barClass: 'bg-orange-500'
    }
  ]
})

// Disk Space Breakdown
const diskMetrics = computed(() => {
  // Images
  const totalImagesSize = props.images.reduce((sum, img) => sum + (img.sizeBytes || 0), 0)
  const unusedImagesSize = props.images
    .filter(img => !img.isUsed)
    .reduce((sum, img) => sum + (img.sizeBytes || 0), 0)
  const usedImagesSize = totalImagesSize - unusedImagesSize
  
  // Volumes
  const totalVolumesSize = props.volumes.reduce((sum, vol) => sum + (vol.sizeBytes || 0), 0)
  const unusedVolumesSize = props.volumes
    .filter(vol => !vol.isUsed)
    .reduce((sum, vol) => sum + (vol.sizeBytes || 0), 0)
  const usedVolumesSize = totalVolumesSize - unusedVolumesSize
  
  return {
    images: {
      total: totalImagesSize,
      used: usedImagesSize,
      unused: unusedImagesSize,
      usedPercent: totalImagesSize > 0 ? (usedImagesSize / totalImagesSize) * 100 : 0,
      unusedPercent: totalImagesSize > 0 ? (unusedImagesSize / totalImagesSize) * 100 : 0
    },
    volumes: {
      total: totalVolumesSize,
      used: usedVolumesSize,
      unused: unusedVolumesSize,
      usedPercent: totalVolumesSize > 0 ? (usedVolumesSize / totalVolumesSize) * 100 : 0,
      unusedPercent: totalVolumesSize > 0 ? (unusedVolumesSize / totalVolumesSize) * 100 : 0
    }
  }
})

// Average Container Uptime
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
  
  // Format average uptime
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

// Expiring Containers Metrics
const expiringContainers = computed(() => {
  const tempContainers = props.containers.filter(c => 
    c.labels && c.labels['yantra.expireAt']
  )
  
  if (tempContainers.length === 0) {
    return { count: 0, nextExpiry: null, nextExpiryFormatted: null, containerName: null }
  }
  
  // Find the container expiring soonest
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
  
  // Format time remaining
  let formatted = 'Expired'
  if (remaining > 0) {
    const hours = Math.floor(remaining / (1000 * 60 * 60))
    const minutes = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60))
    const seconds = Math.floor((remaining % (1000 * 60)) / 1000)
    
    if (hours > 24) {
      const days = Math.floor(hours / 24)
      formatted = `${days}d ${hours % 24}h`
    } else if (hours > 0) {
      formatted = `${hours}h ${minutes}m`
    } else if (minutes > 0) {
      formatted = `${minutes}m ${seconds}s`
    } else {
      formatted = `${seconds}s`
    }
  }
  
  return {
    count: tempContainers.length,
    nextExpiry: soonest,
    nextExpiryFormatted: formatted,
    containerName: soonestContainer?.app?.name || soonestContainer?.name || 'Unknown',
    isExpiringSoon: remaining > 0 && remaining < (60 * 60 * 1000) // < 1 hour
  }
})

// Category Usage Statistics
// Counts UNIQUE apps per category (an app can belong to multiple categories).
const categoryStats = computed(() => {
  // Build a unique app list from containers (avoid counting multiple containers per app)
  const appsById = new Map()

  props.containers.forEach((container) => {
    const app = container?.app
    if (!app) return
    const id = app.id || app.projectId || app.name || container.id
    if (!appsById.has(id)) {
      appsById.set(id, app)
    }
  })

  const categoryToApps = new Map() // categoryKey -> { label: string, apps: Set<string> }

  for (const [appId, app] of appsById.entries()) {
    const raw = (app?.category || 'uncategorized').toString()
    const parts = raw
      .split(',')
      .map((c) => c.trim())
      .filter(Boolean)

    const categories = parts.length > 0 ? parts : ['uncategorized']

    for (const category of categories) {
      const key = category.trim().toLowerCase()
      if (!categoryToApps.has(key)) {
        categoryToApps.set(key, { label: category, apps: new Set() })
      }
      categoryToApps.get(key).apps.add(appId)
    }
  }

  const entries = Array.from(categoryToApps.values()).map((v) => [v.label, v.apps.size])
  if (entries.length === 0) {
    return { mostUsed: null, leastUsed: null, total: 0, all: [] }
  }

  const sorted = entries.sort((a, b) => b[1] - a[1])
  const most = sorted[0]
  const least = sorted[sorted.length - 1]

  return {
    mostUsed: most ? { category: most[0], count: most[1] } : null,
    leastUsed: least ? { category: least[0], count: least[1] } : null,
    total: sorted.length,
    all: sorted,
    appsCount: appsById.size,
  }
})

const diskDonuts = computed(() => {
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

const categoryDonut = computed(() => {
  const rows = Array.isArray(categoryStats.value.all) ? categoryStats.value.all : []
  if (rows.length === 0) {
    return { series: [], labels: [], colors: [], legend: [] }
  }

  const sorted = [...rows].sort((a, b) => (b?.[1] || 0) - (a?.[1] || 0))
  const topN = 6
  const top = sorted.slice(0, topN)
  const rest = sorted.slice(topN)
  const otherCount = rest.reduce((sum, r) => sum + (Number(r?.[1]) || 0), 0)

  const labels = top.map((r) => formatCategory(r[0]))
  const series = top.map((r) => Number(r[1]) || 0)

  if (otherCount > 0) {
    labels.push('Other')
    series.push(otherCount)
  }

  const colors = ['#22c55e', '#10b981', '#0ea5e9', '#6366f1', '#a855f7', '#f59e0b', '#94a3b8']

  return {
    labels,
    series,
    colors: colors.slice(0, labels.length),
    legend: labels.map((label, idx) => ({ label, value: series[idx], color: colors[idx] }))
  }
})

// Helper function to format bytes
function formatBytes(bytes) {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
}

// Helper function to capitalize category names
function formatCategory(category) {
  if (!category) return 'Unknown'
  const cleaned = String(category).trim().replace(/_/g, ' ').replace(/\s+/g, ' ')

  // Title-case words, but keep short words readable
  return cleaned
    .split(/\s|-/)
    .filter(Boolean)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ')
}
</script>

<template>
  <section class="w-full">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
      <div class="flex items-center gap-3">
        <div class="p-2.5 bg-linear-to-br from-purple-100 to-blue-100 text-purple-700 rounded-2xl shadow-sm">
          <Activity :size="22" />
        </div>
        <div>
          <h2 class="text-xl sm:text-2xl font-bold text-gray-900">Quick Metrics</h2>
          <p class="text-sm text-gray-500">Live snapshot of your Docker resources</p>
        </div>
      </div>

      <div class="flex flex-wrap gap-2 sm:justify-end">
        <div class="inline-flex items-center gap-2 rounded-xl border border-gray-200 bg-white px-3 py-1.5 text-xs font-semibold text-gray-700">
          <span class="text-gray-500">Containers</span>
          <span class="text-gray-900">{{ totalContainers }}</span>
        </div>
        <div class="inline-flex items-center gap-2 rounded-xl border border-gray-200 bg-white px-3 py-1.5 text-xs font-semibold text-gray-700">
          <span class="text-gray-500">Images</span>
          <span class="text-gray-900">{{ images.length }}</span>
        </div>
        <div class="inline-flex items-center gap-2 rounded-xl border border-gray-200 bg-white px-3 py-1.5 text-xs font-semibold text-gray-700">
          <span class="text-gray-500">Volumes</span>
          <span class="text-gray-900">{{ volumes.length }}</span>
        </div>
      </div>
    </div>

    <!-- Metrics Grid -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 sm:gap-6 mt-6">
      <!-- Container States -->
      <div class="group relative rounded-2xl border border-gray-100 bg-white p-4 sm:p-5 transition-all duration-300 hover:shadow-xl hover:shadow-gray-900/5">
        <div class="absolute inset-0 rounded-2xl bg-linear-to-br from-gray-50/0 to-gray-50/0 group-hover:from-gray-50/60 group-hover:to-gray-50/20 transition-all duration-300 pointer-events-none"></div>

        <div class="relative z-10">
          <div class="flex items-start justify-between gap-3 mb-4">
            <div class="flex items-center gap-2">
              <div class="p-2 rounded-xl bg-gray-50 text-gray-700 border border-gray-100">
                <Package :size="18" />
              </div>
              <div>
                <h3 class="text-sm font-bold text-gray-900">Container States</h3>
                <p class="text-xs text-gray-500">Status distribution</p>
              </div>
            </div>

            <div class="text-right">
              <div class="text-xs text-gray-500">Total</div>
              <div class="text-sm font-bold text-gray-900">{{ totalContainers }}</div>
            </div>
          </div>

          <div v-if="!hasContainers" class="rounded-xl border border-dashed border-gray-200 bg-gray-50/50 px-4 py-6 text-center">
            <div class="text-sm font-semibold text-gray-700">No containers</div>
            <div class="text-xs text-gray-500 mt-1">Start a stack to see state metrics.</div>
          </div>

          <div v-else class="space-y-3">
            <div v-for="row in containerStateRows" :key="row.key" v-show="row.count > 0" class="space-y-1.5">
              <div class="flex items-center justify-between gap-3 text-sm">
                <div class="flex items-center gap-2 min-w-0">
                  <div class="w-2.5 h-2.5 rounded-full" :class="[row.dotClass, row.key === 'running' ? 'animate-pulse' : '']"></div>
                  <span class="text-gray-700 font-medium truncate">{{ row.label }}</span>
                </div>
                <div class="flex items-center gap-2 shrink-0">
                  <span class="text-xs text-gray-500 tabular-nums">{{ row.percent.toFixed(0) }}%</span>
                  <span class="font-bold text-gray-900 tabular-nums">{{ row.count }}</span>
                </div>
              </div>

              <div class="w-full bg-gray-100 rounded-full h-1.5 overflow-hidden">
                <div class="h-full rounded-full transition-all duration-500" :class="row.barClass" :style="{ width: `${row.percent}%` }"></div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Disk Breakdown -->
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
              <div class="text-sm font-bold text-gray-900">
                {{ formatBytes(diskMetrics.images.total + diskMetrics.volumes.total) }}
              </div>
            </div>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div class="rounded-2xl border border-gray-100 bg-white/60 p-3">
              <div class="flex items-center justify-between gap-3">
                <div class="text-sm font-semibold text-gray-700">Images</div>
                <div class="text-xs font-bold text-gray-900 tabular-nums">{{ formatBytes(diskDonuts.images.total) }}</div>
              </div>

              <div v-if="diskDonuts.images.total === 0" class="mt-3 rounded-xl border border-dashed border-gray-200 bg-gray-50/50 px-4 py-6 text-center">
                <div class="text-sm font-semibold text-gray-700">No image data</div>
                <div class="text-xs text-gray-500 mt-1">Nothing to visualize yet.</div>
              </div>

              <div v-else class="mt-2">
                <DonutChart
                  :series="diskDonuts.images.series"
                  :labels="diskDonuts.images.labels"
                  :colors="diskDonuts.images.colors"
                  :height="185"
                  donut-label="Images"
                  :value-formatter="formatBytes"
                  :total-formatter="() => formatBytes(diskDonuts.images.total)"
                />

                <div class="mt-2 flex items-center justify-between text-xs text-gray-600">
                  <span class="inline-flex items-center gap-1.5">
                    <span class="w-2 h-2 rounded-full" :style="{ backgroundColor: diskDonuts.images.colors[0] }"></span>
                    Used <span class="font-semibold tabular-nums">{{ formatBytes(diskDonuts.images.used) }}</span>
                  </span>
                  <span class="inline-flex items-center gap-1.5">
                    <span class="w-2 h-2 rounded-full" :style="{ backgroundColor: diskDonuts.images.colors[1] }"></span>
                    Unused <span class="font-semibold tabular-nums">{{ formatBytes(diskDonuts.images.unused) }}</span>
                  </span>
                </div>
              </div>
            </div>

            <div class="rounded-2xl border border-gray-100 bg-white/60 p-3">
              <div class="flex items-center justify-between gap-3">
                <div class="text-sm font-semibold text-gray-700">Volumes</div>
                <div class="text-xs font-bold text-gray-900 tabular-nums">{{ formatBytes(diskDonuts.volumes.total) }}</div>
              </div>

              <div v-if="diskDonuts.volumes.total === 0" class="mt-3 rounded-xl border border-dashed border-gray-200 bg-gray-50/50 px-4 py-6 text-center">
                <div class="text-sm font-semibold text-gray-700">No volume data</div>
                <div class="text-xs text-gray-500 mt-1">Nothing to visualize yet.</div>
              </div>

              <div v-else class="mt-2">
                <DonutChart
                  :series="diskDonuts.volumes.series"
                  :labels="diskDonuts.volumes.labels"
                  :colors="diskDonuts.volumes.colors"
                  :height="185"
                  donut-label="Volumes"
                  :value-formatter="formatBytes"
                  :total-formatter="() => formatBytes(diskDonuts.volumes.total)"
                />

                <div class="mt-2 flex items-center justify-between text-xs text-gray-600">
                  <span class="inline-flex items-center gap-1.5">
                    <span class="w-2 h-2 rounded-full" :style="{ backgroundColor: diskDonuts.volumes.colors[0] }"></span>
                    Used <span class="font-semibold tabular-nums">{{ formatBytes(diskDonuts.volumes.used) }}</span>
                  </span>
                  <span class="inline-flex items-center gap-1.5">
                    <span class="w-2 h-2 rounded-full" :style="{ backgroundColor: diskDonuts.volumes.colors[1] }"></span>
                    Unused <span class="font-semibold tabular-nums">{{ formatBytes(diskDonuts.volumes.unused) }}</span>
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Average Uptime -->
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
    </div>

    <!-- Second Row -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 sm:gap-6 mt-6" v-if="expiringContainers.count > 0 || categoryStats.mostUsed">
      <!-- Expiring Containers -->
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
        </div>
      </div>

      <!-- Category Statistics -->
      <div v-if="categoryStats.mostUsed" class="group relative rounded-2xl border border-gray-100 bg-white p-4 sm:p-5 transition-all duration-300 hover:shadow-xl hover:shadow-gray-900/5">
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

              <div v-if="categoryDonut.series.length === 0" class="mt-3 rounded-xl border border-dashed border-gray-200 bg-gray-50/50 px-4 py-6 text-center">
                <div class="text-sm font-semibold text-gray-700">No category data</div>
                <div class="text-xs text-gray-500 mt-1">Apps need categories to chart.</div>
              </div>

              <div v-else class="mt-2 grid grid-cols-1 sm:grid-cols-2 gap-3 items-center">
                <DonutChart
                  :series="categoryDonut.series"
                  :labels="categoryDonut.labels"
                  :colors="categoryDonut.colors"
                  :height="190"
                  donut-label="Apps"
                  :total-formatter="() => `${categoryStats.appsCount}`"
                />

                <div class="space-y-2">
                  <div v-for="row in categoryDonut.legend" :key="row.label" class="flex items-center justify-between gap-3 text-xs">
                    <div class="flex items-center gap-2 min-w-0">
                      <span class="w-2.5 h-2.5 rounded-full" :style="{ backgroundColor: row.color }"></span>
                      <span class="text-gray-700 font-medium truncate" :title="row.label">{{ row.label }}</span>
                    </div>
                    <span class="text-gray-900 font-bold tabular-nums">{{ row.value }}</span>
                  </div>
                  <div class="text-[11px] text-gray-400">Top {{ Math.min(6, categoryStats.total) }} categories</div>
                </div>
              </div>
            </div>

            <div class="rounded-2xl border border-green-100 bg-white/70 p-4">
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
                <div class="text-sm font-bold text-gray-900 tabular-nums">
                  {{ categoryStats.mostUsed.count }}
                </div>
              </div>
              <div class="text-xs text-gray-500 mt-2">
                {{ categoryStats.mostUsed.count === 1 ? 'app' : 'apps' }} in this category
              </div>
            </div>

            <div
              v-if="categoryStats.total > 1 && categoryStats.leastUsed.category !== categoryStats.mostUsed.category"
              class="rounded-2xl border border-blue-100 bg-white/70 p-4"
            >
              <div class="flex items-center justify-between gap-3">
                <div class="min-w-0">
                  <div class="text-xs text-gray-500 font-medium">Least used</div>
                  <div class="text-sm font-extrabold text-blue-700 truncate">
                    {{ formatCategory(categoryStats.leastUsed.category) }}
                  </div>
                </div>
                <div class="text-sm font-bold text-gray-900 tabular-nums">
                  {{ categoryStats.leastUsed.count }}
                </div>
              </div>
              <div class="text-xs text-gray-500 mt-2">
                {{ categoryStats.leastUsed.count === 1 ? 'app' : 'apps' }} in this category
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>
