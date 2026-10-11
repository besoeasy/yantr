<script setup>
import { ref, computed, onMounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import { Server, Activity } from "@lucide/vue";
import { formatBytes, formatBitRate, formatUptime } from "../utils/metrics";
import { useApiUrl } from "../composables/useApiUrl";
import { expectApiSuccess } from "../composables/useApiResponse";
import { usePolling } from '../composables/usePolling'
import Sparkline from '../components/Sparkline.vue';

const { t } = useI18n();
const { apiUrl } = useApiUrl();

const systemInfo = ref(null);
const volumes = ref([]);
const loading = ref(true);
const error = ref(null);

const displayCores = ref(0);

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

// Counts come straight from the /api/system/info payload — no extra
// /api/containers or /api/images round-trips needed for this card.
const podmanInfo = computed(() => systemInfo.value?.podman ?? {});
const runningCount = computed(() => podmanInfo.value?.containers?.running ?? 0);
const stoppedCount = computed(() => podmanInfo.value?.containers?.stopped ?? 0);
const imagesCount = computed(() => podmanInfo.value?.images ?? 0);
const totalVolumes = computed(() => volumes.value.length);
const podmanVersion = computed(() => (podmanInfo.value?.version ? `v${podmanInfo.value.version}` : "--"));

const osInfo = computed(() => {
  if (!systemInfo.value?.os) {
    return {
      name: "Unknown Host",
      type: "--",
      arch: "--",
      kernel: "--",
    };
  }

  return {
    name: systemInfo.value.os.name.replace("Debian GNU/Linux", "Debian"),
    type: systemInfo.value.os.type,
    arch: systemInfo.value.os.arch || systemInfo.value.os.architecture,
    kernel: systemInfo.value.os.kernel,
  };
});

const workloadStats = computed(() => [
  {
    key: "apps",
    label: t("home.overviewPulseCard.apps"),
    value: runningCount.value,
    tone: "text-sky-600 dark:text-sky-400",
  },
  {
    key: "volumes",
    label: t("home.overviewPulseCard.volumes"),
    value: totalVolumes.value,
    tone: "text-sky-600 dark:text-sky-400",
  },
  {
    key: "images",
    label: t("home.overviewPulseCard.images"),
    value: imagesCount.value,
    tone: "text-sky-600 dark:text-sky-400",
  },
  {
    key: "stopped",
    label: t("home.overviewPulseCard.stopped"),
    value: stoppedCount.value,
    tone: "text-zinc-500 dark:text-zinc-400",
  },
]);

watch(systemInfo, (info) => {
  if (!info) return;
  countUpTo(displayCores, info.cpu?.cores ?? 0);
});

async function fetchData() {
  try {
    const [systemRes, volumeRes] = await Promise.all([
      fetch(`${apiUrl.value}/api/system/info`),
      fetch(`${apiUrl.value}/api/volumes`),
    ]);

    const [systemData, volumeData] = await Promise.all([
      expectApiSuccess(systemRes, "Failed to fetch system info"),
      expectApiSuccess(volumeRes, "Failed to fetch volumes"),
    ]);

    systemInfo.value = systemData.info;
    volumes.value = Array.isArray(volumeData.volumes) ? volumeData.volumes : [];
    error.value = null;
  } catch (err) {
    error.value = err.message;
  } finally {
    loading.value = false;
  }
}

// ---------------------------------------------------------------------------
// Live host metrics
//
// Polled on their own timer, deliberately separate from the 30s fetchData
// above: /api/volumes walks every volume on the host and is far slower than a
// few procfs reads, so sharing a timer would make the sparkline stutter every
// time a volume list lands.
// ---------------------------------------------------------------------------

const METRICS_POLL_MS = 2000;
// 120 samples at 2s is a 4 minute window — long enough to read a trend,
// short enough that the ring buffer stays trivial to render.
const HISTORY_POINTS = 120;

const metrics = ref(null);
const cpuHistory = ref([]);

// A metrics failure is not a card failure: the static system info above stays
// valid, so this is tracked separately and only dims the live strip.
const metricsStale = ref(false);

async function fetchMetrics() {
  try {
    const res = await fetch(`${apiUrl.value}/api/system/metrics`);
    const data = await expectApiSuccess(res, "Failed to fetch host metrics");
    metrics.value = data.metrics;
    metricsStale.value = false;

    // Only append once a real rate exists, so the line starts where the
    // backend started measuring rather than from a fake zero.
    if (data.metrics?.cpu?.ready) {
      cpuHistory.value = [...cpuHistory.value, data.metrics.cpu.totalPct].slice(-HISTORY_POINTS);
    }
  } catch {
    // The message is deliberately not surfaced: usePolling already tracks the
    // error, and all this needs to say is "the live strip is no longer live".
    metricsStale.value = true;
  }
}

const degraded = computed(() => new Set(metrics.value?.degraded ?? []));
const isDegraded = (name) => degraded.value.has(name);

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));

