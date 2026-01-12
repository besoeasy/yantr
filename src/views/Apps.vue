<script setup>
import { ref, computed, onMounted } from 'vue'
import { useToast } from 'vue-toastification'
import { Globe, FileCode } from 'lucide-vue-next'

const toast = useToast()

// State
const apps = ref([])
const containers = ref([])
const loading = ref(false)
const deploying = ref(null)
const showEnvModal = ref(false)
const selectedApp = ref(null)
const envValues = ref({})
const appSearch = ref('')
const apiUrl = ref('')
const musthaveapps = ['dufs', 'watchtower']
const temporaryInstall = ref(false)
const expirationHours = ref(24)

// Computed
const installedAppIds = computed(() => {
  const ids = new Set(containers.value.map(c => c.app.id))
  return ids
})

const uninstalledApps = computed(() => {
  const uninstalled = apps.value.filter(app => !installedAppIds.value.has(app.id))
  return shuffleWithSeed(uninstalled).map(app => ({
    ...app,
    isInstalled: false
  }))
})

const allAppsCount = computed(() => uninstalledApps.value.length)

const combinedApps = computed(() => {
  let combined = [...uninstalledApps.value]
  
  if (appSearch.value) {
    const search = appSearch.value.toLowerCase()
    combined = combined.filter(app => {
      return app.name.toLowerCase().includes(search) ||
        app.category.toLowerCase().includes(search) ||
        (app.description && app.description.toLowerCase().includes(search))
    })
  }
  
  return combined
})

// Helper Functions
function getDateHourSeed() {
  const now = new Date()
  const year = now.getFullYear()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  const hour = String(now.getHours()).padStart(2, '0')
  return `${year}-${month}-${day}-${hour}`
}

function seededRandom(seed) {
  let x = Math.sin(seed) * 10000
  return x - Math.floor(x)
}

function shuffleWithSeed(array) {
  const dateHourSeed = getDateHourSeed()
  let numericSeed = 0
  for (let i = 0; i < dateHourSeed.length; i++) {
    numericSeed += dateHourSeed.charCodeAt(i) * (i + 1)
  }
  
  const shuffled = [...array]
  
  for (let i = shuffled.length - 1; i > 0; i--) {
    const j = Math.floor(seededRandom(numericSeed + i) * (i + 1));
    [shuffled[i], shuffled[j]] = [shuffled[j], shuffled[i]]
  }
  
  return shuffled
}

// API Functions
async function fetchApps() {
  try {
    const response = await fetch(`${apiUrl.value}/api/apps`)
    const data = await response.json()
    if (data.success) {
      apps.value = data.apps
    }
  } catch (error) {
    console.error('Failed to fetch apps:', error)
  }
}

async function fetchContainers() {
  try {
    const response = await fetch(`${apiUrl.value}/api/containers`)
    const data = await response.json()
    if (data.success) {
      containers.value = data.containers.filter(c => c.state === 'running')
    }
  } catch (error) {
    console.error('Failed to fetch containers:', error)
  }
}

async function deployApp(appId) {
  const app = apps.value.find(a => a.id === appId)

  if (!app) return

  // Always show modal to allow temporary installation option
  selectedApp.value = app
  envValues.value = {}
  if (app.environment && app.environment.length > 0) {
    app.environment.forEach(env => {
      envValues.value[env.envVar] = env.default
    })
  }
  showEnvModal.value = true
}

async function confirmDeploy(appId, environment) {
  showEnvModal.value = false
  deploying.value = appId
  
  toast.info(`Deploying ${appId}... This may take a few minutes.`)
  
  try {
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 300000)
    
    const requestBody = { appId, environment }
    if (temporaryInstall.value) {
      requestBody.expiresIn = expirationHours.value
    }
    
    const response = await fetch(`${apiUrl.value}/api/deploy`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(requestBody),
      signal: controller.signal
    })

    clearTimeout(timeoutId)
    const data = await response.json()

    if (data.success) {
      if (data.temporary) {
        toast.success(`${appId} deployed as temporary (expires in ${expirationHours.value}h)!`)
      } else {
        toast.success(`${appId} deployed successfully!`)
      }
      await fetchContainers()
    } else {
      toast.error(`Deployment failed: ${data.error}`)
    }
  } catch (error) {
    if (error.name === 'AbortError') {
      toast.error(`Deployment timeout - ${appId} is taking longer than expected`)
    } else {
      toast.error(`Deployment failed: ${error.message}`)
    }
  } finally {
    deploying.value = null
    selectedApp.value = null
    temporaryInstall.value = false
    expirationHours.value = 24
  }
}

function cancelDeploy() {
  showEnvModal.value = false
  selectedApp.value = null
  envValues.value = {}
}

