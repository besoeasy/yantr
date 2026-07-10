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
  { title: t("home.externalLinks.reportIssue"), href: "https://github.com/besoeasy/yantr/issues", icon: ExternalLink },
  { title: t("sponsorCard.label"), href: "https://github.com/sponsors/besoeasy", icon: Heart },
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
  <div class="group relative flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-all duration-300 hover:-translate-y-1 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <div class="flex w-full items-start justify-between">
      <div class="flex items-center gap-4">
        <!-- Minimal Icon Block -->
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 transition-transform duration-300 group-hover:scale-105 dark:border-zinc-800 dark:bg-zinc-900">
          <Github class="h-5 w-5 text-zinc-900 dark:text-zinc-100" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors group-hover:text-black dark:text-white dark:group-hover:text-white">
            {{ t("openSourceCard.label") }}
          </h3>
          <div class="mt-2 flex items-center gap-1.5">
            <span class="text-[10px] font-semibold uppercase tracking-widest text-zinc-500 dark:text-zinc-400">
              {{ t("home.externalLinks.resources") }}
            </span>
          </div>
        </div>
      </div>
      
      <div class="flex h-8 w-8 shrink-0 -translate-y-1 items-center justify-center rounded-full opacity-0 transition-all duration-300 group-hover:translate-y-0 group-hover:opacity-100">
         <Heart class="h-5 w-5 text-zinc-400 transition-colors hover:text-red-500 dark:text-zinc-600 dark:hover:text-red-400" />
      </div>
    </div>

    <!-- Content -->
    <div class="relative z-10 mt-auto flex flex-col pt-6">
      <div class="mb-2 text-4xl font-black leading-[1.1] tracking-tighter text-zinc-900 transition-colors duration-500 group-hover:-translate-y-1 sm:text-5xl dark:text-white">
        {{ t("openSourceCard.title") }}
      </div>
      <div class="mb-6 text-sm leading-relaxed text-zinc-500 line-clamp-2 dark:text-zinc-400">
        {{ t("openSourceCard.description") }}
      </div>

      <div class="flex flex-col gap-2.5">
        <!-- Main CTA -->
        <a
          href="https://github.com/besoeasy/Yantr"
          target="_blank"
          rel="noopener noreferrer"
          class="group/cta relative flex h-12 w-full items-center justify-between rounded-xl bg-zinc-900 px-5 text-white transition-all duration-300 hover:bg-zinc-800 active:scale-[0.98] overflow-hidden dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-100"
        >
          <div class="relative z-10 flex items-center gap-2.5">
            <Github class="h-4 w-4 fill-current" />
            <span class="text-[11px] font-bold uppercase tracking-widest">{{ t('openSourceCard.cta') }}</span>
          </div>
          <ArrowUpRight class="relative z-10 h-4.5 w-4.5 opacity-70 transition-transform duration-300 group-hover/cta:-translate-y-1 group-hover/cta:translate-x-1 group-hover/cta:opacity-100" />
        </a>

        <!-- Minor Links grid -->
        <div class="grid grid-cols-2 gap-2.5">
          <a
            v-for="link in links"
            :key="link.title"
            :href="link.href"
            target="_blank"
            rel="noopener noreferrer"
            class="group/link flex h-10 items-center justify-center gap-2 rounded-xl border border-zinc-200 bg-white text-zinc-600 transition-all duration-300 hover:bg-zinc-50 hover:text-zinc-900 dark:border-zinc-800 dark:bg-black dark:text-zinc-400 dark:hover:bg-zinc-900/50 dark:hover:text-white"
          >
            <component :is="link.icon" class="h-3.5 w-3.5" />
            <span class="text-[9px] font-semibold uppercase tracking-widest">{{ link.title }}</span>
          </a>
        </div>
      </div>

      <!-- Build Info -->
      <div class="mt-4 flex items-center justify-between pt-2">
        <div class="flex items-center gap-1.5 text-[9px] font-semibold uppercase tracking-widest text-zinc-400 dark:text-zinc-600">
          <GitBranch class="h-3 w-3" />
          <span>{{ buildTimeAgo }}</span>
        </div>
        <div class="text-[9px] font-mono font-semibold tracking-widest text-zinc-300 dark:text-zinc-700" :title="buildTimestamp">
          {{ rawBuildTimestamp ? rawBuildTimestamp.split('T')[0] : 'DEV' }}
        </div>
      </div>
    </div>
  </div>
</template>
