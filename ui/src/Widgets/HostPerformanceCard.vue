<script>
// Dashboard engine: this card takes two of the desktop grid's three columns —
// the sparkline, the per-core bars and the load/uptime/temperature tiles give
// each other breathing room, which a single column does not. The .w-span-2
// utility in main.css clamps it per breakpoint; see Home.vue.
export const meta = { cols: 2 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Activity, Thermometer, Timer } from "@lucide/vue";
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
  uptimeText,
  tempText,
} = useHostMetrics();

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));
</script>

<template>
  <div class="group relative flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-violet-200 bg-violet-50 transition-transform duration-200 group-hover:scale-105 dark:border-violet-900/50 dark:bg-violet-900/20">
          <Activity class="h-5 w-5 text-violet-600 dark:text-violet-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors group-hover:text-black dark:text-white dark:group-hover:text-white">
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
        <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.cpu") }}</span>
        <span class="text-4xl font-black tabular-nums leading-[1.1] tracking-tighter text-zinc-900 transition-colors duration-200 group-hover:-translate-y-[3px] dark:text-white sm:text-5xl">
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

      <div class="grid grid-cols-1 gap-2 sm:grid-cols-3">
        <!-- Load average: 1 / 5 / 15 minute -->
        <div class="flex flex-col gap-1 rounded-xl border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
          <div class="flex items-center gap-1.5 text-zinc-500">
            <Activity class="h-3 w-3" />
            <span class="text-[9px] font-bold uppercase tracking-widest">{{ t("quickMetrics.hostMetrics.load") }}</span>
          </div>
          <span class="text-sm font-black tabular-nums tracking-tight text-zinc-900 dark:text-white">
            {{ isDegraded("load") ? unavailable : (loadText ?? "—") }}
          </span>
        </div>

        <!-- Uptime -->
        <div class="flex flex-col gap-1 rounded-xl border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
          <div class="flex items-center gap-1.5 text-zinc-500">
            <Timer class="h-3 w-3" />
            <span class="text-[9px] font-bold uppercase tracking-widest">{{ t("quickMetrics.hostMetrics.uptime") }}</span>
          </div>
          <span class="truncate text-sm font-black tracking-tight text-zinc-900 dark:text-white">{{ uptimeText ?? "—" }}</span>
        </div>

        <!-- Temperature: the usual reason to open this card on a homelab box. -->
        <div class="flex flex-col gap-1 rounded-xl border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
          <div class="flex items-center gap-1.5 text-zinc-500">
            <Thermometer class="h-3 w-3" />
            <span class="text-[9px] font-bold uppercase tracking-widest">{{ t("quickMetrics.hostMetrics.temperature") }}</span>
          </div>
          <span class="text-sm font-black tabular-nums tracking-tight text-zinc-900 dark:text-white">{{ tempText ?? unavailable }}</span>
        </div>
      </div>
    </div>
  </div>
</template>