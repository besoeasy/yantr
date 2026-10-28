<script>
// Dashboard engine: one column — traffic only. Rates, their trend, and what
// every link is moving. Whether a link is up or negotiated slow is the Link
// Status card's question; mixing the two into one tile made a headline that
// explained nothing. See Home.vue for how meta.cols is consumed.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Network, ArrowDown, ArrowUp } from "@lucide/vue";
import Sparkline from "../components/Sparkline.vue";
import { useHostMetrics } from "../composables/useHostMetrics";
import { formatBitRate } from "../utils/metrics";

const { t } = useI18n();

const {
  metricsStale,
  isDegraded,
  primaryNet,
  netInterfaces,
  netText,
  netHistory,
} = useHostMetrics();

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));

// The sparkline scales itself, like the disk I/O line: byte rates have no
// natural ceiling to pin to 100, so the max is the busiest moment in the
// window. The 1 MB/s floor keeps an all-zero window from producing a 0
// ceiling, which would divide the next real sample into a spike to the top.
const sparkMax = computed(() => Math.max(...netHistory.value, 1024 * 1024) * 1.15);
const sparkLabel = computed(() => t("quickMetrics.hostMetrics.netHistory"));

// Split the combined "rx · tx" string so each direction gets its own arrow and
// colour. Kept as two computeds rather than one array destructured once —
// destructuring at setup would freeze the first reading, since setup runs
// before any metric has arrived.
const rxText = computed(() => netText.value?.split("·")[0]?.trim() || "—");
const txText = computed(() => netText.value?.split("·")[1]?.trim() || "—");

// Everything except the headline interface: tunnels and bridges have a story
// (unexpected flows) but no reason to claim the headline. Each row keeps the
// rate only — link state and speed live on the Link Status card.
const secondaryNets = computed(() => {
  const primary = primaryNet.value?.iface;
  return netInterfaces.value.filter((iface) => iface.iface !== primary);
});
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-sky-200 bg-sky-50 transition-transform duration-200 group-hover:scale-105 dark:border-sky-900/50 dark:bg-sky-900/20">
          <Network class="h-5 w-5 text-sky-600 dark:text-sky-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
            {{ t("quickMetrics.hostMetrics.network") }}
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
      <!-- Degraded: the reading is missing, so say so plainly. -->
      <div
        v-if="isDegraded('network')"
        class="flex items-center gap-3 rounded-xl border-red-200 bg-red-50 p-4 dark:border-red-900/50 dark:bg-red-900/20"
      >
        <Network class="h-5 w-5 shrink-0 text-red-500" />
        <div class="text-[11px] font-medium leading-tight text-red-600 dark:text-red-400">
          {{ unavailable }}
        </div>
      </div>

      <template v-else>
        <!-- rx is the headline: the volume a homelab pulls is the more telling
             of the two directions and the first thing you want to see move. -->
        <div class="mb-2 flex items-baseline justify-between gap-3">
          <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
            <ArrowDown class="h-3 w-3 shrink-0 text-sky-500" />
            rx
          </span>
          <span class="text-4xl font-black tabular-nums leading-[1.1] tracking-tighter yantra-text-primary transition-colors duration-200 group-hover:-translate-y-[3px] yantra-text-primary sm:text-5xl">
            {{ rxText }}
          </span>
        </div>

        <!-- Which link this traffic is on: identity, not health — the state
             of this same link is the Link Status card's fact. -->
        <div class="mb-3 truncate text-[9px] font-semibold uppercase tracking-widest yantra-text-secondary">
          {{ primaryNet?.iface ?? "" }}
        </div>

        <!-- 4-minute trend of the busiest interface, rx+tx combined. Sky stays
             this card's own accent so the four host tiles read as four tones —
             line classes and gradient stops already exist in main.css for it. -->
        <div class="mb-5 h-16 w-full">
          <Sparkline :points="netHistory" :max="sparkMax" :label="sparkLabel" />
        </div>

        <!-- tx: the other direction, a row rather than a second headline. -->
        <div class="flex items-center justify-between gap-3 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
          <span class="flex items-center gap-1.5 text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
            <ArrowUp class="h-3 w-3 shrink-0 text-violet-500" />
            tx
          </span>
          <span class="truncate text-sm font-black tabular-nums tracking-tight yantra-text-primary">
            {{ txText }}
          </span>
        </div>

        <!-- The rest of the box's links: what is actually flowing on each.
             No state dots, no speeds — those are link facts, on the Link
             Status card, where a slow negotiation is diagnosable instead of
             decoration. -->
        <div v-if="secondaryNets.length > 0" class="mt-4 border-t border-zinc-200 pt-3 dark:border-zinc-800">
          <div class="mb-2 text-[10px] font-semibold uppercase tracking-widest yantra-text-secondary">
            {{ t("quickMetrics.hostMetrics.netOther") }}
          </div>
          <div class="flex flex-col gap-2">
            <div
              v-for="iface in secondaryNets"
              :key="iface.iface"
              class="flex items-center gap-2 rounded-xl bg-zinc-50 p-2 dark:bg-zinc-900/50"
            >
              <span class="min-w-0 truncate text-[11px] font-bold tracking-tight yantra-text-primary">{{ iface.iface }}</span>
              <span class="ml-auto flex shrink-0 items-center gap-2 text-[9px] font-semibold yantra-text-secondary">
                <span class="flex items-center gap-1"><ArrowDown class="h-3 w-3 shrink-0 text-sky-500" />{{ formatBitRate(iface.rxBps) }}</span>
                <span class="flex items-center gap-1"><ArrowUp class="h-3 w-3 shrink-0 text-violet-500" />{{ formatBitRate(iface.txBps) }}</span>
              </span>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>
