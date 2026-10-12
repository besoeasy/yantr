<script setup>
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { Server, Boxes, Database, Image, Square } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";

const { t } = useI18n();

const {
  loading,
  error,
  podmanVersion,
  runningCount,
  stoppedCount,
  imagesCount,
  totalVolumes,
  osInfo,
  coreCount,
} = useHostMetrics();

function countUpTo(targetRef, targetVal, duration = 900) {
  const startVal = targetRef.value;
  const startTime = Date.now();

  const tick = () => {
    const elapsed = Date.now() - startTime;
    const progress = Math.min(elapsed / duration, 1);
    const eased = 1 - Math.pow(1 - progress, 3);
    targetRef.value = Math.round(startVal + eased * (targetVal - startVal));
    if (progress < 1) requestAnimationFrame(tick);
    else targetRef.value = targetVal;
  };

  requestAnimationFrame(tick);
}

const displayCores = ref(0);
watch(coreCount, (cores) => countUpTo(displayCores, cores));

// Icon per stat keeps the row scannable without four identical glyphs.
const workloadStats = computed(() => [
  { key: "apps", label: t("home.overviewPulseCard.apps"), value: runningCount.value, icon: Boxes, tone: "text-sky-600 dark:text-sky-400" },
  { key: "volumes", label: t("home.overviewPulseCard.volumes"), value: totalVolumes.value, icon: Database, tone: "text-sky-600 dark:text-sky-400" },
  { key: "images", label: t("home.overviewPulseCard.images"), value: imagesCount.value, icon: Image, tone: "text-sky-600 dark:text-sky-400" },
  { key: "stopped", label: t("home.overviewPulseCard.stopped"), value: stoppedCount.value, icon: Square, tone: "text-zinc-500 dark:text-zinc-400" },
]);
</script>

<template>
  <div class="group relative flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-all duration-300 hover:-translate-y-1 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">

    <div v-if="loading" class="relative z-10 flex h-full flex-1 flex-col items-center justify-center gap-3 text-center">
      <div class="flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl bg-zinc-50 dark:bg-zinc-900/50">
        <Server class="h-7 w-7 animate-pulse text-zinc-400 dark:text-zinc-500" />
      </div>
      <span class="text-[10px] font-bold uppercase tracking-[0.18em] text-zinc-500">{{ t("quickMetrics.hostMetrics.scanningHost") }}</span>
    </div>

    <div v-else-if="error" class="relative z-10 flex h-full flex-1 flex-col gap-4">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-red-200 bg-red-50 dark:border-red-900/50 dark:bg-red-900/20">
          <Server class="h-5 w-5 text-red-500" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 dark:text-white">{{ t("quickMetrics.hostMetrics.hostSystem") }}</h3>
          <p class="mt-2 text-[10px] font-semibold uppercase tracking-widest text-red-500">{{ t("quickMetrics.hostMetrics.connectionFailed") }}</p>
        </div>
      </div>
      <div class="mt-auto pb-2 pt-6">
        <div class="text-4xl font-black leading-[1.1] tracking-tighter text-zinc-900 dark:text-white">
          {{ t("common.error") }}
        </div>
        <div class="mt-2 line-clamp-2 text-sm text-zinc-500 dark:text-zinc-400">
          {{ error }}
        </div>
      </div>
    </div>

    <div v-else class="relative z-10 flex h-full flex-col gap-4">
      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-sky-200 bg-sky-50 transition-transform duration-300 group-hover:scale-105 dark:border-sky-900/50 dark:bg-sky-900/20">
            <Server class="h-5 w-5 text-sky-600 dark:text-sky-400" />
          </div>
          <div>
            <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors group-hover:text-black dark:text-white dark:group-hover:text-white">
              {{ t("quickMetrics.hostMetrics.hostSystem") }}
            </h3>
            <div class="mt-2 flex items-center gap-2">
              <span class="relative flex h-2 w-2">
                <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
                <span class="relative inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
              </span>
              <span class="text-[10px] font-semibold uppercase tracking-widest text-emerald-600 dark:text-emerald-500">{{ t("quickMetrics.hostMetrics.online") }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Main Body -->
      <div class="relative z-10 mt-auto flex flex-col pt-4">
        <div class="mb-2 line-clamp-2 text-4xl font-black leading-[1.1] tracking-tighter text-zinc-900 transition-colors duration-500 group-hover:-translate-y-1 dark:text-white sm:text-5xl">
          {{ osInfo.name }}
        </div>

        <div class="mb-4 truncate text-[11px] font-semibold uppercase tracking-widest text-zinc-500">
          {{ osInfo.type }} • {{ osInfo.arch }} • {{ osInfo.kernel }}
        </div>

        <!-- Workload counts -->
        <div class="grid grid-cols-4 gap-2">
          <div
            v-for="stat in workloadStats"
            :key="stat.key"
            class="flex flex-col items-center justify-center gap-1 rounded-lg border border-zinc-200 bg-zinc-50 p-2 dark:border-zinc-800 dark:bg-zinc-900/50"
          >
            <span class="text-center text-[8px] font-bold uppercase leading-none tracking-widest text-zinc-500">{{ stat.label }}</span>
            <span class="text-sm font-black" :class="stat.tone">{{ stat.value }}</span>
          </div>
        </div>

        <!-- Engine facts -->
        <div class="mt-2 grid grid-cols-2 gap-2">
          <div class="flex flex-col gap-1 rounded-xl border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
            <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.processors") }}</span>
            <span class="text-sm font-black tracking-tight text-zinc-900 dark:text-white">
              {{ displayCores }} <span class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.cores") }}</span>
            </span>
          </div>
          <div class="flex flex-col gap-1 rounded-xl border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
            <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.podman") }}</span>
            <span class="truncate text-sm font-black tracking-tight text-zinc-900 dark:text-white">
              {{ podmanVersion }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>