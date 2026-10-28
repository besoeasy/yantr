<script>
// Dashboard engine: one column — three resource rows with their averages
// read cleanly at a quarter of the grid.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Gauge, Cpu, MemoryStick, HardDrive } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";

const { t } = useI18n();

const {
  metricsStale,
  isDegraded,
  pressure,
} = useHostMetrics();

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));

const hasData = computed(() => !!pressure.value?.ready);

// PSI "some" averages are the headline: how much of the window at least one
// task was stalled. Full averages stay secondary.
function fmt(v) {
  return Number.isFinite(v) ? v.toFixed(2) : "—";
}

const rows = computed(() => {
  const p = pressure.value;
  if (!p) return [];
  return [
    { key: "cpu", label: t("quickMetrics.hostMetrics.cpu"), icon: Cpu, entry: p.cpu },
    { key: "mem", label: t("quickMetrics.hostMetrics.memory"), icon: MemoryStick, entry: p.mem },
    { key: "io", label: t("quickMetrics.hostMetrics.disk"), icon: HardDrive, entry: p.io },
  ];
});
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-amber-200 bg-amber-50 transition-transform duration-200 group-hover:scale-105 dark:border-amber-900/50 dark:bg-amber-900/20">
          <Gauge class="h-5 w-5 text-amber-600 dark:text-amber-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
            {{ t("quickMetrics.hostMetrics.pressure") }}
          </h3>
          <div class="mt-2 flex items-center gap-2">
            <span class="relative flex h-2 w-2">
              <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
              <span class="inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
            </span>
            <span class="text-[10px] font-semibold uppercase tracking-widest text-emerald-600 dark:text-emerald-500">{{ t("quickMetrics.hostMetrics.live") }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="relative z-10 mt-auto flex flex-col pt-6 transition-opacity duration-200" :class="metricsStale ? 'opacity-50' : ''">
      <div v-if="isDegraded('pressure') || !hasData" class="flex items-center gap-3 rounded-xl border-red-200 bg-red-50 p-4 dark:border-red-900/50 dark:bg-red-900/20">
        <Gauge class="h-5 w-5 shrink-0 text-red-500" />
        <div class="text-[11px] font-medium leading-tight text-red-600 dark:text-red-400">
          {{ unavailable }}
        </div>
      </div>

      <div v-else class="flex flex-col gap-2">
        <div
          v-for="row in rows"
          :key="row.key"
          class="rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50"
        >
          <div class="flex items-center justify-between gap-3">
            <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
              <component :is="row.icon" class="h-3 w-3 shrink-0" />
              {{ row.label }}
            </span>
            <span class="text-sm font-black tabular-nums tracking-tight yantra-text-primary">
              {{ fmt(row.entry.someAvg10) }}%
            </span>
          </div>
          <div class="mt-1.5 flex items-center justify-between gap-3 text-[10px] yantra-text-secondary tabular-nums">
            <span>some 60s {{ fmt(row.entry.someAvg60) }} · 300s {{ fmt(row.entry.someAvg300) }}</span>
            <span>full {{ fmt(row.entry.fullAvg10) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>