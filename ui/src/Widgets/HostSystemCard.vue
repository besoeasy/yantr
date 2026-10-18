<script>
// Two columns: the host identity and its counters in one readable column, the
// way a machine's spec sheet reads — header row, identity, workload facts,
// then the versioned details. The full-row band had room to sit side by side;
// at half the row the same data reads better stacked, which is why this is a
// redesign rather than a resize.
export const meta = { cols: 2 };
</script>
<script setup>
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { Server } from "@lucide/vue";
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
  loadText,
  uptimeText,
} = useHostMetrics();

// The core count counts up on arrival, one small piece of life in the card —
// the same easing the previous band used, kept: a jump reads as a glitch.
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

// The workload is a fact table, so even a zero is printed — a homelab with
// nothing running is a real state, and an em-dash would be a lie. Stopped is
// the muted cell: the one number here that is not a healthy state.
const workloadStats = computed(() => [
  { key: "apps", label: t("home.overviewPulseCard.apps"), value: runningCount.value },
  { key: "volumes", label: t("home.overviewPulseCard.volumes"), value: totalVolumes.value },
  { key: "images", label: t("home.overviewPulseCard.images"), value: imagesCount.value },
  { key: "stopped", label: t("home.overviewPulseCard.stopped"), value: stoppedCount.value, muted: true },
]);

// The spec rows: label in small caps, value in tabular or mono — a spec
// sheet, not a card. Uptime and load render only when the metrics poll has
// real numbers; undefined values would print "undefined", which is worse than
// a shorter list.
const factRows = computed(() => [
  {
    key: "processors",
    label: t("quickMetrics.hostMetrics.processors"),
    value: `${displayCores.value} ${t("quickMetrics.hostMetrics.cores")}`,
    mono: false,
  },
  {
    key: "podman",
    label: t("quickMetrics.hostMetrics.podman"),
    value: podmanVersion.value,
    mono: true,
  },
  ...(uptimeText.value
    ? [{ key: "uptime", label: t("quickMetrics.hostMetrics.uptime"), value: uptimeText.value, mono: true }]
    : []),
  ...(loadText.value
    ? [{ key: "load", label: t("quickMetrics.hostMetrics.load"), value: loadText.value, mono: true }]
    : []),
]);
</script>

