<script setup>
import { useRoute, useRouter } from "vue-router";
import { 
  Box, Home, Moon, Sun, Check, Layers,
  HardDrive, Terminal, Lock, Globe
} from "@lucide/vue";
import NotificationBanner from './components/NotificationBanner.vue';
import AuthGate from './components/AuthGate.vue';
import { onMounted, onUnmounted, ref, computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { useYantrAuth } from './composables/useYantrAuth';
import { NAV, NAV_ITEMS, dashboard } from './utils/routes.js';

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


// A functional component so each link can carry its own `size` the way the
// lucide icons do, instead of a fixed 24px box sitting in a 40px button.
// fill defaults to currentColor so a brand glyph takes the link's own colour
// and follows the theme. Without it an SVG paints black, which is invisible on
// the dark chrome — and it is invisible only there, so it reads as a light-mode
// pass until someone switches.
const brandIcon = (path, extra) => (props) => {
  const attrs = { ...(props || {}) };
  const size = Number(attrs.size) || 18;
  delete attrs.size;
  return h("svg", {
    xmlns: "http://www.w3.org/2000/svg",
    viewBox: "0 0 24 24",
    width: size,
    height: size,
    fill: "currentColor",
    "aria-hidden": "true",
    ...attrs,
  }, [h("path", { fill: "currentColor", ...extra, d: path })]);
};

const GithubIcon = brandIcon(GITHUB_PATH, { "fill-rule": "evenodd", "clip-rule": "evenodd" });

// GitHub only, and that asymmetry is the point. GitHub is the project's source,
// its licence and its issue tracker: it belongs beside the app's own controls
// because it is part of what this thing is. Discord is a link to a different
// application, and the Home dashboard's Discord card is its discovery path —
// which is the right place for it. This cluster is where a user looks for
// app-level controls, and every off-product destination dilutes the ones next
// to it.
//
// Both names are proper nouns, identical in every locale, so they are not
// routed through i18n — unlike the nav labels, which now are. Same reasoning as
// the hardcoded "Lock Session" below.
const communityLinks = [
  { name: "GitHub", href: "https://github.com/besoeasy/yantr", icon: GithubIcon },
];

// Icons live here, not in the route table: routes.js holds what a page is and
// which nav item owns it, the shell holds what that item looks like.
const NAV_ICONS = {
  [NAV.DASHBOARD]: Home,
  [NAV.APPS]: Box,
  [NAV.STACKS]: Layers,
  [NAV.FLEET]: Globe,
  [NAV.STORAGE]: HardDrive,
  [NAV.LOGS]: Terminal,
};

// Built from the route table's nav list, so a route can no longer name an owner
// that has no item, and adding an item stops meaning "edit this file too".
// Every nav id is also the name of the list route it opens, which
// utils/routes.test.js asserts.
const navItems = NAV_ITEMS.map((item) => ({
  ...item,
  icon: NAV_ICONS[item.id],
  to: { name: item.id },
}));

// Ownership, not equality. The old `route.name === item.name` matched only the
// six list routes, so on /apps/x, /stacks/x or a container page nothing was
// highlighted and the sidebar looked lost.
const isActive = (item) => route.meta?.nav === item.id;

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
  router.push(dashboard());
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
  <div class="min-h-screen flex flex-col bg-white text-slate-900 dark:bg-(--chrome-bg) dark:text-zinc-100 font-sans antialiased">
    <NotificationBanner />
    <AuthGate v-if="authState.booting || !authState.authenticated" />
    <template v-else>

    <!-- Desktop Vertical Sidebar -->
    <aside class="hidden md:flex fixed top-0 bottom-0 left-0 w-20 flex-col items-center justify-between py-6 px-3 bg-white/90 dark:bg-(--chrome-bg)/90 backdrop-blur-xl z-50 transition-colors">
      
      <!-- Center Navigation Items (Top-aligned) -->
      <nav class="flex flex-col items-center gap-3 pt-2">
        <router-link
          v-for="item in navItems"
          :key="item.id"
          :to="item.to"
          :class="[
            'group relative w-12 h-12 rounded-2xl flex items-center justify-center transition-[border-color,box-shadow,transform,opacity] duration-200',
            isActive(item)
              ? 'bg-slate-900 text-white dark:bg-white dark:text-slate-900 shadow-md shadow-slate-900/10 dark:shadow-white/10'
              : 'text-slate-500 dark:text-zinc-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-zinc-800/60'
          ]"
        >
          <component :is="item.icon" :size="20" class="transition-transform duration-200 group-hover:scale-110" />
          
          <!-- Tooltip on hover -->
          <span class="absolute left-full ml-3 px-3 py-1.5 rounded-xl bg-slate-900 dark:bg-white text-white dark:text-slate-900 text-xs font-bold uppercase tracking-wider opacity-0 group-hover:opacity-100 pointer-events-none transition-[border-color,box-shadow,transform,opacity] duration-200 shadow-xl whitespace-nowrap z-50">
            {{ t(item.labelKey) }}
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
          class="group relative w-10 h-10 rounded-xl flex items-center justify-center text-slate-600 dark:text-zinc-400 hover:bg-slate-100 dark:hover:bg-zinc-800/60 hover:text-slate-900 dark:hover:text-white transition-[border-color,box-shadow,transform,opacity] duration-200"
          :title="link.name"
          @click="showLanguageMenu = false"
        >
          <component :is="link.icon" :size="18" class="transition-transform duration-200 group-hover:scale-110" />
          <span class="absolute left-full ml-3 px-3 py-1.5 rounded-xl bg-slate-900 dark:bg-white text-white dark:text-slate-900 text-xs font-bold uppercase tracking-wider opacity-0 group-hover:opacity-100 pointer-events-none transition-[border-color,box-shadow,transform,opacity] duration-200 shadow-xl whitespace-nowrap z-50">
            {{ link.name }}
          </span>
        </a>

        <!-- Language Picker -->
        <div class="relative">
          <button
            type="button"
            @click="toggleLanguageMenu"
            class="group relative w-10 h-10 rounded-xl flex items-center justify-center text-slate-600 dark:text-zinc-400 hover:bg-slate-100 dark:hover:bg-zinc-800/60 transition-[border-color,box-shadow,transform,opacity] duration-200"
            :title="t('nav.language')"
          >
            <span class="text-base select-none">{{ activeLanguage.flag }}</span>
            <span class="absolute left-full ml-3 px-3 py-1.5 rounded-xl bg-slate-900 dark:bg-white text-white dark:text-slate-900 text-xs font-bold uppercase tracking-wider opacity-0 group-hover:opacity-100 pointer-events-none transition-[border-color,box-shadow,transform,opacity] duration-200 shadow-xl whitespace-nowrap z-50">
              {{ activeLanguage.name }}
            </span>
          </button>

          <!-- Language Dropdown Popover -->
          <transition
            enter-active-class="transition-[border-color,box-shadow,transform,opacity] duration-200 ease-out"
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
          class="group relative w-10 h-10 rounded-xl flex items-center justify-center text-slate-600 dark:text-zinc-400 hover:bg-slate-100 dark:hover:bg-zinc-800/60 hover:text-slate-900 dark:hover:text-white transition-[border-color,box-shadow,transform,opacity] duration-200"
          :title="theme === 'dark' ? t('nav.lightMode') : t('nav.darkMode')"
        >
          <component :is="theme === 'dark' ? Sun : Moon" :size="18" class="transition-transform duration-200 group-hover:scale-110" />
          <span class="absolute left-full ml-3 px-3 py-1.5 rounded-xl bg-slate-900 dark:bg-white text-white dark:text-slate-900 text-xs font-bold uppercase tracking-wider opacity-0 group-hover:opacity-100 pointer-events-none transition-[border-color,box-shadow,transform,opacity] duration-200 shadow-xl whitespace-nowrap z-50">
            {{ theme === 'dark' ? t('nav.lightMode') : t('nav.darkMode') }}
          </span>
        </button>

        <!-- Lock Session -->
        <button
          type="button"
          @click="handleLockSession"
          class="group relative w-10 h-10 rounded-xl flex items-center justify-center text-slate-400 hover:text-rose-600 dark:hover:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-950/30 transition-[border-color,box-shadow,transform,opacity] duration-200"
          title="Lock Session"
        >
          <Lock :size="18" />
          <span class="absolute left-full ml-3 px-3 py-1.5 rounded-xl bg-slate-900 dark:bg-white text-white dark:text-slate-900 text-xs font-bold uppercase tracking-wider opacity-0 group-hover:opacity-100 pointer-events-none transition-[border-color,box-shadow,transform,opacity] duration-200 shadow-xl whitespace-nowrap z-50">
            Lock Session
          </span>
        </button>
      </div>
    </aside>

    <!-- Mobile Header (md:hidden) -->
    <header class="flex md:hidden sticky top-0 h-14 z-50 bg-white/90 dark:bg-(--chrome-bg)/90 backdrop-blur-xl border-b border-slate-200 dark:border-zinc-800 px-4 items-center justify-between">
      <router-link :to="dashboard()" class="text-sm font-black tracking-wider uppercase text-slate-900 dark:text-white">
        YANTR
      </router-link>
      <nav class="flex items-center gap-1">
        <router-link
          v-for="item in navItems"
          :key="item.id"
          :to="item.to"
          :class="[
            'p-2 rounded-xl text-xs font-bold transition-all flex items-center justify-center',
            isActive(item)
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
      <!-- Keyed on the path, not the full path. A param change must rebuild the
           view — AppDetail only fetches in onMounted — but a query-only change
           must not: Storage keeps its tab in ?tab=, so keying on fullPath tore
           the page down and refetched it on every tab click.
           out-in sees one leave, then one enter — two pages cannot be dying and
           arriving over each other, and the timing lives in motion.css. -->
      <router-view v-slot="{ Component }">
        <Transition name="route" mode="out-in">
          <component :is="Component" :key="route.path" />
        </Transition>
      </router-view>
    </main>

    </template>
  </div>
</template>

<style scoped>
/* Smooth transitions */
</style>
