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
  <div class="relative group h-full flex flex-col rounded-[24px] overflow-hidden transition-all duration-500 hover:-translate-y-2 hover:shadow-[0_20px_40px_-15px_rgba(236,72,153,0.4)] dark:hover:shadow-[0_20px_40px_-15px_rgba(236,72,153,0.3)] text-left w-full min-h-[320px] border border-pink-100/50 dark:border-white/5 bg-gradient-to-br from-pink-50/50 to-rose-50/50 dark:from-[#1a0b12] dark:to-[#0f090b]">
    
    <!-- Animated Gradient Glow -->
    <div class="absolute -top-[50%] -left-[50%] w-[200%] h-[200%] bg-[radial-gradient(circle_at_50%_50%,_rgba(236,72,153,0.15)_0%,_transparent_50%)] dark:bg-[radial-gradient(circle_at_50%_50%,_rgba(236,72,153,0.12)_0%,_transparent_40%)] opacity-0 group-hover:opacity-100 transition-opacity duration-1000 pointer-events-none group-hover:animate-[spin_20s_linear_infinite]"></div>

    <!-- Massive Watermark Icon -->
    <Heart class="absolute -bottom-8 -right-8 w-64 h-64 text-pink-500/10 dark:text-pink-400/[0.03] rotate-[-20deg] group-hover:rotate-0 group-hover:scale-110 transition-all duration-700 ease-out pointer-events-none" />

    <div class="relative z-10 p-7 flex flex-col h-full gap-4 w-full">
      <!-- header -->
      <div class="flex items-start justify-between w-full">
        <div class="flex items-center gap-4">
          <!-- Gorgeous Icon Block -->
          <div class="relative w-14 h-14 rounded-2xl flex items-center justify-center shrink-0 bg-gradient-to-br from-pink-400 to-rose-600 shadow-xl shadow-pink-500/30 group-hover:shadow-pink-500/50 transition-all duration-500 group-hover:scale-105">
            <div class="absolute inset-0 bg-white/30 dark:bg-white/20 rounded-2xl rounded-bl-none opacity-50 dark:opacity-40 mix-blend-overlay"></div>
            <Heart class="w-7 h-7 text-white drop-shadow-md z-10" />
          </div>
          <div>
            <h3 class="text-xl font-black text-pink-950 dark:text-white tracking-tight leading-none group-hover:text-pink-600 dark:group-hover:text-pink-400 transition-colors">{{ t("sponsorCard.label") }}</h3>
            <div class="flex items-center gap-1.5 mt-2">
              <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-pink-600/70 dark:text-pink-500">{{ t("home.externalLinks.resources") }}</span>
            </div>
          </div>
        </div>
        
        <div class="flex items-center justify-center w-8 h-8 rounded-full opacity-0 group-hover:opacity-100 group-hover:translate-x-1 transition-all duration-300 shrink-0">
           <Github class="w-5 h-5 text-pink-400 dark:text-zinc-500 group-hover:text-pink-500" />
        </div>
      </div>

      <!-- Content -->
      <div class="mt-auto pt-6 flex flex-col relative z-10">
        <div class="text-4xl font-black tracking-tighter text-pink-950 dark:text-white transition-colors line-clamp-2 leading-[1.1] mb-2 drop-shadow-sm group-hover:-translate-y-1 duration-500">
          {{ t("sponsorCard.title") }}
        </div>
        <div class="text-xs font-medium text-pink-900/60 dark:text-zinc-400 line-clamp-2 mb-6">
          {{ t("sponsorCard.description") }}
        </div>

        <div class="flex flex-col gap-2.5">
          <!-- Main CTA -->
          <a
            href="https://github.com/sponsors/besoeasy"
            target="_blank"
            rel="noopener noreferrer"
            class="group/cta relative flex h-12 w-full items-center justify-between rounded-[16px] bg-gradient-to-r from-pink-500 to-rose-600 dark:from-pink-600 dark:to-rose-700 px-5 transition-all duration-300 hover:from-pink-400 hover:to-rose-500 active:scale-[0.98] overflow-hidden text-white shadow-md shadow-pink-500/30 hover:shadow-pink-500/50"
          >
            <div class="flex items-center gap-2.5 relative z-10">
              <Heart class="h-4 w-4 fill-white drop-shadow-sm" />
              <span class="text-[11px] font-bold tracking-widest uppercase drop-shadow-sm">{{ t('sponsorCard.cta') }}</span>
            </div>
            <ArrowUpRight class="h-4.5 w-4.5 opacity-70 transition-transform duration-300 group-hover/cta:translate-x-1 group-hover/cta:-translate-y-1 group-hover/cta:opacity-100 relative z-10" />
            <div class="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent -translate-x-full group-hover/cta:translate-x-full transition-transform duration-700 ease-out mix-blend-overlay"></div>
          </a>

          <!-- Minor Links grid -->
          <div class="grid grid-cols-2 gap-2.5">
            <a
              v-for="link in links"
              :key="link.title"
              :href="link.href"
              target="_blank"
              rel="noopener noreferrer"
              class="group/link flex h-10 items-center justify-center gap-2 rounded-[16px] text-pink-700/60 dark:text-zinc-400 bg-white/40 dark:bg-white/5 backdrop-blur-xl border border-white dark:border-white/5 hover:bg-white/80 dark:hover:bg-zinc-800 transition-all duration-300 hover:text-pink-900 dark:hover:text-white shadow-[0_4px_12px_rgba(0,0,0,0.02)] dark:shadow-none"
            >
              <component :is="link.icon" class="h-3.5 w-3.5" />
              <span class="text-[9px] font-bold uppercase tracking-widest">{{ link.title }}</span>
            </a>
          </div>
        </div>

        <!-- Build Info -->
        <div class="mt-4 flex items-center justify-between pt-2">
          <div class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest text-pink-700/50 dark:text-zinc-500">
            <GitBranch class="h-3 w-3" />
            <span>{{ buildTimeAgo }}</span>
          </div>
          <div class="text-[9px] font-mono font-bold tracking-widest text-pink-700/30 dark:text-zinc-600" :title="buildTimestamp">
            {{ rawBuildTimestamp ? rawBuildTimestamp.split('T')[0] : 'DEV' }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
