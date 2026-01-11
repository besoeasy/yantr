<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()

const containers = ref([])
const loading = ref(false)
const apiUrl = ref('')

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

function viewContainerDetail(container) {
  router.push(`/containers/${container.id}`)
}

onMounted(async () => {
  loading.value = true
  await fetchContainers()
  loading.value = false
  
  // Auto-refresh every 10 seconds
  setInterval(() => {
    fetchContainers()
  }, 10000)
})
</script>

<template>
  <div class="p-6 md:p-10 lg:p-12">
    <h2 class="text-5xl font-bold mb-12 text-gray-900 tracking-tight">Containers</h2>
    <p class="text-gray-600 mb-8 -mt-8">All running Docker containers</p>
    
    <div v-if="loading" class="text-center py-16">
      <div class="text-gray-500 font-medium">Loading containers...</div>
    </div>
    <div v-else-if="containers.length === 0" class="text-center py-16">
      <div class="text-5xl mb-4">🎯</div>
      <div class="text-gray-900 font-bold text-2xl mb-2">Welcome to Yantra!</div>
      <div class="text-gray-500 font-medium mb-4">No containers running yet</div>
      <div class="text-sm text-gray-400 mb-6">Get started by installing apps from the App Store</div>
      <router-link to="/apps"
        class="inline-flex items-center gap-2 px-6 py-3 bg-blue-500 hover:bg-blue-600 text-white rounded-xl font-semibold transition-all">
        <i class="fas fa-store"></i>
        <span>Browse App Store</span>
      </router-link>
    </div>
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <div v-for="container in containers" :key="container.id"
        @click="viewContainerDetail(container)"
        class="bg-white rounded-2xl p-6 border border-gray-200 hover:border-purple-300 transition-all duration-300 hover:shadow-xl cursor-pointer">
        <div class="flex items-start justify-between mb-4">
          <div class="flex-1">
            <div class="flex items-center gap-3 mb-2">
              <img v-if="container.app.logo" :src="container.app.logo" :alt="container.name"
                class="w-12 h-12 rounded-lg object-cover">
              <span v-else class="text-3xl">🐳</span>
              <div>
                <h3 class="font-bold text-gray-900 text-lg">{{ container.name }}</h3>
                <div class="text-xs text-gray-500 font-mono mt-1">{{ container.id.substring(0, 12) }}</div>
              </div>
            </div>
          </div>
          <div :class="container.state === 'running' ? 'bg-green-500' : 'bg-gray-400'" 
            class="w-3 h-3 rounded-full"></div>
        </div>
        
        <div class="space-y-2 mb-4">
          <div class="flex items-center gap-2 text-sm">
            <span class="text-gray-500 font-semibold">Image:</span>
            <span class="text-gray-700 font-mono text-xs truncate">{{ container.image }}</span>
          </div>
          <div class="flex items-center gap-2 text-sm">
            <span class="text-gray-500 font-semibold">Status:</span>
            <span class="text-gray-700">{{ container.status }}</span>
          </div>
        </div>

        <div class="text-sm text-gray-500 flex items-center gap-2">
          <i class="fas fa-arrow-right"></i>
          <span>Click to view details</span>
        </div>
      </div>
    </div>
  </div>
</template>
