<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { MessageCircle, Zap, GitPullRequest, ShieldCheck, ArrowUpRight, Github, Bug, GitBranch, ExternalLink } from "lucide-vue-next";

const { t } = useI18n();

const benefits = computed(() => [
  { icon: MessageCircle, title: t("sponsorCard.benefits.devAccess.title") },
  { icon: Zap, title: t("sponsorCard.benefits.roadmap.title") },
  { icon: GitPullRequest, title: t("sponsorCard.benefits.earlyBuilds.title") },
  { icon: ShieldCheck, title: t("sponsorCard.benefits.badge.title") },
]);

const links = computed(() => [
  { title: t("home.externalLinks.github"), href: "https://github.com/besoeasy/Yantr", icon: Github },
  { title: t("home.externalLinks.reportIssue"), href: "https://github.com/besoeasy/yantr/issues", icon: Bug },
]);

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
  <div class="group flex h-full flex-col rounded-[1.4rem] bg-white p-5 text-(--text-primary) smooth-shadow transition-all duration-300 hover:-translate-y-0.5 hover:smooth-shadow-lg dark:bg-[#0A0A0A] sm:p-6">
    <div class="flex items-start justify-between gap-4">
      <div class="min-w-0">
        <div class="flex flex-wrap items-center gap-2 text-[10px] font-bold uppercase tracking-[0.18em] text-(--text-secondary)">
          <span>{{ t("sponsorCard.label") }}</span>
          <span class="opacity-35">/</span>
          <span>{{ t("home.externalLinks.resources") }}</span>
        </div>
        <h3 class="mt-3 max-w-[16rem] text-xl font-semibold leading-tight tracking-tight text-(--text-primary) sm:text-[1.35rem]">
          {{ t("sponsorCard.title") }}
        </h3>
      </div>
      <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-gray-50 text-(--text-primary) transition-all duration-300 group-hover:scale-105 group-hover:-rotate-3 dark:bg-zinc-900">
        <Github class="h-5 w-5" />
      </div>
    </div>

    <p class="mt-3 max-w-xl text-sm leading-relaxed text-(--text-secondary)">
      {{ t("sponsorCard.description") }}
    </p>

    <div class="mt-5 space-y-3">
      <a
        href="https://github.com/sponsors/besoeasy"
        target="_blank"
        rel="noopener noreferrer"
        :aria-label="t('sponsorCard.cta')"
        class="group/cta flex min-h-11 w-full items-center justify-between rounded-2xl bg-(--text-primary) px-4 py-3 text-(--bg-body) transition-all duration-300 hover:-translate-y-px hover:opacity-92 active:scale-[0.99]"
      >
        <span class="flex items-center gap-2.5">
          <Github class="h-4 w-4 shrink-0" />
          <span class="text-sm font-semibold tracking-tight">{{ t('sponsorCard.cta') }}</span>
        </span>
        <ArrowUpRight class="h-4 w-4 opacity-60 transition-all duration-300 group-hover/cta:translate-x-0.5 group-hover/cta:-translate-y-0.5 group-hover/cta:opacity-100" />
      </a>

      <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
      <a
        v-for="link in links"
        :key="link.title"
        :href="link.href"
        target="_blank"
        rel="noopener noreferrer"
        class="group/link flex min-h-11 items-center justify-between rounded-2xl bg-gray-50 px-3.5 py-3 text-(--text-primary) transition-all duration-300 hover:-translate-y-0.5 hover:smooth-shadow dark:bg-zinc-900/70"
      >
        <span class="flex min-w-0 items-center gap-2.5">
          <component :is="link.icon" class="h-4 w-4 shrink-0 text-(--text-secondary) transition-colors duration-300 group-hover/link:text-(--text-primary)" />
          <span class="truncate text-xs font-semibold">{{ link.title }}</span>
        </span>
        <ExternalLink class="ml-3 h-3.5 w-3.5 shrink-0 text-(--text-secondary) transition-all duration-300 group-hover/link:text-(--text-primary) group-hover/link:translate-x-0.5 group-hover/link:-translate-y-0.5" />
      </a>
      </div>
    </div>

    <div class="mt-5 grid grid-cols-2 gap-2 sm:gap-3">
      <div
        v-for="benefit in benefits"
        :key="benefit.title"
        class="rounded-2xl bg-gray-50 px-3 py-3 dark:bg-zinc-900/70 sm:px-3.5"
      >
        <component :is="benefit.icon" class="h-4 w-4 text-amber-500" />
        <p class="mt-2 text-[11px] font-medium leading-snug text-(--text-secondary) sm:text-xs">
          {{ benefit.title }}
        </p>
      </div>
    </div>

    <div class="mt-auto pt-5">
      <div class="flex items-center justify-between gap-3 rounded-2xl bg-gray-50 px-3.5 py-3 dark:bg-zinc-900/70 sm:px-4">
        <div class="min-w-0">
          <div class="flex items-center gap-2 text-[10px] font-bold uppercase tracking-[0.18em] text-(--text-secondary)">
            <GitBranch class="h-3.5 w-3.5 shrink-0" />
            <span>{{ t("home.externalLinks.buildInfo") }}</span>
          </div>
          <div class="mt-1 text-sm font-semibold tracking-tight text-(--text-primary)">
            {{ buildTimeAgo }}
          </div>
        </div>
        <div class="max-w-42 text-right text-[10px] font-mono tracking-tight text-(--text-secondary)" :title="buildTimestamp">
          {{ buildTimestamp }}
        </div>
      </div>
    </div>
  </div>
</template>
