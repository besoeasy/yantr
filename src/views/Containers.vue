<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Store, ArrowRight, AlertTriangle } from 'lucide-vue-next'

const router = useRouter()

const containers = ref([])
const loading = ref(false)
const apiUrl = ref('')
const currentTime = ref(Date.now())
const watchtowerInstalled = ref(false)

// Computed properties to separate volume browsers from regular containers
const volumeBrowsers = computed(() => 
  containers.value.filter(c => c.labels && c.labels['yantra.volume-browser'])
)

const regularContainers = computed(() => 
  containers.value.filter(c => !c.labels || !c.labels['yantra.volume-browser'])
)

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
      // Show all containers, not just running ones (includes starting, created, restarting states)
      containers.value = data.containers
      
      // Check if Watchtower is installed (check both name and Names fields)
      watchtowerInstalled.value = data.containers.some(c => 
        c.name?.toLowerCase().includes('watchtower') || 
        c.Names?.some(name => name.toLowerCase().includes('watchtower'))
      )
    }
  } catch (error) {
    console.error('Failed to fetch containers:', error)
  }
}

function viewContainerDetail(container) {
  router.push(`/containers/${container.id}`)
}

onMounted(async () => {
  loading.value = true
  await fetchContainers()
  loading.value = false
  
  // Auto-refresh containers every 10 seconds
  setInterval(() => {
    fetchContainers()
  }, 10000)
  
  // Update current time every minute for countdown
  setInterval(() => {
    currentTime.value = Date.now()
  }, 60000)
})
</script>

