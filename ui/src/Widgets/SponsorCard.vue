<script setup>
import { useI18n } from "vue-i18n";
import { ArrowUpRight, Heart } from "@lucide/vue";

const { t } = useI18n();
</script>

<template>
  <div class="group relative flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-all duration-300 hover:-translate-y-1 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <div class="flex w-full items-start justify-between">
      <div class="flex items-center gap-4">
        <!-- Minimal Icon Block -->
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-rose-200 bg-rose-50 transition-transform duration-300 group-hover:scale-105 dark:border-rose-900/50 dark:bg-rose-900/20">
          <Heart class="h-5 w-5 text-rose-500" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors group-hover:text-black dark:text-white dark:group-hover:text-white">
            {{ t("sponsorCard.label") }}
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
        {{ t("sponsorCard.title") }}
      </div>
      <div class="mb-6 text-sm leading-relaxed text-zinc-500 line-clamp-2 dark:text-zinc-400">
        {{ t("sponsorCard.description") }}
      </div>

      <!-- Main CTA -->
      <a
        href="https://github.com/sponsors/besoeasy"
        target="_blank"
        rel="noopener noreferrer"
        class="group/cta relative flex h-12 w-full items-center justify-between rounded-xl bg-rose-500 px-5 text-white transition-all duration-300 hover:bg-rose-600 active:scale-[0.98] overflow-hidden"
      >
        <div class="relative z-10 flex items-center gap-2.5">
          <Heart class="h-4 w-4 fill-current" />
          <span class="text-[11px] font-bold uppercase tracking-widest">{{ t('sponsorCard.cta') }}</span>
        </div>
        <ArrowUpRight class="relative z-10 h-4.5 w-4.5 opacity-70 transition-transform duration-300 group-hover/cta:-translate-y-1 group-hover/cta:translate-x-1 group-hover/cta:opacity-100" />
      </a>
    </div>
  </div>
</template>
