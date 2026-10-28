<script>
// Dashboard engine: one column — the fill-up story and the traffic story for
// the same disk now share one tile, the way the network card absorbed Link
// Status: size and speed answer each other, and on their own each was half a
// card. The disk percentage keeps the headline slot: it is the metric that
// fills up and eventually stops an app from running. Absorbed
// HostDiskIOCard, whose trend line had been silently missing — the template
// named Sparkline without ever importing it. See Home.vue for how meta.cols
// is consumed.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { HardDrive, ArrowDownUp, ArrowDown, ArrowUp, Gauge } from "@lucide/vue";
import Sparkline from "../components/Sparkline.vue";
import { useHostMetrics } from "../composables/useHostMetrics";

const { t } = useI18n();

const {
  metricsStale,
  isDegraded,
  primaryDisk,
  diskFreeText,
  disks,
  primaryDiskIO,
  diskIOText,
  diskIOHistory,
} = useHostMetrics();

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));

// The filesystems beyond the headline one, root-first already sorted by the
// backend. Every mount the host's real mounts table reports lands here — the
// old hardcoded candidate list could only ever see "/" plus four conventional
// paths, so a dedicated data disk wore invisibility as its failure mode.
const otherDisks = computed(() => disks.value.slice(1));

// Device saturation: share of the last window the disk spent busy. Zero for
// the first tick (no baseline yet), and 90%+ is the full-pipe state worth a
// rose tint — the byte rows above cannot distinguish that from a quiet disk.
const utilPct = computed(() => {
  const util = primaryDiskIO.value?.utilPct;
  return Number.isFinite(util) ? util : 0;
});

// Split the combined "read · write" string so each direction gets its own
// arrow and colour — the same split the network card makes on rx · tx, and
// for the same reason: destructuring at setup would freeze the first reading.
const readText = computed(() => diskIOText.value?.split("·")[0]?.trim() || "—");
const writeText = computed(() => diskIOText.value?.split("·")[1]?.trim() || "—");

// The small-IO story the byte headline hides. A device moving one MB in 4k
// chunks and one moving a GB in one block read the same on the headline and
// mean very different things to the disk.
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
          <HardDrive class="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
            {{ t("quickMetrics.hostMetrics.storage") }}
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
      <!-- Disk gets the headline slot: it is the metric that fills up and
           eventually stops an app from running. -->
      <div class="mb-1 flex items-baseline justify-between gap-3">
        <span class="text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">{{ t("quickMetrics.hostMetrics.disk") }}</span>
        <span class="text-4xl font-black tabular-nums leading-[1.1] tracking-tighter yantra-text-primary transition-colors duration-200 group-hover:-translate-y-[3px] yantra-text-primary sm:text-5xl">
          {{ primaryDisk && !isDegraded("disk") ? `${Math.round(primaryDisk.usedPct)}%` : "—" }}
        </span>
      </div>

      <div class="mb-1 truncate text-[9px] font-semibold uppercase tracking-widest yantra-text-secondary">
        {{ primaryDisk?.mount ?? "" }}
      </div>

      <div v-if="!isDegraded('disk') && primaryDisk" class="mb-1 h-1.5 w-full overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
        <div
          class="h-full rounded-full bg-emerald-500 transition-[width] duration-200 dark:bg-emerald-400"
          :style="{ width: `${primaryDisk.usedPct}%` }"
        ></div>
      </div>

      <div class="mb-4 truncate text-[9px] font-semibold yantra-text-secondary">
        {{ isDegraded("disk") ? unavailable : (diskFreeText ? t("quickMetrics.hostMetrics.free", { size: diskFreeText }) : "") }}
      </div>

      <!-- The same disk's traffic, under a hairline. Degrade independently:
           a disk whose IO counters are missing still reports its capacity. -->
      <div class="border-t border-zinc-200 pt-3 dark:border-zinc-800">
        <div class="mb-2 flex items-center gap-1.5 text-[9px] font-semibold uppercase tracking-widest yantra-text-secondary">
          <ArrowDownUp class="h-3 w-3 shrink-0" />
          <span class="truncate">{{ primaryDiskIO?.device ?? "—" }}</span>
          <span class="ml-auto shrink-0 tabular-nums">
            {{ isDegraded("diskio") ? unavailable : iopsText }}
          </span>
        </div>

        <!-- 4-minute trend of the top device, read+write combined, autoscaled
             to the busiest moment in the window — the same shape-over-scale
             deal the CPU sparkline makes, minus the fixed percent ceiling. -->
        <div class="mb-3 h-16 w-full">
          <Sparkline :points="diskIOHistory" :max="sparkMax" :tone="'amber'" :label="sparkLabel" />
        </div>

        <!-- read / write: two rows rather than a second headline. The same
             two arrows the network card uses for rx / tx, amber here so the
             disk's accent stays distinct from the network's sky. -->
        <div class="flex flex-col gap-2">
          <div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
            <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
              <ArrowDown class="h-3 w-3 shrink-0 text-amber-500" />
              {{ t("quickMetrics.hostMetrics.diskRead") }}
            </span>
            <span class="truncate text-sm font-black tabular-nums tracking-tight yantra-text-primary">
              {{ isDegraded("diskio") ? unavailable : readText }}
            </span>
          </div>

          <div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
            <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
              <ArrowUp class="h-3 w-3 shrink-0 text-violet-500" />
              {{ t("quickMetrics.hostMetrics.diskWrite") }}
            </span>
            <span class="truncate text-sm font-black tabular-nums tracking-tight yantra-text-primary">
              {{ isDegraded("diskio") ? unavailable : writeText }}
            </span>
          </div>

          <!-- Saturation: the share of the window the device was busy. The
               honest full-pipe signal — 30 MB/s of small random IOPS saturates
               a spinning disk while it barely registers on this card's byte
               rows. First tick reads 0 (no window yet), like every rate. -->
          <div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
            <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
              <Gauge class="h-3 w-3 shrink-0 text-amber-500" />
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
      </div>

      <!-- The rest of the host's filesystems: mount, its own fill bar and its
           percentage. A dedicated data disk that runs full is exactly as much
           of an event as the root filesystem running full. -->
      <div v-if="otherDisks.length > 0 && !isDegraded('disk')" class="mt-4 border-t border-zinc-200 pt-3 dark:border-zinc-800">
        <div class="mb-2 text-[10px] font-semibold uppercase tracking-widest yantra-text-secondary">
          {{ t("quickMetrics.hostMetrics.otherDisks") }}
        </div>
        <div class="flex flex-col gap-2">
          <div
            v-for="d in otherDisks"
            :key="d.mount"
            class="flex items-center gap-2 rounded-xl bg-zinc-50 p-2 dark:bg-zinc-900/50"
          >
            <span class="min-w-0 flex-1 truncate text-[11px] font-bold tracking-tight yantra-text-primary" :title="d.mount">{{ d.mount }}</span>
            <span class="shrink-0 text-[9px] font-semibold yantra-text-secondary">{{ d.fsType }}</span>
            <span class="h-1 w-14 shrink-0 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
              <span class="block h-full rounded-full bg-emerald-500" :style="{ width: `${d.usedPct}%` }"></span>
            </span>
            <span class="w-10 shrink-0 text-right text-[9px] font-black tabular-nums tracking-tight yantra-text-primary">
              {{ Math.round(d.usedPct) }}%
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
