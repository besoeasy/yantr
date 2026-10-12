import { computed, onMounted, onUnmounted, ref, shallowRef } from "vue";
import { formatBitRate, formatBytes, formatUptime } from "../utils/metrics";
import { useApiUrl } from "./useApiUrl";
import { expectApiSuccess } from "./useApiResponse";

// ---------------------------------------------------------------------------
// Shared host telemetry for the Home dashboard cards.
//
// HostOverviewCard used to own all of this. Splitting it into one card per
// concern means three components now want the same host numbers, and each one
// polling for itself would triple the request rate — the metrics poll alone runs
// every 2s, and on a homelab that is real CPU and socket churn on the host.
//
// So the state below is module-scoped and the two pollers are started once,
// ref-counted against the number of live cards. The first card to mount starts
// them, the last one to unmount stops them.
//
// This does not use the shared usePolling composable, deliberately. That helper
// registers onUnmounted against whichever component instance created it, so a
// poller created by the first card would stop itself when *that* card unmounted,
// even while two sibling cards were still reading from it. It would also warn
// outright when called outside a component instance. The two guards that
// actually matter are kept — no overlapping ticks, and no polling in a hidden
// tab — and the rest of usePolling's API surface is unused here.
// ---------------------------------------------------------------------------

const SYSTEM_POLL_MS = 30000;
const METRICS_POLL_MS = 2000;
// 120 samples at 2s is a 4 minute window — long enough to read a trend,
// short enough that the ring buffer stays trivial to render.
const HISTORY_POINTS = 120;

const systemInfo = shallowRef(null);
const volumes = shallowRef([]);
const metrics = shallowRef(null);
const cpuHistory = shallowRef([]);

const loading = ref(true);
const error = ref(null);

// A metrics failure is not a card failure: the static system info stays valid,
// so this is tracked separately and only dims the live cards.
const metricsStale = ref(false);

let consumers = 0;
let systemTimer = null;
let metricsTimer = null;
let systemInFlight = false;
let metricsInFlight = false;
let hidden = typeof document !== "undefined" ? document.hidden : false;

const { apiUrl } = useApiUrl();

async function fetchSystem() {
  // Never let two ticks overlap: a slow endpoint would otherwise queue requests
  // and deliver them out of order.
  if (systemInFlight) return;
  systemInFlight = true;
  try {
    // /api/volumes walks every volume on the host and is far slower than the
    // few reads /api/system/info does, which is why the metrics poll below runs
    // on its own timer instead of sharing this one.
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
    systemInFlight = false;
  }
}

async function fetchMetrics() {
  if (metricsInFlight) return;
  metricsInFlight = true;
  try {
    const res = await fetch(`${apiUrl.value}/api/system/metrics`);
    const data = await expectApiSuccess(res, "Failed to fetch host metrics");
    metrics.value = data.metrics;
    metricsStale.value = false;

    // Only append once a real rate exists, so the line starts where the backend
    // started measuring rather than from a fake zero.
    if (data.metrics?.cpu?.ready) {
      cpuHistory.value = [...cpuHistory.value, data.metrics.cpu.totalPct].slice(-HISTORY_POINTS);
    }
  } catch {
    // The message is deliberately not surfaced: all this needs to say is "the
    // live readings are no longer live".
    metricsStale.value = true;
  } finally {
    metricsInFlight = false;
  }
}

function startPolling() {
  if (systemTimer === null) {
    fetchSystem();
    systemTimer = setInterval(fetchSystem, SYSTEM_POLL_MS);
  }
  if (metricsTimer === null) {
    fetchMetrics();
    metricsTimer = setInterval(fetchMetrics, METRICS_POLL_MS);
  }
}

function stopPolling() {
  clearInterval(systemTimer);
  clearInterval(metricsTimer);
  systemTimer = null;
  metricsTimer = null;
}

// Pause while the tab is hidden; catch up immediately when it comes back so the
// dashboard is not stale on return. Without this a backgrounded tab would still
// hit the host once every 2s.
function onVisibilityChange() {
  hidden = document.hidden;
  if (hidden) {
    clearInterval(systemTimer);
    clearInterval(metricsTimer);
    systemTimer = null;
    metricsTimer = null;
  } else {
    startPolling();
  }
}

// ---------------------------------------------------------------------------
// Derived values
// ---------------------------------------------------------------------------

// Counts come straight from the /api/system/info payload — no extra
// /api/containers or /api/images round-trips needed for these.
const podmanInfo = computed(() => systemInfo.value?.podman ?? {});
const runningCount = computed(() => podmanInfo.value?.containers?.running ?? 0);
const stoppedCount = computed(() => podmanInfo.value?.containers?.stopped ?? 0);
const imagesCount = computed(() => podmanInfo.value?.images ?? 0);
const totalVolumes = computed(() => volumes.value.length);
const podmanVersion = computed(() => (podmanInfo.value?.version ? `v${podmanInfo.value.version}` : "--"));

const osInfo = computed(() => {
  if (!systemInfo.value?.os) {
    return { name: "Unknown Host", type: "--", arch: "--", kernel: "--" };
  }

  return {
    name: systemInfo.value.os.name.replace("Debian GNU/Linux", "Debian"),
    type: systemInfo.value.os.type,
    arch: systemInfo.value.os.arch || systemInfo.value.os.architecture,
    kernel: systemInfo.value.os.kernel,
  };
});

const coreCount = computed(() => systemInfo.value?.cpu?.cores ?? 0);

const degraded = computed(() => new Set(metrics.value?.degraded ?? []));
const isDegraded = (name) => degraded.value.has(name);

const cpuPct = computed(() => {
  const pct = metrics.value?.cpu?.totalPct;
  return Number.isFinite(pct) ? Math.round(pct * 10) / 10 : null;
});

const coreBars = computed(() => {
  const cores = metrics.value?.cpu?.cores;
  if (!Array.isArray(cores)) return [];
  // Per-core bars stop being readable past this many cores, and a homelab box is
  // rarely smaller — hide rather than render a picket fence.
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

// The primary interface is the one carrying the most traffic; an idle docker0 or
// a down secondary NIC should not be what the cards report.
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

export function useHostMetrics() {
  onMounted(() => {
    consumers += 1;
    startPolling();
    if (consumers === 1 && typeof document !== "undefined") {
      document.addEventListener("visibilitychange", onVisibilityChange);
    }
  });

  onUnmounted(() => {
    consumers -= 1;
    if (consumers === 0) {
      stopPolling();
      if (typeof document !== "undefined") {
        document.removeEventListener("visibilitychange", onVisibilityChange);
      }
    }
  });

  return {
    // state
    systemInfo,
    volumes,
    metrics,
    cpuHistory,
    loading,
    error,
    metricsStale,
    // derived
    podmanInfo,
    podmanVersion,
    runningCount,
    stoppedCount,
    imagesCount,
    totalVolumes,
    osInfo,
    coreCount,
    degraded,
    isDegraded,
    cpuPct,
    coreBars,
    memPct,
    memText,
    loadText,
    uptimeText,
    primaryNet,
    netText,
    primaryDisk,
    diskText,
    diskFreeText,
    tempText,
  };
}