const cpuPct = computed(() => {
  const pct = metrics.value?.cpu?.totalPct;
  return Number.isFinite(pct) ? Math.round(pct * 10) / 10 : null;
});

const coreBars = computed(() => {
  const cores = metrics.value?.cpu?.cores;
  if (!Array.isArray(cores)) return [];
  // Per-core bars stop being readable past this many cores, and a homelab box
  // is rarely smaller — hide rather than render a picket fence.
  return cores.length > 0 && cores.length <= 16 ? cores : [];
});

const memPct = computed(() => {
  const pct = metrics.value?.mem?.usedPct;
  return Number.isFinite(pct) ? pct : null;
});

const memText = computed(() => {
  const mem = metrics.value?.mem;
  if (!mem?.ready) return null;
  return `${formatBytes(mem.usedBytes)} / ${formatBytes(mem.totalBytes)}`;
});

const loadText = computed(() => {
  const load = metrics.value?.load;
  if (!load?.ready) return null;
  return `${load.one.toFixed(2)} · ${load.five.toFixed(2)} · ${load.fifteen.toFixed(2)}`;
});

const uptimeText = computed(() => {
  const secs = metrics.value?.uptimeSec;
  return Number.isFinite(secs) && secs > 0 ? formatUptime(secs * 1000) : null;
});

// The primary interface is the one carrying the most traffic; an idle docker0
// or a down secondary NIC should not be what the card reports.
const primaryNet = computed(() => {
  const net = metrics.value?.net;
  if (!Array.isArray(net) || net.length === 0) return null;
  return [...net].sort((a, b) => b.rxBps + b.txBps - (a.rxBps + a.txBps))[0];
});

const netText = computed(() => {
  const net = primaryNet.value;
  if (!net) return null;
  return `${formatBitRate(net.rxBps)} · ${formatBitRate(net.txBps)}`;
});

const primaryDisk = computed(() => {
  const disks = metrics.value?.disk;
  return Array.isArray(disks) && disks.length > 0 ? disks[0] : null;
});

const diskText = computed(() => {
  const disk = primaryDisk.value;
  if (!disk) return null;
  return `${disk.mount} ${Math.round(disk.usedPct)}%`;
});

const diskFreeText = computed(() => {
  const disk = primaryDisk.value;
  return disk ? formatBytes(disk.freeBytes) : null;
});

const tempText = computed(() => {
  const celsius = metrics.value?.tempC;
  return Number.isFinite(celsius) ? `${Math.round(celsius)}°C` : null;
});

const { start: startHostOverviewCardPolling } = usePolling(fetchData, 30000)
const { start: startMetricsPolling } = usePolling(fetchMetrics, METRICS_POLL_MS)

onMounted(() => {
  fetchData();
  startHostOverviewCardPolling();
  startMetricsPolling();
});
</script>

