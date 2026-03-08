<script setup>
import { useRoute } from "vue-router";
import { Box, Boxes, Layers, HardDrive, ClipboardList, Github, Heart, Home, Moon, Sun, Compass, Check, ChevronDown } from "lucide-vue-next";
import NotificationBanner from './components/NotificationBanner.vue';
import { onMounted, onUnmounted, ref, computed } from "vue";
import { useI18n } from "vue-i18n";

const route = useRoute();
const { locale, t } = useI18n();
const theme = ref("light");
const showLanguageMenu = ref(false);
const languageMenuRef = ref(null);
const mobileLanguageMenuRef = ref(null);

const isActive = (name) => route.name === name;

const setTheme = (nextTheme) => {
  theme.value = nextTheme;
  if (nextTheme === "dark") {
    document.documentElement.classList.add("dark");
  } else {
    document.documentElement.classList.remove("dark");
  }
  document.documentElement.style.colorScheme = nextTheme;
  localStorage.setItem("yantr-theme", nextTheme);
};

const toggleTheme = () => {
  setTheme(theme.value === "dark" ? "light" : "dark");
};

const toggleLanguageMenu = () => {
  showLanguageMenu.value = !showLanguageMenu.value;
};

const setLocale = (newLocale) => {
  locale.value = newLocale;
  localStorage.setItem("yantr-locale", newLocale);
  showLanguageMenu.value = false;
};

const languages = [
  { code: 'en', name: 'English', flag: '🇺🇸' },
  { code: 'zh', name: '中文', flag: '🇨🇳' },
  { code: 'es', name: 'Español', flag: '🇪🇸' },
  { code: 'de', name: 'Deutsch', flag: '🇩🇪' },
  { code: 'fr', name: 'Français', flag: '🇫🇷' },
  { code: 'ja', name: '日本語', flag: '🇯🇵' },
  { code: 'ru', name: 'Русский', flag: '🇷🇺' }
];

const activeLanguage = computed(() => languages.find(l => l.code === locale.value) || languages[0]);

const handleOutsideClick = (e) => {
  const inDesktop = languageMenuRef.value && languageMenuRef.value.contains(e.target);
  const inMobile = mobileLanguageMenuRef.value && mobileLanguageMenuRef.value.contains(e.target);
  if (!inDesktop && !inMobile) {
    showLanguageMenu.value = false;
  }
};

onMounted(() => {
  const stored = localStorage.getItem("yantr-theme");
  const prefersDark = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches;
  setTheme(stored || (prefersDark ? "dark" : "light"));

  // Auto-detect browser language on first visit
  if (!localStorage.getItem('yantr-locale')) {
    const browserLang = navigator.language?.slice(0, 2);
    const match = languages.find(l => l.code === browserLang);
    if (match) setLocale(match.code);
  }

  document.addEventListener('click', handleOutsideClick);
});

onUnmounted(() => {
  document.removeEventListener('click', handleOutsideClick);
});
</script>

