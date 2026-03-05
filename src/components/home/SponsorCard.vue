<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { ArrowUpRight, MessageCircle, Zap, GitPullRequest, ShieldCheck } from "lucide-vue-next";

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
  <div class="group relative h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl overflow-hidden transition-all duration-300 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:border-gray-300 dark:hover:border-zinc-600">

    <!-- Hover accent line -->
    <div class="absolute top-0 left-0 w-full h-[2px] bg-linear-to-r from-transparent via-amber-400 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
    <!-- Dot-grid pattern -->
    <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none mask-[linear-gradient(to_bottom,white,transparent)]"></div>

    <!-- Horizontal layout: left = pitch + CTA, right = benefits grid -->
    <div class="relative z-10 flex flex-col sm:flex-row h-full gap-0">

      <!-- Left: headline + CTA -->
      <div class="flex flex-col justify-between p-6 sm:w-64 shrink-0 sm:border-r border-gray-100 dark:border-zinc-800/80">
        <div>
          <div class="flex items-center gap-2 mb-2">
            <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-amber-500 dark:text-amber-400">{{ $t('sponsorCard.label') }}</span>
            <span class="inline-flex items-center px-1.5 py-0.5 rounded text-[9px] font-bold uppercase tracking-wider bg-amber-50 dark:bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-100 dark:border-amber-500/20">{{ $t('sponsorCard.perksIncluded') }}</span>
          </div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight leading-snug mb-1.5">
            {{ $t('sponsorCard.title') }}
          </h3>
          <p class="text-[11px] text-gray-500 dark:text-zinc-500 leading-relaxed">
            {{ $t('sponsorCard.description') }}
          </p>
        </div>

        <a
          href="https://sponsor.besoeasy.com/"
          target="_blank"
          rel="noopener noreferrer"
          class="group/cta mt-5 flex items-center justify-between w-full px-4 py-3 rounded-lg bg-gray-950 dark:bg-white text-white dark:text-gray-950 transition-all duration-300 hover:bg-gray-800 dark:hover:bg-gray-100 active:scale-[0.98]"
        >
          <div>
            <span class="text-xs font-bold tracking-tight">{{ $t('sponsorCard.cta') }}</span>
          </div>
          <ArrowUpRight class="w-4 h-4 opacity-50 transition-all duration-300 group-hover/cta:opacity-100 group-hover/cta:translate-x-0.5 group-hover/cta:-translate-y-0.5" />
        </a>
      </div>

      <!-- Right: benefits 2×2 grid -->
      <div class="flex-1 grid grid-cols-2 gap-px bg-gray-100 dark:bg-zinc-800/60 border-t sm:border-t-0 border-gray-100 dark:border-zinc-800/60">
        <div
          v-for="b in benefits"
          :key="b.title"
          class="flex flex-col gap-1.5 p-4 bg-white dark:bg-[#0A0A0A] group-hover:bg-gray-50/50 dark:group-hover:bg-zinc-900/40 transition-colors"
        >
          <div class="flex items-center gap-1.5 mb-0.5">
            <component :is="b.icon" class="w-3.5 h-3.5 text-amber-500 dark:text-amber-400 shrink-0" />
            <span class="text-[10px] font-bold uppercase tracking-wider text-gray-700 dark:text-zinc-300">{{ b.title }}</span>
          </div>
          <p class="text-[11px] text-gray-500 dark:text-zinc-500 leading-snug">{{ b.desc }}</p>
        </div>
      </div>

    </div>
  </div>
</template>


