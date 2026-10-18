<script>
// Dashboard engine: one column — the OS line, the four-counter strip and the
// rows below it were designed for a quarter of the grid. See Home.vue for how
// meta.cols is consumed.
export const meta = { cols: 1 };
</script>
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

// One glyph per counter keeps a four-cell row scannable. "stopped" is the one
// muted cell: it is the only number here that is not a healthy state.
const workloadStats = computed(() => [
  { key: "apps", label: t("home.overviewPulseCard.apps"), value: runningCount.value, icon: Boxes },
  { key: "volumes", label: t("home.overviewPulseCard.volumes"), value: totalVolumes.value, icon: Database },
  { key: "images", label: t("home.overviewPulseCard.images"), value: imagesCount.value, icon: Image },
  { key: "stopped", label: t("home.overviewPulseCard.stopped"), value: stoppedCount.value, icon: Square, muted: true },
]);
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl p-7 text-left">
    <div v-if="loading" class="flex h-full flex-1 flex-col items-center justify-center gap-3 text-center">
      <div class="yantra-tile yantra-tile--round h-14 w-14">
        <Server class="h-7 w-7 animate-pulse" />
      </div>
      <span class="yantra-text-secondary text-[10px] font-bold uppercase tracking-[0.18em]">{{ t("quickMetrics.hostMetrics.scanningHost") }}</span>
    </div>

    <div v-else-if="error" class="flex h-full flex-1 flex-col gap-4">
      <div class="flex items-center gap-4">
        <div class="yantra-tile yantra-tile--danger h-12 w-12">
          <Server class="h-5 w-5" />
        </div>
        <div>
          <h3 class="yantra-text-primary text-lg font-bold leading-none tracking-tight">{{ t("quickMetrics.hostMetrics.hostSystem") }}</h3>
          <p class="yantra-text-danger mt-2 text-[10px] font-semibold uppercase tracking-widest">{{ t("quickMetrics.hostMetrics.connectionFailed") }}</p>
        </div>
      </div>
      <div class="mt-auto pb-2 pt-6">
        <div class="yantra-text-primary text-4xl font-black leading-[1.1] tracking-tighter">{{ t("common.error") }}</div>
        <div class="yantra-text-secondary mt-2 line-clamp-2 text-sm">{{ error }}</div>
      </div>
    </div>

    <div v-else class="flex h-full flex-col gap-4">
      <!-- Header -->
      <div class="flex items-center gap-4">
        <div class="yantra-tile yantra-tile--accent h-12 w-12 transition-transform duration-200 group-hover:scale-105">
          <Server class="h-5 w-5" />
        </div>
        <div>
          <h3 class="yantra-text-primary text-lg font-bold leading-none tracking-tight">{{ t("quickMetrics.hostMetrics.hostSystem") }}</h3>
          <div class="mt-2 flex items-center gap-2">
            <span class="relative flex h-2 w-2">
              <span class="host-dot-ping absolute inline-flex h-full w-full animate-ping rounded-full"></span>
              <span class="host-dot relative inline-flex h-2 w-2 rounded-full"></span>
            </span>
            <span class="yantra-text-ok text-[10px] font-semibold uppercase tracking-widest">{{ t("quickMetrics.hostMetrics.online") }}</span>
          </div>
        </div>
      </div>

      <!-- Identity, then the facts panel, both pinned to the bottom so every
           card in the grid lines up whatever its content. -->
      <div class="mt-auto flex flex-col gap-4 pt-4">
        <div>
          <div class="yantra-text-primary line-clamp-2 text-4xl font-black leading-[1.1] tracking-tighter transition-transform duration-200 group-hover:-translate-y-[3px] sm:text-5xl">
            {{ osInfo.name }}
          </div>
          <div class="yantra-text-secondary mt-2 truncate text-[11px] font-semibold uppercase tracking-widest">
            {{ osInfo.type }} • {{ osInfo.arch }} • {{ osInfo.kernel }}
          </div>
        </div>

        <div class="yantra-panel">
          <div class="grid grid-cols-4">
            <div v-for="stat in workloadStats" :key="stat.key" class="host-count">
              <component :is="stat.icon" class="h-3 w-3" :class="stat.muted ? 'yantra-text-secondary' : 'yantra-text-accent'" />
              <span class="yantra-text-secondary text-center text-[8px] font-bold uppercase leading-none tracking-widest">{{ stat.label }}</span>
              <span class="text-sm font-black tabular-nums" :class="stat.muted ? 'yantra-text-secondary' : 'yantra-text-accent'">{{ stat.value }}</span>
            </div>
          </div>

          <div class="yantra-row">
            <span class="yantra-row__label">{{ t("quickMetrics.hostMetrics.processors") }}</span>
            <span class="yantra-row__value">
              {{ displayCores }} <span class="yantra-text-secondary text-[10px] font-bold uppercase tracking-widest">{{ t("quickMetrics.hostMetrics.cores") }}</span>
            </span>
          </div>
          <div class="yantra-row">
            <span class="yantra-row__label">{{ t("quickMetrics.hostMetrics.podman") }}</span>
            <span class="yantra-row__value truncate">{{ podmanVersion }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Chrome, tiles, panel, rows and every text colour come from the shared
   helpers in main.css. What is left here is only what this card owns: the
   counter strip's hairline dividers and the status dot. */
.host-count {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.25rem;
  padding: 0.625rem 0.25rem;
  border-bottom: 1px solid var(--card-border);
}

.host-count:not(:last-child) {
  border-right: 1px solid var(--card-border);
}

.host-dot,
.host-dot-ping {
  background-color: var(--ok);
}
</style>