<template>
  <div class="min-h-screen flex flex-col bg-[#FAFAFA] text-black dark:bg-[#0A0A0A] dark:text-white">
    <NotificationBanner />
    <!-- Desktop Sidebar -->
    <aside
      class="hidden md:flex bg-white dark:bg-[#0A0A0A] flex-col items-center border-r border-gray-200 dark:border-zinc-800 w-20 py-6 px-2 fixed h-screen z-50"
      style="--sidebar-size: 5rem"
    >
      <!-- Navigation (Top) -->
      <nav class="flex flex-col items-center gap-3 mt-1">
        <!-- Home Tab -->
        <router-link
          to="/home"
          :class="
            isActive('home')
              ? 'bg-black text-white shadow-lg shadow-black/10 dark:bg-white dark:text-black dark:shadow-black/20'
              : 'text-gray-600 hover:bg-gray-100 hover:shadow-md hover:shadow-gray-900/10 dark:text-zinc-400 dark:hover:bg-zinc-800/50 dark:hover:shadow-black/40'
          "
          class="nav-item group relative w-12 h-12 rounded-full flex items-center justify-center transition-all duration-300 ease-out smooth-shadow"
          :title="t('nav.home')"
        >
          <Home :size="20" class="group-hover:scale-110 transition-transform duration-300" />
          <span
            class="absolute left-full ml-3 px-3 py-1.5 bg-black text-white text-xs font-semibold rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-300 whitespace-nowrap pointer-events-none dark:bg-white dark:text-black"
            >{{ t('nav.home') }}</span
          >
        </router-link>

        <!-- Apps Tab -->
        <router-link
          to="/apps"
          :class="
            isActive('apps')
              ? 'bg-black text-white shadow-lg shadow-black/10 dark:bg-white dark:text-black dark:shadow-black/20'
              : 'text-gray-600 hover:bg-gray-100 hover:shadow-md hover:shadow-gray-900/10 dark:text-zinc-400 dark:hover:bg-zinc-800/50 dark:hover:shadow-black/40'
          "
          class="nav-item group relative w-12 h-12 rounded-full flex items-center justify-center transition-all duration-300 ease-out smooth-shadow"
          :title="t('nav.apps')"
        >
          <Box :size="20" class="group-hover:scale-110 transition-transform duration-300" />
          <span
            class="absolute left-full ml-3 px-3 py-1.5 bg-black text-white text-xs font-semibold rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-300 whitespace-nowrap pointer-events-none dark:bg-white dark:text-black"
            >{{ t('nav.apps') }}</span
          >
        </router-link>

        
      </nav>

        <!-- Logo (Centered) -->
        <div class="flex-1 flex items-center justify-center">
          <a
            href="https://yantr.org"
            target="_blank"
            rel="noopener noreferrer"
            class="group flex flex-col items-center justify-center text-center select-none rounded-xl px-2 py-2 transition-transform duration-300 ease-out hover:-translate-y-0.5 active:translate-y-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-900/20 dark:focus-visible:ring-white/20"
            aria-label="Open Yantr on GitHub"
            title="Open on GitHub"
          >
            <span
              aria-hidden="true"
              class="flex flex-col items-center gap-[calc(var(--sidebar-size)*0.02)] text-[calc(var(--sidebar-size)*0.22)] font-black uppercase leading-[0.95] text-black dark:text-white"
            >
              <span
                class="block will-change-transform transition-transform duration-300 ease-out group-hover:-translate-y-[calc(var(--sidebar-size)*0.03)] group-hover:scale-110 group-hover:-rotate-6 group-focus-visible:-translate-y-[calc(var(--sidebar-size)*0.03)] group-focus-visible:scale-110 group-focus-visible:-rotate-6"
                style="transition-delay: 0ms"
                >Y</span
              >
              <span
                class="block will-change-transform transition-transform duration-300 ease-out group-hover:-translate-y-[calc(var(--sidebar-size)*0.03)] group-hover:scale-110 group-hover:rotate-6 group-focus-visible:-translate-y-[calc(var(--sidebar-size)*0.03)] group-focus-visible:scale-110 group-focus-visible:rotate-6"
                style="transition-delay: 35ms"
                >A</span
              >
              <span
                class="block will-change-transform transition-transform duration-300 ease-out group-hover:-translate-y-[calc(var(--sidebar-size)*0.03)] group-hover:scale-110 group-hover:-rotate-6 group-focus-visible:-translate-y-[calc(var(--sidebar-size)*0.03)] group-focus-visible:scale-110 group-focus-visible:-rotate-6"
                style="transition-delay: 70ms"
                >N</span
              >
              <span
                class="block will-change-transform transition-transform duration-300 ease-out group-hover:-translate-y-[calc(var(--sidebar-size)*0.03)] group-hover:scale-110 group-hover:rotate-6 group-focus-visible:-translate-y-[calc(var(--sidebar-size)*0.03)] group-focus-visible:scale-110 group-focus-visible:rotate-6"
                style="transition-delay: 105ms"
                >T</span
              >
              <span
                class="block will-change-transform transition-transform duration-300 ease-out group-hover:-translate-y-[calc(var(--sidebar-size)*0.03)] group-hover:scale-110 group-hover:-rotate-6 group-focus-visible:-translate-y-[calc(var(--sidebar-size)*0.03)] group-focus-visible:scale-110 group-focus-visible:-rotate-6"
                style="transition-delay: 140ms"
                >R</span
              >
            </span>
          </a>
        </div>

      <!-- Bottom Actions -->
      <div class="flex flex-col items-center gap-3 mt-4">
        <!-- Language Toggle -->
        <div class="relative" ref="languageMenuRef">
          <button
            type="button"
            @click="toggleLanguageMenu"
            class="action-btn group relative w-12 h-12 rounded-full flex items-center justify-center text-gray-600 hover:bg-gray-100 hover:shadow-md hover:shadow-gray-900/10 dark:text-zinc-300 dark:hover:bg-zinc-800/50 dark:hover:shadow-black/40 transition-all duration-300 ease-out"
            :title="t('nav.language')"
          >
            <span class="text-base leading-none select-none">{{ activeLanguage.flag }}</span>
            <span
              class="absolute left-full ml-3 px-3 py-1.5 bg-black text-white text-xs font-semibold rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-300 whitespace-nowrap pointer-events-none dark:bg-white dark:text-black"
            >
              {{ activeLanguage.name }}
            </span>
          </button>

          <!-- Language Dropdown Menu -->
          <transition
            enter-active-class="transition-all duration-200 ease-out"
            enter-from-class="opacity-0 translate-x-2"
            enter-to-class="opacity-100 translate-x-0"
            leave-active-class="transition-all duration-150 ease-in"
            leave-from-class="opacity-100 translate-x-0"
            leave-to-class="opacity-0 translate-x-2"
          >
            <div
              v-if="showLanguageMenu"
              class="absolute left-full ml-3 bottom-0 bg-white dark:bg-zinc-900 rounded-xl shadow-2xl border border-gray-200 dark:border-zinc-700 py-1.5 min-w-[180px] z-50 overflow-hidden"
            >
              <button
                v-for="lang in languages"
                :key="lang.code"
                @click="setLocale(lang.code)"
                class="w-full px-3 py-2 flex items-center gap-3 transition-colors relative"
                :class="locale === lang.code
                  ? 'bg-gray-50 dark:bg-zinc-800 text-black dark:text-white'
                  : 'text-gray-600 dark:text-zinc-400 hover:bg-gray-50 dark:hover:bg-zinc-800/60 hover:text-black dark:hover:text-white'"
              >
                <span
                  v-if="locale === lang.code"
                  class="absolute left-0 top-1 bottom-1 w-0.5 rounded-full bg-blue-500"
                />
                <span class="text-lg leading-none select-none">{{ lang.flag }}</span>
                <span class="flex-1 text-left text-sm font-medium tracking-tight">{{ lang.name }}</span>
                <Check v-if="locale === lang.code" :size="12" class="text-blue-500 shrink-0" />
              </button>
            </div>
          </transition>
        </div>
        <!-- Theme Toggle -->
        <button
          type="button"
          @click="toggleTheme"
          class="action-btn group relative w-12 h-12 rounded-full flex items-center justify-center text-gray-600 hover:bg-gray-100 hover:shadow-md hover:shadow-gray-900/10 dark:text-zinc-300 dark:hover:bg-zinc-800/50 dark:hover:shadow-black/40 transition-all duration-300 ease-out"
          :title="theme === 'dark' ? t('nav.lightMode') : t('nav.darkMode')"
        >
          <component :is="theme === 'dark' ? Sun : Moon" :size="20" class="group-hover:scale-110 transition-transform duration-300" />
          <span
            class="absolute left-full ml-3 px-3 py-1.5 bg-black text-white text-xs font-semibold rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-300 whitespace-nowrap pointer-events-none dark:bg-white dark:text-black"
          >
            {{ theme === "dark" ? t('nav.lightMode') : t('nav.darkMode') }}
          </span>
        </button>
      </div>
    </aside>

    <!-- Mobile Bottom Navigation -->
    <nav class="md:hidden fixed bottom-0 left-0 right-0 bg-white border-t border-gray-200 dark:bg-[#0A0A0A] dark:border-zinc-800 z-50" style="padding-bottom: env(safe-area-inset-bottom, 0px)">
      <div class="flex items-center justify-around px-1 py-2">
        <router-link
          to="/home"
          :class="isActive('home') ? 'bg-black text-white shadow-lg shadow-black/10 dark:bg-white dark:text-black dark:shadow-white/10' : 'text-gray-600 dark:text-zinc-400'"
          class="flex flex-col items-center gap-1 min-w-[64px] min-h-[52px] px-3 py-2 rounded-xl transition-all active:scale-95 justify-center"
          :title="t('nav.home')"
        >
          <Home :size="22" />
          <span class="text-xs font-medium">{{ t('nav.home') }}</span>
        </router-link>

        <router-link
          to="/apps"
          :class="isActive('apps') ? 'bg-black text-white shadow-lg shadow-black/10 dark:bg-white dark:text-black dark:shadow-white/10' : 'text-gray-600 dark:text-zinc-400'"
          class="flex flex-col items-center gap-1 min-w-[64px] min-h-[52px] px-3 py-2 rounded-xl transition-all active:scale-95 justify-center"
          :title="t('nav.apps')"
        >
          <Box :size="22" />
          <span class="text-xs font-medium">{{ t('nav.apps') }}</span>
        </router-link>

        <div class="relative" ref="mobileLanguageMenuRef">
          <button
            type="button"
            @click="toggleLanguageMenu"
            class="flex flex-col items-center gap-1 min-w-[64px] min-h-[52px] px-3 py-2 rounded-xl transition-all active:scale-95 text-gray-600 dark:text-zinc-400 justify-center"
            :title="activeLanguage.name"
          >
            <span class="text-xl leading-none select-none">{{ activeLanguage.flag }}</span>
            <span class="text-xs font-medium">{{ activeLanguage.name }}</span>
          </button>
          
          <!-- Mobile Language Dropdown Menu -->
          <transition
            enter-active-class="transition-all duration-200 ease-out"
            enter-from-class="opacity-0 translate-y-2"
            enter-to-class="opacity-100 translate-y-0"
            leave-active-class="transition-all duration-150 ease-in"
            leave-from-class="opacity-100 translate-y-0"
            leave-to-class="opacity-0 translate-y-2"
          >
            <div
              v-if="showLanguageMenu"
              class="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 bg-white dark:bg-zinc-900 rounded-xl shadow-2xl border border-gray-200 dark:border-zinc-700 py-1.5 min-w-[180px] z-50 overflow-hidden"
            >
              <button
                v-for="lang in languages"
                :key="lang.code"
                @click="setLocale(lang.code)"
                class="w-full px-3 py-2 flex items-center gap-3 transition-colors relative"
                :class="locale === lang.code
                  ? 'bg-gray-50 dark:bg-zinc-800 text-black dark:text-white'
                  : 'text-gray-600 dark:text-zinc-400 hover:bg-gray-50 dark:hover:bg-zinc-800/60 hover:text-black dark:hover:text-white'"
              >
                <span
                  v-if="locale === lang.code"
                  class="absolute left-0 top-1 bottom-1 w-0.5 rounded-full bg-blue-500"
                />
                <span class="text-lg leading-none select-none">{{ lang.flag }}</span>
                <span class="flex-1 text-left text-sm font-medium tracking-tight">{{ lang.name }}</span>
                <Check v-if="locale === lang.code" :size="12" class="text-blue-500 shrink-0" />
              </button>
            </div>
          </transition>
        </div>

        <button
          type="button"
          @click="toggleTheme"
          class="flex flex-col items-center gap-1 min-w-[64px] min-h-[52px] px-3 py-2 rounded-xl transition-all active:scale-95 text-gray-600 dark:text-zinc-400 justify-center"
          :title="theme === 'dark' ? t('nav.lightMode') : t('nav.darkMode')"
        >
          <component :is="theme === 'dark' ? Sun : Moon" :size="22" />
          <span class="text-xs font-medium">{{ t('nav.theme') }}</span>
        </button>
      </div>
    </nav>

    <!-- Main Content -->
    <main class="flex-1 min-h-screen md:ml-20 pb-24 md:pb-0">
      <router-view :key="route.fullPath" />
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
