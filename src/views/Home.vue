<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { Store, ArrowRight, AlertTriangle, Package, HardDrive, Activity, TrendingUp } from 'lucide-vue-next'
import SystemCleaner from '../components/SystemCleaner.vue'
import WatchtowerAlert from '../components/WatchtowerAlert.vue'

const router = useRouter()

const containers = ref([])
const volumes = ref([])
const loading = ref(false)
const apiUrl = ref('')
const currentTime = ref(Date.now())
const watchtowerInstalled = ref(false)

let containersRefreshInterval = null
let timeRefreshInterval = null

// Metrics computed properties
const totalApps = computed(() => containers.value.length)
const runningApps = computed(() => containers.value.filter(c => c.state === 'running').length)
const totalVolumes = computed(() => volumes.value.length)

// Container Grouping
const volumeContainers = computed(() => {
  return containers.value.filter(c => c.labels && c.labels['yantra.volume-browser'])
})

const yantraContainers = computed(() => {
  return containers.value.filter(c => {
    const isVolume = c.labels && c.labels['yantra.volume-browser']
    const isYantraApp = c.app || (c.labels && c.labels['yantra.app.id'])
    return !isVolume && isYantraApp
  })
})

const otherContainers = computed(() => {
  return containers.value.filter(c => {
    const isVolume = c.labels && c.labels['yantra.volume-browser']
    const isYantraApp = c.app || (c.labels && c.labels['yantra.app.id'])
    return !isVolume && !isYantraApp
  })
})

// Greeting
const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 12) return 'Good morning'
  if (hour < 18) return 'Good afternoon'
  return 'Good evening'
})

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