<template>
  <!-- Watchtower Warning Banner - Sticky Full Width -->
  <div v-if="!loading && !watchtowerInstalled" 
    @click="router.push('/apps/watchtower')"
    class="sticky top-0 z-40 w-full bg-gradient-to-r from-orange-50 to-amber-50 border-b-2 border-orange-500 cursor-pointer hover:shadow-lg transition-all duration-300 group">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 md:px-10 lg:px-12 py-3 sm:py-4">
      <div class="flex items-center gap-3 sm:gap-4">
        <div class="flex-shrink-0">
          <div class="w-8 h-8 sm:w-10 sm:h-10 bg-orange-500 rounded-full flex items-center justify-center group-hover:rotate-12 transition-transform duration-300">
            <AlertTriangle :size="18" class="sm:w-5 sm:h-5 text-white" />
          </div>
        </div>
        <div class="flex-1 min-w-0">
          <h3 class="text-sm sm:text-base font-bold text-gray-900 mb-0.5 group-hover:text-orange-600 transition-colors">
            Watchtower Not Installed
          </h3>
          <p class="text-xs sm:text-sm text-gray-700">
            Install Watchtower to enable automatic updates for Yantra & Installed Apps.
          </p>
        </div>
        <div class="flex-shrink-0 hidden sm:flex items-center gap-2 text-orange-600 font-semibold text-sm group-hover:gap-3 transition-all">
          <span>Install Now</span>
          <ArrowRight :size="16" class="group-hover:translate-x-1 transition-transform" />
        </div>
      </div>
    </div>
  </div>

  <div class="p-4 sm:p-6 md:p-10 lg:p-12">
    <div class="mb-6 md:mb-8">
      <h2 class="text-3xl sm:text-4xl font-bold mb-2 text-gray-900">Running Containers</h2>
      <p class="text-sm sm:text-base text-gray-600">Manage your active Docker containers</p>
    </div>
    
    <div v-if="loading" class="text-center py-16">
      <div class="w-8 h-8 border-3 border-blue-600 border-t-transparent rounded-full animate-spin mx-auto mb-3"></div>
      <div class="text-gray-500 font-medium">Loading containers...</div>
    </div>
    <div v-else-if="containers.length === 0" class="text-center py-12 sm:py-20">
      <div class="text-5xl sm:text-6xl mb-4">🎯</div>
      <div class="text-gray-900 font-bold text-xl sm:text-2xl mb-2">Welcome to Yantra!</div>
      <div class="text-gray-600 font-medium mb-2 text-sm sm:text-base">No containers running yet</div>
      <div class="text-xs sm:text-sm text-gray-500 mb-6">Get started by installing apps from the App Store</div>
      <router-link to="/apps"
        class="inline-flex items-center gap-2 px-5 sm:px-6 py-2.5 sm:py-3 bg-blue-600 hover:bg-blue-700 text-white rounded-xl font-semibold transition-all shadow-md hover:shadow-lg hover:scale-105 active:scale-95 touch-manipulation text-sm sm:text-base">
        <Store :size="18" />
        <span>Browse App Store</span>
      </router-link>
    </div>
    <div v-else class="space-y-8">
      <!-- Regular Containers Section -->
      <div v-if="regularContainers.length > 0">
        <h3 class="text-xl font-bold text-gray-900 mb-4 flex items-center gap-2">
          <span class="text-blue-600">🐳</span>
          Applications
          <span class="text-sm font-normal text-gray-500">({{ regularContainers.length }})</span>
        </h3>
        <div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3 sm:gap-4">
          <div v-for="(container, index) in regularContainers" :key="container.id"
        :style="{ animationDelay: `${index * 50}ms` }"
        @click="viewContainerDetail(container)"
        class="group bg-white/80 backdrop-blur-sm rounded-xl sm:rounded-2xl p-4 sm:p-5 transition-all duration-300 hover:bg-white hover:shadow-xl hover:scale-[1.02] active:scale-[0.98] cursor-pointer animate-fadeIn touch-manipulation">
        
        <!-- Header with Logo and Status -->
        <div class="flex items-start justify-between mb-3 sm:mb-4">
          <div class="flex items-center gap-2.5 sm:gap-3 flex-1 min-w-0">
            <div class="relative flex-shrink-0">
              <img v-if="container.app.logo" :src="container.app.logo" :alt="container.name"
                class="w-12 h-12 sm:w-14 sm:h-14 object-contain transition-all duration-300 group-hover:scale-110 group-hover:rotate-3">
              <span v-else class="text-3xl sm:text-4xl">🐳</span>
            </div>
            <div class="flex-1 min-w-0">
              <h3 class="font-bold text-gray-900 text-base sm:text-lg truncate group-hover:text-blue-600 transition-colors">
                {{ container.name }}
              </h3>
              <div class="text-[10px] sm:text-xs text-gray-500 font-mono mt-0.5">
                {{ container.id.substring(0, 12) }}
              </div>
            </div>
          </div>
          
          <!-- Status Badge -->
          <div class="flex-shrink-0 flex flex-col gap-1">
            <div :class="container.state === 'running' ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600'"
              class="px-2 sm:px-2.5 py-1 rounded-lg text-[10px] sm:text-xs font-semibold flex items-center gap-1 sm:gap-1.5">
              <span :class="container.state === 'running' ? 'bg-green-500' : 'bg-gray-400'" 
                class="w-1.5 h-1.5 sm:w-2 sm:h-2 rounded-full animate-pulse"></span>
              <span class="hidden sm:inline">{{ container.state }}</span>
            </div>
            
            <!-- Temporary Badge -->
            <div v-if="isTemporary(container)" 
              :class="getExpirationInfo(container).isExpiringSoon ? 'bg-red-100 text-red-700' : 'bg-orange-100 text-orange-700'"
              class="px-2 sm:px-2.5 py-1 rounded-lg text-[10px] sm:text-xs font-semibold flex items-center gap-1"
              :title="`Expires at ${new Date(parseInt(container.labels['yantra.expireAt']) * 1000).toLocaleString()}`">
              ⏱️ <span class="hidden sm:inline">{{ getExpirationInfo(container).timeRemaining }}</span>
            </div>
          </div>
        </div>
        
        <!-- Container Info -->
        <div class="space-y-2 sm:space-y-2.5 mb-3 sm:mb-4">
          <div class="flex items-start gap-2">
            <span class="text-[10px] sm:text-xs text-gray-500 font-semibold mt-0.5 flex-shrink-0">Image:</span>
            <span class="text-[10px] sm:text-xs text-gray-700 font-mono break-all">{{ container.image }}</span>
          </div>
          <div class="flex items-center gap-2">
            <span class="text-[10px] sm:text-xs text-gray-500 font-semibold">Status:</span>
            <span class="text-[10px] sm:text-xs text-gray-700">{{ container.status }}</span>
          </div>
          
          <!-- Temporary Installation Warning -->
          <div v-if="isTemporary(container)" 
            :class="getExpirationInfo(container).isExpiringSoon ? 'bg-red-50 border-red-200 text-red-700' : 'bg-orange-50 border-orange-200 text-orange-700'"
            class="mt-2 px-2 py-1.5 rounded-lg border text-[10px] sm:text-xs">
            <div class="font-semibold mb-0.5">⏱️ Temporary Installation</div>
            <div class="opacity-80">
              Auto-deletes in {{ getExpirationInfo(container).timeRemaining }}
            </div>
          </div>
        </div>

        <!-- View Details Footer -->
        <div class="flex items-center justify-between pt-3 border-t border-gray-100">
          <span class="text-xs sm:text-sm text-gray-500 group-hover:text-blue-600 transition-colors font-medium">
            View Details
          </span>
          <ArrowRight :size="14" class="sm:w-4 sm:h-4 text-gray-400 group-hover:text-blue-600 group-hover:translate-x-1 transition-all" />
        </div>
      </div>
        </div>
      </div>

      <!-- Volume Browsers Section -->
      <div v-if="volumeBrowsers.length > 0">
        <h3 class="text-xl font-bold text-gray-900 mb-4 flex items-center gap-2">
          <span class="text-purple-600">📦</span>
          Volume Browsers
          <span class="text-sm font-normal text-gray-500">({{ volumeBrowsers.length }})</span>
        </h3>
        
        <!-- Info Message -->
        <div class="mb-4 p-3 sm:p-4 bg-purple-50 border border-purple-200 rounded-xl">
          <p class="text-xs sm:text-sm text-purple-800">
            <span class="font-semibold">💡 Safe to Close:</span> These are temporary file browsers. Deleting them won't affect your data - the volumes remain intact. You can always start a new browser from the Volumes page.
          </p>
        </div>
        
        <div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3 sm:gap-4">
          <div v-for="(container, index) in volumeBrowsers" :key="container.id"
            :style="{ animationDelay: `${index * 50}ms` }"
            @click="viewContainerDetail(container)"
            class="group bg-purple-50/50 backdrop-blur-sm rounded-xl sm:rounded-2xl p-4 sm:p-5 transition-all duration-300 hover:bg-purple-50 hover:shadow-xl hover:scale-[1.02] active:scale-[0.98] cursor-pointer animate-fadeIn touch-manipulation border border-purple-200">
            
            <!-- Header with Logo and Status -->
            <div class="flex items-start justify-between mb-3 sm:mb-4">
              <div class="flex items-center gap-2.5 sm:gap-3 flex-1 min-w-0">
                <div class="relative flex-shrink-0">
                  <span class="text-3xl sm:text-4xl">📦</span>
                </div>
                <div class="flex-1 min-w-0">
                  <h3 class="font-bold text-gray-900 text-base sm:text-lg truncate group-hover:text-purple-600 transition-colors">
                    {{ container.labels['yantra.volume-browser'] }}
                  </h3>
                  <div class="text-[10px] sm:text-xs text-gray-500 font-mono mt-0.5">
                    Volume Browser
                  </div>
                </div>
              </div>
              
              <!-- Status Badge -->
              <div class="flex-shrink-0">
                <div :class="container.state === 'running' ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600'"
                  class="px-2 sm:px-2.5 py-1 rounded-lg text-[10px] sm:text-xs font-semibold flex items-center gap-1 sm:gap-1.5">
                  <span :class="container.state === 'running' ? 'bg-green-500' : 'bg-gray-400'" 
                    class="w-1.5 h-1.5 sm:w-2 sm:h-2 rounded-full animate-pulse"></span>
                  <span class="hidden sm:inline">{{ container.state }}</span>
                </div>
              </div>
            </div>
            
            <!-- Container Info -->
            <div class="space-y-2 sm:space-y-2.5 mb-3 sm:mb-4">
              <div class="flex items-center gap-2">
                <span class="text-[10px] sm:text-xs text-gray-500 font-semibold">Status:</span>
                <span class="text-[10px] sm:text-xs text-gray-700">{{ container.status }}</span>
              </div>
            </div>

            <!-- View Details Footer -->
            <div class="flex items-center justify-between pt-3 border-t border-purple-200">
              <span class="text-xs sm:text-sm text-gray-500 group-hover:text-purple-600 transition-colors font-medium">
                View Details
              </span>
              <ArrowRight :size="14" class="sm:w-4 sm:h-4 text-gray-400 group-hover:text-purple-600 group-hover:translate-x-1 transition-all" />
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
