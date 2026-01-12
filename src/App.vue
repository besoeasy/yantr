<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { Box, Boxes, Images, ClipboardList, AlertTriangle, Github, Heart, X, RefreshCw } from 'lucide-vue-next'
import { useApi } from './composables/useApi.js'

const route = useRoute()
const { get } = useApi()

const isActive = (name) => route.name === name

// Update notification state
const showUpdateBanner = ref(false)
const updateMessage = ref('')

// Check for pending updates
const checkForUpdates = async () => {
  try {
    const response = await get('/api/updates/pending', { skipCache: true, showError: false })
    if (response.success && response.hasPendingUpdate) {
      showUpdateBanner.value = true
      updateMessage.value = response.message
    }
  } catch (err) {
    // Silently fail - don't disturb user experience
    console.log('Update check skipped:', err.message)
  }
}

const dismissUpdate = () => {
  showUpdateBanner.value = false
  // Store dismissal in localStorage to avoid showing again this session
  localStorage.setItem('updateDismissed', Date.now().toString())
}

onMounted(() => {
  // Check if user dismissed update in this session (within last 24 hours)
  const dismissed = localStorage.getItem('updateDismissed')
  const shouldCheck = !dismissed || (Date.now() - parseInt(dismissed)) > 24 * 60 * 60 * 1000
  
  if (shouldCheck) {
    checkForUpdates()
  }
})
</script>

<template>
  <div class="min-h-screen flex flex-col md:flex-row bg-white text-gray-900">
    <!-- Update Notification Banner -->
    <transition 
      enter-active-class="transition-all duration-300 ease-out"
      enter-from-class="transform -translate-y-full opacity-0"
      enter-to-class="transform translate-y-0 opacity-100"
      leave-active-class="transition-all duration-200 ease-in"
      leave-from-class="transform translate-y-0 opacity-100"
      leave-to-class="transform -translate-y-full opacity-0">
      <div 
        v-if="showUpdateBanner"
        class="fixed top-0 left-0 right-0 z-[100] bg-gradient-to-r from-blue-500 to-blue-600 text-white shadow-lg md:ml-20">
        <div class="container mx-auto px-4 py-3 flex items-center justify-between gap-4">
          <div class="flex items-center gap-3 flex-1">
            <RefreshCw :size="20" class="flex-shrink-0 animate-spin" style="animation-duration: 3s;" />
            <p class="text-sm md:text-base font-medium">
              {{ updateMessage }}
            </p>
          </div>
          <button
            @click="dismissUpdate"
            class="flex-shrink-0 p-2 hover:bg-white/20 rounded-lg transition-colors active:scale-95"
            aria-label="Dismiss">
            <X :size="18" />
          </button>
        </div>
      </div>
    </transition>

    <!-- Desktop Sidebar -->
    <aside class="hidden md:flex bg-white flex-col items-center border-r border-gray-200 w-20 py-6 px-2 fixed h-screen z-50">
      <!-- Logo -->
      <h1 class="text-lg font-bold text-gray-900 mb-8 uppercase leading-tight text-center">
        Yan<br/>tra
      </h1>

      <!-- Navigation -->
      <nav class="flex flex-col items-center gap-3 mb-auto">
        <!-- Apps Tab -->
        <router-link 
          to="/apps"
          :class="isActive('apps') ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-100'"
          class="relative w-12 h-12 rounded-full flex items-center justify-center transition-all smooth-shadow"
          title="Apps">
          <Box :size="20" />
        </router-link>

        <!-- Containers Tab -->
        <router-link 
          to="/containers"
          :class="isActive('containers') ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-100'"
          class="relative w-12 h-12 rounded-full flex items-center justify-center transition-all smooth-shadow"
          title="Containers">
          <Boxes :size="20" />
        </router-link>

        <!-- Images Tab -->
        <router-link 
          to="/images"
          :class="isActive('images') ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-100'"
          class="relative w-12 h-12 rounded-full flex items-center justify-center transition-all smooth-shadow"
          title="Images">
          <Images :size="20" />
        </router-link>

        <!-- Logs Tab -->
        <router-link 
          to="/logs"
          :class="isActive('logs') ? 'bg-gray-900 text-white' : 'text-gray-600 hover:bg-gray-100'"
          class="relative w-12 h-12 rounded-full flex items-center justify-center transition-all smooth-shadow"
          title="Logs">
          <ClipboardList :size="20" />
        </router-link>
      </nav>

      <!-- Bottom Actions -->
      <div class="flex flex-col items-center gap-3 mt-4">
        <!-- Donate -->
        <a 
          href="https://sponsor.besoeasy.com/" 
          target="_blank"
          class="w-12 h-12 rounded-full flex items-center justify-center text-red-500 hover:bg-red-50 transition-all"
          title="Support this project">
          <Heart :size="20" />
        </a>

        <!-- GitHub -->
        <a 
          href="https://github.com/besoeasy/Yantra" 
          target="_blank"
          class="w-12 h-12 rounded-full flex items-center justify-center text-gray-600 hover:bg-gray-100 transition-all"
          title="View on GitHub">
          <Github :size="20" />
        </a>

        <!-- Report Issue -->
        <a 
          href="https://github.com/besoeasy/Yantra/issues/new" 
          target="_blank"
          class="w-12 h-12 rounded-full flex items-center justify-center text-gray-600 hover:bg-gray-100 transition-all"
          title="Report Issue">
          <AlertTriangle :size="20" />
        </a>
      </div>
    </aside>

    <!-- Mobile Bottom Navigation -->
    <nav class="md:hidden fixed bottom-0 left-0 right-0 bg-white border-t border-gray-200 z-50 safe-area-inset-bottom">
      <div class="flex items-center justify-around px-2 py-3">
        <router-link 
          to="/apps"
          :class="isActive('apps') ? 'bg-gray-900 text-white' : 'text-gray-600'"
          class="flex flex-col items-center gap-1 px-4 py-2 rounded-xl transition-all active:scale-95"
          title="Apps">
          <Box :size="20" />
          <span class="text-xs font-medium">Apps</span>
        </router-link>

        <router-link 
          to="/containers"
          :class="isActive('containers') ? 'bg-gray-900 text-white' : 'text-gray-600'"
          class="flex flex-col items-center gap-1 px-4 py-2 rounded-xl transition-all active:scale-95"
          title="Containers">
          <Boxes :size="20" />
          <span class="text-xs font-medium">Containers</span>
        </router-link>

        <router-link 
          to="/images"
          :class="isActive('images') ? 'bg-gray-900 text-white' : 'text-gray-600'"
          class="flex flex-col items-center gap-1 px-4 py-2 rounded-xl transition-all active:scale-95"
          title="Images">
          <Images :size="20" />
          <span class="text-xs font-medium">Images</span>
        </router-link>

        <router-link 
          to="/logs"
          :class="isActive('logs') ? 'bg-gray-900 text-white' : 'text-gray-600'"
          class="flex flex-col items-center gap-1 px-4 py-2 rounded-xl transition-all active:scale-95"
          title="Logs">
          <ClipboardList :size="20" />
          <span class="text-xs font-medium">Logs</span>
        </router-link>
      </div>
    </nav>

    <!-- Main Content -->
    <main 
      class="flex-1 min-h-screen md:ml-20 pb-20 md:pb-0 transition-all duration-300"
      :class="{ 'pt-16': showUpdateBanner }">
      <router-view />
    </main>
  </div>
</template>