<script setup>
import { ref, computed, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import { Store, LayoutGrid, PackageCheck, Container, FolderOpen, Activity } from "lucide-vue-next";
import { formatDuration } from "../utils/metrics";
import { useApiUrl } from "../composables/useApiUrl";
import YantraContainersGrid from "../components/home/YantraContainersGrid.vue";
import VolumeContainersGrid from "../components/home/VolumeContainersGrid.vue";
import OtherContainersGrid from "../components/home/OtherContainersGrid.vue";
import SystemCleaner from "../components/SystemCleaner.vue";
import WatchtowerAlert from "../components/WatchtowerAlert.vue";
import OverviewPulseCard from "../components/home/OverviewPulseCard.vue";
import MachineIdentityCard from "../components/quick-metrics/MachineIdentityCard.vue";
import RotatingDiskUsageCard from "../components/quick-metrics/RotatingDiskUsageCard.vue";
import AverageUptimeCard from "../components/quick-metrics/AverageUptimeCard.vue";
import ExpiringContainersCard from "../components/quick-metrics/ExpiringContainersCard.vue";
import WatchtowerNextCheckCard from "../components/quick-metrics/WatchtowerNextCheckCard.vue";
import HostMetricsCard from "../components/quick-metrics/HostMetricsCard.vue";
import MinioStatusCard from "../components/quick-metrics/MinioStatusCard.vue";

const router = useRouter();
const { apiUrl } = useApiUrl();

const containers = ref([]);
const volumes = ref([]);
const images = ref([]);
const loading = ref(false);
const currentTime = ref(Date.now());
const watchtowerInstalled = ref(false);
const activeFilter = ref("all");

let containersRefreshInterval = null;
let timeRefreshInterval = null;

// Metrics computed properties
const totalApps = computed(() => containers.value.length);
const runningApps = computed(() => containers.value.filter((c) => c.state === "running").length);
const totalVolumes = computed(() => volumes.value.length);

// System Cleaner Visibility
const reclaimableStats = computed(() => {
  if (!images.value || !volumes.value) return { show: false, stats: null };

  const unusedImages = images.value.filter((i) => !i.isUsed);
  const unusedVolumes = volumes.value.filter((v) => !v.isUsed);

  const imageSize = unusedImages.reduce((sum, img) => sum + (img.sizeBytes || 0), 0);
  const volumeSize = unusedVolumes.reduce((sum, vol) => sum + (vol.sizeBytes || 0), 0);

  const totalReclaimable = imageSize + volumeSize;
  const threshold = 100 * 1024 * 1024; // 100 MB

  return {
    show: totalReclaimable > threshold,
    imageStats: {
      unusedCount: unusedImages.length,
      unusedSize: imageSize,
      totalSize: images.value.reduce((sum, img) => sum + (img.sizeBytes || 0), 0),
    },
    volumeStats: {
      unusedCount: unusedVolumes.length,
      unusedSize: volumeSize,
      totalSize: volumes.value.reduce((sum, vol) => sum + (vol.sizeBytes || 0), 0),
    },
  };
});

// Watchtower Visibility
const showWatchtowerAlert = computed(() => !watchtowerInstalled.value);

// Container Grouping
const volumeContainers = computed(() => {
  return containers.value.filter((c) => c.labels && c.labels["yantra.volume-browser"]);
});

const yantraContainers = computed(() => {
  return containers.value.filter((c) => {
    const isVolume = c.labels && c.labels["yantra.volume-browser"];
    const isYantraApp = c.app || (c.labels && c.labels["yantra.app.id"]);
    return !isVolume && isYantraApp;
  });
});

const otherContainers = computed(() => {
  return containers.value.filter((c) => {
    const isVolume = c.labels && c.labels["yantra.volume-browser"];
    const isYantraApp = c.app || (c.labels && c.labels["yantra.app.id"]);
    return !isVolume && !isYantraApp;
  });
});

const temporaryContainersCount = computed(() => containers.value.filter((c) => c?.labels?.["yantra.expireAt"]).length);

// Filter visibility computed properties
const showYantraApps = computed(() => activeFilter.value === "all" || activeFilter.value === "yantra");
const showDockerApps = computed(() => activeFilter.value === "all" || activeFilter.value === "docker");
const showVolumeBrowsers = computed(() => activeFilter.value === "all" || activeFilter.value === "volumes");
const showMetrics = computed(() => activeFilter.value === "all" || activeFilter.value === "metrics");

// Helper function to format time remaining
function formatTimeRemaining(expireAt) {
  const expirationTime = parseInt(expireAt, 10) * 1000; // Convert to milliseconds
  const remaining = expirationTime - currentTime.value;

  if (remaining <= 0) return "Expired";

  const hours = Math.floor(remaining / (1000 * 60 * 60));
  const minutes = Math.floor((remaining % (1000 * 60 * 60)) / (1000 * 60));

  if (hours > 24) {
    const days = Math.floor(hours / 24);
    return `${days}d ${hours % 24}h`;
  } else if (hours > 0) {
    return `${hours}h ${minutes}m`;
  } else {
    return `${minutes}m`;
  }
}

// Check if container is temporary
function isTemporary(container) {
  return container.labels && container.labels["yantra.expireAt"];
}

// Get expiration info
function getExpirationInfo(container) {
  if (!isTemporary(container)) return null;

  const expireAt = container.labels["yantra.expireAt"];
  return {
    expireAt,
    timeRemaining: formatTimeRemaining(expireAt),
    isExpiringSoon: parseInt(expireAt, 10) * 1000 - currentTime.value < 60 * 60 * 1000, // < 1 hour
  };
}

// Format container uptime
function formatUptime(container) {
  if (!container.created || container.state !== "running") return null;

  const createdTime = container.created * 1000;
  const uptime = currentTime.value - createdTime;

  if (uptime <= 0) return "Just started";

  return formatDuration(uptime);
}

async function fetchContainers() {
  try {
    const response = await fetch(`${apiUrl.value}/api/containers`);
    const data = await response.json();
    if (data.success) {
      containers.value = data.containers;

      watchtowerInstalled.value = data.containers.some(
        (c) => c.name?.toLowerCase().includes("watchtower") || c.Names?.some((name) => name.toLowerCase().includes("watchtower")),
      );
    }
  } catch (error) {
    console.error("Failed to fetch containers:", error);
  }
}

async function fetchVolumes() {
  try {
    const response = await fetch(`${apiUrl.value}/api/volumes`);
    const data = await response.json();
    if (data.success) {
      volumes.value = data.volumes || [];
    }
  } catch (error) {
    console.error("Failed to fetch volumes:", error);
  }
}

async function fetchImages() {
  try {
    const response = await fetch(`${apiUrl.value}/api/images`);
    const data = await response.json();
    if (data.success) {
      images.value = data.images || [];
    }
  } catch (error) {
    console.error("Failed to fetch images:", error);
  }
}

async function refreshAll() {
  await Promise.all([fetchContainers(), fetchVolumes(), fetchImages()]);
}

function viewContainerDetail(container) {
  router.push(`/containers/${container.id}`);
}

onMounted(async () => {
  loading.value = true;
  await refreshAll();
  loading.value = false;

  containersRefreshInterval = setInterval(refreshAll, 10000);

  timeRefreshInterval = setInterval(() => {
    currentTime.value = Date.now();
  }, 60000);
});

onUnmounted(() => {
  if (containersRefreshInterval) {
    clearInterval(containersRefreshInterval);
    containersRefreshInterval = null;
  }
  if (timeRefreshInterval) {
    clearInterval(timeRefreshInterval);
    timeRefreshInterval = null;
  }
});
</script>

<template>
  <div class="min-h-screen bg-gray-50/50 dark:bg-slate-900/60">
    <!-- Main Content -->
    <div class="p-3 lg:p-6">
      <div class="space-y-10">
        <!-- Loading State -->
        <div v-if="loading" class="flex flex-col items-center justify-center py-32">
          <div class="w-10 h-10 border-3 border-gray-200 border-t-blue-600 dark:border-slate-700 dark:border-t-blue-400 rounded-full animate-spin mb-4"></div>
          <div class="text-gray-400 dark:text-slate-400 font-medium animate-pulse">Syncing containers...</div>
        </div>

        <!-- Content -->
        <div v-else class="animate-fadeIn">
          <!-- Filter Tabs -->
          <div v-if="containers.length > 0" class="flex gap-2 overflow-x-auto pb-2 mb-6 scrollbar-hide">
            <button
              @click="activeFilter = 'all'"
              :class="[
                'px-4 py-2 rounded-xl font-semibold text-sm whitespace-nowrap transition-all flex items-center gap-2',
                activeFilter === 'all'
                  ? 'bg-gray-900 text-white dark:bg-slate-100 dark:text-slate-900 shadow-lg'
                  : 'bg-white dark:bg-slate-800 text-gray-600 dark:text-slate-400 hover:bg-gray-100 dark:hover:bg-slate-700',
              ]"
            >
              <LayoutGrid :size="16" />
              <span>All</span>
            </button>
            <button
              v-if="yantraContainers.length > 0"
              @click="activeFilter = 'yantra'"
              :class="[
                'px-4 py-2 rounded-xl font-semibold text-sm whitespace-nowrap transition-all flex items-center gap-2',
                activeFilter === 'yantra'
                  ? 'bg-gray-900 text-white dark:bg-slate-100 dark:text-slate-900 shadow-lg'
                  : 'bg-white dark:bg-slate-800 text-gray-600 dark:text-slate-400 hover:bg-gray-100 dark:hover:bg-slate-700',
              ]"
            >
              <PackageCheck :size="16" />
              <span>Yantra Apps</span>
            </button>
            <button
              v-if="otherContainers.length > 0"
              @click="activeFilter = 'docker'"
              :class="[
                'px-4 py-2 rounded-xl font-semibold text-sm whitespace-nowrap transition-all flex items-center gap-2',
                activeFilter === 'docker'
                  ? 'bg-gray-900 text-white dark:bg-slate-100 dark:text-slate-900 shadow-lg'
                  : 'bg-white dark:bg-slate-800 text-gray-600 dark:text-slate-400 hover:bg-gray-100 dark:hover:bg-slate-700',
              ]"
            >
              <Container :size="16" />
              <span>Docker Apps</span>
            </button>
            <button
              v-if="volumeContainers.length > 0"
              @click="activeFilter = 'volumes'"
              :class="[
                'px-4 py-2 rounded-xl font-semibold text-sm whitespace-nowrap transition-all flex items-center gap-2',
                activeFilter === 'volumes'
                  ? 'bg-gray-900 text-white dark:bg-slate-100 dark:text-slate-900 shadow-lg'
                  : 'bg-white dark:bg-slate-800 text-gray-600 dark:text-slate-400 hover:bg-gray-100 dark:hover:bg-slate-700',
              ]"
            >
              <FolderOpen :size="16" />
              <span>Volume Browsers</span>
            </button>
            <button
              @click="activeFilter = 'metrics'"
              :class="[
                'px-4 py-2 rounded-xl font-semibold text-sm whitespace-nowrap transition-all flex items-center gap-2',
                activeFilter === 'metrics'
                  ? 'bg-gray-900 text-white dark:bg-slate-100 dark:text-slate-900 shadow-lg'
                  : 'bg-white dark:bg-slate-800 text-gray-600 dark:text-slate-400 hover:bg-gray-100 dark:hover:bg-slate-700',
              ]"
            >
              <Activity :size="16" />
              <span>Metrics</span>
            </button>
          </div>

          <!-- Empty State -->
          <div v-if="containers.length === 0" class="text-center py-24 bg-white dark:bg-slate-900/70 rounded-3xl shadow-sm dark:shadow-slate-950/60 mb-6">
            <div class="w-24 h-24 bg-gray-50 dark:bg-slate-800 rounded-full flex items-center justify-center mx-auto mb-6">
              <Store :size="40" class="text-gray-300 dark:text-slate-500" />
            </div>
            <h3 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">No Apps Running</h3>
            <p class="text-gray-500 dark:text-slate-400 max-w-md mx-auto mb-8">Your dashboard is looking a bit empty. Visit the App Store to get started.</p>
            <router-link
              to="/apps"
              class="inline-flex items-center gap-2 px-8 py-4 bg-gray-900 hover:bg-black text-white rounded-2xl font-bold transition-all hover:scale-105 shadow-xl shadow-gray-900/20 dark:bg-slate-100 dark:text-slate-900 dark:hover:bg-white dark:shadow-slate-900/20"
            >
              <Store :size="20" />
              <span>Browse App Store</span>
            </router-link>
          </div>

          <!-- Unified Dashboard Grid -->
          <div class="grid grid-cols-1 lg:grid-cols-3 xl:grid-cols-4 gap-3 lg:gap-4 xl:gap-3">
            <!-- Combined Greeting + Operations Pulse -->
            <div v-if="showMetrics" class="lg:col-span-2">
              <OverviewPulseCard
                :running-apps="runningApps"
                :total-volumes="totalVolumes"
                :temporary-count="temporaryContainersCount"
                :images-count="images.length"
              />
            </div>

            <div v-if="showMetrics && showWatchtowerAlert" class="h-full">
              <WatchtowerAlert />
            </div>

            <div v-else-if="showMetrics" class="h-full">
              <WatchtowerNextCheckCard :containers="containers" :current-time="currentTime" :interval-hours="3" />
            </div>

            <div v-if="showMetrics">
              <HostMetricsCard :api-url="apiUrl" />
            </div>

            <div v-if="showMetrics">
              <MinioStatusCard />
            </div>

            <YantraContainersGrid
              v-if="showYantraApps && yantraContainers.length > 0"
              :containers="yantraContainers"
              :format-uptime="formatUptime"
              :is-temporary="isTemporary"
              :get-expiration-info="getExpirationInfo"
              @select="viewContainerDetail"
            />

            <div v-if="showMetrics">
              <AverageUptimeCard :containers="containers" :current-time="currentTime" />
            </div>

            <VolumeContainersGrid
              v-if="showVolumeBrowsers && volumeContainers.length > 0"
              :containers="volumeContainers"
              :is-temporary="isTemporary"
              :get-expiration-info="getExpirationInfo"
              @select="viewContainerDetail"
            />

            <OtherContainersGrid v-if="showDockerApps && otherContainers.length > 0" :containers="otherContainers" @select="viewContainerDetail" />

            <div v-if="showMetrics && reclaimableStats.show" class="h-full lg:col-span-2 xl:col-span-2">
              <SystemCleaner
                :api-url="apiUrl"
                :initial-image-stats="reclaimableStats.imageStats"
                :initial-volume-stats="reclaimableStats.volumeStats"
                @cleaned="refreshAll"
              />
            </div>

            <div v-if="showMetrics && temporaryContainersCount > 0">
              <ExpiringContainersCard :containers="containers" :current-time="currentTime" />
            </div>

            <div v-if="showMetrics && (images.length > 0 || volumes.length > 0)">
              <RotatingDiskUsageCard :images="images" :volumes="volumes" :interval-ms="10000" />
            </div>

            <div v-if="showMetrics">
              <MachineIdentityCard />
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Smooth backdrop blur support */
@supports (backdrop-filter: blur(20px)) or (-webkit-backdrop-filter: blur(20px)) {
  .backdrop-blur-xl {
    -webkit-backdrop-filter: blur(20px);
    backdrop-filter: blur(20px);
  }

  .backdrop-blur-sm {
    -webkit-backdrop-filter: blur(4px);
    backdrop-filter: blur(4px);
  }
}

/* Hide scrollbar for filter tabs */
.scrollbar-hide {
  -ms-overflow-style: none;
  scrollbar-width: none;
}

.scrollbar-hide::-webkit-scrollbar {
  display: none;
}
</style>
