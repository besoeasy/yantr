<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { Store, ArrowRight, AlertTriangle, Package, HardDrive, Activity, TrendingUp } from 'lucide-vue-next'

const router = useRouter()

const containers = ref([])
const volumes = ref([])
const loading = ref(false)
const apiUrl = ref('')
const currentTime = ref(Date.now())
const watchtowerInstalled = ref(false)

let containersRefreshInterval = null
let timeRefreshInterval = null

// Computed properties to separate volume browsers from regular containers
const volumeBrowsers = computed(() => 
  containers.value.filter(c => c.labels && c.labels['yantra.volume-browser'])
)

const regularContainers = computed(() => 
  containers.value.filter(c => !c.labels || !c.labels['yantra.volume-browser'])
)

// Metrics computed properties
const totalApps = computed(() => regularContainers.value.length)
const runningApps = computed(() => regularContainers.value.filter(c => c.state === 'running').length)
const totalVolumes = computed(() => volumes.value.length)
const activeVolumeBrowsers = computed(() => volumeBrowsers.value.length)

// Helper function to format time remaining
function formatTimeRemaining(expireAt) {
  const expirationTime = parseInt(expireAt, 10) * 1000 // Convert to milliseconds
  const remaining = expirationTime - currentTime.value
  
  if (remaining <= 0) return 'Expired'
  
  const hours = Math.floor(remaining / (1000 * 60 * 60))
  const minutes = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60))
  
  if (hours > 24) {
    const days = Math.floor(hours / 24)
    return `${days}d ${hours % 24}h`
  } else if (hours > 0) {
    return `${hours}h ${minutes}m`
  } else {
    return `${minutes}m`
  }
}

// Check if container is temporary
function isTemporary(container) {
  return container.labels && container.labels['yantra.expireAt']
}

// Get expiration info
function getExpirationInfo(container) {
  if (!isTemporary(container)) return null
  
  const expireAt = container.labels['yantra.expireAt']
  return {
    expireAt,
    timeRemaining: formatTimeRemaining(expireAt),
    isExpiringSoon: (parseInt(expireAt, 10) * 1000 - currentTime.value) < (60 * 60 * 1000) // < 1 hour
  }
}

async function fetchContainers() {
  try {
    const response = await fetch(`${apiUrl.value}/api/containers`)
    const data = await response.json()
    if (data.success) {
      containers.value = data.containers
      
      watchtowerInstalled.value = data.containers.some(c => 
        c.name?.toLowerCase().includes('watchtower') || 
        c.Names?.some(name => name.toLowerCase().includes('watchtower'))
      )
    }
  } catch (error) {
    console.error('Failed to fetch containers:', error)
  }
}

async function fetchVolumes() {
  try {
    const response = await fetch(`${apiUrl.value}/api/volumes`)
    const data = await response.json()
    if (data.success) {
      volumes.value = data.volumes || []
    }
  } catch (error) {
    console.error('Failed to fetch volumes:', error)
  }
}

function viewContainerDetail(container) {
  router.push(`/containers/${container.id}`)
}

onMounted(async () => {
  loading.value = true
  await Promise.all([fetchContainers(), fetchVolumes()])
  loading.value = false
  
  containersRefreshInterval = setInterval(() => {
    fetchContainers()
    fetchVolumes()
  }, 10000)
  
  timeRefreshInterval = setInterval(() => {
    currentTime.value = Date.now()
  }, 60000)
})

onUnmounted(() => {
  if (containersRefreshInterval) {
    clearInterval(containersRefreshInterval)
    containersRefreshInterval = null
  }
  if (timeRefreshInterval) {
    clearInterval(timeRefreshInterval)
    timeRefreshInterval = null
  }
})
</script>

