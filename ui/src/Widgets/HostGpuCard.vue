<script>
// Dashboard engine: one column — utilisation as the headline, VRAM as the bar
// and temperature as a row, the same one-number-plus-context shape every host
// tile reads as. Only what a rootless container can read makes it here:
// amdgpu ships gpu_busy_percent and the VRAM pair; Intel and proprietary
// NVIDIA drivers expose none of them, so an absent card is a normal machine
// answered honestly, not a failure. See Home.vue for how meta.cols is
// consumed.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Gpu, Thermometer } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";
import { formatBytes } from "../utils/metrics";

const { t } = useI18n();

const { metricsStale, isDegraded, primaryGpu } = useHostMetrics();

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));

// The driver's own utilisation estimate. A real 0 is an honest idle reading —
// the backend only nulls it when the file itself is missing — so the headline
// prints 0% on an idle GPU rather than a dash that would claim uncertainty.
const busyText = computed(() => {
  const busy = primaryGpu.value?.busyPct;
  return Number.isFinite(busy) ? `${Math.round(busy)}%` : null;
});

// The card and the driver it came from: the one line that survives a bug
// report ("amdgpu, card0") without a device node a container cannot honour.
const captionText = computed(() => {
  const g = primaryGpu.value;
  if (!g) return null;
  return [g.card, g.driver].filter(Boolean).join(" · ") || null;
});

// VRAM: the same story the memory card tells, one device down. The bar rides
// on the driver's own used/total files; a driver that ships none of them hides
// the bar instead of drawing a believable 0.
const vramPct = computed(() => {
  const g = primaryGpu.value;
  if (!g?.vramTotalBytes || g.vramTotalBytes < g.vramUsedBytes) return 0;
  return (g.vramUsedBytes / g.vramTotalBytes) * 100;
});

const vramText = computed(() => {
  const g = primaryGpu.value;
  if (!g?.vramTotalBytes) return null;
  return `${formatBytes(g.vramUsedBytes)} / ${formatBytes(g.vramTotalBytes)}`;
});

// Heat relative to the chip's own trip when the driver ships one, 80°C as the
// stand-alone baseline when it does not — the same rule the Environment card
// learned to use, so both cards agree on what "hot" means.
const isHot = computed(() => {
  const g = primaryGpu.value;
  const temp = g?.tempC;
  if (!Number.isFinite(temp)) return false;
  const tripC = Number.isFinite(g?.tripC) ? g.tripC : null;
  return tripC ? temp >= tripC * 0.85 : temp >= 80;
});
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-amber-200 bg-amber-50 transition-transform duration-200 group-hover:scale-105 dark:border-amber-900/50 dark:bg-amber-900/20">
          <Gpu class="h-5 w-5 text-amber-600 dark:text-amber-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
            {{ t("quickMetrics.hostMetrics.gpu") }}
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
      <!-- Missing sysfs bind: the machine may have a GPU the poller cannot
           see at all, which is a degradation, not a normal state. -->
      <div
        v-if="isDegraded('gpu')"
        class="flex items-center gap-3 rounded-xl border-red-200 bg-red-50 p-4 dark:border-red-900/50 dark:bg-red-900/20"
      >
        <Gpu class="h-5 w-5 shrink-0 text-red-500" />
        <div class="text-[11px] font-medium leading-tight text-red-600 dark:text-red-400">
          {{ unavailable }}
        </div>
      </div>

      <!-- No card exposes these files: the normal Intel-iGPU and headless
           case. Neutral chrome on purpose — nothing here is broken. -->
      <div
        v-else-if="!primaryGpu"
        class="rounded-xl bg-zinc-50 p-4 text-[11px] font-medium leading-tight yantra-text-secondary"
      >
        {{ t("quickMetrics.hostMetrics.gpuNone") }}
      </div>

      <template v-else>
        <!-- Utilisation is the headline: it is the one live rate on the card. -->
        <div class="mb-2 flex items-baseline justify-between gap-3">
          <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
            {{ t("quickMetrics.hostMetrics.gpuBusy") }}
          </span>
          <span class="text-4xl font-black tabular-nums leading-[1.1] tracking-tighter yantra-text-primary transition-colors duration-200 group-hover:-translate-y-[3px] yantra-text-primary sm:text-5xl">
            {{ busyText ?? "—" }}
          </span>
        </div>

        <div class="mb-3 truncate text-[9px] font-semibold uppercase tracking-widest yantra-text-secondary">
          {{ captionText ?? "" }}
        </div>

        <!-- VRAM, whenever the driver ships used/total. -->
        <template v-if="vramText">
          <div class="mb-1 flex items-baseline justify-between gap-3">
            <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
              {{ t("quickMetrics.hostMetrics.gpuVram") }}
            </span>
            <span class="text-[9px] font-semibold tabular-nums yantra-text-secondary">{{ vramText }}</span>
          </div>
          <div class="mb-3 h-2 w-full overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
            <div
              class="h-full rounded-full bg-amber-500 transition-[width] duration-200"
              :style="{ width: `${vramPct}%` }"
            ></div>
          </div>
        </template>

        <!-- Temperature: context, not a second headline, tripped rose against
             the chip's own threshold when the driver ships one. -->
        <div
          v-if="Number.isFinite(primaryGpu.tempC)"
          class="flex items-center justify-between gap-3 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50"
          :title="Number.isFinite(primaryGpu.tripC) ? `${t('quickMetrics.hostMetrics.temperature')} · ${t('quickMetrics.hostMetrics.memoryUsed')} ${Math.round(primaryGpu.tripC)}°C` : undefined"
        >
          <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
            <Thermometer class="h-3 w-3 shrink-0 text-amber-500" />
            {{ t("quickMetrics.hostMetrics.temperature") }}
          </span>
          <span
            class="text-sm font-black tabular-nums tracking-tight"
            :class="isHot ? 'text-rose-600 dark:text-rose-400' : 'yantra-text-primary'"
          >
            {{ Math.round(primaryGpu.tempC) }}°C
          </span>
        </div>
      </template>
    </div>
  </div>
</template>