<template>
  <section
    class="yantra-card yantra-card-min group relative flex h-full w-full flex-col overflow-hidden rounded-2xl p-7 text-left transition-[border-color,box-shadow] duration-200 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <!-- The one hairline accent: the online dot is the card's only colour. -->
    <div
      aria-hidden="true"
      class="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-zinc-300/40 to-transparent dark:via-zinc-700/40"
    ></div>

    <!-- Loading: skeleton slabs that hold the card's shape -->
    <div v-if="loading" class="yantra-card yantra-card-min flex flex-1 flex-col gap-5">
      <div class="flex items-center gap-4">
        <span class="h-12 w-12 shrink-0 animate-pulse rounded-xl bg-zinc-100 dark:bg-zinc-800"></span>
        <span class="flex flex-1 flex-col gap-2">
          <span class="block h-3 w-28 animate-pulse rounded-full bg-zinc-100 dark:bg-zinc-800"></span>
          <span class="block h-2.5 w-20 animate-pulse rounded-full bg-zinc-100 dark:bg-zinc-800"></span>
        </span>
      </div>
      <span class="block h-9 w-48 animate-pulse rounded-lg bg-zinc-100 dark:bg-zinc-800"></span>
      <div class="grid grid-cols-2 overflow-hidden rounded-lg dark:border-zinc-800">
        <span
          v-for="n in 4"
          :key="n"
          class="h-14 animate-pulse border-b border-r yantra-tile"
        ></span>
      </div>
    </div>

    <div v-else-if="error" class="flex flex-1 flex-col gap-4">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-red-200 bg-red-50 dark:border-red-900/50 dark:bg-red-900/10">
          <Server class="h-5 w-5 text-red-600 dark:text-red-500" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight text-red-950 dark:text-red-50">
            {{ t("quickMetrics.hostMetrics.hostSystem") }}
          </h3>
          <p class="mt-2 text-[10px] font-semibold uppercase tracking-widest text-red-600 dark:text-red-400">
            {{ t("quickMetrics.hostMetrics.connectionFailed") }}
          </p>
        </div>
      </div>
      <div class="mt-auto flex flex-col pt-6">
        <div class="text-4xl font-black leading-[1.1] tracking-tighter text-red-950 yantra-text-primary">
          {{ t("common.error") }}
        </div>
        <div class="mt-2 line-clamp-2 text-sm yantra-text-secondary">{{ error }}</div>
      </div>
    </div>

    <template v-else>
      <!-- Header: what this card is, and that it is answering. The status
           sits at the right edge, where a dashboard's eye lands last. -->
      <div class="flex w-full items-start justify-between gap-4">
        <div class="flex items-center gap-4">
          <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-zinc-50 transition-transform duration-200 group-hover:scale-105 dark:border-zinc-800 dark:bg-zinc-900">
            <Server class="h-5 w-5 text-zinc-600 transition-colors group-hover:text-zinc-900 yantra-text-secondary dark:group-hover:text-zinc-100" />
          </div>
          <h3 class="text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
            {{ t("quickMetrics.hostMetrics.hostSystem") }}
          </h3>
        </div>
        <div class="flex shrink-0 items-center gap-2 text-emerald-600 dark:text-emerald-400">
          <span class="relative flex h-2 w-2">
            <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
            <span class="relative inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
          </span>
          <span class="text-[10px] font-semibold uppercase tracking-widest">
            {{ t("quickMetrics.hostMetrics.online") }}
          </span>
        </div>
      </div>

      <!-- Identity: the machine by name, then the machine-readable line -->
      <div class="mt-6 flex flex-col">
        <div class="truncate text-4xl font-black leading-[1.05] tracking-tighter yantra-text-primary">
          {{ osInfo.name }}
        </div>
        <div class="mt-2 truncate font-mono text-[11px] yantra-text-secondary" :title="`${osInfo.type} · ${osInfo.arch} · ${osInfo.kernel}`">
          {{ osInfo.type }} · {{ osInfo.arch }} · {{ osInfo.kernel }}
        </div>
      </div>

      <!-- Workload: a 2x2 fact grid. Hairlines are drawn per cell (the
           utility layer has no divide-* helpers, and silent no-op classes
           are how borders go missing here) — container border plus cell
           right/bottom edges, overflow-hidden to clean the corners. -->
      <div class="mt-6 grid grid-cols-2 overflow-hidden rounded-lg dark:border-zinc-800">
        <div
          v-for="stat in workloadStats"
          :key="stat.key"
          class="flex flex-col items-start gap-1 border-b border-r px-4 py-3 dark:border-zinc-800"
        >
          <span
            class="text-2xl font-bold leading-none tabular-nums"
            :class="stat.muted ? 'yantra-text-secondary' : 'yantra-text-primary'"
          >
            {{ stat.value }}
          </span>
          <span class="text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
            {{ stat.label }}
          </span>
        </div>
      </div>

      <!-- Spec rows: processors, podman, uptime, load. -->
      <div class="mt-auto flex flex-col pt-6">
        <div
          v-for="(row, i) in factRows"
          :key="row.key"
          class="flex items-baseline justify-between gap-4 py-2"
          :class="i < factRows.length - 1 ? 'border-b border-zinc-100 dark:border-zinc-900' : ''"
        >
          <span class="shrink-0 text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
            {{ row.label }}
          </span>
          <span
            class="min-w-0 truncate text-right text-[13px] font-semibold"
            :class="row.mono ? 'font-mono' : 'tabular-nums'"
            :title="typeof row.value === 'string' ? row.value : undefined"
          >
            {{ row.value }}
          </span>
        </div>
      </div>
    </template>
  </section>
</template>