<template>
  <div class="min-h-screen bg-gray-50">
    <div class="p-4 sm:p-6 lg:p-8">
      <div class="max-w-7xl mx-auto space-y-6">
        <!-- Loading State -->
        <div v-if="loading" class="text-center py-20">
          <div class="w-12 h-12 border-4 border-gray-200 border-t-gray-900 rounded-full animate-spin mx-auto mb-4"></div>
          <div class="text-gray-600 font-medium">Loading...</div>
        </div>

        <!-- Empty State -->
        <div v-else-if="containers.length === 0" class="text-center py-16 bg-white rounded-2xl shadow-sm border border-gray-200">
        <div class="text-6xl mb-4">🎯</div>
        <div class="text-gray-900 font-bold text-2xl mb-2">Welcome to Yantra!</div>
        <div class="text-gray-600 mb-6">No containers running yet. Get started by installing apps.</div>
        <router-link to="/apps"
          class="inline-flex items-center gap-2 px-6 py-3 bg-gray-900 hover:bg-gray-800 text-white rounded-xl font-semibold transition-all shadow-lg hover:shadow-xl">
          <Store :size="20" />
          <span>Browse App Store</span>
        </router-link>
      </div>

      <!-- Metrics Grid -->
      <div v-else class="grid gap-4" style="grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); grid-auto-rows: auto;">
        <!-- Total Apps / Watchtower Card -->
        <div v-if="!watchtowerInstalled" 
             @click="router.push('/apps/watchtower')"
             class="col-span-2 bg-orange-50 rounded-2xl p-6 shadow-sm border border-orange-200 hover:shadow-md transition-shadow cursor-pointer group">
          <div class="flex items-center justify-between mb-4">
            <div class="w-12 h-12 bg-orange-100 rounded-xl flex items-center justify-center">
              <AlertTriangle :size="24" class="text-orange-600" />
            </div>
            <span class="text-xs font-medium text-orange-600 uppercase tracking-wide">Action Required</span>
          </div>
          <div class="space-y-1">
            <div class="text-2xl font-black text-gray-900 tracking-tight">
              Watchtower
            </div>
            <div class="text-sm text-orange-700 font-medium group-hover:underline">
              Not Installed →
            </div>
          </div>
        </div>

        <div v-else class="bg-white rounded-2xl p-6 shadow-sm border border-gray-200 hover:shadow-md transition-shadow">
          <div class="flex items-center justify-between mb-4">
            <div class="w-12 h-12 bg-gray-100 rounded-xl flex items-center justify-center">
              <Package :size="24" class="text-gray-900" />
            </div>
            <span class="text-xs font-medium text-gray-500 uppercase tracking-wide">Total Apps</span>
          </div>
          <div class="space-y-1">
            <div class="text-5xl font-black text-gray-900 tracking-tight">
              {{ totalApps }}
            </div>
            <div class="text-sm text-gray-600">
              Installed applications
            </div>
          </div>
        </div>

        <!-- Running Apps Card -->
        <div class="bg-white rounded-2xl p-6 shadow-sm border border-gray-200 hover:shadow-md transition-shadow">
          <div class="flex items-center justify-between mb-4">
            <div class="w-12 h-12 bg-green-50 rounded-xl flex items-center justify-center">
              <Activity :size="24" class="text-green-600" />
            </div>
            <span class="text-xs font-medium text-gray-500 uppercase tracking-wide">Running</span>
          </div>
          <div class="space-y-1">
            <div class="text-5xl font-black text-gray-900 tracking-tight">
              {{ runningApps }}
            </div>
            <div class="text-sm text-gray-600">
              Active containers
            </div>
          </div>
        </div>

        <!-- Total Volumes Card -->
        <div class="bg-white rounded-2xl p-6 shadow-sm border border-gray-200 hover:shadow-md transition-shadow">
          <div class="flex items-center justify-between mb-4">
            <div class="w-12 h-12 bg-blue-50 rounded-xl flex items-center justify-center">
              <HardDrive :size="24" class="text-blue-600" />
            </div>
            <span class="text-xs font-medium text-gray-500 uppercase tracking-wide">Volumes</span>
          </div>
          <div class="space-y-1">
            <div class="text-5xl font-black text-gray-900 tracking-tight">
              {{ totalVolumes }}
            </div>
            <div class="text-sm text-gray-600">
              Storage volumes
            </div>
          </div>
        </div>

        <!-- Volume Browsers Card -->
        <div class="bg-white rounded-2xl p-6 shadow-sm border border-gray-200 hover:shadow-md transition-shadow">
          <div class="flex items-center justify-between mb-4">
            <div class="w-12 h-12 bg-purple-50 rounded-xl flex items-center justify-center">
              <TrendingUp :size="24" class="text-purple-600" />
            </div>
            <span class="text-xs font-medium text-gray-500 uppercase tracking-wide">Browsers</span>
          </div>
          <div class="space-y-1">
            <div class="text-5xl font-black text-gray-900 tracking-tight">
              {{ activeVolumeBrowsers }}
            </div>
            <div class="text-sm text-gray-600">
              Volume browsers
            </div>
          </div>
        </div>
      </div>

      <!-- Applications Section -->
      <div v-if="regularContainers.length > 0" class="space-y-4">
        <div class="flex items-center justify-between">
          <h2 class="text-2xl font-bold text-gray-900">Applications</h2>
          <span class="text-sm text-gray-500 font-medium">{{ regularContainers.length }} installed</span>
        </div>

        <div class="grid gap-4" style="grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); grid-auto-rows: auto;">
          <div v-for="(container, index) in regularContainers" :key="container.id"
            :style="{ animationDelay: `${index * 50}ms` }"
            @click="viewContainerDetail(container)"
            class="group bg-white rounded-xl p-5 border border-gray-200 hover:border-gray-300 hover:shadow-lg transition-all duration-200 cursor-pointer animate-fadeIn">
            
            <div class="flex items-start gap-4 mb-4">
              <div class="shrink-0">
                <img v-if="container.app.logo" :src="container.app.logo" :alt="container.name"
                  class="w-14 h-14 object-contain rounded-lg">
                <span v-else class="text-4xl">🐳</span>
              </div>
              
              <div class="flex-1 min-w-0">
                <h3 class="font-bold text-lg text-gray-900 truncate mb-1">
                  {{ container.name }}
                </h3>
                <div class="flex items-center gap-2">
                  <div :class="container.state === 'running' ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600'"
                    class="px-2 py-0.5 rounded-md text-xs font-semibold inline-flex items-center gap-1">
                    <span :class="container.state === 'running' ? 'bg-green-500' : 'bg-gray-400'" 
                      class="w-1.5 h-1.5 rounded-full"></span>
                    {{ container.state }}
                  </div>
                  <div v-if="isTemporary(container)" 
                    :class="getExpirationInfo(container).isExpiringSoon ? 'bg-red-100 text-red-700' : 'bg-orange-100 text-orange-700'"
                    class="px-2 py-0.5 rounded-md text-xs font-semibold">
                    ⏱️ {{ getExpirationInfo(container).timeRemaining }}
                  </div>
                </div>
              </div>
            </div>

            <div class="space-y-2 text-xs text-gray-600">
              <div class="flex items-start gap-2">
                <span class="font-semibold shrink-0">Image:</span>
                <span class="font-mono break-all">{{ container.image }}</span>
              </div>
              <div v-if="isTemporary(container)" 
                :class="getExpirationInfo(container).isExpiringSoon ? 'bg-red-50 border-red-200 text-red-700' : 'bg-orange-50 border-orange-200 text-orange-700'"
                class="px-2 py-1.5 rounded-lg border">
                <div class="font-semibold mb-0.5">⏱️ Temporary Installation</div>
                <div class="opacity-80 text-[10px]">
                  Auto-deletes in {{ getExpirationInfo(container).timeRemaining }}
                </div>
              </div>
            </div>

            <div class="mt-4 pt-4 border-t border-gray-100 flex items-center justify-between">
              <span class="text-sm text-gray-600 group-hover:text-gray-900 font-medium transition-colors">
                View Details
              </span>
              <ArrowRight :size="16" class="text-gray-400 group-hover:text-gray-900 group-hover:translate-x-1 transition-all" />
            </div>
          </div>
        </div>
      </div>

      <!-- Volume Browsers Section -->
      <div v-if="volumeBrowsers.length > 0" class="space-y-4">
        <div class="flex items-center justify-between">
          <h2 class="text-2xl font-bold text-gray-900">Volume Browsers</h2>
          <span class="text-sm text-gray-500 font-medium">{{ volumeBrowsers.length }} active</span>
        </div>

        <div class="p-4 bg-purple-50 border border-purple-200 rounded-xl">
          <p class="text-sm text-purple-800">
            <span class="font-semibold">💡 Safe to Close:</span> These are temporary file browsers. Deleting them won't affect your data.
          </p>
        </div>

        <div class="grid gap-4" style="grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); grid-auto-rows: auto;">
          <div v-for="(container, index) in volumeBrowsers" :key="container.id"
            :style="{ animationDelay: `${index * 50}ms` }"
            @click="viewContainerDetail(container)"
            class="group bg-purple-50 rounded-xl p-5 border border-purple-200 hover:border-purple-300 hover:shadow-lg transition-all duration-200 cursor-pointer animate-fadeIn">
            
            <div class="flex items-start gap-4 mb-4">
              <div class="shrink-0">
                <span class="text-4xl">📦</span>
              </div>
              
              <div class="flex-1 min-w-0">
                <h3 class="font-bold text-lg text-gray-900 truncate mb-1">
                  {{ container.labels['yantra.volume-browser'] }}
                </h3>
                <div class="flex items-center gap-2">
                  <div :class="container.state === 'running' ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600'"
                    class="px-2 py-0.5 rounded-md text-xs font-semibold inline-flex items-center gap-1">
                    <span :class="container.state === 'running' ? 'bg-green-500' : 'bg-gray-400'" 
                      class="w-1.5 h-1.5 rounded-full"></span>
                    {{ container.state }}
                  </div>
                </div>
              </div>
            </div>

            <div class="mt-4 pt-4 border-t border-purple-200 flex items-center justify-between">
              <span class="text-sm text-purple-700 group-hover:text-purple-900 font-medium transition-colors">
                View Details
              </span>
              <ArrowRight :size="16" class="text-purple-400 group-hover:text-purple-600 group-hover:translate-x-1 transition-all" />
            </div>
          </div>
        </div>
      </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.animate-fadeIn {
  opacity: 0;
  animation: fadeIn 0.5s ease-out forwards;
}
</style>
