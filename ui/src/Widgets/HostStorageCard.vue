<script>
// Dashboard engine: one column — the disk headline was tuned for a single
// cell; wider would only stretch the bar. Memory moved to its own card so the
// breakdown rows it gained would fit; this one stays the fill-up story for
// disks alone. See Home.vue for how meta.cols is consumed.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { HardDrive } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";

const { t } = useI18n();

const {
  metricsStale,
  isDegraded,
  primaryDisk,
  diskFreeText,
} = useHostMetrics();

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));
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

      <div class="mb-5 truncate text-[9px] font-semibold yantra-text-secondary">
        {{ isDegraded("disk") ? unavailable : (diskFreeText ? t("quickMetrics.hostMetrics.free", { size: diskFreeText }) : "") }}
      </div>
    </div>
  </div>
</template>
