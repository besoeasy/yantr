<script>
// Dashboard engine: one column — swap traffic as the headline, page state as
// rows. Swap occupancy is the Memory card's fact ("how full is the swap
// partition"); this card is its event ("is it moving right now"). Every value
// comes from /proc/vmstat through the shared poller, so a host without it
// degrades instead of performing a flat zero line. See Home.vue for how
// meta.cols is consumed.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { ArrowDownUp, Database } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";
import { formatBytes } from "../utils/metrics";

const { t } = useI18n();

const { metrics, metricsStale, isDegraded, vm } = useHostMetrics();

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));

// Swap traffic needs the two-sample window to have elapsed once; the page
// counts are instantaneous and are valid from the first tick.
const ready = computed(() => vm.value?.ready === true);

// A box with no swap space will never produce swap traffic — "None" is a
// configuration fact, not a rate of zero.
const noSwap = computed(() => !((metrics.value?.mem?.swapTotalBytes ?? 0) > 0));

const rateText = (bps) => (Number.isFinite(bps) ? `${formatBytes(bps)}/s` : "—");

const swapOutText = computed(() => (noSwap.value ? null : rateText(vm.value?.swapOutBps)));
const swapInText = computed(() => (noSwap.value ? null : rateText(vm.value?.swapInBps)));

// Swap is moving: that is the thrash alarm the whole card exists for, so the
// headline turns rose on ANY traffic rather than on a threshold — direction
// and presence are the signal, magnitude is context.
const swapping = computed(() => noSwap.value ? false : (vm.value?.swapOutBps ?? 0) > 0 || (vm.value?.swapInBps ?? 0) > 0);

// One class, not a stack: no-swap and not-ready read muted, swap traffic
// reads rose, a quiet-but-configured swap reads normal.
const headlineClass = computed(() => {
  if (noSwap.value || !ready.value) return "yantra-text-secondary";
  return swapping.value ? "text-rose-600 dark:text-rose-400" : "yantra-text-primary";
});
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-slate-200 bg-slate-50 transition-transform duration-200 group-hover:scale-105 dark:border-slate-900/50 dark:bg-slate-900/20">
          <ArrowDownUp class="h-5 w-5 text-slate-600 dark:text-slate-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black dark:group-hover:text-white">
            {{ t("quickMetrics.hostMetrics.vmActivity") }}
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
      <!-- vmstat unreadable: honest degradation, same box every other
           two-sample rate uses. -->
      <div
        v-if="isDegraded('vmstat')"
        class="flex items-center gap-3 rounded-xl border-red-200 bg-red-50 p-4 dark:border-red-900/50 dark:bg-red-900/20"
      >
        <ArrowDownUp class="h-5 w-5 shrink-0 text-red-500" />
        <div class="text-[11px] font-medium leading-tight text-red-600 dark:text-red-400">
          {{ unavailable }}
        </div>
      </div>

      <template v-else>
        <!-- Swap traffic out is the headline: memory being pushed to disk is
             the event. Rose on any traffic — a real 0 B/s is a healthy fact,
             so only movement goes hot. -->
        <div class="mb-2 flex items-baseline justify-between gap-3">
          <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
            {{ t("quickMetrics.hostMetrics.swapOut") }}
          </span>
          <span
            class="text-4xl font-black tabular-nums leading-[1.1] tracking-tighter transition-colors duration-200 group-hover:-translate-y-[3px] sm:text-5xl"
            :class="headlineClass"
          >
            {{ noSwap ? t("quickMetrics.hostMetrics.noSwap") : swapOutText }}
          </span>
        </div>

        <!-- Page state rows: point-in-time, valid from the first tick. -->
        <div class="mt-3 flex flex-col gap-2">
          <div
            v-if="!noSwap"
            class="flex items-center justify-between gap-3 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50"
          >
            <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
              <ArrowDownUp class="h-3 w-3 shrink-0 text-slate-500" />
              {{ t("quickMetrics.hostMetrics.swapIn") }}
            </span>
            <span
              class="text-sm font-black tabular-nums tracking-tight"
              :class="(vm?.swapInBps ?? 0) > 0 ? 'text-rose-600 dark:text-rose-400' : 'yantra-text-primary'"
            >
              {{ swapInText }}
            </span>
          </div>

          <div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
            <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
              <Database class="h-3 w-3 shrink-0 text-slate-500" />
              {{ t("quickMetrics.hostMetrics.vmDirty") }}
            </span>
            <span class="text-sm font-black tabular-nums tracking-tight yantra-text-primary">
              {{ formatBytes(vm?.dirtyBytes ?? 0) }}
            </span>
          </div>

          <div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
            <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
              <Database class="h-3 w-3 shrink-0 text-slate-500" />
              {{ t("quickMetrics.hostMetrics.vmWriteback") }}
            </span>
            <span class="text-sm font-black tabular-nums tracking-tight yantra-text-primary">
              {{ formatBytes(vm?.writebackBytes ?? 0) }}
            </span>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>
