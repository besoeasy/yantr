<script>
// Dashboard engine: one column — the CPU headline, sparkline and the load
// tile were designed around a single cell and stay legible there. Uptime and
// temperature used to ride along as tiles in a three-wide grid; they were
// pulled out into HostEnvironmentCard, which is why this card no longer
// imports Timer or Thermometer and the grid below holds a single row. See
// Home.vue for how meta.cols is consumed.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Activity } from "@lucide/vue";
import Sparkline from "../components/Sparkline.vue";
import { useHostMetrics } from "../composables/useHostMetrics";

const { t } = useI18n();

const {
  metricsStale,
  isDegraded,
  cpuPct,
  coreBars,
  cpuHistory,
  loadText,
} = useHostMetrics();

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-violet-200 bg-violet-50 transition-transform duration-200 group-hover:scale-105 dark:border-violet-900/50 dark:bg-violet-900/20">
          <Activity class="h-5 w-5 text-violet-600 dark:text-violet-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
            {{ t("quickMetrics.hostMetrics.performance") }}
          </h3>
          <div class="mt-2 flex items-center gap-2">
            <span class="relative flex h-2 w-2">
              <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
              <span class="relative inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
            </span>
            <span class="text-[10px] font-semibold uppercase tracking-widest text-emerald-600 dark:text-emerald-500">{{ t("quickMetrics.hostMetrics.live") }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Whole body dims while the metrics poll is failing, so a stale reading
         never looks live. -->
    <div class="relative z-10 mt-auto flex flex-col pt-6 transition-opacity duration-200" :class="metricsStale ? 'opacity-50' : ''">
      <!-- CPU: the headline number gets the full card width because it is the
           one metric here that is a rate, not a static total. -->
      <div class="mb-2 flex items-baseline justify-between gap-3">
        <span class="text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">{{ t("quickMetrics.hostMetrics.cpu") }}</span>
        <span class="text-4xl font-black tabular-nums leading-[1.1] tracking-tighter yantra-text-primary transition-colors duration-200 group-hover:-translate-y-[3px] yantra-text-primary sm:text-5xl">
          {{ cpuPct !== null ? `${cpuPct}%` : "—" }}
        </span>
      </div>

      <div class="mb-2 h-16 w-full">
        <Sparkline :points="cpuHistory" :max="100" :tone="'violet'" :label="t('quickMetrics.hostMetrics.cpuHistory')" />
      </div>

      <div v-if="coreBars.length" class="mb-5 flex h-5 items-end gap-0.5" aria-hidden="true">
        <div
          v-for="(core, i) in coreBars"
          :key="i"
          class="flex-1 rounded-t-sm bg-violet-500/70 dark:bg-violet-400/70"
          :style="{ height: `${Math.max(core, 2)}%` }"
        ></div>
      </div>
      <div v-else class="mb-5"></div>

      <div class="grid grid-cols-1 gap-2">
        <!-- Load average: 1 / 5 / 15 minute. Full row now: with uptime and
             temperature pulled out, the old three-wide grid would leave two
             empty cells behind. -->
        <div class="flex flex-col gap-1 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
          <div class="flex items-center gap-1.5 yantra-text-secondary">
            <Activity class="h-3 w-3" />
            <span class="text-[9px] font-bold uppercase tracking-widest">{{ t("quickMetrics.hostMetrics.load") }}</span>
          </div>
          <span class="text-sm font-black tabular-nums tracking-tight yantra-text-primary">
            {{ isDegraded("load") ? unavailable : (loadText ?? "—") }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>