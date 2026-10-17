<script setup>
import { onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { ArrowUpRight, GitBranch } from "@lucide/vue";
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
  <div class="yantra-card group relative flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl p-7 text-left">
    <div class="flex w-full items-start justify-between">
      <div class="flex items-center gap-4">
        <!-- Minimal Icon Block -->
        <div class="yantra-tile yantra-tile--accent h-12 w-12 transition-transform duration-300 group-hover:scale-105">
          <Github class="h-5 w-5" />
        </div>
        <div>
          <h3 class="yantra-text-primary text-lg font-bold leading-none tracking-tight">
            {{ t("githubCard.label") }}
          </h3>
          <p class="yantra-text-secondary mt-2 text-[10px] font-semibold uppercase tracking-widest">
            {{ t("githubCard.subtitle") }}
          </p>
        </div>
      </div>
    </div>

    <!-- Content -->
    <div class="relative z-10 mt-auto flex flex-col pt-6">
      <div class="yantra-text-primary mb-2 text-4xl font-black leading-[1.1] tracking-tighter transition-transform duration-500 group-hover:-translate-y-1 sm:text-5xl">
        {{ t("githubCard.title") }}
      </div>
      <div class="yantra-text-secondary mb-6 line-clamp-2 text-sm leading-relaxed">
        {{ t("githubCard.description") }}
      </div>

      <div class="flex flex-col gap-2.5">
        <!-- Ask for the star -->
        <a
          href="https://github.com/besoeasy/yantr"
          target="_blank"
          rel="noopener noreferrer"
          class="gh-cta group/cta relative flex h-12 w-full items-center justify-between rounded-xl px-5 transition-all duration-300 active:scale-[0.98]"
        >
          <div class="relative z-10 flex items-center gap-2.5">
            <Github class="h-4 w-4 fill-current" />
            <span class="text-[11px] font-bold uppercase tracking-widest">{{ t('githubCard.starCta') }}</span>
          </div>
          <ArrowUpRight class="relative z-10 h-4.5 w-4.5 opacity-70 transition-transform duration-300 group-hover/cta:-translate-y-1 group-hover/cta:translate-x-1 group-hover/cta:opacity-100" />
        </a>
      </div>

      <!-- Build info: a hairline separates the strip from the buttons, and the
           date sits in a mono chip, so it reads as a footer rather than as
           stray text. It also uses the theme's secondary colour rather than
           zinc-300/400, which was around 2:1 on the card at 9px. -->
      <div class="gh-build mt-4 flex items-center justify-between pt-3">
        <span class="yantra-text-secondary flex items-center gap-1.5 text-[9px] font-semibold uppercase tracking-widest">
          <GitBranch class="h-3 w-3" />
          <span>{{ buildTimeAgo }}</span>
        </span>
        <span class="gh-build__stamp yantra-text-secondary rounded-md px-2 py-0.5 font-mono text-[9px] font-semibold tracking-widest" :title="buildTimestamp">
          {{ rawBuildTimestamp ? rawBuildTimestamp.split('T')[0] : 'DEV' }}
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Card chrome, the icon tile and every text colour come from the shared
   helpers in main.css. A card owns only its own pieces, which here are the
   two calls to action and the build footer. */
.gh-cta {
  background-color: var(--cta-bg);
  color: var(--cta-text);
}

.gh-cta:hover {
  background-color: var(--cta-bg-hover);
}

.gh-cta-ghost {
  border-color: var(--card-border);
  background-color: var(--card-bg);
  color: var(--text-secondary);
}

.gh-cta-ghost:hover {
  border-color: var(--card-border-strong);
  background-color: var(--card-inset);
  color: var(--text-primary);
}

.gh-build {
  border-top: 1px solid var(--card-border);
}

.gh-build__stamp {
  background-color: var(--card-inset);
}
</style>
