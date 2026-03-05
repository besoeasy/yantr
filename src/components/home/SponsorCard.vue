<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Heart, MessageCircle, Zap, GitPullRequest, ShieldCheck, ArrowUpRight } from "lucide-vue-next";

const { t } = useI18n();

const benefits = computed(() => [
  {
    icon: MessageCircle,
    title: t("sponsorCard.benefits.devAccess.title"),
    desc: t("sponsorCard.benefits.devAccess.desc"),
  },
  {
    icon: Zap,
    title: t("sponsorCard.benefits.roadmap.title"),
    desc: t("sponsorCard.benefits.roadmap.desc"),
  },
  {
    icon: GitPullRequest,
    title: t("sponsorCard.benefits.earlyBuilds.title"),
    desc: t("sponsorCard.benefits.earlyBuilds.desc"),
  },
  {
    icon: ShieldCheck,
    title: t("sponsorCard.benefits.badge.title"),
    desc: t("sponsorCard.benefits.badge.desc"),
  },
]);
</script>

<template>
  <div class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl overflow-hidden transition-all duration-400 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:border-gray-300 dark:hover:border-zinc-600">
    <!-- Hover Accents -->
    <div class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-amber-400 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
    <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none [mask-image:linear-gradient(to_bottom,white,transparent)]"></div>

    <div class="relative z-10 flex flex-col h-full p-6">
      <div class="flex items-center gap-3 mb-6">
        <div class="w-10 h-10 rounded-lg bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center shrink-0 group-hover:scale-105 transition-all duration-500">
          <Heart class="w-5 h-5 text-gray-400 dark:text-zinc-500 group-hover:text-amber-500 transition-colors" />
        </div>
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight group-hover:text-amber-600 dark:group-hover:text-amber-400 transition-colors">{{ t('sponsorCard.title') }}</h3>
          <div class="text-[11px] font-medium text-gray-500 dark:text-zinc-400 uppercase tracking-wider mt-1">{{ t('sponsorCard.label') }}</div>
        </div>
      </div>

      <div class="flex flex-col gap-2">
        <div
          v-for="b in benefits"
          :key="b.title"
          class="flex items-start gap-3 p-3 rounded-lg bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50"
        >
          <component :is="b.icon" class="w-4 h-4 text-amber-500 shrink-0 mt-0.5" />
          <div class="flex flex-col">
            <span class="text-xs font-semibold text-gray-900 dark:text-white">{{ b.title }}</span>
            <span class="text-[10px] text-gray-500 dark:text-zinc-400">{{ b.desc }}</span>
          </div>
        </div>
      </div>

      <div class="mt-4 pt-4 border-t border-gray-100 dark:border-zinc-800/80">
        <a
          href="https://sponsor.besoeasy.com/"
          target="_blank"
          rel="noopener noreferrer"
          class="group/cta flex items-center justify-between w-full px-4 py-3 rounded-lg bg-gray-950 dark:bg-white text-white dark:text-gray-950 transition-all duration-300 hover:bg-gray-800 dark:hover:bg-gray-100 active:scale-[0.98]"
        >
          <span class="text-xs font-bold tracking-tight">{{ t('sponsorCard.cta') }}</span>
          <ArrowUpRight class="w-4 h-4 opacity-50 transition-all duration-300 group-hover/cta:opacity-100 group-hover/cta:translate-x-0.5 group-hover/cta:-translate-y-0.5" />
        </a>
      </div>
    </div>
  </div>
</template>
