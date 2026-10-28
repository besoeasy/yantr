<script>
// One concern: the container census — how many apps are running, how many
// lie stopped, and the volumes and images behind them, plus the engine
// version that qualifies the numbers. What the machine is, is the Host
// card's answer; what each container does, is TopConsumers' and the events'.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Boxes } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";
import WidgetSkeleton from "../components/WidgetSkeleton.vue";

const { t } = useI18n();
const { loading, error, runningCount, stoppedCount, imagesCount, totalVolumes, podmanVersion } = useHostMetrics();

// The workload is a fact table, so even a zero is printed — a homelab with
// nothing running is a real state, and an em-dash would be a lie. Stopped is
// the muted cell: the one number here that is not a healthy state.
const workloadStats = computed(() => [
  { key: "apps", label: t("home.overviewPulseCard.apps"), value: runningCount.value },
  { key: "volumes", label: t("home.overviewPulseCard.volumes"), value: totalVolumes.value },
  { key: "images", label: t("home.overviewPulseCard.images"), value: imagesCount.value },
  { key: "stopped", label: t("home.overviewPulseCard.stopped"), value: stoppedCount.value, muted: true },
]);
</script>

<template>
  <WidgetSkeleton v-if="loading" />

  <section
    v-else-if="error"
    class="yantra-card yantra-card-min group relative flex h-full w-full flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow] duration-200 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <div class="flex items-center gap-4">
      <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-red-200 bg-red-50 dark:border-red-900/50 dark:bg-red-900/20">
        <Boxes class="h-5 w-5 text-red-600 dark:text-red-500" />
      </div>
      <div>
        <h3 class="text-lg font-bold leading-none tracking-tight text-red-950 dark:text-red-50">
          {{ t("home.hostCard.stack") }}
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
      class="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-zinc-300/40 to-transparent dark:via-zinc-700/40"
    ></div>

    <!-- Header -->
    <div class="flex w-full items-start justify-between gap-4">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-zinc-50 transition-transform duration-200 group-hover:scale-105 dark:border-zinc-800 dark:bg-zinc-900">
          <Boxes class="h-5 w-5 text-zinc-600 transition-colors group-hover:text-zinc-900 yantra-text-secondary dark:group-hover:text-zinc-100" />
        </div>
        <h3 class="text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
          {{ t("home.hostCard.stack") }}
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

    <!-- Census: a 2x2 fact grid. Hairlines are drawn per cell (the utility
         layer has no divide-* helpers, and silent no-op classes are how
         borders go missing here) — container border plus cell right/bottom
         edges, overflow-hidden to clean the corners. -->
    <div class="mt-6 grid grid-cols-2 overflow-hidden rounded-lg dark:border-zinc-800">
      <div
        v-for="stat in workloadStats"
        :key="stat.key"
        class="flex flex-col items-start gap-1 border-b border-r px-4 py-3 dark:border-zinc-800"
      >
        <span
          class="text-2xl font-bold leading-none tabular-nums"
          :class="stat.muted ? 'yantra-text-secondary' : 'yantra-text-primary'"
        >
          {{ stat.value }}
        </span>
        <span class="text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
          {{ stat.label }}
        </span>
      </div>
    </div>

    <!-- The engine version rides with the census: it is what these numbers
         are numbers OF. -->
    <div class="mt-auto flex items-baseline justify-between gap-4 border-t border-zinc-100 pb-2 pt-6 dark:border-zinc-900">
      <span class="shrink-0 text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
        {{ t("quickMetrics.hostMetrics.podman") }}
      </span>
      <span class="min-w-0 truncate text-right font-mono text-[13px] font-semibold yantra-text-primary">
        {{ podmanVersion }}
      </span>
    </div>
  </section>
</template>
