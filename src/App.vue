<script setup>
import { useRoute } from 'vue-router'
import { Box, Boxes, Layers, HardDrive, ClipboardList, Send, Github, Heart, Home } from 'lucide-vue-next'

const route = useRoute()

const isActive = (name) => route.name === name
</script>

<template>
  <div class="min-h-screen flex flex-col md:flex-row bg-white text-gray-900">
    <!-- Desktop Sidebar -->
    <aside class="hidden md:flex bg-white flex-col items-center border-r border-gray-200 w-20 py-6 px-2 fixed h-screen z-50">
      <!-- Logo -->
      <h1 class="text-lg font-bold text-gray-900 mb-8 uppercase leading-tight text-center">
        Yan<br/>tra
      </h1>

      <!-- Navigation -->
      <nav class="flex flex-col items-center gap-3 mb-auto">
        <!-- Home Tab -->
        <router-link 
          to="/home"
          :class="isActive('home') ? 'bg-gray-900 text-white shadow-lg shadow-gray-900/20' : 'text-gray-600 hover:bg-gray-100 hover:shadow-md hover:shadow-gray-900/10'"
          class="nav-item group relative w-12 h-12 rounded-full flex items-center justify-center transition-all duration-300 ease-out smooth-shadow"
          title="Home">
          <Home :size="20" class="group-hover:scale-110 transition-transform duration-300" />
          <span class="absolute left-full ml-3 px-3 py-1.5 bg-gray-900 text-white text-xs font-semibold rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-300 whitespace-nowrap pointer-events-none">Home</span>
        </router-link>

        <!-- Apps Tab -->
        <router-link 
          to="/apps"
          :class="isActive('apps') ? 'bg-gray-900 text-white shadow-lg shadow-gray-900/20' : 'text-gray-600 hover:bg-gray-100 hover:shadow-md hover:shadow-gray-900/10'"
          class="nav-item group relative w-12 h-12 rounded-full flex items-center justify-center transition-all duration-300 ease-out smooth-shadow"
          title="Apps">
          <Box :size="20" class="group-hover:scale-110 transition-transform duration-300" />
          <span class="absolute left-full ml-3 px-3 py-1.5 bg-gray-900 text-white text-xs font-semibold rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-300 whitespace-nowrap pointer-events-none">Apps</span>
        </router-link>

     

        <!-- Images Tab -->
        <router-link 
          to="/images"
          :class="isActive('images') ? 'bg-gray-900 text-white shadow-lg shadow-gray-900/20' : 'text-gray-600 hover:bg-gray-100 hover:shadow-md hover:shadow-gray-900/10'"
          class="nav-item group relative w-12 h-12 rounded-full flex items-center justify-center transition-all duration-300 ease-out smooth-shadow"
          title="Images">
          <Layers :size="20" class="group-hover:scale-110 transition-transform duration-300" />
          <span class="absolute left-full ml-3 px-3 py-1.5 bg-gray-900 text-white text-xs font-semibold rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-300 whitespace-nowrap pointer-events-none">Images</span>
        </router-link>

        <!-- Volumes Tab -->
        <router-link 
          to="/volumes"
          :class="isActive('volumes') ? 'bg-gray-900 text-white shadow-lg shadow-gray-900/20' : 'text-gray-600 hover:bg-gray-100 hover:shadow-md hover:shadow-gray-900/10'"
          class="nav-item group relative w-12 h-12 rounded-full flex items-center justify-center transition-all duration-300 ease-out smooth-shadow"
          title="Volumes">
          <HardDrive :size="20" class="group-hover:scale-110 transition-transform duration-300" />
          <span class="absolute left-full ml-3 px-3 py-1.5 bg-gray-900 text-white text-xs font-semibold rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-300 whitespace-nowrap pointer-events-none">Volumes</span>
        </router-link>

        <!-- Logs Tab -->
        <router-link 
          to="/logs"
          :class="isActive('logs') ? 'bg-gray-900 text-white shadow-lg shadow-gray-900/20' : 'text-gray-600 hover:bg-gray-100 hover:shadow-md hover:shadow-gray-900/10'"
          class="nav-item group relative w-12 h-12 rounded-full flex items-center justify-center transition-all duration-300 ease-out smooth-shadow"
          title="Logs">
          <ClipboardList :size="20" class="group-hover:scale-110 transition-transform duration-300" />
          <span class="absolute left-full ml-3 px-3 py-1.5 bg-gray-900 text-white text-xs font-semibold rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-300 whitespace-nowrap pointer-events-none">Logs</span>
        </router-link>
      </nav>

      <!-- Bottom Actions -->
      <div class="flex flex-col items-center gap-3 mt-4">
        <!-- Donate -->
        <a 
          href="https://sponsor.besoeasy.com/" 
          target="_blank"
          class="action-btn group relative w-12 h-12 rounded-full flex items-center justify-center text-red-500 hover:bg-red-50 hover:text-red-600 hover:shadow-md hover:shadow-red-500/20 transition-all duration-300 ease-out"
          title="Support this project">
          <Heart :size="20" class="group-hover:scale-110 group-hover:-rotate-12 transition-transform duration-300" />
          <span class="absolute left-full ml-3 px-3 py-1.5 bg-gray-900 text-white text-xs font-semibold rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-300 whitespace-nowrap pointer-events-none">Sponsor</span>
        </a>

        <!-- GitHub -->
        <a 
          href="https://github.com/besoeasy/Yantra" 
          target="_blank"
          class="action-btn group relative w-12 h-12 rounded-full flex items-center justify-center text-gray-600 hover:bg-gray-100 hover:shadow-md hover:shadow-gray-900/10 transition-all duration-300 ease-out"
          title="View on GitHub">
          <Github :size="20" class="group-hover:scale-110 group-hover:rotate-12 transition-transform duration-300" />
          <span class="absolute left-full ml-3 px-3 py-1.5 bg-gray-900 text-white text-xs font-semibold rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-300 whitespace-nowrap pointer-events-none">GitHub</span>
        </a>

        <!-- Telegram Group -->
        <a 
          href="https://t.me/+Qu06yCZHBAU3NTk1" 
          target="_blank"
          class="action-btn group relative w-12 h-12 rounded-full flex items-center justify-center text-blue-500 hover:bg-blue-50 hover:shadow-md hover:shadow-blue-500/20 transition-all duration-300 ease-out"
          title="Join Telegram Group">
          <Send :size="20" class="group-hover:scale-110 group-hover:translate-x-1 group-hover:-translate-y-1 transition-transform duration-300" />
          <span class="absolute left-full ml-3 px-3 py-1.5 bg-gray-900 text-white text-xs font-semibold rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-300 whitespace-nowrap pointer-events-none">Telegram</span>
        </a>
      </div>
    </aside>

    <!-- Mobile Bottom Navigation -->
    <nav class="md:hidden fixed bottom-0 left-0 right-0 bg-white border-t border-gray-200 z-50 safe-area-inset-bottom">
      <div class="flex items-center justify-around px-2 py-3">
        <router-link 
          to="/home"
          :class="isActive('home') ? 'bg-gray-900 text-white' : 'text-gray-600'"
          class="flex flex-col items-center gap-1 px-4 py-2 rounded-xl transition-all active:scale-95"
          title="Home">
          <Home :size="20" />
          <span class="text-xs font-medium">Home</span>
        </router-link>

        <router-link 
          to="/apps"
          :class="isActive('apps') ? 'bg-gray-900 text-white' : 'text-gray-600'"
          class="flex flex-col items-center gap-1 px-4 py-2 rounded-xl transition-all active:scale-95"
          title="Apps">
          <Box :size="20" />
          <span class="text-xs font-medium">Apps</span>
        </router-link>


        <router-link 
          to="/images"
          :class="isActive('images') ? 'bg-gray-900 text-white' : 'text-gray-600'"
          class="flex flex-col items-center gap-1 px-4 py-2 rounded-xl transition-all active:scale-95"
          title="Images">
          <Layers :size="20" />
          <span class="text-xs font-medium">Images</span>
        </router-link>

        <router-link 
          to="/volumes"
          :class="isActive('volumes') ? 'bg-gray-900 text-white' : 'text-gray-600'"
          class="flex flex-col items-center gap-1 px-4 py-2 rounded-xl transition-all active:scale-95"
          title="Volumes">
          <HardDrive :size="20" />
          <span class="text-xs font-medium">Volumes</span>
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
    <main class="flex-1 min-h-screen md:ml-20 pb-20 md:pb-0">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
