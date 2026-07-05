<script setup>
import { useI18n } from "vue-i18n";
import { ArrowUpRight, GitBranch, ExternalLink, Heart } from "@lucide/vue";
import { h } from 'vue';

const Github = (props, context) => h('svg', {
  xmlns: "http://www.w3.org/2000/svg",
  width: "24",
  height: "24",
  viewBox: "0 0 24 24",
  fill: "none",
  stroke: "currentColor",
  'stroke-width': "2",
  'stroke-linecap': "round",
  'stroke-linejoin': "round",
  ...props,
  ...context.attrs
}, [
  h('path', { d: "M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4" }),
  h('path', { d: "M9 18c-4.51 2-5-2-7-2" })
]);

const { t } = useI18n();

const links = [
  { title: t("home.externalLinks.github"), href: "https://github.com/besoeasy/Yantr", icon: Github },
  { title: t("home.externalLinks.reportIssue"), href: "https://github.com/besoeasy/yantr/issues", icon: ExternalLink },
];

const rawBuildTimestamp = import.meta.env.VITE_BUILD_TIMESTAMP;
const buildDate = rawBuildTimestamp ? new Date(rawBuildTimestamp) : null;
const buildTimestamp =
  buildDate && !Number.isNaN(buildDate.getTime())
    ? buildDate.toISOString().replace("T", " ").replace("Z", " UTC")
    : "Unknown";

function formatTimeAgo(date) {
  if (!(date instanceof Date) || Number.isNaN(date.getTime())) return "Unknown";
  const diffMs = Date.now() - date.getTime();
  const diffSeconds = Math.abs(Math.round(diffMs / 1000));
  if (diffSeconds < 60) return "just now";
  const units = [
    { seconds: 60 * 60 * 24 * 365, label: "year" },
    { seconds: 60 * 60 * 24 * 30, label: "month" },
    { seconds: 60 * 60 * 24, label: "day" },
    { seconds: 60 * 60, label: "hour" },
    { seconds: 60, label: "minute" },
  ];
  for (const unit of units) {
    if (diffSeconds >= unit.seconds) {
      const value = Math.floor(diffSeconds / unit.seconds);
      return `${value} ${unit.label}${value === 1 ? "" : "s"} ago`;
    }
  }
  return "just now";
}

const buildTimeAgo = formatTimeAgo(buildDate);
</script>

<template>
  <div class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] rounded-xl overflow-hidden transition-all duration-300 hover:-translate-y-0.5 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 border border-gray-100 dark:border-zinc-800 text-left w-full min-h-72">
    <!-- top accent line -->
    <div class="absolute top-0 left-0 w-full h-0.5 bg-pink-500 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>

    <div class="relative z-10 p-5 flex flex-col h-full gap-4">
      <!-- header -->
      <div class="flex items-center justify-between gap-3">
        <div class="flex items-center gap-3 min-w-0">
          <div class="w-9 h-9 rounded-lg bg-pink-50 dark:bg-pink-500/10 border border-pink-100 dark:border-pink-500/20 flex items-center justify-center shrink-0 transition-transform group-hover:scale-110">
            <Heart class="w-4.5 h-4.5 text-pink-600 dark:text-pink-400 group-hover:text-pink-500 transition-colors" />
          </div>
          <div class="min-w-0">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight leading-none truncate">{{ t("sponsorCard.label") }}</h3>
            <div class="flex items-center gap-1.5 mt-1.5 text-pink-600 dark:text-pink-400">
              <span class="text-[10px] font-bold uppercase tracking-wider">{{ t("home.externalLinks.resources") }}</span>
            </div>
          </div>
        </div>
        <div class="flex items-center justify-center w-8 h-8 rounded-full bg-gray-50 dark:bg-zinc-900/50 opacity-0 group-hover:opacity-100 group-hover:translate-x-1 transition-all duration-300 shrink-0">
           <Github class="w-4 h-4 text-gray-400 dark:text-zinc-500 group-hover:text-pink-500" />
        </div>
      </div>

      <!-- Content -->
      <div class="mt-auto pt-6 flex flex-col relative z-10">
        <div class="text-3xl sm:text-4xl md:text-5xl font-black tracking-tighter text-gray-900 dark:text-white transition-colors line-clamp-2 leading-[1.1] mb-2">
          {{ t("sponsorCard.title") }}
        </div>
        <div class="text-xs text-gray-500 dark:text-zinc-400 font-medium line-clamp-2 mb-6">
          {{ t("sponsorCard.description") }}
        </div>

        <div class="flex flex-col gap-2.5">
          <!-- Main CTA -->
          <a
            href="https://github.com/sponsors/besoeasy"
            target="_blank"
            rel="noopener noreferrer"
            class="group/cta relative flex h-12 w-full items-center justify-between rounded-xl bg-pink-500 dark:bg-pink-600 px-4 transition-all duration-300 hover:bg-pink-600 dark:hover:bg-pink-500 active:scale-[0.98] overflow-hidden text-white shadow-md shadow-pink-500/20"
          >
            <div class="flex items-center gap-2.5 relative z-10">
              <Heart class="h-4 w-4 fill-current" />
              <span class="text-sm font-bold tracking-wide">{{ t('sponsorCard.cta') }}</span>
            </div>
            <ArrowUpRight class="h-4.5 w-4.5 opacity-70 transition-transform duration-300 group-hover/cta:translate-x-1 group-hover/cta:-translate-y-1 group-hover/cta:opacity-100 relative z-10" />
            <div class="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent -translate-x-full group-hover/cta:translate-x-full transition-transform duration-700 ease-out"></div>
          </a>

          <!-- Minor Links grid -->
          <div class="grid grid-cols-2 gap-2.5">
            <a
              v-for="link in links"
              :key="link.title"
              :href="link.href"
              target="_blank"
              rel="noopener noreferrer"
              class="group/link flex h-10 items-center justify-center gap-2 rounded-xl bg-gray-50 dark:bg-zinc-900/70 text-gray-600 dark:text-zinc-400 hover:bg-gray-100 dark:hover:bg-zinc-800 transition-all duration-300 hover:text-gray-900 dark:hover:text-white border border-transparent dark:hover:border-zinc-700/50"
            >
              <component :is="link.icon" class="h-3.5 w-3.5" />
              <span class="text-[10px] font-bold uppercase tracking-widest">{{ link.title }}</span>
            </a>
          </div>
        </div>

        <!-- Build Info -->
        <div class="mt-6 flex items-center justify-between border-t border-gray-100 dark:border-zinc-800/60 pt-4">
          <div class="flex items-center gap-2 text-[10px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500">
            <GitBranch class="h-3.5 w-3.5" />
            <span>{{ buildTimeAgo }}</span>
          </div>
          <div class="text-[10px] font-mono font-bold tracking-wider text-gray-300 dark:text-zinc-600" :title="buildTimestamp">
            {{ rawBuildTimestamp ? rawBuildTimestamp.split('T')[0] : 'DEV' }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
