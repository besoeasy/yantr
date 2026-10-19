<script>
// Dashboard engine: two columns — five ranked rows with bars buckle below
// half the row, and one column cannot fit a name, two numbers and a bar per
// line comfortably. Width honours the two-column ceiling every widget lives
// by; see Home.vue for how meta.cols is consumed.
export const meta = { cols: 2 };
</script>
<script setup>
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { Flame, Cpu, MemoryStick } from "@lucide/vue";
import { formatBytes } from "../utils/metrics";
import { createCpuRateCalculator } from "../utils/cpu";
import { useApiUrl } from "../composables/useApiUrl";
import { expectApiSuccess } from "../composables/useApiResponse";
import { useContainers } from "../composables/useDashboardData";
import WidgetSkeleton from "../components/WidgetSkeleton.vue";

// The stats endpoint returns cumulative ns counters, not percentages (see
// handleContainerStats for why), so a real CPU figure needs two samples
// differenced in the browser — utils/cpu.js owns that arithmetic.
const SAMPLE_ROUND_MS = 3000;

// The fan-out is the cost here: one request per running container per round.
// Twelve bounds the worst case on a busy homelab while covering what a
// five-row leaderboard can show with margin; the list is newest-first, which
// is where a stack someone just deployed would appear.
const MAX_SAMPLED = 12;
const ROW_COUNT = 5;

const { t } = useI18n();
const { apiUrl } = useApiUrl();
const { containers, containersLoaded, containersLoadFailed } = useContainers({ live: true });

// Per-container state lives in plain maps keyed by id — ref() would wrap
// every 3s sample in a deep reactive proxy for numbers this card only reads
// in two computeds.
const samples = new Map(); // id -> { pct, memUsage, name }
const calculators = new Map(); // id -> createCpuRateCalculator()
const version = ref(0); // bump to recompute rows; see rows notes below.

let timer = null;
let roundInFlight = false;

function samplingTargets() {
  return containers.value
    .filter((c) => c.state === "running")
    .sort((a, b) => (b.created ?? 0) - (a.created ?? 0))
    .slice(0, MAX_SAMPLED);
}

async function sampleRound() {
  if (roundInFlight || document.hidden) return;
  const targets = samplingTargets();
  if (targets.length === 0) return;
  roundInFlight = true;

  const seen = new Set();
  await Promise.all(
    targets.map(async (c) => {
      seen.add(c.id);
      try {
        const res = await fetch(`${apiUrl.value}/api/containers/${c.id}/stats`);
        const data = await expectApiSuccess(res, "Failed to fetch container stats");
        const cpu = data.stats?.cpu ?? {};
        const sample = {
          usage: cpu.usage,
          sampledAtMs: cpu.sampledAtMs,
          onlineCpus: cpu.onlineCpus,
        };
        let calc = calculators.get(c.id);
        if (!calc) {
          calc = createCpuRateCalculator();
          calculators.set(c.id, calc);
        }
        const pct = calc.push(sample);
        // A container that left the list keeps its old numbers until pruning
        // below, so a mid-round restart never flashes an empty row.
        const prev = samples.get(c.id) ?? {};
        samples.set(c.id, {
          name: c.app?.name || c.name || "Unknown",
          pct: typeof pct === "number" ? pct : prev.pct,
          memUsage: data.stats?.memory?.usage ?? prev.memUsage,
        });
      } catch {
        // One bad sample is a skipped tick, not a card failure: the next
        // round overwrites it, and dropping the whole leaderboard because a
        // short-lived container 404'd would be the wrong trade.
      }
    })
  );

  // Prune rows for containers that are no longer in the sampling set, so a
  // removed stack cannot keep a leaderboard slot forever.
  for (const id of [...samples.keys()]) {
    if (!seen.has(id)) {
      samples.delete(id);
      calculators.delete(id);
    }
  }

  version.value += 1;
  roundInFlight = false;
}

onMounted(() => {
  sampleRound();
  timer = setInterval(sampleRound, SAMPLE_ROUND_MS);
});
onUnmounted(() => clearInterval(timer));

// version in the dependency chain makes rows recompute after mutations of the
// plain (non-reactive) maps, without making those maps reactive themselves.
const rows = computed(() => {
  void version.value;
  const ranked = [...samples.values()]
    .filter((s) => typeof s.pct === "number" && s.pct > 0)
    .sort((a, b) => b.pct - a.pct)
    .slice(0, ROW_COUNT);
  const max = ranked[0]?.pct ?? 0;
  return ranked.map((s, i) => ({
    key: `${s.name}-${i}`,
    name: s.name,
    pct: s.pct,
    memUsage: s.memUsage,
    bar: max > 0 ? Math.min((s.pct / max) * 100, 100) : 0,
  }));
});

const waiting = computed(
  () => !containersLoadFailed.value && (containers.value.filter((c) => c.state === "running").length > 0) && rows.value.length === 0
);
const hasLoad = computed(() => rows.value.length > 0);
</script>

<template>
  <WidgetSkeleton v-if="!containersLoaded && !containersLoadFailed" />
  <div v-else class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex w-full items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-rose-200 bg-rose-50 transition-transform duration-200 group-hover:scale-105 dark:border-rose-900/50 dark:bg-rose-900/20">
          <Flame class="h-5 w-5 text-rose-600 dark:text-rose-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
            {{ t("quickMetrics.topConsumers.title") }}
          </h3>
          <div class="mt-2 flex items-center gap-1.5">
            <div class="h-1.5 w-1.5 rounded-full bg-rose-500"></div>
            <span class="text-[10px] font-semibold uppercase tracking-widest text-rose-600 dark:text-rose-400">
              {{ t("quickMetrics.topConsumers.subtitle") }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Rows -->
    <div class="relative z-10 mt-auto flex flex-col gap-2.5 pt-6">
      <template v-if="hasLoad">
        <div v-for="row in rows" :key="row.key" class="rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
          <div class="mb-1.5 flex items-baseline justify-between gap-3">
            <span class="truncate text-sm font-bold yantra-text-primary">{{ row.name }}</span>
            <span class="shrink-0 text-sm font-black tabular-nums tracking-tight text-rose-600 dark:text-rose-400">
              {{ row.pct.toFixed(1) }}%
            </span>
          </div>
          <div class="h-1 w-full overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
            <div class="h-full rounded-full bg-rose-500 transition-[width] duration-500" :style="{ width: `${row.bar}%` }"></div>
          </div>
          <div class="mt-1.5 flex items-center gap-1 text-[10px] font-semibold uppercase tracking-widest yantra-text-secondary">
            <MemoryStick class="h-3 w-3 shrink-0" />
            {{ formatBytes(row.memUsage) }}
          </div>
        </div>
      </template>

      <!-- Empty states: a dashboard claims nothing it has not measured. No
           running containers, a host still being sampled, and a failed list
           load are different facts — each renders its own line rather than a
           shared "no data". -->
      <div v-if="!hasLoad && !waiting" class="flex items-center gap-2 rounded-xl bg-zinc-50 p-4 text-sm yantra-text-secondary dark:border-zinc-800 dark:bg-zinc-900/50">
        <Cpu class="h-4 w-4 shrink-0" />
        {{ t("quickMetrics.topConsumers.none") }}
      </div>
      <div v-else-if="!hasLoad" class="flex items-center gap-2 rounded-xl bg-zinc-50 p-4 text-sm yantra-text-secondary dark:border-zinc-800 dark:bg-zinc-900/50">
        <Cpu class="h-4 w-4 shrink-0 animate-pulse" />
        {{ t("quickMetrics.topConsumers.sampling") }}
      </div>
    </div>
  </div>
</template>
