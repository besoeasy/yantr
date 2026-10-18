<script>
// Dashboard engine: two of the desktop grid's four columns — the disk
// headline plus the memory and network tiles each carry a reading, and one
// column squeezes all three into truncated labels. .w-span-2 in main.css does
// the responsive clamping; see Home.vue.
export const meta = { cols: 2 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { HardDrive, Network, ArrowDown, ArrowUp } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";

const { t } = useI18n();

const {
  metricsStale,
  isDegraded,
  memPct,
  memText,
  primaryNet,
  netText,
  primaryDisk,
  diskFreeText,
} = useHostMetrics();

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));

// Split the combined "rx · tx" string so each direction gets its own arrow and
// colour instead of two bare numbers in one row. Kept as two computeds rather
// than one array destructured once — destructuring at setup would freeze the
// first reading, since setup runs before any metric has arrived.
const rxText = computed(() => netText.value?.split("·")[0]?.trim() || "—");
const txText = computed(() => netText.value?.split("·")[1]?.trim() || "—");
</script>

<template>
  <div class="group relative flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-emerald-200 bg-emerald-50 transition-transform duration-200 group-hover:scale-105 dark:border-emerald-900/50 dark:bg-emerald-900/20">
          <HardDrive class="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors group-hover:text-black dark:text-white dark:group-hover:text-white">
            {{ t("quickMetrics.hostMetrics.storageNetwork") }}
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

    <div class="relative z-10 mt-auto flex flex-col pt-6 transition-opacity duration-200" :class="metricsStale ? 'opacity-50' : ''">
      <!-- Disk gets the headline slot: it is the metric that fills up and eventually
           stops an app from running. -->
      <div class="mb-1 flex items-baseline justify-between gap-3">
        <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.disk") }}</span>
        <span class="text-4xl font-black tabular-nums leading-[1.1] tracking-tighter text-zinc-900 transition-colors duration-200 group-hover:-translate-y-[3px] dark:text-white sm:text-5xl">
          {{ primaryDisk && !isDegraded("disk") ? `${Math.round(primaryDisk.usedPct)}%` : "—" }}
        </span>
      </div>

      <div class="mb-1 truncate text-[9px] font-semibold uppercase tracking-widest text-zinc-500">
        {{ primaryDisk?.mount ?? "" }}
      </div>

      <div v-if="!isDegraded('disk') && primaryDisk" class="mb-1 h-1.5 w-full overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
        <div
          class="h-full rounded-full bg-emerald-500 transition-[width] duration-200 dark:bg-emerald-400"
          :style="{ width: `${primaryDisk.usedPct}%` }"
        ></div>
      </div>

      <div class="mb-5 truncate text-[9px] font-semibold text-zinc-500">
        {{ isDegraded("disk") ? unavailable : (diskFreeText ? t("quickMetrics.hostMetrics.free", { size: diskFreeText }) : "") }}
      </div>

      <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
        <!-- Memory -->
        <div class="flex flex-col gap-1.5 rounded-xl border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
          <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.memory") }}</span>
          <span class="text-sm font-black tabular-nums tracking-tight text-zinc-900 dark:text-white">
            {{ isDegraded("memory") ? unavailable : (memPct !== null ? `${Math.round(memPct)}%` : "—") }}
          </span>
          <div class="h-1 w-full overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
            <div
              class="h-full rounded-full bg-emerald-500 transition-[width] duration-200 dark:bg-emerald-400"
              :style="{ width: `${memPct ?? 0}%` }"
            ></div>
          </div>
          <span class="truncate text-[9px] font-semibold tabular-nums text-zinc-500">{{ memText ?? "" }}</span>
        </div>

        <!-- Network, one row per direction -->
        <div class="flex flex-col gap-1.5 rounded-xl border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
          <div class="flex items-center gap-1.5 text-zinc-500">
            <Network class="h-3 w-3 shrink-0" />
            <span class="truncate text-[9px] font-semibold uppercase tracking-widest">{{ primaryNet?.iface ?? "" }}</span>
          </div>
          <div class="flex items-center gap-1.5">
            <ArrowDown class="h-3 w-3 shrink-0 text-sky-500" />
            <span class="truncate text-sm font-black tabular-nums tracking-tight text-zinc-900 dark:text-white">
              {{ isDegraded("network") ? unavailable : rxText }}
            </span>
          </div>
          <div class="flex items-center gap-1.5">
            <ArrowUp class="h-3 w-3 shrink-0 text-violet-500" />
            <span class="truncate text-sm font-black tabular-nums tracking-tight text-zinc-900 dark:text-white">
              {{ isDegraded("network") ? unavailable : txText }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>