/* Navigation item hover animations */
.nav-item {
  position: relative;
}

.nav-item::before {
  content: '';
  position: absolute;
  inset: -2px;
  border-radius: 9999px;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.1), rgba(139, 92, 246, 0.1));
  opacity: 0;
  transition: opacity 300ms ease-out;
  pointer-events: none;
  z-index: -1;
}

.nav-item:hover::before {
  opacity: 1;
}

/* Action button animations */
.action-btn {
  position: relative;
}

.action-btn::before {
  content: '';
  position: absolute;
  inset: -2px;
  border-radius: 9999px;
  background: currentColor;
  opacity: 0;
  transition: opacity 300ms ease-out;
  pointer-events: none;
  z-index: -1;
  filter: brightness(0.9);
}

.action-btn:hover::before {
  opacity: 0.05;
}

/* Tooltip animations */
.nav-item span,
.action-btn span {
  animation: tooltipSlideIn 300ms ease-out forwards;
}

@keyframes tooltipSlideIn {
  from {
    opacity: 0;
    transform: translateX(-8px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

/* Optional: Add ripple effect on click */
.nav-item,
.action-btn {
  overflow: hidden;
}

.nav-item::after,
.action-btn::after {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 9999px;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.4) 0%, transparent 70%);
  opacity: 0;
  pointer-events: none;
}

.nav-item:active::after,
.action-btn:active::after {
  animation: ripple 600ms ease-out;
}

@keyframes ripple {
  from {
    opacity: 1;
    transform: scale(0.5);
  }
  to {
    opacity: 0;
    transform: scale(2.5);
  }
}
</style>