// Lifecycle
onMounted(async () => {
  loading.value = true
  await Promise.all([fetchApps(), fetchContainers()])
  loading.value = false
  
  // Auto-refresh every 10 seconds
  setInterval(() => {
    fetchContainers()
  }, 10000)
})
</script>

<template>
  <div class="p-4 sm:p-6 md:p-10 lg:p-12">
    <h2 class="text-3xl sm:text-4xl md:text-5xl font-bold mb-6 md:mb-12 text-gray-900 tracking-tight">App Store</h2>
    <p class="text-gray-600 mb-6 md:mb-8 -mt-3 md:-mt-8">Browse and install apps</p>
    
    <!-- Search Bar -->
    <div class="mb-6 md:mb-8">
      <div class="relative">
        <input v-model="appSearch" type="text"
          placeholder="Search apps..."
          class="w-full px-4 sm:px-5 py-3 sm:py-4 pl-11 sm:pl-12 bg-white border border-gray-200 rounded-xl text-gray-900 placeholder-gray-400 transition-all text-base">
        <span class="absolute left-3 sm:left-4 top-1/2 -translate-y-1/2 text-lg sm:text-xl">🔍</span>
        <button v-if="appSearch" @click="appSearch = ''"
          class="absolute right-3 sm:right-4 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors p-1 touch-manipulation"
          title="Clear search">
          ✕
        </button>
      </div>
      <div v-if="combinedApps.length < allAppsCount" class="mt-2 text-sm text-gray-500">
        Showing {{ combinedApps.length }} of {{ allAppsCount }} apps
      </div>
    </div>

    <div v-if="loading" class="text-center py-16">
      <div class="text-gray-500 font-medium">Loading apps...</div>
    </div>
    <div v-else-if="combinedApps.length === 0" class="text-center py-16">
      <div class="text-5xl mb-4">🔍</div>
      <div class="text-gray-500 font-medium">No apps found</div>
      <div class="text-sm text-gray-400 mt-2">Try a different search term</div>
    </div>
    <div v-else class="grid grid-cols-1 gap-3 sm:gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
      <div v-for="(app, index) in combinedApps" :key="app.id"
        :style="{ animationDelay: `${index * 30}ms` }"
        class="group bg-white/80 backdrop-blur-sm rounded-2xl sm:rounded-3xl p-4 sm:p-6 transition-all duration-500 ease-out hover:bg-white hover:shadow-xl hover:scale-[1.02] active:scale-[0.98] cursor-pointer flex flex-col animate-fadeIn">
        
        <!-- App Logo & Name Section -->
        <div class="flex items-center gap-3 sm:gap-4 mb-3 sm:mb-4">
          <div class="relative flex-shrink-0">
            <!-- Recommended Badge on Logo -->
            <div v-if="musthaveapps.includes(app.id)" 
              class="absolute -top-1 -right-1 sm:-top-1.5 sm:-right-1.5 w-4 h-4 sm:w-5 sm:h-5 bg-gradient-to-br from-yellow-400 to-yellow-500 rounded-full flex items-center justify-center shadow-lg z-10 animate-pulse">
              <span class="text-[10px] sm:text-xs">⭐</span>
            </div>
            <img :src="app.logo" :alt="app.name"
              class="w-12 h-12 sm:w-16 sm:h-16 object-contain transition-all duration-300 group-hover:scale-110 group-hover:rotate-3">
          </div>
          
          <div class="flex-1 min-w-0">
            <h3 class="font-bold text-base sm:text-lg text-gray-900 truncate mb-1 group-hover:text-blue-600 transition-colors">
              {{ app.name }}
            </h3>
            <div class="flex flex-wrap gap-1">
              <span v-for="(cat, idx) in app.category.split(',').slice(0, 2)" :key="cat"
                class="text-xs px-2 py-0.5 rounded-md bg-blue-50/30 text-blue-700 font-medium">
                {{ cat.trim() }}
              </span>
            </div>
          </div>
        </div>

        <!-- Description -->
        <p class="text-sm text-gray-600 leading-relaxed mb-3 sm:mb-4 line-clamp-2 flex-1">
          {{ app.description || 'No description available' }}
        </p>

        <!-- Actions -->
        <div class="flex items-center gap-2">
          <button @click="deployApp(app.id)" :disabled="deploying === app.id"
            :class="deploying === app.id ? 'bg-gray-400' : 'bg-blue-600 hover:bg-blue-700 active:scale-95'"
            class="flex-1 px-3 sm:px-4 py-2 sm:py-2.5 text-white rounded-xl text-sm font-semibold transition-all duration-200 shadow-sm hover:shadow-md disabled:cursor-not-allowed touch-manipulation">
            <span v-if="deploying === app.id" class="inline-flex items-center gap-2">
              <span class="w-3 h-3 sm:w-4 sm:h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
              <span class="hidden sm:inline">Installing...</span>
              <span class="sm:hidden">...</span>
            </span>
            <span v-else>Install</span>
          </button>
          
          <a v-if="app.website" :href="app.website" target="_blank"
            class="p-2 sm:p-2.5 rounded-xl bg-gray-100 hover:bg-blue-100 text-gray-600 hover:text-blue-600 transition-all duration-200 hover:scale-110 active:scale-95 touch-manipulation"
            title="Visit Website">
            <Globe :size="16" class="sm:w-[18px] sm:h-[18px]" />
          </a>
          
          <a :href="`https://github.com/besoeasy/yantra/blob/main/apps/${app.id}/compose.yml`"
            target="_blank"
            class="p-2 sm:p-2.5 rounded-xl bg-gray-100 hover:bg-purple-100 text-gray-600 hover:text-purple-600 transition-all duration-200 hover:scale-110 active:scale-95 touch-manipulation"
            title="View Source">
            <FileCode :size="16" class="sm:w-[18px] sm:h-[18px]" />
          </a>
        </div>
      </div>
    </div>

    <!-- Environment Variables Modal -->
    <div v-if="showEnvModal"
      class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4"
      @click.self="cancelDeploy">
      <div class="bg-white rounded-2xl sm:rounded-3xl p-6 sm:p-8 max-w-md w-full shadow-2xl">
        <h2 class="text-xl sm:text-2xl font-bold mb-2 text-gray-900">
          {{ selectedApp?.environment?.length > 0 ? 'Configure' : 'Deploy' }} {{ selectedApp?.name }}
        </h2>
        <p class="text-sm text-gray-600 mb-6">
          {{ selectedApp?.environment?.length > 0 ? 'Set environment variables for deployment' : 'Choose deployment options' }}
        </p>

        <div class="space-y-4 mb-6 sm:mb-8">
          <!-- Environment Variables Section (only if app has env vars) -->
          <div v-if="selectedApp?.environment?.length > 0" class="space-y-4 mb-4">
            <div v-for="env in selectedApp?.environment" :key="env.envVar" class="space-y-2">
              <label class="block text-sm font-semibold text-gray-700">
                {{ env.name }}
                <span v-if="env.default" class="text-gray-400 text-xs font-normal ml-1">(default: {{ env.default }})</span>
              </label>
              <input v-model="envValues[env.envVar]" type="text" :placeholder="env.default"
                class="w-full px-4 py-3 glass rounded-xl text-gray-900 placeholder-gray-400 transition-all text-base">
            </div>
          </div>
          
          <!-- Temporary Installation Option -->
          <div :class="selectedApp?.environment?.length > 0 ? 'pt-4 border-t border-gray-200' : ''">
            <label class="flex items-center gap-3 cursor-pointer group">
              <input type="checkbox" v-model="temporaryInstall" 
                class="w-5 h-5 rounded border-gray-300 text-blue-600 focus:ring-2 focus:ring-blue-500 cursor-pointer">
              <div class="flex-1">
                <span class="text-sm font-semibold text-gray-700 group-hover:text-blue-600 transition-colors">
                  ⏱️ Temporary Installation
                </span>
                <p class="text-xs text-gray-500 mt-0.5">Auto-delete after specified time</p>
              </div>
            </label>
            
            <div v-if="temporaryInstall" class="mt-3 ml-8 space-y-2 animate-fadeIn">
              <label class="block text-xs font-medium text-gray-600">Expires in (hours)</label>
              <select v-model.number="expirationHours" 
                class="w-full px-3 py-2 glass rounded-lg text-sm text-gray-900 transition-all">
                <option :value="1">1 hour</option>
                <option :value="6">6 hours</option>
                <option :value="12">12 hours</option>
                <option :value="24">24 hours (1 day)</option>
                <option :value="72">72 hours (3 days)</option>
                <option :value="168">168 hours (1 week)</option>
              </select>
            </div>
          </div>
        </div>

        <div class="flex gap-3">
          <button @click="confirmDeploy(selectedApp.id, envValues)"
            class="flex-1 px-4 sm:px-5 py-3 bg-blue-600 hover:bg-blue-700 text-white rounded-xl font-semibold transition-all shadow-md hover:shadow-lg active:scale-95 touch-manipulation">
            Deploy
          </button>
          <button @click="cancelDeploy"
            class="px-4 sm:px-5 py-3 bg-gray-100 hover:bg-gray-200 text-gray-700 rounded-xl font-medium transition-all active:scale-95 touch-manipulation">
            Cancel
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