<template>
  <div class="group relative flex h-full w-full min-h-[420px] flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-all duration-300 hover:-translate-y-1 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">

    <div v-if="loading" class="relative z-10 flex h-full flex-1 flex-col items-center justify-center gap-3 text-center">
      <div class="flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl bg-zinc-50 dark:bg-zinc-900/50">
        <Server class="h-7 w-7 text-zinc-400 animate-pulse dark:text-zinc-500" />
      </div>
      <span class="text-[10px] font-bold uppercase tracking-[0.18em] text-zinc-500">{{ t("quickMetrics.hostMetrics.scanningHost") }}</span>
    </div>

    <div v-else-if="error" class="relative z-10 flex h-full flex-1 flex-col gap-4">
      <div class="flex items-center gap-4">
          <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-red-200 bg-red-50 dark:border-red-900/50 dark:bg-red-900/20">
            <Activity class="h-5 w-5 text-red-500" />
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
          <!-- Minimal Icon Block -->
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

        <div class="mb-2 text-4xl font-black leading-[1.1] tracking-tighter text-zinc-900 transition-colors duration-500 group-hover:-translate-y-1 sm:text-5xl dark:text-white line-clamp-2">
          {{ osInfo.name }}
        </div>

        <div class="mb-4 truncate text-[11px] font-semibold uppercase tracking-widest text-zinc-500">
          {{ osInfo.type }} • {{ osInfo.arch }} • {{ osInfo.kernel }}
        </div>

        <!-- Nested Minimal Containers -->
        <div class="grid grid-cols-3 gap-2">
           <div class="col-span-3 grid grid-cols-4 gap-2">
              <!-- Workload stats pills -->
              <div v-for="stat in workloadStats" :key="stat.key" class="flex flex-col items-center justify-center gap-1 rounded-lg border border-zinc-200 bg-zinc-50 p-2 dark:border-zinc-800 dark:bg-zinc-900/50">
                 <span class="text-center text-[8px] font-bold uppercase leading-none tracking-widest text-zinc-500">{{ stat.label }}</span>
                 <span class="text-sm font-black" :class="stat.tone">{{ stat.value }}</span>
              </div>
           </div>

           <div class="col-span-3 grid grid-cols-3 gap-2 mt-2">
              <div class="flex flex-col gap-1 rounded-xl border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
                  <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.processors") }}</span>
                  <span class="text-sm font-black tracking-tight text-zinc-900 dark:text-white">{{ displayCores }} <span class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.cores") }}</span></span>
              </div>
              <!-- Memory is live: MemTotal never changes, so animating it was
                   counting up to a constant. -->
              <div class="flex flex-col gap-1 rounded-xl border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
                  <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.memory") }}</span>
                  <span class="text-sm font-black tabular-nums tracking-tight text-zinc-900 dark:text-white">
                    {{ isDegraded("memory") ? unavailable : (memPct !== null ? `${Math.round(memPct)}%` : "—") }}
                  </span>
                  <div class="h-1 w-full overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
                    <div
                      class="h-full rounded-full bg-sky-500 transition-[width] duration-500 dark:bg-sky-400"
                      :style="{ width: `${memPct ?? 0}%` }"
                    ></div>
                  </div>
                  <span class="truncate text-[9px] font-semibold tabular-nums text-zinc-500">{{ memText ?? "" }}</span>
              </div>
              <div class="flex flex-col gap-1 truncate rounded-xl border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
                  <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.podman") }}</span>
                  <span class="truncate text-sm font-black tracking-tight text-zinc-900 dark:text-white">
                    {{ podmanVersion }}
                  </span>
              </div>
           </div>
        </div>
      </div>

      <!-- Live host metrics. Every tile degrades to an explicit "unavailable"
           rather than a zero: a missing /host bind has to be visibly missing,
           not silently render as an empty disk. The whole strip dims while the
           metrics poll is failing so a stale reading never looks live. -->
      <div
        class="relative z-10 mt-4 grid grid-cols-1 gap-2 border-t border-zinc-200 pt-4 transition-opacity duration-500 sm:grid-cols-2 lg:grid-cols-3 dark:border-zinc-800"
        :class="metricsStale ? 'opacity-50' : ''"
      >
        <!-- CPU with sparkline. Spans two columns because a line chart needs
             the horizontal room more than the tiles do. -->
        <div class="flex flex-col gap-2 rounded-xl border border-zinc-200 bg-zinc-50 p-3 sm:col-span-2 dark:border-zinc-800 dark:bg-zinc-900/50">
          <div class="flex items-baseline justify-between">
            <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.cpu") }}</span>
            <span class="text-sm font-black tabular-nums tracking-tight text-zinc-900 dark:text-white">
              {{ cpuPct !== null ? `${cpuPct}%` : "—" }}
            </span>
          </div>
          <div class="h-8 w-full">
            <Sparkline :points="cpuHistory" :max="100" :label="t('quickMetrics.hostMetrics.cpuHistory')" />
          </div>
          <div v-if="coreBars.length" class="flex h-4 items-end gap-0.5" aria-hidden="true">
            <div
              v-for="(core, i) in coreBars"
              :key="i"
              class="flex-1 rounded-t-sm bg-sky-500/70 dark:bg-sky-400/70"
              :style="{ height: `${Math.max(core, 2)}%` }"
            ></div>
          </div>
        </div>

        <!-- Load, with uptime underneath: both are scalars the host always has. -->
        <div class="flex flex-col gap-1 rounded-xl border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
          <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.load") }}</span>
          <span class="text-sm font-black tabular-nums tracking-tight text-zinc-900 dark:text-white">
            {{ isDegraded("load") ? unavailable : (loadText ?? "—") }}
          </span>
          <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.uptime") }}</span>
          <span class="truncate text-sm font-black tracking-tight text-zinc-900 dark:text-white">{{ uptimeText ?? "—" }}</span>
        </div>

        <!-- Disk -->
        <div class="flex flex-col gap-1 rounded-xl border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
          <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.disk") }}</span>
          <span class="text-sm font-black tracking-tight text-zinc-900 dark:text-white">
            {{ isDegraded("disk") ? unavailable : (diskText ?? "—") }}
          </span>
          <div v-if="!isDegraded('disk') && primaryDisk" class="mt-1 h-1 w-full overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
            <div
              class="h-full rounded-full bg-sky-500 transition-[width] duration-500 dark:bg-sky-400"
              :style="{ width: `${primaryDisk.usedPct}%` }"
            ></div>
          </div>
          <span class="truncate text-[9px] font-semibold text-zinc-500">
            {{ diskFreeText ? t("quickMetrics.hostMetrics.free", { size: diskFreeText }) : "" }}
          </span>
        </div>

        <!-- Network -->
        <div class="flex flex-col gap-1 rounded-xl border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
          <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.network") }}</span>
          <span class="truncate text-[9px] font-bold uppercase tracking-widest text-zinc-500">{{ primaryNet?.iface ?? "" }}</span>
          <span class="truncate text-sm font-black tabular-nums tracking-tight text-zinc-900 dark:text-white">
            {{ isDegraded("network") ? unavailable : (netText ?? "—") }}
          </span>
          <span class="truncate text-[9px] font-semibold text-zinc-500">{{ t("quickMetrics.hostMetrics.rxTx") }}</span>
        </div>

        <!-- Temperature -->
        <div class="flex flex-col gap-1 rounded-xl border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
          <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.temperature") }}</span>
          <span class="text-sm font-black tabular-nums tracking-tight text-zinc-900 dark:text-white">{{ tempText ?? unavailable }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
@media (prefers-reduced-motion: reduce) {
  .animate-ping {
    animation: none !important;
  }
}
</style>