<script setup>
import { ref, computed, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import { Store, ArrowRight, Package, HardDrive } from "lucide-vue-next";
import SystemCleaner from "../components/SystemCleaner.vue";
import WatchtowerAlert from "../components/WatchtowerAlert.vue";
import GreetingCard from "../components/home/GreetingCard.vue";
import AppCategoriesCard from "../components/quick-metrics/AppCategoriesCard.vue";
import BiggestStorageCard from "../components/quick-metrics/BiggestStorageCard.vue";
import MachineIdentityCard from "../components/quick-metrics/MachineIdentityCard.vue";
import RotatingDiskUsageCard from "../components/quick-metrics/RotatingDiskUsageCard.vue";
import AverageUptimeCard from "../components/quick-metrics/AverageUptimeCard.vue";
import ExpiringContainersCard from "../components/quick-metrics/ExpiringContainersCard.vue";
import WatchtowerNextCheckCard from "../components/quick-metrics/WatchtowerNextCheckCard.vue";

const router = useRouter();

const containers = ref([]);
const volumes = ref([]);
const images = ref([]);
const loading = ref(false);
const apiUrl = ref("");
const currentTime = ref(Date.now());
const watchtowerInstalled = ref(false);

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

  const createdTime = container.created * 1000; // Convert to milliseconds
  const uptime = currentTime.value - createdTime;

  if (uptime <= 0) return "Just started";

  const days = Math.floor(uptime / (1000 * 60 * 60 * 24));
  const hours = Math.floor((uptime % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
  const minutes = Math.floor((uptime % (1000 * 60 * 60)) / (1000 * 60));

  if (days > 0) {
    return `${days}d ${hours}h`;
  } else if (hours > 0) {
    return `${hours}h ${minutes}m`;
  } else {
    return `${minutes}m`;
  }
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
  <div class="min-h-screen bg-gray-50/50">
    <!-- Main Content -->
    <div class="p-6 sm:p-10 lg:p-14">
      <div class="space-y-10">
        <!-- Loading State -->
        <div v-if="loading" class="flex flex-col items-center justify-center py-32">
          <div class="w-10 h-10 border-3 border-gray-200 border-t-blue-600 rounded-full animate-spin mb-4"></div>
          <div class="text-gray-400 font-medium animate-pulse">Syncing containers...</div>
        </div>

        <!-- Content -->
        <div v-else class="space-y-12 animate-fadeIn">
          <!-- Empty State -->
          <div v-if="containers.length === 0" class="text-center py-24 bg-white rounded-3xl shadow-sm">
            <div class="w-24 h-24 bg-gray-50 rounded-full flex items-center justify-center mx-auto mb-6">
              <Store :size="40" class="text-gray-300" />
            </div>
            <h3 class="text-2xl font-bold text-gray-900 mb-2">No Apps Running</h3>
            <p class="text-gray-500 max-w-md mx-auto mb-8">Your dashboard is looking a bit empty. Visit the App Store to get started.</p>
            <router-link
              to="/apps"
              class="inline-flex items-center gap-2 px-8 py-4 bg-gray-900 hover:bg-black text-white rounded-2xl font-bold transition-all hover:scale-105 shadow-xl shadow-gray-900/20"
            >
              <Store :size="20" />
              <span>Browse App Store</span>
            </router-link>
          </div>

          <!-- Quick Metrics (Cards) -->
          <div class="space-y-6">
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-5">
              <div class="lg:col-span-2 xl:col-span-2">
                <GreetingCard :running-apps="runningApps" :total-volumes="totalVolumes" />
              </div>

              <template v-if="yantraContainers.length > 0">
                <div
                  v-for="(container, index) in yantraContainers"
                  :key="container.id"
                  :style="{ animationDelay: `${index * 50}ms` }"
                  @click="viewContainerDetail(container)"
                  class="relative h-full overflow-hidden group rounded-2xl cursor-pointer transition-all duration-500 hover:-translate-y-1 animate-fadeIn"
                >
                  <div class="absolute inset-0 bg-gray-900">
                    <div class="absolute inset-0 bg-linear-to-br from-slate-600/25 via-blue-600/10 to-gray-900 z-10"></div>
                    <div
                      class="absolute top-0 right-0 w-64 h-64 bg-blue-500/15 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2 group-hover:bg-blue-500/25 transition-colors duration-700"
                    ></div>
                    <div
                      class="absolute bottom-0 left-0 w-48 h-48 bg-slate-500/20 rounded-full blur-3xl translate-y-1/2 -translate-x-1/2 group-hover:bg-slate-500/30 transition-colors duration-700"
                    ></div>
                  </div>

                  <div
                    class="relative z-20 h-full p-6 flex flex-col justify-between border border-white/5 rounded-2xl backdrop-blur-sm group-hover:border-blue-500/30 transition-colors duration-500"
                  >
                    <!-- Card Header -->
                    <div class="flex items-start justify-between mb-6">
                      <div class="flex items-center gap-4 min-w-0">
                        <div class="relative shrink-0">
                          <div class="absolute inset-0 bg-blue-500/15 rounded-2xl blur-lg group-hover:blur-xl transition-all duration-500"></div>

                          <div class="relative w-16 h-16 rounded-2xl bg-white/5 border border-white/10 flex items-center justify-center overflow-hidden">
                            <img
                              v-if="container.app && container.app.logo"
                              :src="container.app.logo"
                              :alt="container.name"
                              class="w-14 h-14 object-contain filter group-hover:brightness-110 transition-all"
                            />
                            <div v-else class="text-3xl">🐳</div>
                          </div>

                          <!-- Status Indicator Dot -->
                          <div class="absolute -bottom-1 -right-1 w-5 h-5 bg-gray-900/90 border border-white/10 rounded-full flex items-center justify-center">
                            <div :class="container.state === 'running' ? 'bg-emerald-400' : 'bg-gray-400'" class="w-3 h-3 rounded-full animate-pulse"></div>
                          </div>
                        </div>

                        <div class="min-w-0">
                          <h3 class="font-bold text-xl text-white line-clamp-1 mb-1 group-hover:text-blue-200 transition-colors">
                            {{ container.app ? container.app.name : container.name.replace(/^\//, "") }}
                          </h3>
                          <div class="flex items-center gap-2 flex-wrap">
                            <span
                              class="text-xs font-semibold px-2.5 py-1 rounded-lg border"
                              :class="
                                container.state === 'running'
                                  ? 'bg-emerald-500/10 text-emerald-200 border-emerald-500/20'
                                  : 'bg-white/5 text-gray-300 border-white/10'
                              "
                            >
                              {{ container.state }}
                            </span>
                            <span
                              v-if="isTemporary(container)"
                              class="text-xs font-semibold px-2.5 py-1 rounded-lg border bg-orange-500/10 text-orange-200 border-orange-500/20"
                            >
                              Temp
                            </span>
                          </div>
                        </div>
                      </div>
                    </div>

                    <!-- Metrics -->
                    <div class="space-y-3 mb-6 flex-1">
                      <div
                        v-if="container.state === 'running' && formatUptime(container)"
                        class="flex items-center justify-between text-sm py-2 border-b border-white/10 last:border-0 border-dashed"
                      >
                        <span class="text-gray-400 font-medium">Uptime</span>
                        <span class="text-gray-200 font-semibold font-mono tabular-nums">{{ formatUptime(container) }}</span>
                      </div>

                      <div
                        v-if="isTemporary(container)"
                        class="flex items-center justify-between text-sm py-2 border-b border-white/10 last:border-0 border-dashed"
                      >
                        <span class="text-gray-400 font-medium">Expires</span>
                        <span
                          :class="getExpirationInfo(container).isExpiringSoon ? 'text-red-300 animate-pulse' : 'text-orange-200'"
                          class="font-bold font-mono tabular-nums"
                        >
                          {{ getExpirationInfo(container).timeRemaining }}
                        </span>
                      </div>
                    </div>

                    <!-- Footer -->
                    <div class="mt-auto pt-4 flex items-center justify-between text-sm border-t border-white/5 group-hover:border-white/10 transition-colors">
                      <span class="text-gray-400 font-medium group-hover:text-blue-200 transition-colors">Manage App</span>
                      <div
                        class="w-9 h-9 rounded-full bg-white/5 border border-white/10 flex items-center justify-center text-white group-hover:border-blue-500/30 group-hover:bg-blue-500/10 transition-colors"
                      >
                        <ArrowRight :size="16" class="transform group-hover:translate-x-0.5 transition-transform" />
                      </div>
                    </div>
                  </div>
                </div>
              </template>

              <template v-if="volumeContainers.length > 0">
                <div
                  v-for="(container, index) in volumeContainers"
                  :key="container.id"
                  :style="{ animationDelay: `${index * 50}ms` }"
                  @click="viewContainerDetail(container)"
                  class="relative h-full overflow-hidden group rounded-2xl cursor-pointer transition-all duration-500 hover:-translate-y-1 animate-fadeIn"
                >
                  <div class="absolute inset-0 bg-gray-900">
                    <div class="absolute inset-0 bg-linear-to-br from-indigo-600/25 via-purple-600/10 to-gray-900 z-10"></div>
                    <div
                      class="absolute top-0 right-0 w-64 h-64 bg-indigo-500/15 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2 group-hover:bg-indigo-500/25 transition-colors duration-700"
                    ></div>
                    <div
                      class="absolute bottom-0 left-0 w-48 h-48 bg-purple-600/15 rounded-full blur-3xl translate-y-1/2 -translate-x-1/2 group-hover:bg-purple-600/25 transition-colors duration-700"
                    ></div>
                  </div>

                  <div
                    class="relative z-20 h-full p-6 flex flex-col justify-between border border-white/5 rounded-2xl backdrop-blur-sm group-hover:border-indigo-500/30 transition-colors duration-500"
                  >
                    <div class="flex items-center gap-4 mb-4">
                      <div class="relative shrink-0">
                        <div class="absolute inset-0 bg-indigo-500/20 rounded-2xl blur-lg group-hover:blur-xl transition-all duration-500"></div>
                        <div
                          class="relative w-14 h-14 bg-linear-to-br from-indigo-600 to-purple-600 rounded-2xl flex items-center justify-center shadow-lg group-hover:scale-110 transition-transform duration-500"
                        >
                          <span class="text-2xl">📂</span>
                        </div>
                      </div>

                      <div class="min-w-0">
                        <h3 class="font-bold text-lg text-white line-clamp-1 mb-1 group-hover:text-indigo-200 transition-colors">
                          {{ container.labels?.["yantra.volume-browser"] || container.name }}
                        </h3>
                        <span class="text-xs font-semibold px-2.5 py-1 rounded-lg border bg-emerald-500/10 text-emerald-200 border-emerald-500/20">
                          Active
                        </span>
                      </div>
                    </div>

                    <!-- Expiration Timer -->
                    <div v-if="isTemporary(container)" class="mb-4 px-1">
                      <div class="flex items-center justify-between text-xs py-1.5 px-3 rounded-lg border border-orange-500/20 bg-orange-500/10">
                        <span class="font-medium text-orange-200">Expires in</span>
                        <span
                          :class="getExpirationInfo(container).isExpiringSoon ? 'text-red-300 animate-pulse font-bold' : 'text-orange-200 font-semibold'"
                          class="font-mono tabular-nums"
                        >
                          {{ getExpirationInfo(container).timeRemaining }}
                        </span>
                      </div>
                    </div>

                    <div class="mt-auto pt-3 flex items-center justify-between text-sm border-t border-white/5 group-hover:border-white/10 transition-colors">
                      <span class="text-gray-400 font-medium group-hover:text-indigo-200 transition-colors">Manage Files</span>
                      <ArrowRight :size="16" class="text-white/70 group-hover:text-indigo-200 transform group-hover:translate-x-0.5 transition-all" />
                    </div>
                  </div>
                </div>
              </template>

              <div v-if="showWatchtowerAlert" class="h-full">
                <WatchtowerAlert />
              </div>

              <div v-else class="h-full">
                <WatchtowerNextCheckCard :containers="containers" :current-time="currentTime" :interval-hours="3" />
              </div>

              <div v-if="reclaimableStats.show" class="h-full lg:col-span-2 xl:col-span-2">
                <SystemCleaner
                  :api-url="apiUrl"
                  :initial-image-stats="reclaimableStats.imageStats"
                  :initial-volume-stats="reclaimableStats.volumeStats"
                  @cleaned="refreshAll"
                />
              </div>

              <div>
                <AverageUptimeCard :containers="containers" :current-time="currentTime" />
              </div>

              <div v-if="temporaryContainersCount > 0" class="lg:col-span-2 xl:col-span-2">
                <ExpiringContainersCard :containers="containers" :current-time="currentTime" />
              </div>

              <div class="lg:col-span-2 xl:col-span-2" v-if="images.length > 0 || volumes.length > 0">
                <RotatingDiskUsageCard :images="images" :volumes="volumes" :interval-ms="10000" />
              </div>

              <div class="lg:col-span-2 xl:col-span-2">
                <MachineIdentityCard />
              </div>

              <div v-if="images.length > 0" class="lg:col-span-2 xl:col-span-2">
                <BiggestStorageCard :images="images" />
              </div>

              
              <div v-if="containers.length > 0" class="lg:col-span-2 xl:col-span-2">
                <AppCategoriesCard :containers="containers" />
              </div>
            </div>
          </div>
        </div>

        <!-- Other Containers Section -->
        <div v-if="otherContainers.length > 0">
          <div class="flex items-center gap-3 mb-6">
            <div class="p-2 bg-gray-200 text-gray-600 rounded-xl">
              <div class="font-bold text-lg px-1">#</div>
            </div>
            <h2 class="text-2xl font-bold text-gray-900">Other Containers</h2>
            <span class="px-2.5 py-0.5 rounded-full bg-gray-100 text-gray-600 text-xs font-bold">{{ otherContainers.length }}</span>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-2 xl:grid-cols-3 gap-5">
            <div
              v-for="(container, index) in otherContainers"
              :key="container.id"
              :style="{ animationDelay: `${index * 50}ms` }"
              @click="viewContainerDetail(container)"
              class="group bg-white rounded-3xl p-6 shadow-sm border border-gray-200 hover:shadow-lg transition-all duration-300 cursor-pointer animate-fadeIn relative overflow-hidden flex flex-col h-full hover:-translate-y-1"
            >
              <div class="flex items-start gap-4 mb-4">
                <div class="w-12 h-12 bg-gray-100 rounded-xl flex items-center justify-center text-2xl grayscale opacity-70">🐳</div>
                <div class="overflow-hidden">
                  <h3 class="font-bold text-lg text-gray-700 truncate mb-1" :title="container.name">
                    {{ container.name.replace(/^\//, "") }}
                  </h3>
                  <span
                    class="text-xs font-semibold px-2 py-0.5 rounded-md"
                    :class="container.state === 'running' ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-600'"
                  >
                    {{ container.state }}
                  </span>
                </div>
              </div>

              <div class="mt-auto pt-4 border-t border-gray-100">
                <div class="flex items-center justify-between text-xs text-gray-500">
                  <span class="font-mono truncate max-w-37.5">{{ container.image.split(":")[0] }}</span>
                  <ArrowRight :size="14" class="group-hover:translate-x-1 transition-transform" />
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.animate-fadeIn {
  opacity: 0;
  animation: fadeIn 0.5s ease-out forwards;
}

/* Apple-like hover lift effect */
.hover-lift {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.hover-lift:hover {
  transform: translateY(-4px) scale(1.01);
  box-shadow:
    0 20px 40px -12px rgba(0, 0, 0, 0.15),
    0 8px 16px -8px rgba(124, 58, 237, 0.2);
}

.hover-lift:active {
  transform: translateY(-2px) scale(1);
  transition: all 0.1s cubic-bezier(0.4, 0, 0.2, 1);
}

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
</style>
