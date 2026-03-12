<script setup>
import { ref, computed, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import { Store, LayoutGrid, PackageCheck, Container, FolderOpen, Activity } from "lucide-vue-next";

// Auto-load all widget .vue files from src/Widgets/, shuffle order
const widgetModules = import.meta.glob("../Widgets/*.vue", { eager: true });
const widgets = Object.values(widgetModules)
  .map((m) => m.default)
  .sort(() => Math.random() - 0.5);
import { useApiUrl } from "../composables/useApiUrl";
import { useI18n } from "vue-i18n";
import YantraContainersGrid from "../components/YantraContainersGrid.vue";
import VolumeContainersGrid from "../components/VolumeContainersGrid.vue";
import OtherContainersGrid from "../components/OtherContainersGrid.vue";

const { apiUrl } = useApiUrl();
const { t } = useI18n();
const router = useRouter();

const containers = ref([]);
const volumes = ref([]);
const volumeBrowsers = ref([]);
const loading = ref(false);
const activeFilter = ref("all");

let containersRefreshInterval = null;

// Container Grouping
const volumeContainers = computed(() => volumeBrowsers.value);

const yantrContainers = computed(() => {
  return containers.value.filter((c) => c.appLabels?.app);
});

const otherContainers = computed(() => {
  return containers.value.filter((c) => !c.appLabels?.app);
});

const temporaryContainersCount = computed(() => volumeBrowsers.value.filter((b) => b.expireAt).length);

// Filter visibility computed properties
const showYantrApps = computed(() => activeFilter.value === "all" || activeFilter.value === "yantr");
const showDockerApps = computed(() => activeFilter.value === "all" || activeFilter.value === "docker");
const showVolumeBrowsers = computed(() => activeFilter.value === "all" || activeFilter.value === "volumes");
const showMetrics = computed(() => activeFilter.value === "all" || activeFilter.value === "metrics");

async function fetchContainers() {
  try {
    const response = await fetch(`${apiUrl.value}/api/containers`);
    const data = await response.json();
    if (data.success) {
      containers.value = data.containers;
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

async function fetchVolumeBrowsers() {
  try {
    const response = await fetch(`${apiUrl.value}/api/volumes/browsers`);
    volumeBrowsers.value = await response.json();
  } catch (error) {
    console.error("Failed to fetch volume browsers:", error);
  }
}

async function stopBrowser(volumeName) {
  try {
    await fetch(`${apiUrl.value}/api/volumes/${volumeName}/browse`, { method: "DELETE" });
    await fetchVolumeBrowsers();
  } catch (error) {
    console.error("Failed to stop browser:", error);
  }
}

function viewContainerDetail(container) {
  router.push(`/containers/${container.id}`);
}

async function refreshAll() {
  await Promise.all([fetchContainers(), fetchVolumes(), fetchVolumeBrowsers()]);
}

onMounted(async () => {
  loading.value = true;
  await refreshAll();
  loading.value = false;

  containersRefreshInterval = setInterval(refreshAll, 10000);
});

onUnmounted(() => {
  if (containersRefreshInterval) {
    clearInterval(containersRefreshInterval);
    containersRefreshInterval = null;
  }
});
</script>

<template>
  <div class="min-h-screen bg-white dark:bg-[#0A0A0A] text-gray-900 dark:text-white font-sans">
    <!-- Main Content -->
    <div class="p-3 sm:p-4 lg:p-8 max-w-400 mx-auto">
      <div class="space-y-8">
        <!-- Loading State -->
        <div v-if="loading" class="flex flex-col items-center justify-center py-32">
          <div class="w-8 h-8 border-2 border-gray-200 dark:border-zinc-800 border-t-blue-500 rounded-full animate-spin mb-4"></div>
          <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-500 dark:text-zinc-500 animate-pulse">{{ t("home.syncing") }}</div>
        </div>

        <!-- Content -->
        <div v-else class="animate-fadeIn">
          <!-- Filter Tabs -->
          <div v-if="containers.length > 0" class="flex gap-2 overflow-x-auto pb-4 mb-4 scrollbar-hide">
            <button
              @click="activeFilter = 'all'"
              :class="[
                'px-4 py-2 rounded-lg text-xs font-bold uppercase tracking-wider whitespace-nowrap transition-all flex items-center gap-2 border',
                activeFilter === 'all'
                  ? 'bg-gray-900 text-white border-gray-900 dark:bg-white dark:text-gray-900 dark:border-white shadow-sm'
                  : 'bg-white dark:bg-[#0A0A0A] border-gray-200 dark:border-zinc-800 text-gray-600 dark:text-zinc-400 hover:border-gray-300 dark:hover:border-zinc-600',
              ]"
            >
              <LayoutGrid :size="14" />
              <span>{{ t("home.all") }}</span>
            </button>
            <button
              v-if="yantrContainers.length > 0"
              @click="activeFilter = 'yantr'"
              :class="[
                'px-4 py-2 rounded-lg text-xs font-bold uppercase tracking-wider whitespace-nowrap transition-all flex items-center gap-2 border',
                activeFilter === 'yantr'
                  ? 'bg-gray-900 text-white border-gray-900 dark:bg-white dark:text-gray-900 dark:border-white shadow-sm'
                  : 'bg-white dark:bg-[#0A0A0A] border-gray-200 dark:border-zinc-800 text-gray-600 dark:text-zinc-400 hover:border-gray-300 dark:hover:border-zinc-600',
              ]"
            >
              <PackageCheck :size="14" />
              <span>{{ t("home.yantrApps") }}</span>
            </button>
            <button
              v-if="otherContainers.length > 0"
              @click="activeFilter = 'docker'"
              :class="[
                'px-4 py-2 rounded-lg text-xs font-bold uppercase tracking-wider whitespace-nowrap transition-all flex items-center gap-2 border',
                activeFilter === 'docker'
                  ? 'bg-gray-900 text-white border-gray-900 dark:bg-white dark:text-gray-900 dark:border-white shadow-sm'
                  : 'bg-white dark:bg-[#0A0A0A] border-gray-200 dark:border-zinc-800 text-gray-600 dark:text-zinc-400 hover:border-gray-300 dark:hover:border-zinc-600',
              ]"
            >
              <Container :size="14" />
              <span>{{ t("home.dockerApps") }}</span>
            </button>
            <button
              v-if="volumeContainers.length > 0"
              @click="activeFilter = 'volumes'"
              :class="[
                'px-4 py-2 rounded-lg text-xs font-bold uppercase tracking-wider whitespace-nowrap transition-all flex items-center gap-2 border',
                activeFilter === 'volumes'
                  ? 'bg-gray-900 text-white border-gray-900 dark:bg-white dark:text-gray-900 dark:border-white shadow-sm'
                  : 'bg-white dark:bg-[#0A0A0A] border-gray-200 dark:border-zinc-800 text-gray-600 dark:text-zinc-400 hover:border-gray-300 dark:hover:border-zinc-600',
              ]"
            >
              <FolderOpen :size="14" />
              <span>{{ t("home.volumeBrowsers") }}</span>
            </button>
            <button
              @click="activeFilter = 'metrics'"
              :class="[
                'px-4 py-2 rounded-lg text-xs font-bold uppercase tracking-wider whitespace-nowrap transition-all flex items-center gap-2 border',
                activeFilter === 'metrics'
                  ? 'bg-gray-900 text-white border-gray-900 dark:bg-white dark:text-gray-900 dark:border-white shadow-sm'
                  : 'bg-white dark:bg-[#0A0A0A] border-gray-200 dark:border-zinc-800 text-gray-600 dark:text-zinc-400 hover:border-gray-300 dark:hover:border-zinc-600',
              ]"
            >
              <Activity :size="14" />
              <span>{{ t("home.metrics") }}</span>
            </button>
          </div>

          <!-- Empty State -->
          <div
            v-if="containers.length === 0"
            class="text-center py-32 bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl mb-6 flex flex-col items-center"
          >
            <div class="w-20 h-20 bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 rounded-2xl flex items-center justify-center mb-6">
              <Store :size="32" class="text-gray-400 dark:text-zinc-500" />
            </div>
            <h3 class="text-xl font-semibold tracking-tight text-gray-900 dark:text-white mb-2">{{ t("home.noAppsRunning") }}</h3>
            <p class="text-sm font-medium text-gray-500 dark:text-zinc-400 max-w-md mx-auto mb-8">{{ t("home.dashboardEmpty") }}</p>
            <router-link
              to="/apps"
              class="inline-flex items-center gap-2 px-6 py-3 bg-gray-900 text-white dark:bg-white dark:text-gray-900 rounded-lg text-xs font-bold uppercase tracking-wider transition-all hover:-translate-y-0.5 shadow-sm"
            >
              <Store :size="16" />
              <span>{{ t("home.browseAppStore") }}</span>
            </router-link>
          </div>

          <!-- Unified Dashboard Grid -->
          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3 sm:gap-4">
            <YantraContainersGrid v-if="showYantrApps && yantrContainers.length > 0" :containers="yantrContainers" />

            <VolumeContainersGrid v-if="showVolumeBrowsers && volumeContainers.length > 0" :containers="volumeContainers" @stop-browser="stopBrowser" />

            <OtherContainersGrid v-if="showDockerApps && otherContainers.length > 0" :containers="otherContainers" @select="viewContainerDetail" />

            <!-- Dynamic Widgets from src/Widgets/ -->
            <template v-if="showMetrics">
              <component v-for="(widget, i) in widgets" :is="widget" :key="i" />
            </template>
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
