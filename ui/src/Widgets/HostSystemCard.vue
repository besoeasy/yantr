<script>
// Full-width band: all four of the desktop grid's four columns. Host identity
// is the hinge of every other card on the dashboard — it earns the whole row,
// and the counters read like a data table only when nothing is squeezed
// beside them.
export const meta = { cols: 4 };
</script>
<script setup>
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { Server } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";

// Same design system as the other full-width bands (Featured Today,
// Developer of the day): monochrome chrome, one hairline system, numbers in
// tabular figures, identifiers in mono. The only accent anywhere is the
// online dot — a status signal, not decoration.

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

// The core count counts up on arrival, one small piece of life in the band —
// the same easing the previous card used, kept: a jump reads as a glitch.
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

// The workload row is a fact table, so even a zero is printed — a homelab
// with nothing running is a real state, and an em-dash would be a lie.
// Stopped is the muted cell: the one number here that is not a healthy state.
const workloadStats = computed(() => [
  { key: "apps", label: t("home.overviewPulseCard.apps"), value: runningCount.value },
  { key: "volumes", label: t("home.overviewPulseCard.volumes"), value: totalVolumes.value },
  { key: "images", label: t("home.overviewPulseCard.images"), value: imagesCount.value },
  { key: "stopped", label: t("home.overviewPulseCard.stopped"), value: stoppedCount.value, muted: true },
]);

// Fact rows on the right: label in small caps, value in tabular/mono — a
// spec sheet, not a card. Uptime and load are rendered only when the metrics
// poll has real numbers; undefined values would print "undefined", which is
// worse than a shorter list.
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
    class="group relative flex h-full w-full min-h-[280px] flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-[border-color,box-shadow] duration-200 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <div
      aria-hidden="true"
      class="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-zinc-300/40 to-transparent dark:via-zinc-700/40"
    ></div>

    <!-- Loading: two skeleton rows, so the band keeps its shape -->
    <div v-if="loading" class="flex flex-1 flex-col gap-6">
      <div class="flex items-center gap-4">
        <span class="h-12 w-12 shrink-0 animate-pulse rounded-xl bg-zinc-100 dark:bg-zinc-800"></span>
        <span class="mt-0.5 flex-1 space-y-2">
          <span class="block h-3 w-24 animate-pulse rounded-full bg-zinc-100 dark:bg-zinc-800"></span>
          <span class="block h-2.5 w-40 animate-pulse rounded-full bg-zinc-100 dark:bg-zinc-800"></span>
        </span>
      </div>
      <div class="mt-auto flex gap-6">
        <span class="block h-14 w-64 flex-1 animate-pulse rounded-lg bg-zinc-100 dark:bg-zinc-800"></span>
        <span class="block h-14 w-64 flex-1 animate-pulse rounded-lg bg-zinc-100 dark:bg-zinc-800"></span>
      </div>
    </div>

    <div v-else-if="error" class="flex flex-1 flex-col gap-4">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-red-200 bg-red-50 dark:border-red-900/50 dark:bg-red-900/10">
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
        <div class="text-4xl font-black leading-[1.1] tracking-tighter text-red-950 dark:text-white">
          {{ t("common.error") }}
        </div>
        <div class="mt-2 line-clamp-2 text-sm text-zinc-500 dark:text-zinc-400">{{ error }}</div>
      </div>
    </div>

    <template v-else>
      <!-- Header row: what this band is, and that it is answering -->
      <div class="flex w-full items-start justify-between gap-4">
        <div class="flex items-center gap-4">
          <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 transition-transform duration-200 group-hover:scale-105 dark:border-zinc-800 dark:bg-zinc-900">
            <Server class="h-5 w-5 text-zinc-600 transition-colors group-hover:text-zinc-900 dark:text-zinc-400 dark:group-hover:text-zinc-100" />
          </div>
          <div>
            <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 dark:text-white">
              {{ t("quickMetrics.hostMetrics.hostSystem") }}
            </h3>
            <div class="mt-2 flex items-center gap-2 text-emerald-600 dark:text-emerald-400">
              <span class="relative flex h-2 w-2">
                <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
                <span class="relative inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
              </span>
              <span class="text-[10px] font-semibold uppercase tracking-widest">
                {{ t("quickMetrics.hostMetrics.online") }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Body: identity left, spec sheet right -->
      <div class="flex flex-1 flex-col gap-8 lg:flex-row lg:gap-0 lg:pt-2">
        <div class="flex flex-1 flex-col lg:pr-8">
          <div class="flex min-w-0 flex-col">
            <div class="truncate text-5xl font-black leading-[1.05] tracking-tighter text-zinc-900 sm:text-6xl dark:text-white">
              {{ osInfo.name }}
            </div>
            <div class="mt-3 truncate font-mono text-[11px] text-zinc-400" :title="`${osInfo.type} · ${osInfo.arch} · ${osInfo.kernel}`">
              {{ osInfo.type }} · {{ osInfo.arch }} · {{ osInfo.kernel }}
            </div>
          </div>

          <!-- Workload fact table. Hairline dividers between cells; numbers
               in tabular figures; the value is printed even when it is 0. -->
          <div class="mt-auto grid grid-cols-4 divide-x divide-zinc-200 border-y border-zinc-200 dark:divide-zinc-800 dark:border-zinc-800">
            <div
              v-for="stat in workloadStats"
              :key="stat.key"
              class="flex flex-col items-center gap-1 px-2 py-3"
            >
              <span
                class="text-base font-bold tabular-nums"
                :class="stat.muted ? 'text-zinc-400 dark:text-zinc-500' : 'text-zinc-900 dark:text-white'"
              >
                {{ stat.value }}
              </span>
              <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-500 dark:text-zinc-400">
                {{ stat.label }}
              </span>
            </div>
          </div>
        </div>

        <!-- Spec rows: hairline rules, small-caps labels, tabular values -->
        <div class="flex flex-col justify-center lg:w-64 lg:shrink-0 lg:border-l lg:border-zinc-200 lg:pl-8 dark:lg:border-zinc-800">
          <div
            v-for="row in factRows"
            :key="row.key"
            class="flex items-baseline justify-between gap-4 border-b border-zinc-100 py-2.5 last:border-b-0 dark:border-zinc-900"
          >
            <span class="shrink-0 text-[10px] font-bold uppercase tracking-widest text-zinc-500 dark:text-zinc-400">
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
      </div>
    </template>
  </section>
</template>
