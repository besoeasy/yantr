<script>
// Dashboard engine: one column — link facts only. Whether a link is up, how
// it negotiated, and at what duplex are point-in-time answers to "is the
// cable healthy"; the rates live on the Network card, which asks a different
// question. One fact per card keeps both readable. See Home.vue for how
// meta.cols is consumed.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { EthernetPort } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";

const { t } = useI18n();

const {
  metricsStale,
  isDegraded,
  primaryNet,
  netInterfaces,
  primaryNetSpeed,
  primaryNetDuplex,
  primaryNetState,
} = useHostMetrics();

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));

// The headline link: state is the colour carrier — up emerald, down rose,
// unknown muted, because a link that has not reported state is a question,
// not an alarm. Duplex is the fact the merged card used to drop: a gigabit
// NIC negotiating half duplex quietly caps every transfer at a tenth of its
// rate, and no traffic number explains it.
const linkState = computed(() => primaryNetState.value || "—");

const stateDotColor = computed(() =>
  linkState.value === "up"
    ? "var(--color-emerald-500)"
    : linkState.value === "down"
      ? "var(--color-rose-500)"
      : "var(--color-zinc-500)");

const stateTextClass = computed(() =>
  linkState.value === "up"
    ? "text-emerald-600 dark:text-emerald-400"
    : linkState.value === "down"
      ? "text-rose-600 dark:text-rose-400"
      : "yantra-text-secondary");

const speedText = computed(() =>
  primaryNetSpeed.value !== null ? `${primaryNetSpeed.value} Mbps` : "—");

// The kernel answers "full"/"half"; anything else (virtual links report "")
// is an em-dash, not a guess.
const duplexText = computed(() => {
  const duplex = primaryNetDuplex.value;
  if (duplex === "full") return t("quickMetrics.hostMetrics.duplexFull");
  if (duplex === "half") return t("quickMetrics.hostMetrics.duplexHalf");
  return "—";
});

// Every other link on the box, same facts as the headline: state and speed.
// No rates here — that is the Network card's question.
const secondaryLinks = computed(() => {
  const primary = primaryNet.value?.iface;
  return netInterfaces.value.filter((iface) => iface.iface !== primary);
});

function speedLabel(iface) {
  return iface.speedMbps ? `${iface.speedMbps} Mbps` : "—";
}

function stateColor(state) {
  return state === "up"
    ? "var(--color-emerald-500)"
    : state === "down"
      ? "var(--color-rose-500)"
      : "var(--color-zinc-500)";
}
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-blue-200 bg-blue-50 transition-transform duration-200 group-hover:scale-105 dark:border-blue-900/50 dark:bg-blue-900/20">
          <EthernetPort class="h-5 w-5 text-blue-600 dark:text-blue-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black dark:group-hover:text-white">
            {{ t("quickMetrics.hostMetrics.linkStatus") }}
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
      <!-- Degraded: /proc/net/dev unreadable, same source the traffic card
           shares — degraded together or not at all. -->
      <div
        v-if="isDegraded('network')"
        class="flex items-center gap-3 rounded-xl border-red-200 bg-red-50 p-4 dark:border-red-900/50 dark:bg-red-900/20"
      >
        <EthernetPort class="h-5 w-5 shrink-0 text-red-500" />
        <div class="text-[11px] font-medium leading-tight text-red-600 dark:text-red-400">
          {{ unavailable }}
        </div>
      </div>

      <template v-else>
        <!-- The headline link: name and state on one row, the negotiation
             facts under it. The state dot is the first thing read from across
             the room; speed and duplex explain it on the way past. -->
        <div class="rounded-xl bg-zinc-50 p-4 dark:border-zinc-800 dark:bg-zinc-900/50">
          <div class="mb-3 flex items-center justify-between gap-3">
            <span class="min-w-0 truncate text-base font-black tracking-tight yantra-text-primary">
              {{ primaryNet?.iface ?? "—" }}
            </span>
            <span class="flex shrink-0 items-center gap-1.5">
              <span class="h-1.5 w-1.5 shrink-0 rounded-full" :style="{ backgroundColor: stateDotColor }"></span>
              <span class="text-[10px] font-black uppercase tracking-widest" :class="stateTextClass">
                {{ linkState }}
              </span>
            </span>
          </div>

          <div class="flex flex-col gap-2">
            <div class="flex items-center justify-between gap-3">
              <span class="text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
                {{ t("quickMetrics.hostMetrics.linkSpeed") }}
              </span>
              <span class="text-sm font-black tabular-nums tracking-tight yantra-text-primary">
                {{ speedText }}
              </span>
            </div>
            <div class="flex items-center justify-between gap-3">
              <span class="text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
                {{ t("quickMetrics.hostMetrics.linkDuplex") }}
              </span>
              <span class="text-sm font-black tabular-nums tracking-tight" :class="primaryNetDuplex === 'half' ? 'text-amber-600 dark:text-amber-400' : 'yantra-text-primary'">
                {{ duplexText }}
              </span>
            </div>
          </div>
        </div>

        <!-- The box's other links, the same two facts each. -->
        <div v-if="secondaryLinks.length > 0" class="mt-4 border-t border-zinc-200 pt-3 dark:border-zinc-800">
          <div class="mb-2 text-[10px] font-semibold uppercase tracking-widest yantra-text-secondary">
            {{ t("quickMetrics.hostMetrics.netOther") }}
          </div>
          <div class="flex flex-col gap-2">
            <div
              v-for="iface in secondaryLinks"
              :key="iface.iface"
              class="flex items-center gap-2 rounded-xl bg-zinc-50 p-2 dark:bg-zinc-900/50"
            >
              <span class="h-1.5 w-1.5 shrink-0 rounded-full" :style="{ backgroundColor: stateColor(iface.operState) }"></span>
              <span class="min-w-0 flex-1 truncate text-[11px] font-bold tracking-tight yantra-text-primary">{{ iface.iface }}</span>
              <span class="shrink-0 text-[9px] font-semibold tabular-nums yantra-text-secondary">{{ speedLabel(iface) }}</span>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>
