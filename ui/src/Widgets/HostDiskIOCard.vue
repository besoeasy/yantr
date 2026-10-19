<script>
// Dashboard engine: one column — the read headline and the write row are the
// same two-row shape the network card reads, so the fourth host tile scans
// exactly like its siblings. See Home.vue for how meta.cols is consumed.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { ArrowDownUp, ArrowDown, ArrowUp } from "@lucide/vue";
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

// Split the combined "read · write" string so each direction gets its own
// arrow and colour — the same split HostNetworkCard makes on rx · tx, and for
// the same reason: destructuring at setup would freeze the first reading.
const readText = computed(() => diskIOText.value?.split("·")[0]?.trim() || "—");
const writeText = computed(() => diskIOText.value?.split("·")[1]?.trim() || "—");

// Candidates for a disk: mid-window devices (and degraded counters, and the
// whole array on the first sample) are excluded server-side, so an empty array
// here means only "the rates have not arrived yet" or "the host has no disk
// counters" — both render as the em-dash state rather than fake zeros.
const iopsText = computed(() => {
  const io = primaryDiskIO.value;
  if (!io) return "—";
  const fmt = (v) => (v >= 1000 ? `${(v / 1000).toFixed(1)}k` : Math.round(v).toString());
  return `${fmt(io.readsPerSec)} r/s · ${fmt(io.writesPerSec)} w/s`;
});

// The sparkline scales itself: byte rates have no natural ceiling to pin to 100
// like a CPU percentage does, so the max is the busiest moment in the window —
// the shape is the information, the absolute scale lives in the headline above.
// The 1 MB/s floor keeps an all-zero window from producing a 0 ceiling, which
// would divide the next real sample into a spike to the top of the card.
const sparkMax = computed(() => Math.max(...diskIOHistory.value, 1024 * 1024) * 1.15);
const sparkLabel = computed(() => t("quickMetrics.hostMetrics.diskIOHistory"));
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-amber-200 bg-amber-50 transition-transform duration-200 group-hover:scale-105 dark:border-amber-900/50 dark:bg-amber-900/20">
          <ArrowDownUp class="h-5 w-5 text-amber-600 dark:text-amber-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
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
      <!-- The device the numbers belong to. Like the network card, the
           composable takes the busiest entry and the backend has already
           filtered partitions and loop noise out. -->
      <div class="mb-5 flex items-center gap-1.5 text-[9px] font-semibold uppercase tracking-widest yantra-text-secondary">
        <ArrowDownUp class="h-3 w-3 shrink-0" />
        <span class="truncate">{{ primaryDiskIO?.device ?? "—" }}</span>
      </div>

      <!-- read is the headline: database and torrent workloads show here
           before writes do, and it is the first number you want to see move. -->
      <div class="mb-2 flex items-baseline justify-between gap-3">
        <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
          <ArrowDown class="h-3 w-3 shrink-0 text-amber-500" />
          {{ t("quickMetrics.hostMetrics.diskRead") }}
        </span>
        <span class="text-4xl font-black tabular-nums leading-[1.1] tracking-tighter yantra-text-primary transition-colors duration-200 group-hover:-translate-y-[3px] yantra-text-primary sm:text-5xl">
          {{ isDegraded("diskio") ? unavailable : readText }}
        </span>
      </div>

      <!-- 4-minute trend of the top device, read+write combined, autoscaled
           to the busiest moment in the window — the same shape-over-scale deal
           the CPU sparkline makes, minus the fixed percent ceiling. -->
      <div class="mb-5 h-16 w-full">
        <Sparkline :points="diskIOHistory" :max="sparkMax" :tone="'amber'" :label="sparkLabel" />
      </div>

      <!-- write: the other direction, a row rather than a second headline. -->
      <div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
        <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
          <ArrowUp class="h-3 w-3 shrink-0 text-violet-500" />
          {{ t("quickMetrics.hostMetrics.diskWrite") }}
        </span>
        <span class="truncate text-sm font-black tabular-nums tracking-tight yantra-text-primary">
          {{ isDegraded("diskio") ? unavailable : writeText }}
        </span>
      </div>

      <!-- IOPS: the small-IO story the byte headline hides. A device moving
           one MB in 4k chunks and one moving a GB in one block mean the same
           thing here and very different things to the disk. -->
      <div class="mt-3 text-right text-[10px] font-semibold uppercase tracking-widest yantra-text-secondary tabular-nums">
        {{ isDegraded("diskio") ? unavailable : iopsText }}
      </div>
    </div>
  </div>
</template>
