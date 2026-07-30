<script setup>
import { useRoute, useRouter } from "vue-router";
import { 
  Box, Home, Moon, Sun, Check, Shield, 
  HardDrive, Terminal, Network, Lock, Sparkles
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

    <!-- Desktop Vertical Sidebar -->
    <aside class="hidden md:flex fixed top-0 bottom-0 left-0 w-20 flex-col items-center justify-between py-6 px-3 bg-white/90 dark:bg-[#09090b]/90 backdrop-blur-xl border-r border-slate-200/80 dark:border-zinc-800/80 z-50 transition-colors">
      
      <!-- Top: Brand Logo -->
      <div class="flex flex-col items-center">
        <router-link
          to="/home"
          class="group flex flex-col items-center justify-center select-none rounded-xl p-2 transition-transform duration-300 hover:scale-105"
          title="YANTR Dashboard"
        >
          <span class="text-xs font-black tracking-widest uppercase text-slate-900 dark:text-white leading-tight text-center">
            YAN<br/>TR
          </span>
        </router-link>
      </div>

      <!-- Center Navigation Items -->
      <nav class="flex flex-col items-center gap-3">
        <router-link
          v-for="item in navItems"
          :key="item.name"
          :to="item.path"
          :class="[
            'group relative w-12 h-12 rounded-2xl flex items-center justify-center transition-all duration-300',
            isActive(item.name)
              ? 'bg-slate-900 text-white dark:bg-white dark:text-slate-900 shadow-md shadow-slate-900/10 dark:shadow-white/10'
              : 'text-slate-500 dark:text-zinc-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-zinc-800/60'
          ]"
        >
          <component :is="item.icon" :size="20" class="transition-transform duration-300 group-hover:scale-110" />
          
          <!-- Tooltip on hover -->
          <span class="absolute left-full ml-3 px-3 py-1.5 rounded-xl bg-slate-900 dark:bg-white text-white dark:text-slate-900 text-xs font-bold uppercase tracking-wider opacity-0 group-hover:opacity-100 pointer-events-none transition-all duration-200 shadow-xl whitespace-nowrap z-50">
            {{ item.label }}
          </span>
        </router-link>
      </nav>

      <!-- Bottom Actions -->
      <div class="flex flex-col items-center gap-3.5" ref="languageMenuRef">
        
        <!-- Language Picker -->
        <div class="relative">
          <button
            type="button"
            @click="toggleLanguageMenu"
            class="group relative w-10 h-10 rounded-xl flex items-center justify-center text-slate-600 dark:text-zinc-400 hover:bg-slate-100 dark:hover:bg-zinc-800/60 transition-all duration-200"
            :title="t('nav.language')"
          >
            <span class="text-base select-none">{{ activeLanguage.flag }}</span>
            <span class="absolute left-full ml-3 px-3 py-1.5 rounded-xl bg-slate-900 dark:bg-white text-white dark:text-slate-900 text-xs font-bold uppercase tracking-wider opacity-0 group-hover:opacity-100 pointer-events-none transition-all duration-200 shadow-xl whitespace-nowrap z-50">
              {{ activeLanguage.name }}
            </span>
          </button>

          <!-- Language Dropdown Popover -->
          <transition
            enter-active-class="transition-all duration-200 ease-out"
            enter-from-class="opacity-0 scale-95 translate-x-2"
            enter-to-class="opacity-100 scale-100 translate-x-0"
            leave-active-class="transition-all duration-150 ease-in"
            leave-from-class="opacity-100 scale-100 translate-x-0"
            leave-to-class="opacity-0 scale-95 translate-x-2"
          >
            <div
              v-if="showLanguageMenu"
              class="absolute left-full bottom-0 ml-3 bg-white dark:bg-zinc-900 rounded-2xl shadow-2xl border border-slate-100 dark:border-zinc-800 py-1.5 min-w-44 z-50 overflow-hidden"
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

        <!-- Theme Toggle -->
        <button
          type="button"
          @click="toggleTheme"
          class="group relative w-10 h-10 rounded-xl flex items-center justify-center text-slate-600 dark:text-zinc-400 hover:bg-slate-100 dark:hover:bg-zinc-800/60 hover:text-slate-900 dark:hover:text-white transition-all duration-200"
          :title="theme === 'dark' ? t('nav.lightMode') : t('nav.darkMode')"
        >
          <component :is="theme === 'dark' ? Sun : Moon" :size="18" class="transition-transform duration-300 group-hover:scale-110" />
          <span class="absolute left-full ml-3 px-3 py-1.5 rounded-xl bg-slate-900 dark:bg-white text-white dark:text-slate-900 text-xs font-bold uppercase tracking-wider opacity-0 group-hover:opacity-100 pointer-events-none transition-all duration-200 shadow-xl whitespace-nowrap z-50">
            {{ theme === 'dark' ? t('nav.lightMode') : t('nav.darkMode') }}
          </span>
        </button>

        <!-- Lock Session -->
        <button
          type="button"
          @click="handleLockSession"
          class="group relative w-10 h-10 rounded-xl flex items-center justify-center text-slate-400 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/30 transition-all duration-200"
          title="Lock Session"
        >
          <Lock :size="18" />
          <span class="absolute left-full ml-3 px-3 py-1.5 rounded-xl bg-slate-900 dark:bg-white text-white dark:text-slate-900 text-xs font-bold uppercase tracking-wider opacity-0 group-hover:opacity-100 pointer-events-none transition-all duration-200 shadow-xl whitespace-nowrap z-50">
            Lock Session
          </span>
        </button>
      </div>
    </aside>

    <!-- Mobile Header (md:hidden) -->
    <header class="flex md:hidden fixed top-0 left-0 right-0 h-14 z-50 bg-white/90 dark:bg-[#09090b]/90 backdrop-blur-xl border-b border-slate-200 dark:border-zinc-800 px-4 items-center justify-between">
      <router-link to="/home" class="text-sm font-black tracking-wider uppercase text-slate-900 dark:text-white">
        YANTR
      </router-link>
      <nav class="flex items-center gap-1">
        <router-link
          v-for="item in navItems"
          :key="item.name"
          :to="item.path"
          :class="[
            'p-2 rounded-xl text-xs font-bold transition-all flex items-center justify-center',
            isActive(item.name)
              ? 'bg-slate-900 text-white dark:bg-white dark:text-slate-900 shadow-sm'
              : 'text-slate-500 dark:text-zinc-400'
          ]"
        >
          <component :is="item.icon" :size="18" />
        </router-link>
        <button
          type="button"
          @click="toggleTheme"
          class="p-2 rounded-xl text-slate-500 dark:text-zinc-400"
        >
          <component :is="theme === 'dark' ? Sun : Moon" :size="18" />
        </button>
        <button
          type="button"
          @click="handleLockSession"
          class="p-2 rounded-xl text-slate-400 hover:text-rose-500"
        >
          <Lock :size="18" />
        </button>
      </nav>
    </header>

    <!-- Main Content Area with Desktop Sidebar Offset -->
    <main class="flex-1 min-h-screen md:pl-20 pt-14 md:pt-0">
      <router-view :key="route.fullPath" />
    </main>

    </template>
  </div>
</template>

<style scoped>
/* Smooth transitions */
</style>
