<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from "vue";
import { useI18n } from "vue-i18n";
({ colSpan: 1 });
import { Layers, Database, Box, HardDrive, Cpu, MemoryStick, Server, Activity } from "lucide-vue-next";
import { formatBytes } from "../utils/metrics";
import { useApiUrl } from "../composables/useApiUrl";

const { t } = useI18n();
const { apiUrl } = useApiUrl();

const systemInfo = ref(null);
const containers = ref([]);
const volumes = ref([]);
const images = ref([]);
const loading = ref(true);
const error = ref(null);
let refreshInterval = null;

const displayCores = ref(0);
const displayMemBytes = ref(0);
const displayStoragePercent = ref(0);

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

const runningApps = computed(() => containers.value.filter((container) => container.state === "running").length);
const totalVolumes = computed(() => volumes.value.length);
const imagesCount = computed(() => images.value.length);
const temporaryCount = computed(() => containers.value.filter((container) => container?.labels?.["yantr.expireAt"]).length);

const greeting = computed(() => {
  const hour = new Date().getHours();
  if (hour < 5) return t("home.overviewPulseCard.lateNightCoding");
  if (hour < 12) return t("home.overviewPulseCard.goodMorning");
  if (hour < 18) return t("home.overviewPulseCard.goodAfternoon");
  return t("home.overviewPulseCard.goodEvening");
});

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
    name: systemInfo.value.os.name.replace("Debian GNU/Linux", "Debian").replace("Ubuntu", "Ubuntu"),
    type: systemInfo.value.os.type,
    arch: systemInfo.value.os.arch || systemInfo.value.os.architecture,
    kernel: systemInfo.value.os.kernel,
  };
});

const storageInfo = computed(() => {
  if (!systemInfo.value?.storage) {
    return { used: 0, total: 0, percent: 0, usedFormatted: "0 B", totalFormatted: "0 B", hasData: false };
  }

  const { used, total } = systemInfo.value.storage;
  if (used && used > 0) {
    if (total && total > 0) {
      const percent = Math.round((used / total) * 100);
      return { used, total, percent, usedFormatted: formatBytes(used), totalFormatted: formatBytes(total), hasData: true };
    }
    return { used, total: 0, percent: 0, usedFormatted: formatBytes(used), totalFormatted: null, hasData: true };
  }

  return { used: 0, total: 0, percent: 0, usedFormatted: "0 B", totalFormatted: "0 B", hasData: false };
});

const displayMemFormatted = computed(() => formatBytes(displayMemBytes.value));

const displayMemParts = computed(() => {
  const [value = "0", unit = "B"] = displayMemFormatted.value.split(" ");
  return { value, unit };
});

const overviewStats = computed(() => [
  {
    key: "apps",
    label: t("home.overviewPulseCard.apps"),
    value: runningApps.value,
    icon: Layers,
    tone: "text-blue-500",
  },
  {
    key: "volumes",
    label: t("home.overviewPulseCard.volumes"),
    value: totalVolumes.value,
    icon: HardDrive,
    tone: "text-violet-500",
  },
  {
    key: "images",
    label: t("home.overviewPulseCard.images"),
    value: imagesCount.value,
    icon: Database,
    tone: "text-green-500",
  },
  {
    key: "temp",
    label: t("home.overviewPulseCard.temp"),
    value: temporaryCount.value,
    icon: Box,
    tone: "text-amber-500",
  },
]);

const hostStats = computed(() => [
  {
    key: "cpu",
    label: t("quickMetrics.hostMetrics.processors"),
    value: String(displayCores.value),
    suffix: t("quickMetrics.hostMetrics.cores"),
    icon: Cpu,
    tone: "text-blue-500",
  },
  {
    key: "memory",
    label: t("quickMetrics.hostMetrics.memory"),
    value: displayMemParts.value.value,
    suffix: displayMemParts.value.unit,
    icon: MemoryStick,
    tone: "text-violet-500",
  },
  {
    key: "storage",
    label: t("quickMetrics.hostMetrics.dockerVol"),
    value: storageInfo.value.total > 0
      ? `${storageInfo.value.usedFormatted} / ${storageInfo.value.totalFormatted}`
      : storageInfo.value.hasData
        ? storageInfo.value.usedFormatted
        : "0 B",
    suffix: storageInfo.value.total > 0 ? `${displayStoragePercent.value}%` : "",
    icon: HardDrive,
    tone: "text-emerald-500",
  },
]);

