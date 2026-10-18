<script>
// Dashboard engine: one column — the interface and its two rates read
// cleanly at a quarter of the grid, which is exactly the width the combined
// card gave them before the split. See Home.vue for how meta.cols is
// consumed.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Network, ArrowDown, ArrowUp } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";

const { t } = useI18n();

const {
  metricsStale,
  isDegraded,
  primaryNet,
  netText,
} = useHostMetrics();

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));

// Split the combined "rx · tx" string so each direction gets its own arrow and
// colour. Kept as two computeds rather than one array destructured once —
// destructuring at setup would freeze the first reading, since setup runs
// before any metric has arrived.
const rxText = computed(() => netText.value?.split("·")[0]?.trim() || "—");
const txText = computed(() => netText.value?.split("·")[1]?.trim() || "—");
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden rounded-2xl p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-sky-200 bg-sky-50 transition-transform duration-200 group-hover:scale-105 dark:border-sky-900/50 dark:bg-sky-900/20">
          <Network class="h-5 w-5 text-sky-600 dark:text-sky-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
            {{ t("quickMetrics.hostMetrics.network") }}
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
      <!-- The interface the numbers belong to. The composable picks the
           busiest one, so an idle docker0 never becomes the headline. -->
      <div class="mb-5 flex items-center gap-1.5 text-[9px] font-semibold uppercase tracking-widest yantra-text-secondary">
        <Network class="h-3 w-3 shrink-0" />
        <span class="truncate">{{ primaryNet?.iface ?? "—" }}</span>
      </div>

      <!-- rx is the headline: the volume a homelab pulls is the more telling
           of the two directions and the first thing you want to see move. -->
      <div class="mb-5 flex items-baseline justify-between gap-3">
        <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
          <ArrowDown class="h-3 w-3 shrink-0 text-sky-500" />
          rx
        </span>
        <span class="text-4xl font-black tabular-nums leading-[1.1] tracking-tighter yantra-text-primary transition-colors duration-200 group-hover:-translate-y-[3px] yantra-text-primary sm:text-5xl">
          {{ isDegraded("network") ? unavailable : rxText }}
        </span>
      </div>

      <!-- tx: the other direction, a row rather than a second headline. -->
      <div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
        <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
          <ArrowUp class="h-3 w-3 shrink-0 text-violet-500" />
          tx
        </span>
        <span class="truncate text-sm font-black tabular-nums tracking-tight yantra-text-primary">
          {{ isDegraded("network") ? unavailable : txText }}
        </span>
      </div>
    </div>
  </div>
</template>
