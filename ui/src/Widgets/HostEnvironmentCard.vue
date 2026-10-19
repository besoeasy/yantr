<script>
// Dashboard engine: one column — temperature as the live headline with
// uptime and fan speed as rows beneath it. Uptime barely moves and fan speed
// is one short number each; widening the card would only stretch strings.
// Split out of HostPerformanceCard, where all three were cramped tiles. See
// Home.vue for how meta.cols is consumed.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Timer, Thermometer, Fan } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";

const { t } = useI18n();

const {
  metricsStale,
  isDegraded,
  uptimeText,
  tempText,
  fanRPM,
} = useHostMetrics();

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));

// Temperature is the card's headline: it is the one live number here and the
// usual reason to look on a homelab box — uptime barely moves. The value came
// out of readTemperature filtered through the plausibility window, so a dead
// sensor renders the unavailable state instead of a convincing zero.
const headline = computed(() => (isDegraded("temperature") ? unavailable : (tempText.value ?? unavailable)));
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-indigo-200 bg-indigo-50 transition-transform duration-200 group-hover:scale-105 dark:border-indigo-900/50 dark:bg-indigo-900/20">
          <Thermometer class="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
            {{ t("quickMetrics.hostMetrics.environment") }}
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
      <!-- Temperature headline: a live reading first, identity second. -->
      <div class="mb-5 flex items-baseline justify-between gap-3">
        <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
          <Thermometer class="h-3 w-3 shrink-0 text-indigo-600" />
          {{ t("quickMetrics.hostMetrics.temperature") }}
        </span>
        <span class="text-4xl font-black tabular-nums leading-[1.1] tracking-tighter yantra-text-primary transition-colors duration-200 group-hover:-translate-y-[3px] yantra-text-primary sm:text-5xl">
          {{ headline }}
        </span>
      </div>

      <!-- Uptime: the slow counter, a row rather than a second headline. -->
      <div class="mb-2 flex items-center justify-between gap-3 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
        <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
          <Timer class="h-3 w-3 shrink-0 text-slate-500" />
          {{ t("quickMetrics.hostMetrics.uptime") }}
        </span>
        <span class="text-sm font-black tracking-tight yantra-text-primary">{{ uptimeText ?? "—" }}</span>
      </div>

      <!-- Fans: the physical response to the temperature above. A card that
           shows heat but not what answers it stops half the story. -->
      <div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
        <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
          <Fan class="h-3 w-3 shrink-0 text-sky-500" />
          {{ t("quickMetrics.hostMetrics.fans") }}
        </span>
        <span class="text-sm font-black tabular-nums tracking-tight yantra-text-primary">
          {{ isDegraded("fans") ? unavailable : (fanText ?? "—") }}
        </span>
      </div>
    </div>
  </div>
</template>