watch(systemInfo, (info) => {
  if (!info) return;
  countUpTo(displayCores, info.cpu?.cores ?? 0);
  countUpTo(displayMemBytes, info.memory?.total ?? 0, 1000);

  if (info.storage?.used > 0 && info.storage?.total > 0) {
    countUpTo(displayStoragePercent, Math.round((info.storage.used / info.storage.total) * 100));
  } else {
    displayStoragePercent.value = 0;
  }
});

async function fetchData() {
  try {
    const [systemRes, containerRes, volumeRes, imageRes] = await Promise.all([
      fetch(`${apiUrl.value}/api/system/info`),
      fetch(`${apiUrl.value}/api/containers`),
      fetch(`${apiUrl.value}/api/volumes`),
      fetch(`${apiUrl.value}/api/images`),
    ]);

    const [systemData, containerData, volumeData, imageData] = await Promise.all([
      systemRes.json(),
      containerRes.json(),
      volumeRes.json(),
      imageRes.json(),
    ]);

    if (systemData.success) {
      systemInfo.value = systemData.info;
      error.value = null;
    } else {
      error.value = systemData.error || "Failed to fetch system info";
    }

    if (containerData.success) containers.value = containerData.containers;
    if (volumeData.success) volumes.value = volumeData.volumes || [];
    if (imageData.success) images.value = imageData.images || [];
  } catch (err) {
    error.value = err.message;
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  fetchData();
  refreshInterval = setInterval(fetchData, 30000);
});

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval);
});
</script>

