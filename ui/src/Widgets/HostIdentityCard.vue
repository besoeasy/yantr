<script>
// One concern: the host's software identity and the fact that it is
// answering — the OS by name, the machine-readable line beneath it, and how
// long it has been awake. The silicon is the Hardware card's question; the
// container census is the Stack card's. Red was the old Host System sheet's
// failure colour; it stays the identity family's colour, now for health too.
export const meta = { cols: 1 };
</script>
<script setup>
import { useI18n } from "vue-i18n";
import { Server } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";
import WidgetSkeleton from "../components/WidgetSkeleton.vue";

const { t } = useI18n();
const { loading, error, osInfo, uptimeText } = useHostMetrics();
</script>

<template>
  <WidgetSkeleton v-if="loading" />

  <section
    v-else-if="error"
    class="yantra-card yantra-card-min group relative flex h-full w-full flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow] duration-200 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <div class="flex items-center gap-4">
      <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-red-200 bg-red-50 dark:border-red-900/50 dark:bg-red-900/20">
        <Server class="h-5 w-5 text-red-600 dark:text-red-500" />
      </div>
      <div>
        <h3 class="text-lg font-bold leading-none tracking-tight text-red-950 dark:text-red-50">
          {{ t("home.hostCard.host") }}
        </h3>
        <p class="mt-2 text-[10px] font-semibold uppercase tracking-widest text-red-600 dark:text-red-400">
          {{ t("quickMetrics.hostMetrics.connectionFailed") }}
        </p>
      </div>
    </div>
    <div class="mt-auto flex flex-col pt-6">
      <div class="text-4xl font-black leading-[1.1] tracking-tighter yantra-text-primary">
        {{ t("common.error") }}
      </div>
      <div class="mt-2 line-clamp-2 text-sm yantra-text-secondary">{{ error }}</div>
    </div>
  </section>

  <section
    v-else
    class="yantra-card yantra-card-min group relative flex h-full w-full flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow] duration-200 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <div
      aria-hidden="true"
      class="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-red-500/30 to-transparent dark:via-red-500/40"
    ></div>

    <!-- Header: what this card is, and that it is answering. The status
         sits at the right edge, where a dashboard's eye lands last. -->
    <div class="flex w-full items-start justify-between gap-4">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-red-200 bg-red-50 transition-transform duration-200 group-hover:scale-105 dark:border-red-900/50 dark:bg-red-900/20">
          <Server class="h-5 w-5 text-red-600 dark:text-red-400" />
        </div>
        <h3 class="text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
          {{ t("home.hostCard.host") }}
        </h3>
      </div>
      <div class="flex shrink-0 items-center gap-2 text-emerald-600 dark:text-emerald-400">
        <span class="relative flex h-2 w-2">
          <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
          <span class="relative inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
        </span>
        <span class="text-[10px] font-semibold uppercase tracking-widest">
          {{ t("quickMetrics.hostMetrics.online") }}
        </span>
      </div>
    </div>

    <!-- Identity: the machine by name, then the machine-readable line -->
    <div class="mt-6 flex flex-col">
      <div class="truncate text-4xl font-black leading-[1.05] tracking-tighter yantra-text-primary">
        {{ osInfo.name }}
      </div>
      <div class="mt-2 truncate font-mono text-[11px] yantra-text-secondary" :title="`${osInfo.type} · ${osInfo.arch} · ${osInfo.kernel}`">
        {{ osInfo.type }} · {{ osInfo.arch }} · {{ osInfo.kernel }}
      </div>
    </div>

    <!-- Awake time. Uptime renders only when the poll has real numbers —
         undefined values would print "undefined", which is worse than a
         shorter list. -->
    <div v-if="uptimeText" class="mt-auto flex items-baseline justify-between gap-4 border-t border-zinc-100 py-2 dark:border-zinc-900">
      <span class="shrink-0 text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
        {{ t("quickMetrics.hostMetrics.uptime") }}
      </span>
      <span class="min-w-0 truncate text-right font-mono text-[13px] font-semibold yantra-text-primary">
        {{ uptimeText }}
      </span>
    </div>
  </section>
</template>
