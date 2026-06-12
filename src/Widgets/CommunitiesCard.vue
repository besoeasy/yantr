<script setup>
import { useI18n } from "vue-i18n";
import { ArrowUpRight } from "lucide-vue-next";

const { t } = useI18n();

const DISCORD_URL = "https://discord.gg/Z9FYUEH2zX";
</script>

<template>
  <div class="group flex h-full flex-col rounded-[1.4rem] bg-white p-5 text-(--text-primary) smooth-shadow transition-all duration-300 hover:-translate-y-0.5 hover:smooth-shadow-lg dark:bg-[#0A0A0A] sm:p-6">
    <div class="flex items-start justify-between gap-4">
      <div class="min-w-0">
        <div class="flex flex-wrap items-center gap-2 text-[10px] font-bold uppercase tracking-[0.18em] text-(--text-secondary)">
          <span>{{ t("communitiesCard.label") }}</span>
          <span class="opacity-35">/</span>
          <span>{{ t("communitiesCard.badge") }}</span>
        </div>
        <h3 class="mt-3 max-w-[16rem] text-xl font-semibold leading-tight tracking-tight text-(--text-primary) sm:text-[1.35rem]">
          {{ t("communitiesCard.title") }}
        </h3>
      </div>
      <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-[#5865F2]/10 text-[#5865F2] transition-all duration-300 group-hover:scale-105 group-hover:-rotate-3">
        <svg class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
          <path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028 14.09 14.09 0 0 0 1.226-1.994.076.076 0 0 0-.041-.106 13.107 13.107 0 0 1-1.872-.892.077.077 0 0 1-.008-.128 10.2 10.2 0 0 0 .372-.292.074.074 0 0 1 .077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.892.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.157 2.418z" />
        </svg>
      </div>
    </div>

    <p class="mt-3 max-w-xl text-sm leading-relaxed text-(--text-secondary)">
      {{ t("communitiesCard.description") }}
    </p>

    <div class="mt-5">
      <a
        :href="DISCORD_URL"
        target="_blank"
        rel="noopener noreferrer"
        :aria-label="t('communitiesCard.cta')"
        class="group/cta flex min-h-11 w-full items-center justify-between rounded-2xl bg-[#5865F2] px-4 py-3 text-white transition-all duration-300 hover:-translate-y-px hover:bg-[#4752C4] active:scale-[0.99]"
      >
        <span class="flex items-center gap-2.5">
          <svg class="h-4 w-4 shrink-0" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028 14.09 14.09 0 0 0 1.226-1.994.076.076 0 0 0-.041-.106 13.107 13.107 0 0 1-1.872-.892.077.077 0 0 1-.008-.128 10.2 10.2 0 0 0 .372-.292.074.074 0 0 1 .077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.892.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.157 2.418z" />
          </svg>
          <span class="text-sm font-semibold tracking-tight">{{ t("communitiesCard.cta") }}</span>
        </span>
        <ArrowUpRight class="h-4 w-4 opacity-60 transition-all duration-300 group-hover/cta:translate-x-0.5 group-hover/cta:-translate-y-0.5 group-hover/cta:opacity-100" />
      </a>
    </div>

    <div class="mt-auto pt-5">
      <div class="rounded-2xl bg-gray-50 px-3.5 py-3 dark:bg-zinc-900/70 sm:px-4">
        <div class="text-[10px] font-bold uppercase tracking-[0.18em] text-(--text-secondary)">
          {{ t("communitiesCard.footer") }}
        </div>
        <div class="mt-1 text-sm font-semibold tracking-tight text-(--text-primary)">
          {{ t("communitiesCard.links.discord.desc") }}
        </div>
      </div>
    </div>
  </div>
</template>