// Format container uptime
function formatUptime(container) {
  if (!container.created || container.state !== 'running') return null
  
  const createdTime = container.created * 1000 // Convert to milliseconds
  const uptime = currentTime.value - createdTime
  
  if (uptime <= 0) return 'Just started'
  
  const days = Math.floor(uptime / (1000 * 60 * 60 * 24))
  const hours = Math.floor((uptime % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60))
  const minutes = Math.floor((uptime % (1000 * 60 * 60)) / (1000 * 60))
  
  if (days > 0) {
    return `${days}d ${hours}h`
  } else if (hours > 0) {
    return `${hours}h ${minutes}m`
  } else {
    return `${minutes}m`
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
  <div class="min-h-screen bg-gray-50/50">
    <!-- Main Content -->
    <div class="p-6 sm:p-10 lg:p-14">
      <div class="space-y-10">
        
        <!-- Header & Stats -->
        <div class="flex flex-col md:flex-row md:items-end justify-between gap-6">
          <div>
            <h1 class="text-4xl sm:text-5xl font-bold text-gray-900 tracking-tight mb-2">
              {{ greeting }}
            </h1>
            <p class="text-lg text-gray-500 font-medium">
              Here's what's happening with your system today.
            </p>
          </div>
          
          <div class="flex items-center gap-4">
            <div class="flex flex-col items-end px-6 py-3 bg-white rounded-2xl shadow-sm border border-gray-100/50">
              <span class="text-3xl font-bold text-gray-900 leading-none">{{ runningApps }}</span>
              <span class="text-xs font-semibold text-gray-400 uppercase tracking-wider mt-1">Running Apps</span>
            </div>
            <div class="flex flex-col items-end px-6 py-3 bg-white rounded-2xl shadow-sm border border-gray-100/50">
              <span class="text-3xl font-bold text-gray-900 leading-none">{{ totalVolumes }}</span>
              <span class="text-xs font-semibold text-gray-400 uppercase tracking-wider mt-1">Volumes</span>
            </div>
          </div>
        </div>

        <!-- Loading State -->
        <div v-if="loading" class="flex flex-col items-center justify-center py-32">
          <div class="w-10 h-10 border-3 border-gray-200 border-t-blue-600 rounded-full animate-spin mb-4"></div>
          <div class="text-gray-400 font-medium animate-pulse">Syncing containers...</div>
        </div>

        <!-- Content -->
        <div v-else class="space-y-12 animate-fadeIn">
          
          <!-- Alerts Grid -->
          <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8" v-if="!loading">
            <div class="h-full">
              <SystemCleaner :api-url="apiUrl" />
            </div>
            <div class="h-full" v-if="!watchtowerInstalled && containers.length > 0">
              <WatchtowerAlert />
            </div>
          </div>

          <!-- Empty State -->
          <div v-if="containers.length === 0" class="text-center py-24 bg-white rounded-3xl shadow-sm">
            <div class="w-24 h-24 bg-gray-50 rounded-full flex items-center justify-center mx-auto mb-6">
              <Store :size="40" class="text-gray-300" />
            </div>
            <h3 class="text-2xl font-bold text-gray-900 mb-2">No Apps Running</h3>
            <p class="text-gray-500 max-w-md mx-auto mb-8">Your dashboard is looking a bit empty. Visit the App Store to get started.</p>
            <router-link to="/apps"
              class="inline-flex items-center gap-2 px-8 py-4 bg-gray-900 hover:bg-black text-white rounded-2xl font-bold transition-all hover:scale-105 shadow-xl shadow-gray-900/20">
              <Store :size="20" />
              <span>Browse App Store</span>
            </router-link>
          </div>

          <!-- Volume Browsers Section -->
          <div v-if="volumeContainers.length > 0">
            <div class="flex items-center gap-3 mb-6">
              <div class="p-2 bg-indigo-100 text-indigo-600 rounded-xl">
                <HardDrive :size="24" />
              </div>
              <h2 class="text-2xl font-bold text-gray-900">Volume Managers</h2>
              <span class="px-2.5 py-0.5 rounded-full bg-gray-100 text-gray-600 text-xs font-bold">{{ volumeContainers.length }}</span>
            </div>
            
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-2 xl:grid-cols-3 gap-5">
              <div v-for="(container, index) in volumeContainers" :key="container.id"
                :style="{ animationDelay: `${index * 50}ms` }"
                @click="viewContainerDetail(container)"
                class="group bg-white rounded-3xl p-6 shadow-[0_2px_15px_rgb(0,0,0,0.03)] border border-indigo-50 hover:border-indigo-100 hover:shadow-[0_8px_30px_rgb(79,70,229,0.06)] transition-all duration-300 cursor-pointer animate-fadeIn relative overflow-hidden flex flex-col h-full hover:-translate-y-1">
                
                <div class="flex items-center gap-4 mb-4">
                  <div class="w-14 h-14 bg-indigo-50 rounded-2xl flex items-center justify-center text-3xl group-hover:scale-105 transition-transform">
                    📂
                  </div>
                  <div>
                    <h3 class="font-bold text-lg text-gray-900 line-clamp-1 mb-1">
                      {{ container.labels?.['yantra.volume-browser'] || container.name }}
                    </h3>
                    <span class="text-xs font-semibold px-2.5 py-1 rounded-lg bg-green-50 text-green-700">
                      Active
                    </span>
                  </div>
                </div>

                <!-- Expiration Timer -->
                <div v-if="isTemporary(container)" class="mb-4 px-1">
                   <div class="flex items-center justify-between text-xs py-1.5 px-3 bg-orange-50 text-orange-800 rounded-lg border border-orange-100">
                    <span class="font-medium">Expires in</span>
                    <span :class="getExpirationInfo(container).isExpiringSoon ? 'text-red-600 animate-pulse font-bold' : 'font-semibold'">
                      {{ getExpirationInfo(container).timeRemaining }}
                    </span>
                  </div>
                </div>
                
                 <div class="mt-auto pt-3 flex items-center justify-between text-sm border-t border-gray-50">
                  <span class="text-gray-400 font-medium group-hover:text-indigo-600 transition-colors">Manage Files</span>
                  <ArrowRight :size="16" class="text-gray-300 group-hover:text-indigo-600 transform group-hover:translate-x-0.5 transition-all" />
                </div>
              </div>
            </div>
          </div>

          <!-- Yantra Apps Section -->
          <div v-if="yantraContainers.length > 0">
             <div class="flex items-center gap-3 mb-6">
              <div class="p-2 bg-blue-100 text-blue-600 rounded-xl">
                <Package :size="24" />
              </div>
              <h2 class="text-2xl font-bold text-gray-900">Installed Apps</h2>
              <span class="px-2.5 py-0.5 rounded-full bg-gray-100 text-gray-600 text-xs font-bold">{{ yantraContainers.length }}</span>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-2 xl:grid-cols-3 gap-5">
              <div v-for="(container, index) in yantraContainers" :key="container.id"
                :style="{ animationDelay: `${index * 50}ms` }"
                @click="viewContainerDetail(container)"
                 class="group bg-white rounded-3xl p-6 shadow-[0_2px_20px_rgb(0,0,0,0.04)] hover:shadow-[0_8px_30px_rgb(0,0,0,0.08)] transition-all duration-300 cursor-pointer animate-fadeIn relative overflow-hidden flex flex-col h-full hover:-translate-y-1">
                
                <!-- Card Header -->
                <div class="flex items-start justify-between mb-6">
                  <div class="flex items-center gap-4">
                    <div class="relative">
                      <img v-if="container.app && container.app.logo" :src="container.app.logo" :alt="container.name"
                        class="w-16 h-16 object-contain filter group-hover:brightness-105 transition-all">
                      <div v-else class="w-16 h-16 bg-blue-50 rounded-2xl flex items-center justify-center text-3xl">
                        🐳
                      </div>
                      <!-- Status Indicator Dot -->
                      <div class="absolute -bottom-1 -right-1 w-5 h-5 bg-white rounded-full flex items-center justify-center shadow-sm">
                        <div :class="container.state === 'running' ? 'bg-green-500' : 'bg-gray-400'" 
                             class="w-3 h-3 rounded-full animate-pulse"></div>
                      </div>
                    </div>
                    
                    <div>
                      <h3 class="font-bold text-xl text-gray-900 line-clamp-1 mb-1">
                        {{ container.app ? container.app.name : (container.name.replace(/^\//, '')) }}
                      </h3>
                      <div class="flex items-center gap-2">
                         <span class="text-xs font-semibold px-2.5 py-1 rounded-lg"
                               :class="container.state === 'running' ? 'bg-green-50 text-green-700' : 'bg-gray-100 text-gray-600'">
                           {{ container.state }}
                         </span>
                         <span v-if="isTemporary(container)" class="text-xs font-semibold px-2.5 py-1 rounded-lg bg-orange-50 text-orange-700">
                           Temp
                         </span>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- Metrics -->
                <div class="space-y-3 mb-6 flex-1">
                  <div v-if="container.state === 'running' && formatUptime(container)" 
                       class="flex items-center justify-between text-sm py-2 border-b border-gray-50 last:border-0 border-dashed">
                    <span class="text-gray-400 font-medium">Uptime</span>
                    <span class="text-gray-700 font-semibold font-mono">{{ formatUptime(container) }}</span>
                  </div>
                  
                  <div v-if="isTemporary(container)" 
                       class="flex items-center justify-between text-sm py-2 border-b border-gray-50 last:border-0 border-dashed">
                    <span class="text-gray-400 font-medium">Expires</span>
                    <span :class="getExpirationInfo(container).isExpiringSoon ? 'text-red-600 animate-pulse' : 'text-orange-600'" 
                          class="font-bold font-mono">
                      {{ getExpirationInfo(container).timeRemaining }}
                    </span>
                  </div>
                </div>

                <!-- Footer -->
                <div class="mt-auto pt-4 flex items-center justify-between text-sm border-t border-gray-50">
                  <span class="text-gray-400 font-medium group-hover:text-blue-600 transition-colors">Manage App</span>
                  <div class="w-8 h-8 rounded-full bg-gray-50 flex items-center justify-center group-hover:bg-blue-50 group-hover:text-blue-600 transition-colors">
                    <ArrowRight :size="16" class="transform group-hover:translate-x-0.5 transition-transform" />
                  </div>
                </div>
              </div>
            </div>
          </div>
          </div>

          <!-- Other Containers Section -->
          <div v-if="otherContainers.length > 0">
             <div class="flex items-center gap-3 mb-6">
              <div class="p-2 bg-gray-200 text-gray-600 rounded-xl">
                <div class="font-bold text-lg px-1">#</div>
              </div>
              <h2 class="text-2xl font-bold text-gray-900">Other Containers</h2>
              <span class="px-2.5 py-0.5 rounded-full bg-gray-100 text-gray-600 text-xs font-bold">{{ otherContainers.length }}</span>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-2 xl:grid-cols-3 gap-5">
              <div v-for="(container, index) in otherContainers" :key="container.id"
                :style="{ animationDelay: `${index * 50}ms` }"
                @click="viewContainerDetail(container)"
                 class="group bg-white rounded-3xl p-6 shadow-sm border border-gray-200 hover:shadow-lg transition-all duration-300 cursor-pointer animate-fadeIn relative overflow-hidden flex flex-col h-full hover:-translate-y-1">
                
                <div class="flex items-start gap-4 mb-4">
                  <div class="w-12 h-12 bg-gray-100 rounded-xl flex items-center justify-center text-2xl grayscale opacity-70">
                    🐳
                  </div>
                  <div class="overflow-hidden">
                    <h3 class="font-bold text-lg text-gray-700 truncate mb-1" :title="container.name">
                      {{ container.name.replace(/^\//, '') }}
                    </h3>
                     <span class="text-xs font-semibold px-2 py-0.5 rounded-md"
                           :class="container.state === 'running' ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-600'">
                       {{ container.state }}
                     </span>
                  </div>
                </div>

                <div class="mt-auto pt-4 border-t border-gray-100">
                    <div class="flex items-center justify-between text-xs text-gray-500">
                        <span class="font-mono truncate max-w-[150px]">{{ container.image.split(':')[0] }}</span>
                        <ArrowRight :size="14" class="group-hover:translate-x-1 transition-transform" />
                    </div>
                </div>
              </div>
            </div>
          </div>

        </div>
      
      <!-- Right Sidebar Area Removed -->
      
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

/* Apple-like hover lift effect */
.hover-lift {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.hover-lift:hover {
  transform: translateY(-4px) scale(1.01);
  box-shadow: 0 20px 40px -12px rgba(0, 0, 0, 0.15), 0 8px 16px -8px rgba(124, 58, 237, 0.2);
}

.hover-lift:active {
  transform: translateY(-2px) scale(1);
  transition: all 0.1s cubic-bezier(0.4, 0, 0.2, 1);
}

/* Smooth backdrop blur support */
@supports (backdrop-filter: blur(20px)) or (-webkit-backdrop-filter: blur(20px)) {
  .backdrop-blur-xl {
    -webkit-backdrop-filter: blur(20px);
    backdrop-filter: blur(20px);
  }
  
  .backdrop-blur-sm {
    -webkit-backdrop-filter: blur(4px);
    backdrop-filter: blur(4px);
  }
}
</style>
