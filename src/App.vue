<script setup>
import { useRoute } from "vue-router";
import { Box, Home, Moon, Sun, Check } from "@lucide/vue";
import NotificationBanner from './components/NotificationBanner.vue';
import AuthGate from './components/AuthGate.vue';
import { onMounted, onUnmounted, ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useYantrAuth } from './composables/useYantrAuth';

const route = useRoute();
const { locale, t } = useI18n();
const { authState, bootstrapYantrAuth } = useYantrAuth();
const theme = ref("dark");
const showLanguageMenu = ref(false);
const languageMenuRef = ref(null);

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
  if (languageMenuRef.value && !languageMenuRef.value.contains(e.target)) {
    showLanguageMenu.value = false;
  }
};

onMounted(() => {
  const stored = localStorage.getItem("yantr-theme");
  const prefersLight = window.matchMedia && window.matchMedia("(prefers-color-scheme: light)").matches;
  setTheme(stored || (prefersLight ? "light" : "dark"));
  bootstrapYantrAuth();

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
    <AuthGate v-if="authState.booting || !authState.authenticated" />
    <template v-else>

    <!-- Top Bar -->
    <header class="fixed top-0 left-0 right-0 h-14 z-50 bg-white/90 dark:bg-[#0A0A0A]/90 backdrop-blur-md border-b border-gray-200 dark:border-zinc-800">
      <div class="h-full flex items-center px-4 gap-2">


        <!-- Nav Links -->
        <nav class="flex items-center gap-1">
          <router-link
            to="/home"
            :class="isActive('home')
              ? 'bg-black text-white dark:bg-white dark:text-black shadow-sm'
              : 'text-gray-600 dark:text-zinc-400 hover:bg-gray-100 dark:hover:bg-zinc-800/60 hover:text-black dark:hover:text-white'"
            class="nav-item flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm font-medium transition-all duration-200"
          >
            <Home :size="15" />
            <span>{{ t('nav.home') }}</span>
          </router-link>

          <router-link
            to="/apps"
            :class="isActive('apps')
              ? 'bg-black text-white dark:bg-white dark:text-black shadow-sm'
              : 'text-gray-600 dark:text-zinc-400 hover:bg-gray-100 dark:hover:bg-zinc-800/60 hover:text-black dark:hover:text-white'"
            class="nav-item flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm font-medium transition-all duration-200"
          >
            <Box :size="15" />
            <span>{{ t('nav.apps') }}</span>
          </router-link>
        </nav>

        <!-- Spacer -->
        <div class="flex-1"></div>

        <!-- Right Actions -->
        <div class="flex items-center gap-1">
          <!-- Language Picker -->
          <div class="relative" ref="languageMenuRef">
            <button
              type="button"
              @click="toggleLanguageMenu"
              class="action-btn flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-sm text-gray-600 dark:text-zinc-400 hover:bg-gray-100 dark:hover:bg-zinc-800/60 hover:text-black dark:hover:text-white transition-all duration-200"
              :title="t('nav.language')"
            >
              <span class="text-base leading-none select-none">{{ activeLanguage.flag }}</span>
              <span class="text-xs font-medium hidden sm:inline">{{ activeLanguage.name }}</span>
            </button>

            <!-- Language Dropdown -->
            <transition
              enter-active-class="transition-all duration-200 ease-out"
              enter-from-class="opacity-0 -translate-y-2"
              enter-to-class="opacity-100 translate-y-0"
              leave-active-class="transition-all duration-150 ease-in"
              leave-from-class="opacity-100 translate-y-0"
              leave-to-class="opacity-0 -translate-y-2"
            >
              <div
                v-if="showLanguageMenu"
                class="absolute right-0 top-full mt-2 bg-white dark:bg-zinc-900 rounded-xl shadow-2xl border border-gray-200 dark:border-zinc-700 py-1.5 min-w-44 z-50 overflow-hidden"
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
                  <span v-if="locale === lang.code" class="absolute left-0 top-1 bottom-1 w-0.5 rounded-full bg-blue-500" />
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
            class="action-btn w-9 h-9 flex items-center justify-center rounded-lg text-gray-600 dark:text-zinc-400 hover:bg-gray-100 dark:hover:bg-zinc-800/60 hover:text-black dark:hover:text-white transition-all duration-200"
            :title="theme === 'dark' ? t('nav.lightMode') : t('nav.darkMode')"
          >
            <component :is="theme === 'dark' ? Sun : Moon" :size="17" class="transition-transform duration-300 group-hover:scale-110" />
          </button>
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <main class="flex-1 min-h-screen pt-14">
      <router-view :key="route.fullPath" />
    </main>

    </template>
  </div>
</template>

<style scoped>
/* Navigation item active/hover */
.nav-item {
  position: relative;
  overflow: hidden;
}

.nav-item::after {
  content: "";
  position: absolute;
  inset: 0;
  border-radius: 0.5rem;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.35) 0%, transparent 70%);
  opacity: 0;
  pointer-events: none;
}

.nav-item:active::after {
  animation: ripple 500ms ease-out;
}

/* Action button */
.action-btn {
  position: relative;
  overflow: hidden;
}

.action-btn::after {
  content: "";
  position: absolute;
  inset: 0;
  border-radius: 0.5rem;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.35) 0%, transparent 70%);
  opacity: 0;
  pointer-events: none;
}

.action-btn:active::after {
  animation: ripple 500ms ease-out;
}

@keyframes ripple {
  from { opacity: 1; transform: scale(0.5); }
  to   { opacity: 0; transform: scale(2.5); }
}
</style>
