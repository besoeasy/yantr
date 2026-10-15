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
  <div class="host-card group relative flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl p-7 text-left">
    <div v-if="loading" class="flex h-full flex-1 flex-col items-center justify-center gap-3 text-center">
      <div class="host-icon host-icon--lg h-14 w-14">
        <Server class="h-7 w-7 animate-pulse" />
      </div>
      <span class="host-sub text-[10px] font-bold uppercase tracking-[0.18em]">{{ t("quickMetrics.hostMetrics.scanningHost") }}</span>
    </div>

    <div v-else-if="error" class="flex h-full flex-1 flex-col gap-4">
      <div class="flex items-center gap-4">
        <div class="host-icon host-icon--danger h-12 w-12">
          <Server class="h-5 w-5" />
        </div>
        <div>
          <h3 class="host-value text-lg font-bold leading-none tracking-tight">{{ t("quickMetrics.hostMetrics.hostSystem") }}</h3>
          <p class="host-danger mt-2 text-[10px] font-semibold uppercase tracking-widest">{{ t("quickMetrics.hostMetrics.connectionFailed") }}</p>
        </div>
      </div>
      <div class="mt-auto pb-2 pt-6">
        <div class="host-value text-4xl font-black leading-[1.1] tracking-tighter">{{ t("common.error") }}</div>
        <div class="host-sub mt-2 line-clamp-2 text-sm">{{ error }}</div>
      </div>
    </div>

    <div v-else class="flex h-full flex-col gap-4">
      <!-- Header -->
      <div class="flex items-center gap-4">
        <div class="host-icon host-icon--accent h-12 w-12 transition-transform duration-300 group-hover:scale-105">
          <Server class="h-5 w-5" />
        </div>
        <div>
          <h3 class="host-value text-lg font-bold leading-none tracking-tight">{{ t("quickMetrics.hostMetrics.hostSystem") }}</h3>
          <div class="mt-2 flex items-center gap-2">
            <span class="relative flex h-2 w-2">
              <span class="host-ok-ping absolute inline-flex h-full w-full animate-ping rounded-full"></span>
              <span class="host-ok-dot relative inline-flex h-2 w-2 rounded-full"></span>
            </span>
            <span class="host-ok text-[10px] font-semibold uppercase tracking-widest">{{ t("quickMetrics.hostMetrics.online") }}</span>
          </div>
        </div>
      </div>

      <!-- Identity, then the facts panel, both pinned to the bottom so every
           card in the grid lines up whatever its content. -->
      <div class="mt-auto flex flex-col gap-4 pt-4">
        <div>
          <div class="host-value line-clamp-2 text-4xl font-black leading-[1.1] tracking-tighter transition-transform duration-500 group-hover:-translate-y-1 sm:text-5xl">
            {{ osInfo.name }}
          </div>
          <div class="host-sub mt-2 truncate text-[11px] font-semibold uppercase tracking-widest">
            {{ osInfo.type }} • {{ osInfo.arch }} • {{ osInfo.kernel }}
          </div>
        </div>

        <div class="host-panel">
          <div class="host-counts">
            <div v-for="stat in workloadStats" :key="stat.key" class="host-count">
              <component :is="stat.icon" class="h-3 w-3" :class="stat.muted ? 'host-count__icon--muted' : 'host-count__icon'" />
              <span class="host-count__label">{{ stat.label }}</span>
              <span class="host-count__value" :class="{ 'host-count__value--muted': stat.muted }">{{ stat.value }}</span>
            </div>
          </div>

          <div class="host-row">
            <span class="host-row__label">{{ t("quickMetrics.hostMetrics.processors") }}</span>
            <span class="host-row__value">
              {{ displayCores }} <span class="host-row__unit">{{ t("quickMetrics.hostMetrics.cores") }}</span>
            </span>
          </div>
          <div class="host-row">
            <span class="host-row__label">{{ t("quickMetrics.hostMetrics.podman") }}</span>
            <span class="host-row__value truncate">{{ podmanVersion }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Card chrome and every colour in this card come from the tokens in
   main.css (--card-*, --accent-*, --ok-*, --danger-*, --text-*), so the theme
   is defined in one place instead of a second palette spelled out in classes.
   Layout and type stay on the shared utility classes. */
.host-card {
  background-color: var(--card-bg);
  border: 1px solid var(--card-border);
  transition:
    border-color var(--yantra-dur) var(--yantra-ease),
    box-shadow var(--yantra-dur) var(--yantra-ease),
    transform var(--yantra-dur) var(--yantra-ease);
}

.host-card:hover {
  border-color: var(--card-border-strong);
  box-shadow: var(--card-shadow-hover);
  transform: translateY(-4px);
}

.host-icon {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--card-border);
  border-radius: var(--radius-xl);
  background-color: var(--card-inset);
  color: var(--text-secondary);
}
.host-icon--lg {
  border-radius: var(--radius-2xl);
}
.host-icon--accent {
  border-color: var(--accent-border);
  background-color: var(--accent-soft);
  color: var(--accent);
}
.host-icon--danger {
  border-color: var(--danger-border);
  background-color: var(--danger-soft);
  color: var(--danger-text);
}

/* Text colours only — size, weight and tracking come from utilities. */
.host-value {
  color: var(--text-primary);
}
.host-sub {
  color: var(--text-secondary);
}
.host-ok {
  color: var(--ok-text);
}
.host-ok-dot,
.host-ok-ping {
  background-color: var(--ok);
}
.host-danger {
  color: var(--danger-text);
}

/* The facts panel: one bordered block of label/value cells with hairline
   dividers, rather than a patchwork of tiles. */
.host-panel {
  overflow: hidden;
  border: 1px solid var(--card-border);
  border-radius: var(--radius-xl);
}

.host-counts {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

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

.host-count__icon {
  color: var(--accent);
}
.host-count__icon--muted {
  color: var(--text-secondary);
}
.host-count__label {
  color: var(--text-secondary);
  font-size: 8px;
  font-weight: 700;
  line-height: 1;
  text-align: center;
  text-transform: uppercase;
  letter-spacing: 0.14em;
}
.host-count__value {
  color: var(--accent);
  font-size: 0.875rem;
  font-weight: 900;
  font-variant-numeric: tabular-nums;
}
.host-count__value--muted {
  color: var(--text-secondary);
}

.host-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.625rem 0.875rem;
}
.host-row + .host-row {
  border-top: 1px solid var(--card-border);
}

.host-row__label {
  color: var(--text-secondary);
  font-size: 9px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.1em;
}
.host-row__value {
  color: var(--text-primary);
  font-size: 0.875rem;
  font-weight: 900;
  font-variant-numeric: tabular-nums;
}
.host-row__unit {
  color: var(--text-secondary);
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.1em;
}

@media (prefers-reduced-motion: reduce) {
  .host-card,
  .host-card:hover {
    transform: none;
  }
}
</style>
