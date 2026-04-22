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

const links = [
  { title: t("home.externalLinks.github"), href: "https://github.com/besoeasy/Yantr", icon: Github },
  { title: t("home.externalLinks.reportIssue"), href: "https://github.com/besoeasy/yantr/issues", icon: Bug },
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
  <div class="group h-full flex flex-col bg-white dark:bg-[#0A0A0A] rounded-xl p-5 transition-all duration-300 hover:-translate-y-0.5 smooth-shadow">
    <!-- Header -->
    <div class="flex items-start gap-3 mb-4">
      <div class="w-9 h-9 rounded-lg bg-gray-100 dark:bg-zinc-800 flex items-center justify-center shrink-0">
        <Github class="w-5 h-5 text-gray-800 dark:text-white" />
      </div>
      <div class="min-w-0">
        <div class="text-[10px] font-bold uppercase tracking-[0.18em] text-gray-400 dark:text-zinc-500">{{ t('sponsorCard.label') }}</div>
        <h3 class="text-sm font-semibold text-gray-900 dark:text-white leading-snug mt-0.5">{{ t('sponsorCard.title') }}</h3>
      </div>
    </div>

    <!-- Quick Links -->
    <div class="flex gap-2 mb-4">
      <a
        v-for="link in links"
        :key="link.title"
        :href="link.href"
        target="_blank"
        rel="noopener noreferrer"
        class="flex-1 flex items-center justify-center gap-1.5 py-2 px-3 rounded-lg bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50 hover:border-gray-300 dark:hover:border-zinc-700 transition-all duration-300 group/link"
      >
        <component :is="link.icon" class="w-3.5 h-3.5 text-gray-600 dark:text-zinc-400" />
        <span class="text-[11px] font-semibold text-gray-800 dark:text-zinc-200">{{ link.title }}</span>
        <ExternalLink class="w-3 h-3 text-gray-300 dark:text-zinc-600 group-hover/link:text-gray-700 dark:group-hover/link:text-zinc-300 transition-colors ml-auto" />
      </a>
    </div>

    <!-- Benefits -->
    <ul class="flex-1 flex flex-col gap-2 mb-4">
      <li
        v-for="benefit in benefits"
        :key="benefit.title"
        class="flex items-center gap-2 text-[11px] text-gray-600 dark:text-zinc-400"
      >
        <component :is="benefit.icon" class="w-3.5 h-3.5 shrink-0 text-amber-500" />
        <span>{{ benefit.title }}</span>
      </li>
    </ul>

    <!-- Sponsor CTA -->
    <a
      href="https://github.com/sponsors/besoeasy"
      target="_blank"
      rel="noopener noreferrer"
      :aria-label="t('sponsorCard.cta')"
      class="group/cta flex items-center justify-between w-full px-4 py-3 rounded-lg bg-gray-950 dark:bg-white text-white dark:text-gray-950 transition-all duration-300 hover:bg-gray-800 dark:hover:bg-gray-100 active:scale-[0.98] mb-4"
    >
      <span class="flex items-center gap-2">
        <Github class="w-4 h-4 shrink-0 opacity-80" />
        <span class="text-xs font-bold tracking-tight">{{ t('sponsorCard.cta') }}</span>
      </span>
      <ArrowUpRight class="w-4 h-4 opacity-40 transition-all duration-300 group-hover/cta:opacity-100 group-hover/cta:translate-x-0.5 group-hover/cta:-translate-y-0.5" />
    </a>

    <!-- Build Info -->
    <div class="pt-3 border-t border-gray-100 dark:border-zinc-800/80 flex items-center justify-between">
      <div class="flex items-center gap-1.5 text-gray-500 dark:text-zinc-400">
        <GitBranch class="w-3.5 h-3.5" />
        <span class="text-[10px] font-bold uppercase tracking-wider">{{ t("home.externalLinks.buildInfo") }}</span>
      </div>
      <div class="text-right" :title="buildTimestamp">
        <div class="text-[10px] font-semibold text-gray-700 dark:text-zinc-200 tracking-tight">{{ buildTimeAgo }}</div>
        <div class="mt-0.5 text-[10px] font-mono text-gray-400 dark:text-zinc-500 tracking-tighter">{{ buildTimestamp }}</div>
      </div>
    </div>
  </div>
</template>
