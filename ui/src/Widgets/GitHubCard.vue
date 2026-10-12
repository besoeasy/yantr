<script setup>
import { onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { ArrowUpRight, GitBranch, Bug } from "@lucide/vue";
import { h } from 'vue';
import { useRepoStats } from "../composables/useRepoStats";

// @lucide/vue dropped brand icons, so the GitHub mark is inlined here — the
// same approach the other widgets use for it.
const Github = (props, context) => h('svg', {
  xmlns: "http://www.w3.org/2000/svg",
  viewBox: "0 0 24 24",
  fill: "currentColor",
  "aria-hidden": "true",
  ...props,
  ...context.attrs
}, [
  h('path', { "fill-rule": "evenodd", "clip-rule": "evenodd", d: "M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" })
]);

const { t } = useI18n();

// The card shows no counters, but the stats are still fetched so the backend
// cache stays warm for anything that does want them.
const { fetchStats } = useRepoStats();
onMounted(() => { fetchStats(); });

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
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-sky-200 bg-sky-50 transition-transform duration-300 group-hover:scale-105 dark:border-sky-900/50 dark:bg-sky-900/20">
          <Github class="h-5 w-5 text-sky-600 dark:text-sky-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors group-hover:text-black dark:text-white dark:group-hover:text-white">
            {{ t("githubCard.label") }}
          </h3>
          <div class="mt-2 flex items-center gap-1.5">
            <span class="text-[10px] font-semibold uppercase tracking-widest text-zinc-500 dark:text-zinc-400">
              {{ t("githubCard.subtitle") }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Content -->
    <div class="relative z-10 mt-auto flex flex-col pt-6">
      <div class="mb-2 text-4xl font-black leading-[1.1] tracking-tighter text-zinc-900 transition-colors duration-500 group-hover:-translate-y-1 sm:text-5xl dark:text-white">
        {{ t("githubCard.title") }}
      </div>
      <div class="mb-6 text-sm leading-relaxed text-zinc-500 line-clamp-2 dark:text-zinc-400">
        {{ t("githubCard.description") }}
      </div>

      <div class="flex flex-col gap-2.5">
        <!-- Ask for the star -->
        <a
          href="https://github.com/besoeasy/yantr"
          target="_blank"
          rel="noopener noreferrer"
          class="group/cta relative flex h-12 w-full items-center justify-between rounded-xl bg-sky-600 px-5 text-white transition-all duration-300 hover:bg-sky-700 active:scale-[0.98] overflow-hidden dark:bg-sky-500 dark:hover:bg-sky-400 dark:text-zinc-900"
        >
          <div class="relative z-10 flex items-center gap-2.5">
            <Github class="h-4 w-4 fill-current" />
            <span class="text-[11px] font-bold uppercase tracking-widest">{{ t('githubCard.starCta') }}</span>
          </div>
          <ArrowUpRight class="relative z-10 h-4.5 w-4.5 opacity-70 transition-transform duration-300 group-hover/cta:-translate-y-1 group-hover/cta:translate-x-1 group-hover/cta:opacity-100" />
        </a>

        <!-- Report a problem -->
        <a
          href="https://github.com/besoeasy/yantr/issues/new"
          target="_blank"
          rel="noopener noreferrer"
          class="group/link flex h-12 w-full items-center justify-center gap-2 rounded-xl border border-zinc-200 bg-white text-zinc-600 transition-all duration-300 hover:bg-zinc-50 hover:text-zinc-900 dark:border-zinc-800 dark:bg-black dark:text-zinc-400 dark:hover:bg-zinc-900/50 dark:hover:text-white"
        >
          <Bug class="h-4 w-4" />
          <span class="text-[11px] font-bold uppercase tracking-widest">{{ t('githubCard.issueCta') }}</span>
        </a>
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