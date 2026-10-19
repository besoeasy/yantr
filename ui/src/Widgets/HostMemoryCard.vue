<script>
// Dashboard engine: one column — the memory bar and its four where-the-bytes-
// went rows stack into a single readable rail; wider would only stretch the
// bar. Split out of HostStorageCard, which keeps the fill-up story for disks
// alone. See Home.vue for how meta.cols is consumed.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { MemoryStick } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";

const { t } = useI18n();

const {
  metricsStale,
  isDegraded,
  memPct,
  memText,
  swapText,
  cachedText,
  buffersText,
  slabText,
} = useHostMetrics();

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));

// Swap answers the question the other rows don't: is the box pressed against
// its backstop? A machine that never configured swap is telling the truth, so
// that renders a word — an em-dash would claim the reading was missing, and
// 0 / 0 would claim a sensor that does not exist.
const swapValue = computed(() => {
  if (isDegraded("memory")) return unavailable;
  return swapText.value ?? t("quickMetrics.hostMetrics.noSwap");
});

// The non-working-set pages the kernel holds: reclaimable in a pinch, and the
// reason a near-full bar is usually fine. Each row degrades to an em-dash —
// "unknown" — because a breakdown row reading 0 could just mean an old kernel
// line was absent, and a fake zero here would look like reclaimed memory.
const detailRows = computed(() => [
  { key: "swap", label: t("quickMetrics.hostMetrics.memorySwap"), value: swapValue.value },
  { key: "cached", label: t("quickMetrics.hostMetrics.memoryCached"), value: isDegraded("memory") ? "—" : (cachedText.value ?? "—") },
  { key: "buffers", label: t("quickMetrics.hostMetrics.memoryBuffers"), value: isDegraded("memory") ? "—" : (buffersText.value ?? "—") },
  { key: "slab", label: t("quickMetrics.hostMetrics.memorySlab"), value: isDegraded("memory") ? "—" : (slabText.value ?? "—") },
]);
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-rose-200 bg-rose-50 transition-transform duration-200 group-hover:scale-105 dark:border-rose-900/50 dark:bg-rose-900/20">
          <MemoryStick class="h-5 w-5 text-rose-600 dark:text-rose-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
            {{ t("quickMetrics.hostMetrics.memory") }}
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
      <!-- Headline: the same number the storage card carried when memory lived
           there, so the move does not change what a glance takes in. -->
      <div class="mb-1 flex items-baseline justify-between gap-3">
        <span class="text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">{{ t("quickMetrics.hostMetrics.memory") }}</span>
        <span class="text-4xl font-black tabular-nums leading-[1.1] tracking-tighter yantra-text-primary transition-colors duration-200 group-hover:-translate-y-[3px] yantra-text-primary sm:text-5xl">
          {{ !isDegraded("memory") && memPct !== null ? `${Math.round(memPct)}%` : "—" }}
        </span>
      </div>

      <div v-if="!isDegraded('memory') && memPct !== null" class="mb-1 h-1.5 w-full overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
        <div
          class="h-full rounded-full bg-rose-500 transition-[width] duration-200 dark:bg-rose-400"
          :style="{ width: `${memPct}%` }"
        ></div>
      </div>

      <div class="mb-5 truncate text-[9px] font-semibold tabular-nums yantra-text-secondary">
        {{ isDegraded("memory") ? "" : (memText ?? "") }}
      </div>

      <!-- Breakdown rows: swap first, because it is the one that says the box
           is truly out of room; cache, buffers and slab follow as the "relaxed,
           not really used" half of the picture. -->
      <div
        v-for="(row, i) in detailRows"
        :key="row.key"
        class="flex items-baseline justify-between gap-2 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50"
        :class="i < detailRows.length - 1 ? 'mb-2' : ''"
      >
        <span class="text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
          {{ row.label }}
        </span>
        <span class="truncate text-sm font-black tabular-nums tracking-tight yantra-text-primary">
          {{ row.value }}
        </span>
      </div>
    </div>
  </div>
</template>
