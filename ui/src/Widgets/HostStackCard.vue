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
import { AlertCircle, Boxes, CirclePause, HardDrive, Layers, Package } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";
import WidgetSkeleton from "../components/WidgetSkeleton.vue";

const { t } = useI18n();
const { loading, error, runningCount, stoppedCount, imagesCount, totalVolumes, podmanVersion } = useHostMetrics();

// The workload is a fact table, so even a zero is printed — a homelab with
// nothing running is a real state, and an em-dash would be a lie. Stopped is
// highlighted when containers are paused/stopped to draw operational attention.
const workloadStats = computed(() => [
  {
    key: "apps",
    label: t("home.overviewPulseCard.apps"),
    value: runningCount.value,
    icon: Package,
    iconColor: "text-emerald-600 dark:text-emerald-400",
    valueColor: "text-emerald-600 dark:text-emerald-400",
  },
  {
    key: "volumes",
    label: t("home.overviewPulseCard.volumes"),
    value: totalVolumes.value,
    icon: HardDrive,
    iconColor: "text-indigo-600 dark:text-indigo-400",
    valueColor: "yantra-text-primary",
  },
  {
    key: "images",
    label: t("home.overviewPulseCard.images"),
    value: imagesCount.value,
    icon: Layers,
    iconColor: "text-indigo-600 dark:text-indigo-400",
    valueColor: "yantra-text-primary",
  },
  {
    key: "stopped",
    label: t("home.overviewPulseCard.stopped"),
    value: stoppedCount.value,
    icon: CirclePause,
    iconColor: stoppedCount.value > 0 ? "text-amber-500" : "yantra-text-secondary",
    valueColor: stoppedCount.value > 0 ? "text-amber-600 dark:text-amber-400" : "yantra-text-secondary",
    muted: stoppedCount.value === 0,
  },
]);
</script>

<template>
  <WidgetSkeleton v-if="loading" />

  <section
    v-else-if="error"
    class="yantra-card yantra-card-min group relative flex h-full w-full flex-col overflow-hidden border-red-200 p-7 text-left dark:border-red-900/50"
  >
    <div class="flex items-center gap-3">
      <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border-red-200 bg-red-50 dark:border-red-900/50 dark:bg-red-900/10">
        <AlertCircle class="h-4 w-4 text-red-600 dark:text-red-500" />
      </div>
      <div>
        <h3 class="text-[10px] font-bold uppercase tracking-widest text-red-600 dark:text-red-400">
          {{ t("home.hostCard.stack") }}
        </h3>
        <p class="mt-0.5 text-[10px] font-semibold uppercase tracking-wider text-red-500 dark:text-red-400">
          {{ t("quickMetrics.hostMetrics.connectionFailed") }}
        </p>
      </div>
    </div>
    <div class="mt-auto flex flex-col">
      <div class="text-[10px] font-semibold uppercase tracking-widest text-red-500 dark:text-red-400">
        Failed to load
      </div>
      <div class="mt-2 text-3xl font-black leading-[1.1] tracking-tighter text-red-950 yantra-text-primary">
        {{ t("common.error") }}
      </div>
    </div>
  </section>

  <section
    v-else
    class="yantra-card yantra-card-min group relative flex h-full w-full flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <div
      aria-hidden="true"
      class="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-indigo-500/30 to-transparent dark:via-indigo-500/40"
    ></div>

    <!-- Header -->
    <div class="flex w-full items-start justify-between gap-4">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-indigo-200 bg-indigo-50 transition-transform duration-200 group-hover:scale-105 dark:border-indigo-900/50 dark:bg-indigo-900/20">
          <Boxes class="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
        </div>
        <div>
          <h3 class="text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
            {{ t("home.hostCard.stack") }}
          </h3>
          <div class="mt-1 flex items-baseline gap-1.5">
            <span class="text-xl font-black tracking-tight yantra-text-primary">
              {{ runningCount }}
            </span>
            <span class="text-[11px] font-medium yantra-text-secondary">
              {{ t("home.overviewPulseCard.apps") }}
            </span>
          </div>
        </div>
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

    <!-- Census: 2x2 stat grid -->
    <div class="mt-6 grid grid-cols-2 gap-2.5">
      <div
        v-for="stat in workloadStats"
        :key="stat.key"
        class="flex flex-col rounded-xl border border-zinc-200 bg-zinc-50 p-3.5 transition-colors duration-200 hover:border-zinc-300 dark:border-zinc-800 dark:bg-zinc-900/50 dark:hover:border-zinc-700"
      >
        <div class="flex items-center justify-between gap-1.5">
          <span class="text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
            {{ stat.label }}
          </span>
          <component :is="stat.icon" class="h-3.5 w-3.5 shrink-0 opacity-70" :class="stat.iconColor" />
        </div>
        <div
          class="mt-2 text-2xl font-black tabular-nums tracking-tight"
          :class="stat.valueColor"
        >
          {{ stat.value }}
        </div>
      </div>
    </div>

    <!-- Podman engine version footer -->
    <div class="mt-auto flex items-center justify-between gap-4 border-t border-zinc-100 pt-4 dark:border-zinc-800">
      <div class="flex items-center gap-1.5">
        <span class="h-1.5 w-1.5 rounded-full bg-indigo-500"></span>
        <span class="text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
          {{ t("quickMetrics.hostMetrics.podman") }}
        </span>
      </div>
      <span class="font-mono text-xs font-semibold tabular-nums yantra-text-primary">
        {{ podmanVersion || "—" }}
      </span>
    </div>
  </section>
</template>
