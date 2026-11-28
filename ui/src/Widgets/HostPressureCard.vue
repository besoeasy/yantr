<script>
// Dashboard engine: one column — PSI reads as a status card, not a
// percentage gauge. The rows answer "is anything stalled, and where?"
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Gauge, Cpu, MemoryStick, HardDrive } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";

const { t } = useI18n();

const { metricsStale, isDegraded, pressure } = useHostMetrics();

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));
const hasData = computed(() => !!pressure.value?.ready);

function fmt(v) {
  return Number.isFinite(v) ? v.toFixed(2) : "—";
}

function widthPct(v) {
  if (!Number.isFinite(v) || v <= 0) return 0;
  // PSI is % of time stalled; cap at 100 for bar width, but keep low values visible.
  return Math.min(100, Math.max(v, 0.8));
}

const rows = computed(() => {
  const p = pressure.value;
  if (!p) return [];
  return [
    { key: "cpu", label: t("quickMetrics.hostMetrics.cpu"), icon: Cpu, entry: p.cpu },
    { key: "mem", label: t("quickMetrics.hostMetrics.memory"), icon: MemoryStick, entry: p.mem },
    { key: "io", label: t("quickMetrics.hostMetrics.disk"), icon: HardDrive, entry: p.io },
  ].map((row) => ({
    ...row,
    hasFull: Number.isFinite(row.entry.fullAvg10) && row.entry.fullAvg10 > 0,
    hasSome: Number.isFinite(row.entry.someAvg10) && row.entry.someAvg10 > 0,
  }));
});

const state = computed(() => {
  if (isDegraded("pressure") || !hasData.value) {
    return {
      key: "unavailable",
      label: unavailable.value,
      dot: "bg-red-500",
      ping: "hidden",
      text: "text-red-600 dark:text-red-400",
      iconWrap: "border-red-200 bg-red-50 dark:border-red-900/50 dark:bg-red-900/20",
      icon: "text-red-600 dark:text-red-400",
    };
  }
  if (metricsStale.value) {
    return {
      key: "stale",
      label: t("quickMetrics.hostMetrics.pressureStale"),
      dot: "bg-amber-500",
      ping: "hidden",
      text: "text-amber-600 dark:text-amber-500",
      iconWrap: "border-amber-200 bg-amber-50 dark:border-amber-900/50 dark:bg-amber-900/20",
      icon: "text-amber-600 dark:text-amber-400",
    };
  }
  const hasFull = rows.value.some((r) => r.hasFull);
  const hasSome = rows.value.some((r) => r.hasSome);
  if (hasFull) {
    return {
      key: "full",
      label: t("quickMetrics.hostMetrics.pressureFullStalls"),
      dot: "bg-rose-500",
      ping: "bg-rose-500",
      text: "text-rose-600 dark:text-rose-400",
      iconWrap: "border-rose-200 bg-rose-50 dark:border-rose-900/50 dark:bg-rose-900/20",
      icon: "text-rose-600 dark:text-rose-400",
    };
  }
  if (hasSome) {
    return {
      key: "some",
      label: t("quickMetrics.hostMetrics.pressureSomeStalls"),
      dot: "bg-amber-500",
      ping: "bg-amber-400",
      text: "text-amber-600 dark:text-amber-500",
      iconWrap: "border-amber-200 bg-amber-50 dark:border-amber-900/50 dark:bg-amber-900/20",
      icon: "text-amber-600 dark:text-amber-400",
    };
  }
  return {
    key: "none",
    label: t("quickMetrics.hostMetrics.pressureNoStalls"),
    dot: "bg-emerald-500",
    ping: "bg-emerald-400",
    text: "text-emerald-600 dark:text-emerald-500",
    iconWrap: "border-emerald-200 bg-emerald-50 dark:border-emerald-900/50 dark:bg-emerald-900/20",
    icon: "text-emerald-600 dark:text-emerald-400",
  };
});
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl transition-transform duration-200 group-hover:scale-105" :class="state.iconWrap">
          <Gauge class="h-5 w-5" :class="state.icon" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black dark:group-hover:text-white">
            {{ t("quickMetrics.hostMetrics.pressure") }}
          </h3>
          <div class="mt-2 flex items-center gap-2">
            <span class="relative flex h-2 w-2">
              <span v-if="state.ping !== 'hidden'" class="absolute inline-flex h-full w-full animate-ping rounded-full opacity-75" :class="state.ping"></span>
              <span class="inline-flex h-2 w-2 rounded-full" :class="state.dot"></span>
            </span>
            <span class="text-[10px] font-semibold uppercase tracking-widest" :class="state.text">{{ state.label }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="relative z-10 mt-auto flex flex-col pt-6 transition-opacity duration-200" :class="metricsStale ? 'opacity-50' : ''">
      <div v-if="state.key === 'unavailable'" class="flex items-center gap-3 rounded-xl border-red-200 bg-red-50 p-4 dark:border-red-900/50 dark:bg-red-900/20">
        <Gauge class="h-5 w-5 shrink-0 text-red-500" />
        <div class="text-[11px] font-medium leading-tight text-red-600 dark:text-red-400">
          {{ unavailable }}
        </div>
      </div>

      <div v-else class="flex flex-col">
        <div
          v-for="(row, idx) in rows"
          :key="row.key"
          class="py-3"
          :class="idx < rows.length - 1 ? 'border-b border-zinc-100 dark:border-zinc-900' : ''"
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

          <div class="mt-2 flex flex-col gap-1.5">
            <div class="flex items-center gap-2">
              <span class="w-8 shrink-0 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">{{ t("quickMetrics.hostMetrics.pressureSome") }}</span>
              <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-900">
                <div class="h-full rounded-full bg-amber-500 transition-[width] duration-300" :style="{ width: `${widthPct(row.entry.someAvg10)}%` }"></div>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <span class="w-8 shrink-0 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">{{ t("quickMetrics.hostMetrics.pressureFull") }}</span>
              <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-900">
                <div class="h-full rounded-full bg-rose-500 transition-[width] duration-300" :style="{ width: `${widthPct(row.entry.fullAvg10)}%` }"></div>
              </div>
            </div>
          </div>

          <div class="mt-2 flex items-center justify-between gap-3 text-[10px] yantra-text-secondary tabular-nums">
            <span>60s {{ fmt(row.entry.someAvg60) }}% · 300s {{ fmt(row.entry.someAvg300) }}%</span>
            <span>{{ t("quickMetrics.hostMetrics.pressureFull") }} {{ fmt(row.entry.fullAvg10) }}%</span>
          </div>
        </div>

        <div class="mt-2 flex flex-col gap-0.5 text-[9px] yantra-text-secondary">
          <span>{{ t("quickMetrics.hostMetrics.pressureLegendSome") }}</span>
          <span>{{ t("quickMetrics.hostMetrics.pressureLegendFull") }}</span>
        </div>
      </div>
    </div>
  </div>
</template>