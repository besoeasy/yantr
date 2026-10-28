<script>
// Dashboard engine: one column — a per-interface list reads best at a
// quarter of the grid, the same width as the other host tiles.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Cable, Activity, ArrowDown, ArrowUp } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";
import { formatBitRate } from "../utils/metrics";

const { t } = useI18n();

const {
  metricsStale,
  isDegraded,
  netInterfaces,
} = useHostMetrics();

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));

const hasData = computed(() => netInterfaces.value.length > 0);

function speedLabel(iface) {
  return iface.speedMbps ? `${iface.speedMbps} Mbps` : "—";
}

function stateClass(state) {
  return state === "up"
    ? "text-emerald-600 dark:text-emerald-400"
    : state === "down"
      ? "text-rose-600 dark:text-rose-400"
      : "yantra-text-secondary";
}
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-sky-200 bg-sky-50 transition-transform duration-200 group-hover:scale-105 dark:border-sky-900/50 dark:bg-sky-900/20">
          <Cable class="h-5 w-5 text-sky-600 dark:text-sky-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
            {{ t("quickMetrics.hostMetrics.linkStatus") }}
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
      <div v-if="isDegraded('network') || !hasData" class="flex items-center gap-3 rounded-xl border-red-200 bg-red-50 p-4 dark:border-red-900/50 dark:bg-red-900/20">
        <Activity class="h-5 w-5 shrink-0 text-red-500" />
        <div class="text-[11px] font-medium leading-tight text-red-600 dark:text-red-400">
          {{ unavailable }}
        </div>
      </div>

      <div v-else class="flex flex-col gap-2">
        <div
          v-for="iface in netInterfaces"
          :key="iface.iface"
          class="rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50"
        >
          <div class="flex items-center justify-between gap-3">
            <span class="truncate text-sm font-black tabular-nums tracking-tight yantra-text-primary">
              {{ iface.iface }}
            </span>
            <span class="shrink-0 text-[10px] font-bold uppercase tracking-widest" :class="stateClass(iface.operState)">
              {{ iface.operState || "—" }}
            </span>
          </div>
          <div class="mt-1.5 flex items-center justify-between gap-3 text-[11px] yantra-text-secondary">
            <span>{{ speedLabel(iface) }} · {{ iface.duplex || "—" }}</span>
            <span class="flex items-center gap-2 tabular-nums">
              <span class="flex items-center gap-1"><ArrowDown class="h-3 w-3 text-sky-500" />{{ formatBitRate(iface.rxBps) }}</span>
              <span class="flex items-center gap-1"><ArrowUp class="h-3 w-3 text-violet-500" />{{ formatBitRate(iface.txBps) }}</span>
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>