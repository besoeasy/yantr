<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from "vue";
import { useI18n } from "vue-i18n";
import { Cpu, Server, Activity } from "@lucide/vue";
import { formatBytes } from "../utils/metrics";
import { useApiUrl } from "../composables/useApiUrl";
import { expectApiSuccess } from "../composables/useApiResponse";

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

const workloadStats = computed(() => [
  {
    key: "apps",
    label: t("home.overviewPulseCard.apps"),
    value: runningApps.value,
    tone: "text-blue-500",
  },
  {
    key: "volumes",
    label: t("home.overviewPulseCard.volumes"),
    value: totalVolumes.value,
    tone: "text-violet-500",
  },
  {
    key: "images",
    label: t("home.overviewPulseCard.images"),
    value: imagesCount.value,
    tone: "text-green-500",
  },
  {
    key: "temp",
    label: t("home.overviewPulseCard.temp"),
    value: temporaryCount.value,
    tone: "text-amber-500",
  },
]);

const hostSummaryStats = computed(() => [
  {
    key: "cpu",
    label: t("quickMetrics.hostMetrics.processors"),
    value: String(displayCores.value),
    suffix: t("quickMetrics.hostMetrics.cores"),
    tone: "text-blue-500",
  },
  {
    key: "memory",
    label: t("quickMetrics.hostMetrics.memory"),
    value: displayMemParts.value.value,
    suffix: displayMemParts.value.unit,
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
      expectApiSuccess(systemRes, "Failed to fetch system info"),
      expectApiSuccess(containerRes, "Failed to fetch containers"),
      expectApiSuccess(volumeRes, "Failed to fetch volumes"),
      expectApiSuccess(imageRes, "Failed to fetch images"),
    ]);

    systemInfo.value = systemData.info;
    containers.value = Array.isArray(containerData.containers) ? containerData.containers : [];
    volumes.value = Array.isArray(volumeData.volumes) ? volumeData.volumes : [];
    images.value = Array.isArray(imageData.images) ? imageData.images : [];
    error.value = null;
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
  <div class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] rounded-xl overflow-hidden transition-all duration-300 hover:-translate-y-0.5 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 text-left w-full min-h-72">
    <!-- top accent line -->
    <div class="absolute top-0 left-0 w-full h-0.5 bg-blue-500 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>

    <div v-if="loading" class="relative z-10 flex flex-1 flex-col items-center justify-center gap-3 p-5 h-full text-center">
      <div class="w-9 h-9 rounded-lg flex items-center justify-center shrink-0">
        <Server class="w-4.5 h-4.5 text-blue-600 dark:text-blue-400 animate-pulse" />
      </div>
      <span class="text-[10px] font-bold uppercase tracking-[0.18em] text-gray-500 dark:text-zinc-500">{{ t("quickMetrics.hostMetrics.scanningHost") }}</span>
    </div>

    <div v-else-if="error" class="relative z-10 flex flex-1 flex-col gap-4 p-5 h-full">
      <div class="flex items-center gap-3 min-w-0">
        <div class="w-9 h-9 rounded-lg flex items-center justify-center shrink-0">
          <Activity class="w-4.5 h-4.5 text-red-600 dark:text-red-400" />
        </div>
        <div class="min-w-0">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight leading-none">{{ t("quickMetrics.hostMetrics.hostSystem") }}</h3>
          <span class="text-[10px] font-bold uppercase tracking-wider text-red-600 dark:text-red-400 mt-1.5 block">{{ t("quickMetrics.hostMetrics.connectionFailed") }}</span>
        </div>
      </div>
      <div class="mt-auto pt-6 pb-2">
        <div class="text-3xl font-black tracking-tighter text-gray-900 dark:text-white leading-[1.1]">
          {{ t("common.error") }}
        </div>
        <div class="text-sm font-medium text-gray-500 dark:text-zinc-400 mt-2 line-clamp-2">
          {{ error }}
        </div>
      </div>
    </div>

    <div v-else class="relative z-10 p-5 flex flex-col gap-4 h-full">
      <!-- header -->
      <div class="flex items-center justify-between gap-3">
        <div class="flex items-center gap-3 min-w-0">
          <div class="w-9 h-9 rounded-lg flex items-center justify-center shrink-0 transition-transform group-hover:scale-110">
            <Server class="w-4.5 h-4.5 text-blue-600 dark:text-blue-400" />
          </div>
          <div class="min-w-0">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight leading-none">{{ t("quickMetrics.hostMetrics.hostSystem") }}</h3>
            <div class="flex items-center gap-1.5 mt-1.5 text-emerald-600 dark:text-emerald-400">
              <div class="w-1.5 h-1.5 rounded-full bg-current animate-pulse"></div>
              <span class="text-[10px] font-bold uppercase tracking-wider">{{ t("quickMetrics.hostMetrics.online") }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="mt-auto pt-6 pb-2 flex flex-col relative z-10 pr-4">
        <div class="text-3xl sm:text-4xl md:text-5xl font-black tracking-tighter text-gray-900 dark:text-white group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors line-clamp-2 leading-[1.1] mb-2">
          {{ osInfo.name }}
        </div>
        
        <div class="text-xs text-gray-500 dark:text-zinc-400 line-clamp-1 truncate font-medium">
          {{ osInfo.type }} • {{ osInfo.arch }} • {{ osInfo.kernel }}
        </div>

        <div class="flex flex-wrap items-center gap-3 mt-3">
           <template v-for="stat in workloadStats" :key="stat.key">
              <div class="flex items-center gap-1.5 text-[11px] font-medium">
                 <span class="text-gray-400 dark:text-zinc-500 uppercase tracking-widest">{{ stat.label }}</span>
                 <span :class="['font-bold', stat.tone]">{{ stat.value }}</span>
              </div>
           </template>
        </div>

        <div class="grid grid-cols-3 gap-3 mt-4 pt-4">
           <div class="flex flex-col">
              <span class="text-[10px] font-bold text-gray-400 dark:text-zinc-500 uppercase tracking-widest mb-1">{{ t("quickMetrics.hostMetrics.processors") }}</span>
              <span class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight">{{ displayCores }} <span class="text-gray-400 dark:text-zinc-500 font-medium text-xs">{{ t("quickMetrics.hostMetrics.cores") }}</span></span>
           </div>
           <div class="flex flex-col">
              <span class="text-[10px] font-bold text-gray-400 dark:text-zinc-500 uppercase tracking-widest mb-1">{{ t("quickMetrics.hostMetrics.memory") }}</span>
              <span class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight">{{ displayMemParts.value }} <span class="text-gray-400 dark:text-zinc-500 font-medium text-xs">{{ displayMemParts.unit }}</span></span>
           </div>
           <div class="flex flex-col min-w-0">
              <span class="text-[10px] font-bold text-gray-400 dark:text-zinc-500 uppercase tracking-widest mb-1">{{ t("quickMetrics.hostMetrics.dockerVol") }}</span>
              <span class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight truncate">
                {{ storageInfo.usedFormatted }}
                <span v-if="storageInfo.total > 0" class="text-gray-400 dark:text-zinc-500 font-medium text-xs">/ {{ displayStoragePercent }}%</span>
              </span>
           </div>
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