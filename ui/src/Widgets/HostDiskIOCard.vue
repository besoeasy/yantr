<script>
// Dashboard engine: one column — the traffic story for the top device, split
// back out of Storage: how full a disk is and how hard it is working are two
// questions on two rhythms, and one tile answered neither cleanly. Shares the
// storage family's emerald on purpose: same device, two questions. See
// Home.vue for how meta.cols is consumed.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { ArrowDownUp, ArrowDown, ArrowUp, Gauge } from "@lucide/vue";
import Sparkline from "../components/Sparkline.vue";
import { useHostMetrics } from "../composables/useHostMetrics";

const { t } = useI18n();

const {
  metricsStale,
  isDegraded,
  primaryDiskIO,
  diskIOText,
  diskIOHistory,
} = useHostMetrics();

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));

// Device saturation: share of the last window the disk spent busy. Zero for
// the first tick (no baseline yet), and 90%+ is the full-pipe state worth a
// rose tint — the byte rows cannot distinguish that from a quiet disk.
const utilPct = computed(() => {
  const util = primaryDiskIO.value?.utilPct;
  return Number.isFinite(util) ? util : 0;
});

// Split the combined "read · write" string so each direction gets its own
// arrow and colour — the same split the network card makes on rx · tx, and
// for the same reason: destructuring at setup would freeze the first reading.
const readText = computed(() => diskIOText.value?.split("·")[0]?.trim() || "—");
const writeText = computed(() => diskIOText.value?.split("·")[1]?.trim() || "—");

// The small-IO story the byte rows hide. A device moving one MB in 4k chunks
// and one moving a GB in one block read the same on the rows and mean very
// different things to the disk.
const iopsText = computed(() => {
  const io = primaryDiskIO.value;
  if (!io) return "—";
  const fmt = (v) => (v >= 1000 ? `${(v / 1000).toFixed(1)}k` : Math.round(v).toString());
  return `${fmt(io.readsPerSec)} r/s · ${fmt(io.writesPerSec)} w/s`;
});

// The sparkline scales itself: byte rates have no natural ceiling to pin to
// 100 like a CPU percentage does, so the max is the busiest moment in the
// window — the shape is the information, the absolute scale lives in the
// read/write rows. The 1 MB/s floor keeps an all-zero window from producing a
// 0 ceiling, which would divide the next real sample into a spike to the top.
const sparkMax = computed(() => Math.max(...diskIOHistory.value, 1024 * 1024) * 1.15);
const sparkLabel = computed(() => t("quickMetrics.hostMetrics.diskIOHistory"));
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-emerald-200 bg-emerald-50 transition-transform duration-200 group-hover:scale-105 dark:border-emerald-900/50 dark:bg-emerald-900/20">
          <ArrowDownUp class="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black dark:group-hover:text-white">
            {{ t("quickMetrics.hostMetrics.diskIO") }}
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
      <!-- Degraded: the counters are missing, so say so plainly. Capacity
           keeps reporting on the Storage card regardless. -->
      <div
        v-if="isDegraded('diskio')"
        class="flex items-center gap-3 rounded-xl border-red-200 bg-red-50 p-4 dark:border-red-900/50 dark:bg-red-900/20"
      >
        <ArrowDownUp class="h-5 w-5 shrink-0 text-red-500" />
        <div class="text-[11px] font-medium leading-tight text-red-600 dark:text-red-400">
          {{ unavailable }}
        </div>
      </div>

      <template v-else>
        <!-- Which device, and its small-IO fingerprint: the row names the
             subject, the IOPS right side is the story the bytes hide. -->
        <div class="mb-2 flex items-center gap-1.5 text-[9px] font-semibold uppercase tracking-widest yantra-text-secondary">
          <ArrowDownUp class="h-3 w-3 shrink-0" />
          <span class="truncate">{{ primaryDiskIO?.device ?? "—" }}</span>
          <span class="ml-auto shrink-0 tabular-nums">
            {{ iopsText }}
          </span>
        </div>

        <!-- 4-minute trend of the top device, read+write combined, autoscaled
             to the busiest moment in the window — the same shape-over-scale
             deal the CPU sparkline makes, minus the fixed percent ceiling. -->
        <div class="mb-3 h-16 w-full">
          <Sparkline :points="diskIOHistory" :max="sparkMax" tone="emerald" :label="sparkLabel" />
        </div>

        <!-- read / write: two rows rather than a second headline. -->
        <div class="flex flex-col gap-2">
          <div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
            <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
              <ArrowDown class="h-3 w-3 shrink-0 text-amber-500" />
              {{ t("quickMetrics.hostMetrics.diskRead") }}
            </span>
            <span class="truncate text-sm font-black tabular-nums tracking-tight yantra-text-primary">
              {{ readText }}
            </span>
          </div>

          <div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
            <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
              <ArrowUp class="h-3 w-3 shrink-0 text-violet-500" />
              {{ t("quickMetrics.hostMetrics.diskWrite") }}
            </span>
            <span class="truncate text-sm font-black tabular-nums tracking-tight yantra-text-primary">
              {{ writeText }}
            </span>
          </div>

          <!-- Saturation: the share of the window the device was busy. The
               honest full-pipe signal — 30 MB/s of small random IOPS saturates
               a spinning disk while it barely registers on the byte rows
               above. First tick reads 0 (no window yet), like every rate. -->
          <div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
            <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
              <Gauge class="h-3 w-3 shrink-0 text-emerald-500" />
              {{ t("quickMetrics.hostMetrics.diskUtil") }}
            </span>
            <span class="flex items-center gap-2">
              <span class="h-1 w-14 shrink-0 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
                <span
                  class="block h-full rounded-full transition-[width] duration-200"
                  :class="utilPct >= 90 ? 'bg-rose-500' : 'bg-emerald-500'"
                  :style="{ width: `${utilPct}%` }"
                ></span>
              </span>
              <span
                class="w-10 shrink-0 text-right text-sm font-black tabular-nums tracking-tight"
                :class="utilPct >= 90 ? 'text-rose-600 dark:text-rose-400' : 'yantra-text-primary'"
              >
                {{ Math.round(utilPct) }}%
              </span>
            </span>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>
