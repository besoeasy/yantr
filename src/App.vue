<script setup>
import { useRoute } from "vue-router";
import { Box, Boxes, Layers, HardDrive, ClipboardList, Send, Github, Heart, Home, Moon, Sun, Compass } from "lucide-vue-next";
import { onMounted, ref } from "vue";

const route = useRoute();
const theme = ref("light");

const isActive = (name) => route.name === name;

const setTheme = (nextTheme) => {
  theme.value = nextTheme;
  if (nextTheme === "dark") {
    document.documentElement.classList.add("dark");
  } else {
    document.documentElement.classList.remove("dark");
  }
  document.documentElement.style.colorScheme = nextTheme;
  localStorage.setItem("yantra-theme", nextTheme);
};

const toggleTheme = () => {
  setTheme(theme.value === "dark" ? "light" : "dark");
};

onMounted(() => {
  const stored = localStorage.getItem("yantra-theme");
  const prefersDark = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches;
  setTheme(stored || (prefersDark ? "dark" : "light"));
});
</script>

<template>
  <div class="min-h-screen flex flex-col md:flex-row bg-slate-50 text-slate-900 dark:bg-slate-950 dark:text-slate-100">
    <!-- Desktop Sidebar -->
    <aside
      class="hidden md:flex bg-white dark:bg-slate-950 flex-col items-center border-r border-gray-200 dark:border-slate-800 w-20 py-6 px-2 fixed h-screen z-50"
    >
      <!-- Navigation (Top) -->
      <nav class="flex flex-col items-center gap-3 mt-1">
        <!-- Home Tab -->
        <router-link
          to="/home"
          :class="
            isActive('home')
              ? 'bg-gray-900 text-white shadow-lg shadow-gray-900/20 dark:bg-slate-100 dark:text-slate-900 dark:shadow-slate-900/20'
              : 'text-gray-600 hover:bg-gray-100 hover:shadow-md hover:shadow-gray-900/10 dark:text-slate-400 dark:hover:bg-slate-900 dark:hover:shadow-slate-900/40'
          "
          class="nav-item group relative w-12 h-12 rounded-full flex items-center justify-center transition-all duration-300 ease-out smooth-shadow"
          title="Home"
        >
          <Home :size="20" class="group-hover:scale-110 transition-transform duration-300" />
          <span
            class="absolute left-full ml-3 px-3 py-1.5 bg-gray-900 text-white text-xs font-semibold rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-300 whitespace-nowrap pointer-events-none dark:bg-slate-100 dark:text-slate-900"
            >Home</span
          >
        </router-link>

        <!-- Apps Tab -->
        <router-link
          to="/apps"
          :class="
            isActive('apps')
              ? 'bg-gray-900 text-white shadow-lg shadow-gray-900/20 dark:bg-slate-100 dark:text-slate-900 dark:shadow-slate-900/20'
              : 'text-gray-600 hover:bg-gray-100 hover:shadow-md hover:shadow-gray-900/10 dark:text-slate-400 dark:hover:bg-slate-900 dark:hover:shadow-slate-900/40'
          "
          class="nav-item group relative w-12 h-12 rounded-full flex items-center justify-center transition-all duration-300 ease-out smooth-shadow"
          title="Apps"
        >
          <Box :size="20" class="group-hover:scale-110 transition-transform duration-300" />
          <span
            class="absolute left-full ml-3 px-3 py-1.5 bg-gray-900 text-white text-xs font-semibold rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-300 whitespace-nowrap pointer-events-none dark:bg-slate-100 dark:text-slate-900"
            >Apps</span
          >
        </router-link>

        <!-- Extra Tab -->
        <router-link
          to="/extra"
          :class="
            isActive('extra')
              ? 'bg-gray-900 text-white shadow-lg shadow-gray-900/20 dark:bg-slate-100 dark:text-slate-900 dark:shadow-slate-900/20'
              : 'text-gray-600 hover:bg-gray-100 hover:shadow-md hover:shadow-gray-900/10 dark:text-slate-400 dark:hover:bg-slate-900 dark:hover:shadow-slate-900/40'
          "
          class="nav-item group relative w-12 h-12 rounded-full flex items-center justify-center transition-all duration-300 ease-out smooth-shadow"
          title="Extra"
        >
          <Compass :size="20" class="group-hover:scale-110 transition-transform duration-300" />
          <span
            class="absolute left-full ml-3 px-3 py-1.5 bg-gray-900 text-white text-xs font-semibold rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-300 whitespace-nowrap pointer-events-none dark:bg-slate-100 dark:text-slate-900"
            >Extra</span
          >
        </router-link>
      </nav>

        <!-- Logo (Centered) -->
        <div class="flex-1 flex items-center justify-center">
          <router-link
            to="/home"
            class="group flex flex-col items-center justify-center text-center select-none rounded-xl px-2 py-2 transition-transform duration-300 ease-out hover:-translate-y-0.5 active:translate-y-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-900/20 dark:focus-visible:ring-slate-100/20"
            aria-label="Yantra home"
            title="Yantra"
          >
            <span
              aria-hidden="true"
              class="flex flex-col items-center text-[11px] font-black uppercase leading-[1.05] text-gray-900 dark:text-white"
            >
              <span>Y</span>
              <span>A</span>
              <span>N</span>
              <span>T</span>
              <span>R</span>
              <span>A</span>
            </span>
          </router-link>
        </div>

      <!-- Bottom Actions -->
      <div class="flex flex-col items-center gap-3 mt-4">
        <!-- Theme Toggle -->
        <button
          type="button"
          @click="toggleTheme"
          class="action-btn group relative w-12 h-12 rounded-full flex items-center justify-center text-gray-600 hover:bg-gray-100 hover:shadow-md hover:shadow-gray-900/10 dark:text-slate-300 dark:hover:bg-slate-900 dark:hover:shadow-slate-900/40 transition-all duration-300 ease-out"
          :title="`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`"
        >
          <component :is="theme === 'dark' ? Sun : Moon" :size="20" class="group-hover:scale-110 transition-transform duration-300" />
          <span
            class="absolute left-full ml-3 px-3 py-1.5 bg-gray-900 text-white text-xs font-semibold rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-300 whitespace-nowrap pointer-events-none dark:bg-slate-100 dark:text-slate-900"
          >
            {{ theme === "dark" ? "Light mode" : "Dark mode" }}
          </span>
        </button>
      </div>
    </aside>

    <!-- Mobile Bottom Navigation -->
    <nav class="md:hidden fixed bottom-0 left-0 right-0 bg-white border-t border-gray-200 dark:bg-slate-950 dark:border-slate-800 z-50 safe-area-inset-bottom">
      <div class="flex items-center justify-around px-2 py-3">
        <router-link
          to="/home"
          :class="isActive('home') ? 'bg-gray-900 text-white dark:bg-slate-100 dark:text-slate-900' : 'text-gray-600 dark:text-slate-400'"
          class="flex flex-col items-center gap-1 px-4 py-2 rounded-xl transition-all active:scale-95"
          title="Home"
        >
          <Home :size="20" />
          <span class="text-xs font-medium">Home</span>
        </router-link>

        <router-link
          to="/apps"
          :class="isActive('apps') ? 'bg-gray-900 text-white dark:bg-slate-100 dark:text-slate-900' : 'text-gray-600 dark:text-slate-400'"
          class="flex flex-col items-center gap-1 px-4 py-2 rounded-xl transition-all active:scale-95"
          title="Apps"
        >
          <Box :size="20" />
          <span class="text-xs font-medium">Apps</span>
        </router-link>

        <router-link
          to="/extra"
          :class="isActive('extra') ? 'bg-gray-900 text-white dark:bg-slate-100 dark:text-slate-900' : 'text-gray-600 dark:text-slate-400'"
          class="flex flex-col items-center gap-1 px-4 py-2 rounded-xl transition-all active:scale-95"
          title="Extra"
        >
          <Compass :size="20" />
          <span class="text-xs font-medium">Extra</span>
        </router-link>

        <button
          type="button"
          @click="toggleTheme"
          class="flex flex-col items-center gap-1 px-4 py-2 rounded-xl transition-all active:scale-95 text-gray-600 dark:text-slate-400"
          :title="`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`"
        >
          <component :is="theme === 'dark' ? Sun : Moon" :size="20" />
          <span class="text-[10px] font-medium">Theme</span>
        </button>
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
  content: "";
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
  content: "";
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
  content: "";
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
