<script setup>
import { useRoute, useRouter } from "vue-router";
import { 
  Box, Home, Moon, Sun, Check, Shield, 
  HardDrive, Terminal, Network, Lock, Sparkles, Globe
} from "@lucide/vue";
import NotificationBanner from './components/NotificationBanner.vue';
import AuthGate from './components/AuthGate.vue';
import { onMounted, onUnmounted, ref, computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { useYantrAuth } from './composables/useYantrAuth';

const route = useRoute();
const router = useRouter();
const { locale, t } = useI18n();
const { authState, bootstrapYantrAuth, logoutYantr } = useYantrAuth();
const theme = ref("dark");
const showLanguageMenu = ref(false);
const languageMenuRef = ref(null);

// @lucide/vue dropped brand icons, which is why the widgets inline their own
// GitHub mark. Same reason here: both sidebar links are brand marks.
const GITHUB_PATH = "M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z";

const DISCORD_PATH = "M20.317 4.3698a19.7913 19.7913 0 0 0-4.8851-1.5152.0741.0741 0 0 0-.0785.0371c-.211.3753-.4447.8648-.6083 1.2495-1.8447-.2762-3.68-.2762-5.4868 0-.1636-.3933-.4058-.8742-.6177-1.2495a.077.077 0 0 0-.0785-.037 19.7363 19.7363 0 0 0-4.8852 1.515.0699.0699 0 0 0-.0321.0277C.5334 9.0458-.319 13.5799.0992 18.0578a.0824.0824 0 0 0 .0312.0561c2.0528 1.5076 4.0413 2.4228 5.9929 3.0294a.0777.0777 0 0 0 .0842-.0276c.4616-.6304.8731-1.2952 1.226-1.9942a.076.076 0 0 0-.0416-.1057c-.6528-.2476-1.2743-.5495-1.8722-.8923a.077.077 0 0 1-.0076-.1277c.1258-.0943.2517-.1923.3718-.2914a.0743.0743 0 0 1 .0776-.0105c3.9278 1.7933 8.18 1.7933 12.0614 0a.0739.0739 0 0 1 .0785.0095c.1202.099.246.1981.3728.2924a.077.077 0 0 1-.0066.1276 12.2986 12.2986 0 0 1-1.873.8914.0766.0766 0 0 0-.0407.1067c.3604.698.7719 1.3628 1.225 1.9932a.076.076 0 0 0 .0842.0286c1.961-.6067 3.9495-1.5219 6.0023-3.0294a.077.077 0 0 0 .0313-.0552c.5004-5.177-.8382-9.6739-3.5485-13.6604a.061.061 0 0 0-.0312-.0286zM8.02 15.3312c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9555-2.4189 2.157-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.9555 2.4189-2.1569 2.4189zm7.9748 0c-1.1825 0-2.1569-1.0857-2.1569-2.419 0-1.3332.9554-2.4189 2.1569-2.4189 1.2108 0 2.1757 1.0952 2.1568 2.419 0 1.3332-.946 2.4189-2.1568 2.4189Z";

// A functional component so each link can carry its own `size` the way the
// lucide icons do, instead of a fixed 24px box sitting in a 40px button.
const brandIcon = (path, extra) => (props) => {
  const attrs = { ...(props || {}) };
  const size = Number(attrs.size) || 18;
  delete attrs.size;
  return h("svg", {
    xmlns: "http://www.w3.org/2000/svg",
    viewBox: "0 0 24 24",
    width: size,
    height: size,
    "aria-hidden": "true",
    ...attrs,
  }, [h("path", { ...extra, d: path })]);
};

const GithubIcon = brandIcon(GITHUB_PATH, { "fill-rule": "evenodd", "clip-rule": "evenodd" });
const DiscordIcon = brandIcon(DISCORD_PATH, { fill: "currentColor" });

// Both names are proper nouns, identical in every locale, so they are not
// routed through i18n — same reasoning as the hardcoded "Lock Session" below.
const communityLinks = [
  { name: "GitHub", href: "https://github.com/besoeasy/yantr", icon: GithubIcon },
  { name: "Discord", href: "https://discord.gg/YZG5NVUcAt", icon: DiscordIcon },
];

const isActive = (name) => route.name === name;

const navItems = [
  { name: "home", path: "/home", label: "Home", icon: Home },
  { name: "apps", path: "/apps", label: "Apps", icon: Box },
  { name: "telemetry", path: "/telemetry", label: "Fleet", icon: Globe },
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
    <aside class="hidden md:flex fixed top-0 bottom-0 left-0 w-20 flex-col items-center justify-between py-6 px-3 bg-white/90 dark:bg-[#09090b]/90 backdrop-blur-xl z-50 transition-colors">
      
      <!-- Center Navigation Items (Top-aligned) -->
      <nav class="flex flex-col items-center gap-3 pt-2">
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

        <!-- Community Links -->
        <a
          v-for="link in communityLinks"
          :key="link.name"
          :href="link.href"
          target="_blank"
          rel="noopener noreferrer"
          class="group relative w-10 h-10 rounded-xl flex items-center justify-center text-slate-600 dark:text-zinc-400 hover:bg-slate-100 dark:hover:bg-zinc-800/60 hover:text-slate-900 dark:hover:text-white transition-all duration-200"
          :title="link.name"
          @click="showLanguageMenu = false"
        >
          <component :is="link.icon" :size="18" class="transition-transform duration-300 group-hover:scale-110" />
          <span class="absolute left-full ml-3 px-3 py-1.5 rounded-xl bg-slate-900 dark:bg-white text-white dark:text-slate-900 text-xs font-bold uppercase tracking-wider opacity-0 group-hover:opacity-100 pointer-events-none transition-all duration-200 shadow-xl whitespace-nowrap z-50">
            {{ link.name }}
          </span>
        </a>

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
    <header class="flex md:hidden sticky top-0 h-14 z-50 bg-white/90 dark:bg-[#09090b]/90 backdrop-blur-xl border-b border-slate-200 dark:border-zinc-800 px-4 items-center justify-between">
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
    <main class="flex-1 min-h-screen md:pl-20">
      <router-view :key="route.fullPath" />
    </main>

    </template>
  </div>
</template>

<style scoped>
/* Smooth transitions */
</style>
