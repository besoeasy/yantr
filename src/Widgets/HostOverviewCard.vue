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
  <div class="relative group h-full flex flex-col rounded-[24px] overflow-hidden transition-all duration-500 hover:-translate-y-2 hover:shadow-[0_20px_40px_-15px_rgba(59,130,246,0.4)] dark:hover:shadow-[0_20px_40px_-15px_rgba(59,130,246,0.3)] text-left w-full min-h-[320px] border border-blue-100/50 dark:border-white/5 bg-gradient-to-br from-blue-50/50 to-indigo-50/50 dark:from-[#111827] dark:to-[#09090b]">
    
    <!-- Animated Gradient Glow -->
    <div class="absolute -top-[50%] -left-[50%] w-[200%] h-[200%] bg-[radial-gradient(circle_at_50%_50%,_rgba(59,130,246,0.15)_0%,_transparent_50%)] dark:bg-[radial-gradient(circle_at_50%_50%,_rgba(59,130,246,0.12)_0%,_transparent_40%)] opacity-0 group-hover:opacity-100 transition-opacity duration-1000 pointer-events-none group-hover:animate-[spin_20s_linear_infinite]"></div>

    <!-- Massive Watermark Icon -->
    <Server class="absolute -bottom-8 -right-8 w-64 h-64 text-blue-500/10 dark:text-blue-400/[0.03] rotate-[-20deg] group-hover:rotate-0 group-hover:scale-110 transition-all duration-700 ease-out pointer-events-none" />

    <div v-if="loading" class="relative z-10 flex flex-1 flex-col items-center justify-center gap-3 p-7 h-full text-center">
      <div class="w-14 h-14 rounded-2xl flex items-center justify-center shrink-0 bg-blue-500/10 dark:bg-blue-500/20">
        <Server class="w-7 h-7 text-blue-600 dark:text-blue-400 animate-pulse" />
      </div>
      <span class="text-[10px] font-bold uppercase tracking-[0.18em] text-blue-500/60 dark:text-zinc-500">{{ t("quickMetrics.hostMetrics.scanningHost") }}</span>
    </div>

    <div v-else-if="error" class="relative z-10 flex flex-1 flex-col gap-4 p-7 h-full">
      <div class="flex items-center gap-4">
          <div class="relative w-14 h-14 rounded-2xl flex items-center justify-center shrink-0 bg-gradient-to-br from-red-400 to-rose-600 shadow-xl shadow-red-500/30">
            <div class="absolute inset-0 bg-white/30 dark:bg-white/20 rounded-2xl rounded-bl-none opacity-50 dark:opacity-40 mix-blend-overlay"></div>
            <Activity class="w-7 h-7 text-white drop-shadow-md z-10" />
          </div>
          <div>
            <h3 class="text-xl font-black text-red-950 dark:text-white tracking-tight leading-none">{{ t("quickMetrics.hostMetrics.hostSystem") }}</h3>
            <p class="text-[10px] font-bold uppercase tracking-[0.2em] text-red-600/70 dark:text-red-400 mt-2">{{ t("quickMetrics.hostMetrics.connectionFailed") }}</p>
          </div>
      </div>
      <div class="mt-auto pt-6 pb-2">
        <div class="text-4xl font-black tracking-tighter text-red-950 dark:text-white leading-[1.1]">
          {{ t("common.error") }}
        </div>
        <div class="text-sm font-medium text-red-800/60 dark:text-zinc-400 mt-2 line-clamp-2">
          {{ error }}
        </div>
      </div>
    </div>

    <div v-else class="relative z-10 p-7 flex flex-col gap-4 h-full">
      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Gorgeous Icon Block -->
          <div class="relative w-14 h-14 rounded-2xl flex items-center justify-center shrink-0 bg-gradient-to-br from-blue-400 to-indigo-600 shadow-xl shadow-blue-500/30 group-hover:shadow-blue-500/50 transition-all duration-500 group-hover:scale-105">
            <div class="absolute inset-0 bg-white/30 dark:bg-white/20 rounded-2xl rounded-bl-none opacity-50 dark:opacity-40 mix-blend-overlay"></div>
            <Server class="w-7 h-7 text-white drop-shadow-md z-10" />
          </div>
          <div>
            <h3 class="text-xl font-black text-blue-950 dark:text-white tracking-tight leading-none group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
              {{ t("quickMetrics.hostMetrics.hostSystem") }}
            </h3>
            <div class="flex items-center gap-1.5 mt-2">
              <div class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></div>
              <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-emerald-600/90 dark:text-emerald-500">{{ t("quickMetrics.hostMetrics.online") }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Main Body -->
      <div class="mt-auto pt-4 flex flex-col relative z-10">
        
        <div class="text-4xl sm:text-5xl font-black tracking-tighter text-blue-950 dark:text-white group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors line-clamp-2 leading-[1.1] mb-2 drop-shadow-sm group-hover:-translate-y-1 duration-500">
          {{ osInfo.name }}
        </div>
        
        <div class="text-[11px] font-bold uppercase tracking-[0.15em] text-blue-700/60 dark:text-zinc-500 line-clamp-1 truncate drop-shadow-sm mb-4">
          {{ osInfo.type }} • {{ osInfo.arch }} • {{ osInfo.kernel }}
        </div>

        <!-- Nested Glass Containers -->
        <div class="grid grid-cols-3 gap-2">
           <div class="col-span-3 grid grid-cols-4 gap-2">
              <!-- Workload stats pills -->
              <div v-for="stat in workloadStats" :key="stat.key" class="p-2 rounded-[12px] bg-white/60 dark:bg-black/40 backdrop-blur-xl border border-white dark:border-white/5 flex flex-col items-center justify-center gap-1 shadow-[0_4px_12px_rgba(0,0,0,0.02)] dark:shadow-sm">
                 <span class="text-[8px] font-bold text-gray-500 dark:text-zinc-500 uppercase tracking-widest text-center leading-none">{{ stat.label }}</span>
                 <span :class="['text-sm font-black', stat.tone]">{{ stat.value }}</span>
              </div>
           </div>
           
           <div class="col-span-3 grid grid-cols-3 gap-2 mt-2">
              <div class="p-3 rounded-[16px] bg-gradient-to-r from-blue-100/50 to-indigo-100/50 dark:from-blue-950/40 dark:to-indigo-950/40 backdrop-blur-xl border border-white dark:border-blue-900/50 flex flex-col gap-1 shadow-[0_4px_12px_rgba(59,130,246,0.05)] dark:shadow-none">
                  <span class="text-[9px] font-bold text-blue-700/60 dark:text-zinc-500 uppercase tracking-widest">{{ t("quickMetrics.hostMetrics.processors") }}</span>
                  <span class="text-sm font-black text-blue-950 dark:text-white tracking-tight">{{ displayCores }} <span class="text-blue-700/60 dark:text-zinc-500 font-bold text-[10px] uppercase tracking-widest">{{ t("quickMetrics.hostMetrics.cores") }}</span></span>
              </div>
              <div class="p-3 rounded-[16px] bg-gradient-to-r from-blue-100/50 to-indigo-100/50 dark:from-blue-950/40 dark:to-indigo-950/40 backdrop-blur-xl border border-white dark:border-blue-900/50 flex flex-col gap-1 shadow-[0_4px_12px_rgba(59,130,246,0.05)] dark:shadow-none">
                  <span class="text-[9px] font-bold text-blue-700/60 dark:text-zinc-500 uppercase tracking-widest">{{ t("quickMetrics.hostMetrics.memory") }}</span>
                  <span class="text-sm font-black text-blue-950 dark:text-white tracking-tight">{{ displayMemParts.value }} <span class="text-blue-700/60 dark:text-zinc-500 font-bold text-[10px] uppercase tracking-widest">{{ displayMemParts.unit }}</span></span>
              </div>
              <div class="p-3 rounded-[16px] bg-gradient-to-r from-blue-100/50 to-indigo-100/50 dark:from-blue-950/40 dark:to-indigo-950/40 backdrop-blur-xl border border-white dark:border-blue-900/50 flex flex-col gap-1 shadow-[0_4px_12px_rgba(59,130,246,0.05)] dark:shadow-none truncate">
                  <span class="text-[9px] font-bold text-blue-700/60 dark:text-zinc-500 uppercase tracking-widest">{{ t("quickMetrics.hostMetrics.dockerVol") }}</span>
                  <span class="text-sm font-black text-blue-950 dark:text-white tracking-tight truncate">
                    {{ storageInfo.usedFormatted }}
                    <span v-if="storageInfo.total > 0" class="text-blue-700/60 dark:text-zinc-500 font-bold text-[10px] uppercase tracking-widest">/ {{ displayStoragePercent }}%</span>
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