<template>
  <div class="group relative flex h-full flex-col overflow-hidden rounded-[1.4rem] bg-white dark:bg-[#0A0A0A] smooth-shadow transition-all duration-300 hover:-translate-y-0.5 hover:smooth-shadow-lg">
    <div class="absolute right-5 top-5 h-16 w-16 rounded-full bg-blue-500/6 blur-2xl transition-all duration-500 group-hover:scale-125 group-hover:bg-blue-500/10"></div>

    <div v-if="loading" class="relative z-10 flex min-h-56 flex-1 flex-col items-center justify-center gap-3 p-6 text-center">
      <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-gray-50 dark:bg-zinc-900 text-blue-500 transition-transform duration-500 group-hover:scale-105 group-hover:-rotate-6">
        <Server class="h-5 w-5 animate-pulse" />
      </div>
      <span class="text-[10px] font-semibold uppercase tracking-[0.18em] text-(--text-secondary)">{{ t("quickMetrics.hostMetrics.scanningHost") }}</span>
    </div>

    <div v-else-if="error" class="relative z-10 flex min-h-56 flex-1 flex-col justify-center p-6">
      <div class="rounded-[1.25rem] bg-red-500/10 p-5 text-red-600 dark:text-red-300">
        <div class="flex items-center gap-3">
          <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-red-500/12">
            <Activity class="h-4.5 w-4.5" />
          </div>
          <div class="min-w-0">
            <p class="text-[10px] font-bold uppercase tracking-[0.18em]">{{ t("quickMetrics.hostMetrics.connectionFailed") }}</p>
            <p class="mt-1 line-clamp-2 text-sm opacity-85 wrap-break-word">{{ error }}</p>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="relative z-10 flex h-full flex-col p-5 sm:p-6">
      <div class="flex items-start justify-between gap-4">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2 text-[10px] font-bold uppercase tracking-[0.18em] text-(--text-secondary)">
            <span>{{ t("quickMetrics.hostMetrics.hostSystem") }}</span>
            <span class="opacity-35">/</span>
            <span class="inline-flex items-center gap-1.5 text-emerald-600 dark:text-emerald-400">
              <span class="status-dot h-1.5 w-1.5 rounded-full bg-current"></span>
              {{ t("quickMetrics.hostMetrics.online") }}
            </span>
          </div>

          <h3 class="mt-3 truncate text-[1.45rem] font-semibold tracking-tight text-(--text-primary)">
            {{ osInfo.name }}
          </h3>
          <p class="mt-2 max-w-xl text-sm leading-relaxed text-(--text-secondary)">
            {{ greeting }}. {{ t("home.overviewPulseCard.stackRunningSmoothly") }}
          </p>
        </div>

        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-[1.1rem] bg-gray-50 dark:bg-zinc-900 text-blue-500 transition-all duration-500 group-hover:scale-105 group-hover:-rotate-6">
          <Server class="h-5 w-5" />
        </div>
      </div>

      <div class="mt-4 flex flex-wrap gap-2">
        <div class="inline-flex min-h-9 items-center gap-2 rounded-full bg-gray-50 dark:bg-zinc-900/70 px-3 py-2 text-[11px] font-medium text-(--text-secondary)">
          <Server class="h-3.5 w-3.5 text-blue-500" />
          <span>{{ osInfo.type }}</span>
        </div>
        <div class="inline-flex min-h-9 items-center gap-2 rounded-full bg-gray-50 dark:bg-zinc-900/70 px-3 py-2 text-[11px] font-medium text-(--text-secondary)">
          <Cpu class="h-3.5 w-3.5 text-violet-500" />
          <span>{{ osInfo.arch }}</span>
        </div>
        <div class="inline-flex min-h-9 items-center gap-2 rounded-full bg-gray-50 dark:bg-zinc-900/70 px-3 py-2 text-[11px] font-medium text-(--text-secondary)">
          <Activity class="h-3.5 w-3.5 text-emerald-500" />
          <span class="font-mono tracking-tight">{{ osInfo.kernel }}</span>
        </div>
      </div>

      <div class="mt-5 grid grid-cols-2 gap-2 sm:gap-3">
        <div
          v-for="stat in overviewStats"
          :key="stat.key"
          class="group/tile rounded-[1.15rem] bg-gray-50 dark:bg-zinc-900/70 px-3 py-3 transition-all duration-300 hover:-translate-y-0.5 hover:smooth-shadow"
        >
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <p class="text-[10px] font-bold uppercase tracking-[0.18em] text-(--text-secondary)">{{ stat.label }}</p>
              <p class="mt-2 truncate text-xl font-semibold tracking-tight text-(--text-primary)">{{ stat.value }}</p>
            </div>
            <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-2xl bg-white dark:bg-[#0A0A0A] transition-transform duration-300 group-hover/tile:scale-105 group-hover/tile:-rotate-6">
              <component :is="stat.icon" :class="['h-4 w-4 transition-transform duration-300 group-hover/tile:scale-110', stat.tone]" />
            </div>
          </div>
        </div>
      </div>

      <div class="mt-5 space-y-2">
        <div
          v-for="stat in hostStats"
          :key="stat.key"
          class="group/metric flex items-center justify-between gap-3 rounded-[1.15rem] bg-gray-50 dark:bg-zinc-900/70 px-3.5 py-3 transition-all duration-300 hover:-translate-y-0.5 hover:smooth-shadow"
        >
          <div class="flex min-w-0 items-center gap-3">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-white dark:bg-[#0A0A0A] transition-transform duration-300 group-hover/metric:scale-105 group-hover/metric:-rotate-6">
              <component :is="stat.icon" :class="['h-4.5 w-4.5 transition-transform duration-300 group-hover/metric:scale-110', stat.tone]" />
            </div>

            <div class="min-w-0">
              <p class="text-[10px] font-bold uppercase tracking-[0.18em] text-(--text-secondary)">{{ stat.label }}</p>
              <p class="mt-1 truncate text-sm font-semibold tracking-tight text-(--text-primary)">{{ stat.value }}</p>
            </div>
          </div>

          <div class="shrink-0 text-right">
            <p v-if="stat.suffix" class="text-sm font-semibold tracking-tight text-(--text-primary)">{{ stat.suffix }}</p>
            <p v-else class="text-sm font-semibold tracking-tight text-(--text-primary)">--</p>
          </div>
        </div>
      </div>

      <div class="mt-auto pt-5 text-[11px] text-(--text-secondary)">
        <div class="flex items-center justify-between gap-3 rounded-[1.15rem] bg-gray-50 dark:bg-zinc-900/70 px-3.5 py-3">
          <span class="inline-flex items-center gap-2 font-medium">
            <Activity class="h-3.5 w-3.5 text-emerald-500" />
            {{ t("home.overviewPulseCard.active") }} · {{ t("home.overviewPulseCard.healthy") }}
          </span>
          <span class="truncate font-mono text-[10px] tracking-tight">{{ t("home.overviewPulseCard.systemOnline") }}</span>
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