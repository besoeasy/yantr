<script setup>
import { computed } from 'vue'
import { Activity, Package, HardDrive, Clock, Timer, Layers } from 'lucide-vue-next'

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

const containerStatePercentages = computed(() => {
  if (totalContainers.value === 0) return {}
  
  return {
    running: (containersByState.value.running / totalContainers.value) * 100,
    stopped: (containersByState.value.stopped / totalContainers.value) * 100,
    paused: (containersByState.value.paused / totalContainers.value) * 100,
    exited: (containersByState.value.exited / totalContainers.value) * 100
  }
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
const categoryStats = computed(() => {
  const categoryCounts = {}
  
  props.containers.forEach(container => {
    const category = container.app?.category || 'uncategorized'
    categoryCounts[category] = (categoryCounts[category] || 0) + 1
  })
  
  if (Object.keys(categoryCounts).length === 0) {
    return { mostUsed: null, leastUsed: null, total: 0 }
  }
  
  // Sort categories by usage
  const sorted = Object.entries(categoryCounts)
    .sort((a, b) => b[1] - a[1])
  
  return {
    mostUsed: sorted[0] ? { category: sorted[0][0], count: sorted[0][1] } : null,
    leastUsed: sorted[sorted.length - 1] ? { category: sorted[sorted.length - 1][0], count: sorted[sorted.length - 1][1] } : null,
    total: Object.keys(categoryCounts).length,
    all: sorted
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
  return category
    .split('-')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ')
}
</script>

<template>
  <div class="bg-white rounded-3xl p-6 shadow-[0_2px_20px_rgb(0,0,0,0.04)] border border-gray-100/50">
    <!-- Header -->
    <div class="flex items-center gap-3 mb-6">
      <div class="p-2 bg-gradient-to-br from-purple-100 to-blue-100 text-purple-600 rounded-xl">
        <Activity :size="24" />
      </div>
      <h2 class="text-2xl font-bold text-gray-900">Quick Metrics</h2>
    </div>

    <!-- Metrics Grid -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
      
      <!-- Container States -->
      <div class="space-y-3">
        <div class="flex items-center gap-2 mb-3">
          <Package :size="18" class="text-gray-400" />
          <h3 class="text-sm font-bold text-gray-500 uppercase tracking-wider">Container States</h3>
        </div>
        
        <div v-if="totalContainers === 0" class="text-sm text-gray-400 py-4 text-center">
          No containers
        </div>
        
        <div v-else class="space-y-2">
          <!-- Running -->
          <div v-if="containersByState.running > 0" class="flex items-center justify-between text-sm">
            <div class="flex items-center gap-2">
              <div class="w-2 h-2 rounded-full bg-green-500 animate-pulse"></div>
              <span class="text-gray-600 font-medium">Running</span>
            </div>
            <span class="font-bold text-gray-900">{{ containersByState.running }}</span>
          </div>
          <div v-if="containersByState.running > 0" class="w-full bg-gray-100 rounded-full h-1.5 overflow-hidden">
            <div class="bg-gradient-to-r from-green-500 to-green-400 h-full rounded-full transition-all duration-500"
                 :style="{ width: `${containerStatePercentages.running}%` }"></div>
          </div>
          
          <!-- Stopped -->
          <div v-if="containersByState.stopped > 0" class="flex items-center justify-between text-sm">
            <div class="flex items-center gap-2">
              <div class="w-2 h-2 rounded-full bg-gray-400"></div>
              <span class="text-gray-600 font-medium">Stopped</span>
            </div>
            <span class="font-bold text-gray-900">{{ containersByState.stopped }}</span>
          </div>
          <div v-if="containersByState.stopped > 0" class="w-full bg-gray-100 rounded-full h-1.5 overflow-hidden">
            <div class="bg-gray-400 h-full rounded-full transition-all duration-500"
                 :style="{ width: `${containerStatePercentages.stopped}%` }"></div>
          </div>
          
          <!-- Exited -->
          <div v-if="containersByState.exited > 0" class="flex items-center justify-between text-sm">
            <div class="flex items-center gap-2">
              <div class="w-2 h-2 rounded-full bg-red-500"></div>
              <span class="text-gray-600 font-medium">Exited</span>
            </div>
            <span class="font-bold text-gray-900">{{ containersByState.exited }}</span>
          </div>
          <div v-if="containersByState.exited > 0" class="w-full bg-gray-100 rounded-full h-1.5 overflow-hidden">
            <div class="bg-red-500 h-full rounded-full transition-all duration-500"
                 :style="{ width: `${containerStatePercentages.exited}%` }"></div>
          </div>
          
          <!-- Paused -->
          <div v-if="containersByState.paused > 0" class="flex items-center justify-between text-sm">
            <div class="flex items-center gap-2">
              <div class="w-2 h-2 rounded-full bg-orange-500"></div>
              <span class="text-gray-600 font-medium">Paused</span>
            </div>
            <span class="font-bold text-gray-900">{{ containersByState.paused }}</span>
          </div>
          <div v-if="containersByState.paused > 0" class="w-full bg-gray-100 rounded-full h-1.5 overflow-hidden">
            <div class="bg-orange-500 h-full rounded-full transition-all duration-500"
                 :style="{ width: `${containerStatePercentages.paused}%` }"></div>
          </div>
        </div>
      </div>

      <!-- Disk Breakdown -->
      <div class="space-y-3">
        <div class="flex items-center gap-2 mb-3">
          <HardDrive :size="18" class="text-gray-400" />
          <h3 class="text-sm font-bold text-gray-500 uppercase tracking-wider">Disk Usage</h3>
        </div>
        
        <div class="space-y-3">
          <!-- Images -->
          <div>
            <div class="flex items-center justify-between text-sm mb-1.5">
              <span class="text-gray-600 font-medium">Images</span>
              <span class="font-bold text-gray-900">{{ formatBytes(diskMetrics.images.total) }}</span>
            </div>
            <div class="w-full bg-gray-100 rounded-full h-2 overflow-hidden">
              <div class="flex h-full">
                <div class="bg-gradient-to-r from-blue-500 to-blue-400 transition-all duration-500"
                     :style="{ width: `${diskMetrics.images.usedPercent}%` }"
                     :title="`Used: ${formatBytes(diskMetrics.images.used)}`"></div>
                <div class="bg-gradient-to-r from-orange-300 to-orange-200 transition-all duration-500"
                     :style="{ width: `${diskMetrics.images.unusedPercent}%` }"
                     :title="`Unused: ${formatBytes(diskMetrics.images.unused)}`"></div>
              </div>
            </div>
            <div class="flex items-center justify-between text-xs text-gray-400 mt-1">
              <span>Used: {{ formatBytes(diskMetrics.images.used) }}</span>
              <span>Unused: {{ formatBytes(diskMetrics.images.unused) }}</span>
            </div>
          </div>
          
          <!-- Volumes -->
          <div>
            <div class="flex items-center justify-between text-sm mb-1.5">
              <span class="text-gray-600 font-medium">Volumes</span>
              <span class="font-bold text-gray-900">{{ formatBytes(diskMetrics.volumes.total) }}</span>
            </div>
            <div class="w-full bg-gray-100 rounded-full h-2 overflow-hidden">
              <div class="flex h-full">
                <div class="bg-gradient-to-r from-indigo-500 to-indigo-400 transition-all duration-500"
                     :style="{ width: `${diskMetrics.volumes.usedPercent}%` }"
                     :title="`Used: ${formatBytes(diskMetrics.volumes.used)}`"></div>
                <div class="bg-gradient-to-r from-orange-300 to-orange-200 transition-all duration-500"
                     :style="{ width: `${diskMetrics.volumes.unusedPercent}%` }"
                     :title="`Unused: ${formatBytes(diskMetrics.volumes.unused)}`"></div>
              </div>
            </div>
            <div class="flex items-center justify-between text-xs text-gray-400 mt-1">
              <span>Used: {{ formatBytes(diskMetrics.volumes.used) }}</span>
              <span>Unused: {{ formatBytes(diskMetrics.volumes.unused) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Average Uptime -->
      <div class="space-y-3">
        <div class="flex items-center gap-2 mb-3">
          <Clock :size="18" class="text-gray-400" />
          <h3 class="text-sm font-bold text-gray-500 uppercase tracking-wider">Avg Uptime</h3>
        </div>
        
        <div v-if="averageUptime.count === 0" class="text-sm text-gray-400 py-4 text-center">
          No running containers
        </div>
        
        <div v-else class="flex flex-col items-center justify-center py-2">
          <div class="text-4xl font-bold text-transparent bg-clip-text bg-gradient-to-r from-purple-600 to-blue-600 mb-2">
            {{ averageUptime.formatted }}
          </div>
          <div class="text-xs text-gray-500 font-medium">
            Across {{ averageUptime.count }} running {{ averageUptime.count === 1 ? 'container' : 'containers' }}
          </div>
          
          <!-- Visual indicator -->
          <div class="mt-4 flex items-center gap-1">
            <div v-for="i in Math.min(averageUptime.count, 10)" :key="i"
                 class="w-1.5 h-8 bg-gradient-to-t from-purple-200 to-purple-500 rounded-full"
                 :style="{ 
                   animationDelay: `${i * 100}ms`,
                   height: `${Math.random() * 20 + 20}px`
                 }"
                 :class="i <= 5 ? 'animate-pulse' : ''">
            </div>
            <span v-if="averageUptime.count > 10" class="text-xs text-gray-400 ml-1">+{{ averageUptime.count - 10 }}</span>
          </div>
        </div>
      </div>

    </div>

    <!-- Second Row: Expiring Containers & Category Stats -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mt-6" v-if="expiringContainers.count > 0 || categoryStats.mostUsed">
      
      <!-- Expiring Containers -->
      <div class="space-y-3" v-if="expiringContainers.count > 0">
        <div class="flex items-center gap-2 mb-3">
          <Timer :size="18" class="text-gray-400" />
          <h3 class="text-sm font-bold text-gray-500 uppercase tracking-wider">Expiring Containers</h3>
        </div>
        
        <div class="space-y-3">
          <!-- Count -->
          <div class="flex items-center justify-between">
            <span class="text-sm text-gray-600 font-medium">Total Expiring</span>
            <span class="px-3 py-1 bg-orange-100 text-orange-700 rounded-lg font-bold text-sm">{{ expiringContainers.count }}</span>
          </div>
          
          <!-- Next Expiry Countdown -->
          <div class="bg-gradient-to-br from-orange-50 to-red-50 rounded-2xl p-4 border border-orange-100">
            <div class="text-xs text-gray-500 font-medium mb-1">Next to Expire</div>
            <div class="text-2xl font-bold mb-1"
                 :class="expiringContainers.isExpiringSoon ? 'text-red-600 animate-pulse' : 'text-orange-600'">
              {{ expiringContainers.nextExpiryFormatted }}
            </div>
            <div class="text-xs text-gray-600 truncate" :title="expiringContainers.containerName">
              {{ expiringContainers.containerName }}
            </div>
          </div>
        </div>
      </div>

      <!-- Category Statistics -->
      <div class="space-y-3" v-if="categoryStats.mostUsed">
        <div class="flex items-center gap-2 mb-3">
          <Layers :size="18" class="text-gray-400" />
          <h3 class="text-sm font-bold text-gray-500 uppercase tracking-wider">App Categories</h3>
        </div>
        
        <div class="space-y-3">
          <!-- Most Used -->
          <div>
            <div class="flex items-center justify-between text-sm mb-2">
              <span class="text-gray-600 font-medium">Most Used</span>
              <span class="font-bold text-gray-900">{{ categoryStats.mostUsed.count }} {{ categoryStats.mostUsed.count === 1 ? 'app' : 'apps' }}</span>
            </div>
            <div class="bg-gradient-to-r from-green-50 to-emerald-50 rounded-xl px-4 py-2 border border-green-100">
              <div class="text-sm font-bold text-green-700">🏆 {{ formatCategory(categoryStats.mostUsed.category) }}</div>
            </div>
          </div>
          
          <!-- Least Used (only show if more than 1 category) -->
          <div v-if="categoryStats.total > 1 && categoryStats.leastUsed.category !== categoryStats.mostUsed.category">
            <div class="flex items-center justify-between text-sm mb-2">
              <span class="text-gray-600 font-medium">Least Used</span>
              <span class="font-bold text-gray-900">{{ categoryStats.leastUsed.count }} {{ categoryStats.leastUsed.count === 1 ? 'app' : 'apps' }}</span>
            </div>
            <div class="bg-gradient-to-r from-blue-50 to-indigo-50 rounded-xl px-4 py-2 border border-blue-100">
              <div class="text-sm font-bold text-blue-700">{{ formatCategory(categoryStats.leastUsed.category) }}</div>
            </div>
          </div>
        </div>
      </div>

    </div>
  </div>
</template>

<style scoped>
@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

.animate-pulse {
  animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}
</style>
