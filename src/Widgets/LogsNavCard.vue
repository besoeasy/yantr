<script setup>
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { ChevronRight, ClipboardList } from "lucide-vue-next";

const router = useRouter();
const { t } = useI18n();
</script>

<template>
  <button
    type="button"
    @click="router.push('/logs')"
    class="group flex h-full w-full flex-col rounded-2xl p-6 text-left transition-all duration-300 hover:-translate-y-0.5 hover:smooth-shadow-lg bg-white dark:bg-[#0A0A0A] text-(--text-primary)"
  >
    <div class="flex h-full flex-col gap-6">
      <div class="flex items-start justify-between gap-4">
        <div class="flex min-w-0 items-start gap-3.5">
          <div
            class="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl"
            style="background: var(--surface-muted)"
          >
            <ClipboardList class="h-5 w-5 text-emerald-500 transition-transform duration-300 group-hover:scale-105" />
          </div>

          <div class="min-w-0">
            <h3 class="text-sm font-semibold tracking-tight">
              {{ t("home.toolsNavCard.logs") }}
            </h3>
            <p class="mt-1 text-xs leading-relaxed" style="color: var(--text-secondary)">
              {{ t("home.toolsNavCard.logsDesc") }}
            </p>
          </div>
        </div>

        <span class="inline-flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-[0.18em] text-emerald-500">
          <span class="h-2 w-2 rounded-full bg-emerald-500"></span>
          Live
        </span>
      </div>

      <div
        class="mt-auto flex items-center justify-between rounded-xl px-4 py-3 transition-colors duration-300 group-hover:bg-emerald-500/5"
        style="background: var(--surface-muted)"
      >
        <div class="min-w-0">
          <div class="text-[10px] font-bold uppercase tracking-[0.18em]" style="color: var(--text-secondary)">
            System Output
          </div>
          <div class="mt-1 text-sm font-semibold">
            {{ t("home.toolsNavCard.logs") }}
          </div>
        </div>

        <ChevronRight class="h-4 w-4 shrink-0 text-emerald-500 transition-transform duration-300 group-hover:translate-x-1" />
      </div>
    </div>
  </button>
</template>