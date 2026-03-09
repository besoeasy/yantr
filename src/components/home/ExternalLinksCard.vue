<script setup>
import { useI18n } from "vue-i18n";
import { Github, Bug, ExternalLink, Link2, GitBranch, Send } from "lucide-vue-next";

const { t } = useI18n();

const links = [
  {
    title: t("home.externalLinks.github"),
    href: "https://github.com/besoeasy/Yantr",
    icon: Github,
    color: "text-gray-700 dark:text-zinc-300",
  },
  {
    title: t("home.externalLinks.reportIssue"),
    href: "https://github.com/besoeasy/yantr/issues",
    icon: Bug,
    color: "text-amber-500",
  },
  {
    title: t("home.externalLinks.telegram"),
    href: "https://t.me/+h4RvCk63PxUyODQ1",
    icon: Send,
    color: "text-sky-500",
  },
];

const rawBuildTimestamp = import.meta.env.VITE_BUILD_TIMESTAMP;
const buildDate = rawBuildTimestamp ? new Date(rawBuildTimestamp) : null;
const buildTimestamp = buildDate && !Number.isNaN(buildDate.getTime())
  ? buildDate.toISOString().replace("T", " ").replace("Z", " UTC")
  : "Unknown";

function formatTimeAgo(date) {
  if (!(date instanceof Date) || Number.isNaN(date.getTime())) return "Unknown";

  const diffMs = Date.now() - date.getTime();
  const future = diffMs < 0;
  const diffSeconds = Math.abs(Math.round(diffMs / 1000));

  if (diffSeconds < 60) return future ? "in a few seconds" : "just now";

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
      const plural = value === 1 ? "" : "s";
      return future
        ? `in ${value} ${unit.label}${plural}`
        : `${value} ${unit.label}${plural} ago`;
    }
  }

  return future ? "in a few seconds" : "just now";
}

const buildTimeAgo = formatTimeAgo(buildDate);
</script>

<template>
  <div class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl overflow-hidden transition-all duration-400 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:border-gray-300 dark:hover:border-zinc-600">
    <!-- Hover Accents -->
    <div class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-gray-400 dark:via-zinc-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
    <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none [mask-image:linear-gradient(to_bottom,white,transparent)]"></div>

    <div class="relative z-10 flex flex-col h-full p-6">
      <div class="flex items-center gap-3 mb-6">
        <div class="w-10 h-10 rounded-lg bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center shrink-0 group-hover:scale-105 transition-all duration-500">
           <Link2 class="w-5 h-5 text-gray-400 dark:text-zinc-500 group-hover:text-gray-900 dark:group-hover:text-white transition-colors" />
        </div>
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight">{{ t("home.externalLinks.resources") }}</h3>
          <div class="text-[11px] font-medium text-gray-500 dark:text-zinc-400 uppercase tracking-wider mt-1">{{ t("home.externalLinks.buildData") }}</div>
        </div>
      </div>

      <div class="flex flex-wrap gap-2 mb-6">
        <a
          v-for="link in links"
          :key="link.title"
          :href="link.href"
          target="_blank"
          rel="noopener noreferrer"
          class="flex-1 flex items-center justify-center gap-2 py-2.5 px-3 rounded-lg bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50 hover:border-gray-300 dark:hover:border-zinc-700 transition-all duration-300 focus:outline-none group/link"
        >
          <component :is="link.icon" :class="['w-4 h-4', link.color]" />
          <span class="text-xs font-semibold text-gray-900 dark:text-white">{{ link.title }}</span>
          <ExternalLink class="w-3 h-3 text-gray-400 dark:text-zinc-600 group-hover/link:text-gray-900 dark:group-hover/link:text-white transition-colors ml-auto" />
        </a>
      </div>
      
      <div class="mt-auto pt-4 border-t border-gray-100 dark:border-zinc-800/80 flex items-center justify-between">
        <div class="flex items-center gap-2 text-gray-500 dark:text-zinc-400">
          <GitBranch class="w-3.5 h-3.5" />
          <span class="text-[10px] font-bold uppercase tracking-wider">{{ t("home.externalLinks.buildInfo") }}</span>
        </div>
        <div class="text-right" :title="buildTimestamp">
          <div class="text-[10px] font-semibold text-gray-700 dark:text-zinc-200 tracking-tight">
            {{ buildTimeAgo }}
          </div>
          <div class="mt-1 text-[10px] font-mono text-gray-500 dark:text-zinc-400 tracking-tighter bg-gray-50 dark:bg-zinc-900/80 px-2 py-1 rounded">
            {{ buildTimestamp }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
