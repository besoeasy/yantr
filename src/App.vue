<script setup>
import { useRoute, useRouter } from "vue-router";
import { 
  Box, Home, Moon, Sun, Check, Shield, 
  HardDrive, Terminal, Network, Lock
} from "@lucide/vue";
import NotificationBanner from './components/NotificationBanner.vue';
import AuthGate from './components/AuthGate.vue';
import { onMounted, onUnmounted, ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useYantrAuth } from './composables/useYantrAuth';

const route = useRoute();
const router = useRouter();
const { locale, t } = useI18n();
const { authState, bootstrapYantrAuth, logoutYantr } = useYantrAuth();
const theme = ref("dark");
const showLanguageMenu = ref(false);
const languageMenuRef = ref(null);

const isActive = (name) => route.name === name;

const navItems = [
  { name: "home", path: "/home", label: "Home", icon: Home },
  { name: "apps", path: "/apps", label: "Apps", icon: Box },
  { name: "logs", path: "/logs", label: "Logs", icon: Terminal },
];

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

function handleLockSession() {
  logoutYantr();
  router.push("/");
}

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
  <div class="min-h-screen flex flex-col bg-white text-slate-900 dark:bg-[#09090b] dark:text-zinc-100 font-sans antialiased">
    <NotificationBanner />
    <AuthGate v-if="authState.booting || !authState.authenticated" />
    <template v-else>

    <!-- Sleek Full-Width Minimalist Header -->
    <header class="fixed top-0 left-0 right-0 h-16 z-50 bg-white/80 dark:bg-[#09090b]/80 backdrop-blur-xl border-b border-slate-100 dark:border-zinc-800/80 transition-colors">
      <div class="h-full max-w-7xl mx-auto flex items-center justify-between px-4 sm:px-6 lg:px-8">
        
        <!-- Left: Brand Identifier -->
        <div class="flex items-center gap-6">
          <router-link 
            to="/home" 
            class="text-base font-black tracking-wider uppercase text-slate-900 dark:text-white hover:opacity-80 transition-opacity select-none"
          >
            YANTR
          </router-link>

          <!-- Divider -->
          <div class="hidden sm:block h-5 w-px bg-slate-200 dark:bg-zinc-800"></div>

          <!-- Main Nav Pills (Desktop) -->
          <nav class="hidden md:flex items-center gap-1.5">
            <router-link
              v-for="item in navItems"
              :key="item.name"
              :to="item.path"
              :class="[
                'px-3.5 py-1.5 rounded-xl text-xs font-bold uppercase tracking-wider transition-all duration-200 flex items-center gap-2',
                isActive(item.name)
                  ? 'bg-slate-900 text-white dark:bg-white dark:text-slate-900 shadow-sm'
                  : 'text-slate-500 hover:text-slate-900 dark:text-zinc-400 dark:hover:text-white hover:bg-slate-100/80 dark:hover:bg-zinc-800/60'
              ]"
            >
              <component :is="item.icon" :size="14" />
              <span>{{ item.label }}</span>
            </router-link>
          </nav>
        </div>

        <!-- Mobile Nav Pills -->
        <nav class="flex md:hidden items-center gap-1 overflow-x-auto py-1">
          <router-link
            v-for="item in navItems"
            :key="item.name"
            :to="item.path"
            :class="[
              'p-2 rounded-xl text-xs font-bold transition-all flex items-center justify-center',
              isActive(item.name)
                ? 'bg-slate-900 text-white dark:bg-white dark:text-slate-900 shadow-sm'
                : 'text-slate-500 dark:text-zinc-400 hover:bg-slate-100 dark:hover:bg-zinc-800'
            ]"
            :title="item.label"
          >
            <component :is="item.icon" :size="16" />
          </router-link>
        </nav>

        <!-- Right Controls & Actions -->
        <div class="flex items-center gap-2">
          <!-- Language Picker Dropdown -->
          <div class="relative" ref="languageMenuRef">
            <button
              type="button"
              @click="toggleLanguageMenu"
              class="flex items-center gap-2 px-3 py-1.5 rounded-xl text-xs font-semibold text-slate-600 dark:text-zinc-300 hover:bg-slate-100 dark:hover:bg-zinc-800/80 transition-all duration-200"
              :title="t('nav.language')"
            >
              <span class="text-sm select-none">{{ activeLanguage.flag }}</span>
              <span class="hidden sm:inline font-bold uppercase tracking-wider text-[11px]">{{ activeLanguage.code }}</span>
            </button>

            <!-- Popover Menu -->
            <transition
              enter-active-class="transition-all duration-200 ease-out"
              enter-from-class="opacity-0 scale-95 -translate-y-1"
              enter-to-class="opacity-100 scale-100 translate-y-0"
              leave-active-class="transition-all duration-150 ease-in"
              leave-from-class="opacity-100 scale-100 translate-y-0"
              leave-to-class="opacity-0 scale-95 -translate-y-1"
            >
              <div
                v-if="showLanguageMenu"
                class="absolute right-0 top-full mt-2 bg-white dark:bg-zinc-900 rounded-2xl shadow-xl border border-slate-100 dark:border-zinc-800 py-1.5 min-w-44 z-50 overflow-hidden"
              >
                <button
                  v-for="lang in languages"
                  :key="lang.code"
                  @click="setLocale(lang.code)"
                  class="w-full px-3.5 py-2 flex items-center gap-3 transition-colors relative"
                  :class="locale === lang.code
                    ? 'bg-slate-50 dark:bg-zinc-800/80 text-slate-900 dark:text-white font-bold'
                    : 'text-slate-600 dark:text-zinc-400 hover:bg-slate-50 dark:hover:bg-zinc-800/50 hover:text-slate-900 dark:hover:text-white font-medium'"
                >
                  <span v-if="locale === lang.code" class="absolute left-0 top-1.5 bottom-1.5 w-1 rounded-r-full bg-indigo-600" />
                  <span class="text-base leading-none select-none">{{ lang.flag }}</span>
                  <span class="flex-1 text-left text-xs font-semibold tracking-tight">{{ lang.name }}</span>
                  <Check v-if="locale === lang.code" :size="14" class="text-indigo-600 dark:text-indigo-400 shrink-0" />
                </button>
              </div>
            </transition>
          </div>

          <!-- Theme Switcher -->
          <button
            type="button"
            @click="toggleTheme"
            class="w-9 h-9 flex items-center justify-center rounded-xl text-slate-600 dark:text-zinc-400 hover:bg-slate-100 dark:hover:bg-zinc-800/80 hover:text-slate-900 dark:hover:text-white transition-all duration-200"
            :title="theme === 'dark' ? t('nav.lightMode') : t('nav.darkMode')"
          >
            <component :is="theme === 'dark' ? Sun : Moon" :size="16" class="transition-transform duration-300 hover:scale-110" />
          </button>

          <!-- Lock / Logout Button -->
          <button
            type="button"
            @click="handleLockSession"
            class="w-9 h-9 flex items-center justify-center rounded-xl text-slate-400 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/30 transition-all duration-200"
            title="Lock Yantr Session"
          >
            <Lock :size="16" />
          </button>
        </div>
      </div>
    </header>

    <!-- Main Content Container -->
    <main class="flex-1 min-h-screen pt-16">
      <router-view :key="route.fullPath" />
    </main>

    </template>
  </div>
</template>

<style scoped>
/* Smooth transition effects */
</style>
