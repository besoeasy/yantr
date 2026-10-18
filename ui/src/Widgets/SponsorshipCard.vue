<script>
// Dashboard engine: two of the desktop grid's four columns — the ask and the
// CTA share one row, so the card reads as a single decision instead of a page
// of it. Same shape as the report card beside it; the .w-span-2 utility in
// main.css clamps it per breakpoint, see Home.vue.
export const meta = { cols: 2 };
</script>
<script setup>
import { useI18n } from "vue-i18n";
import { ArrowUpRight, Heart } from "@lucide/vue";

const { t } = useI18n();
</script>

<template>
  <div class="group relative flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl bg-white p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:bg-black dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)] sm:border sm:border-zinc-200 dark:sm:border-zinc-800">
    <!-- Header: identity only. The ask lives with the CTA below it. -->
    <div class="flex w-full items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-rose-200 bg-rose-50 transition-transform duration-200 group-hover:scale-105 dark:border-rose-900/50 dark:bg-rose-900/20">
          <Heart class="h-5 w-5 fill-current text-rose-500" />
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

      <div class="flex h-8 w-8 shrink-0 -translate-y-1 items-center justify-center rounded-full opacity-0 transition-[border-color,box-shadow,transform] duration-200 group-hover:translate-y-0 group-hover:opacity-100">
        <Heart class="h-5 w-5 text-zinc-400 transition-colors hover:text-red-500 dark:text-zinc-600 dark:hover:text-red-400" />
      </div>
    </div>

    <!-- Body: the ask left, the CTA right — one decision per glance. -->
    <div class="relative z-10 mt-auto flex flex-wrap items-end justify-between gap-x-8 gap-y-4 pt-5">
      <div class="min-w-0 max-w-[38ch] flex-1">
        <div class="mb-1.5 text-2xl font-black leading-[1.2] tracking-tight text-zinc-900 transition-colors duration-200 group-hover:-translate-y-[3px] dark:text-white">
          {{ t("sponsorCard.title") }}
        </div>
        <div class="line-clamp-2 text-xs leading-relaxed text-zinc-500 dark:text-zinc-400">
          {{ t("sponsorCard.description") }}
        </div>
      </div>

      <a
        href="https://github.com/sponsors/besoeasy"
        target="_blank"
        rel="noopener noreferrer"
        class="group/cta relative flex h-11 shrink-0 items-center justify-between gap-3 rounded-xl bg-rose-500 px-5 text-white transition-[border-color,box-shadow,transform] duration-200 hover:bg-rose-600 active:scale-[0.98] overflow-hidden"
      >
        <div class="relative z-10 flex items-center gap-2.5">
          <Heart class="h-4 w-4 shrink-0 fill-current" />
          <span class="text-[11px] font-bold uppercase tracking-widest">{{ t('sponsorCard.cta') }}</span>
        </div>
        <ArrowUpRight class="relative z-10 h-4.5 w-4.5 shrink-0 opacity-70 transition-transform duration-200 group-hover/cta:-translate-y-1 group-hover/cta:translate-x-1 group-hover/cta:opacity-100" />
      </a>
    </div>
  </div>
</template>
