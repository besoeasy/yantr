<script>
// Dashboard engine: one column — swap occupancy only. How full the swap
// device (or zram) is answers "did the box already hit the wall once"; the
// swap TRAFFIC question lives on the Swap & writeback card, and the working
// set lives on Memory. One subject per card. See Home.vue for how meta.cols
// is consumed.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { ArrowLeftRight } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";
import { formatBytes } from "../utils/metrics";

const { t } = useI18n();

const { metrics, metricsStale, isDegraded } = useHostMetrics();

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));
const mem = computed(() => metrics.value?.mem ?? null);

// A machine with no swap configured is telling the truth, not missing it:
// "None" is a configuration fact, so it gets the headline slot muted rather
// than a fake 0%.
const noSwap = computed(() => !((mem.value?.swapTotalBytes ?? 0) > 0));

const swapPct = computed(() => {
  const m = mem.value;
  if (!m?.ready || !m.swapTotalBytes) return null;
  return Math.min(100, (m.swapUsedBytes / m.swapTotalBytes) * 100);
});

const isHot = computed(() => (swapPct.value ?? 0) >= 90);

const headline = computed(() => {
  if (isDegraded("memory")) return unavailable;
  if (noSwap.value) return t("quickMetrics.hostMetrics.noSwap");
  return swapPct.value !== null ? `${Math.round(swapPct.value)}%` : "—";
});

const headlineClass = computed(() => {
  if (isDegraded("memory")) return "yantra-text-secondary";
  if (noSwap.value) return "yantra-text-secondary";
  return isHot.value ? "text-rose-600 dark:text-rose-400" : "yantra-text-primary";
});

const captionText = computed(() => {
  const m = mem.value;
  if (isDegraded("memory") || !m?.swapTotalBytes) return "";
  return `${formatBytes(m.swapUsedBytes)} / ${formatBytes(m.swapTotalBytes)}`;
});
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-purple-200 bg-purple-50 transition-transform duration-200 group-hover:scale-105 dark:border-purple-900/50 dark:bg-purple-900/20">
          <ArrowLeftRight class="h-5 w-5 text-purple-600 dark:text-purple-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black dark:group-hover:text-white">
            {{ t("quickMetrics.hostMetrics.memorySwap") }}
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
      <!-- Degraded: swap numbers come out of the same meminfo read as the
           Memory card, so they degrade together. -->
      <div
        v-if="isDegraded('memory')"
        class="flex items-center gap-3 rounded-xl border-red-200 bg-red-50 p-4 dark:border-red-900/50 dark:bg-red-900/20"
      >
        <ArrowLeftRight class="h-5 w-5 shrink-0 text-red-500" />
        <div class="text-[11px] font-medium leading-tight text-red-600 dark:text-red-400">
          {{ unavailable }}
        </div>
      </div>

      <template v-else>
        <!-- Headline: occupancy of the swap device, bytes alongside. -->
        <div class="mb-2 flex items-baseline justify-between gap-3">
          <span
            class="text-4xl font-black tabular-nums leading-[1.1] tracking-tighter transition-colors duration-200 group-hover:-translate-y-[3px] sm:text-5xl"
            :class="headlineClass"
          >
            {{ headline }}
          </span>
          <span class="truncate text-[9px] font-semibold tabular-nums yantra-text-secondary">
            {{ captionText }}
          </span>
        </div>

        <!-- The swap rail: purple ink is what the swap meter always wore
             inside the Memory card; 90%+ goes rose like every other bar. -->
        <div class="h-2 w-full overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
          <div
            class="h-full rounded-full transition-[width] duration-200"
            :class="isHot ? 'bg-rose-500' : 'bg-purple-500'"
            :style="{ width: `${swapPct ?? 0}%` }"
          ></div>
        </div>
      </template>
    </div>
  </div>
</template>
