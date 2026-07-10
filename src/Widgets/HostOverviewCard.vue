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
    tone: "text-emerald-500",
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
  <div class="group relative flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-all duration-300 hover:-translate-y-1 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    
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
          <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 transition-transform duration-300 group-hover:scale-105 dark:border-zinc-800 dark:bg-zinc-900">
            <Server class="h-5 w-5 text-zinc-900 dark:text-zinc-100" />
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
                 <span class="text-sm font-black text-zinc-900 dark:text-white">{{ stat.value }}</span>
              </div>
           </div>
           
           <div class="col-span-3 grid grid-cols-3 gap-2 mt-2">
              <div class="flex flex-col gap-1 rounded-xl border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
                  <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.processors") }}</span>
                  <span class="text-sm font-black tracking-tight text-zinc-900 dark:text-white">{{ displayCores }} <span class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.cores") }}</span></span>
              </div>
              <div class="flex flex-col gap-1 rounded-xl border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
                  <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.memory") }}</span>
                  <span class="text-sm font-black tracking-tight text-zinc-900 dark:text-white">{{ displayMemParts.value }} <span class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ displayMemParts.unit }}</span></span>
              </div>
              <div class="flex flex-col gap-1 truncate rounded-xl border border-zinc-200 bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
                  <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-500">{{ t("quickMetrics.hostMetrics.dockerVol") }}</span>
                  <span class="truncate text-sm font-black tracking-tight text-zinc-900 dark:text-white">
                    {{ storageInfo.usedFormatted }}
                    <span v-if="storageInfo.total > 0" class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">/ {{ displayStoragePercent }}%</span>
                  </span>
              </div>
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