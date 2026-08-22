<script setup>
import { ref, computed, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import { Store, LayoutGrid, PackageCheck, Container, FolderOpen, Activity } from "@lucide/vue";
import HomelabKits from "../components/HomelabKits.vue";

// Auto-load all widget .vue files from src/Widgets/, shuffle once per day
const widgetModules = import.meta.glob("../Widgets/*.vue", { eager: true });
function dailyShuffle(arr) {
  const today = new Date();
  const seed = today.getFullYear() * 10000 + (today.getMonth() + 1) * 100 + today.getDate();
  const seeded = arr.slice();
  for (let i = seeded.length - 1; i > 0; i--) {
    const j = ((seed * (i + 1)) ^ (seed >> 3)) % (i + 1);
    [seeded[i], seeded[j]] = [seeded[j], seeded[i]];
  }
  return seeded;
}
// All dashboard cards now use 1 column only.
const widgets = dailyShuffle(Object.values(widgetModules).map((m) => m.default));
import { useApiUrl } from "../composables/useApiUrl";
import { expectApiSuccess } from "../composables/useApiResponse";
import { useNotification } from "../composables/useNotification";
import { useI18n } from "vue-i18n";
import YantraContainersGrid from "../components/YantraContainersGrid.vue";
import VolumeContainersGrid from "../components/VolumeContainersGrid.vue";
import OtherContainersGrid from "../components/OtherContainersGrid.vue";

const { apiUrl } = useApiUrl();
const { t } = useI18n();
const toast = useNotification();
const router = useRouter();

const containers = ref([]);
const volumeBrowsers = ref([]);
const catalogApps = ref([]);
const loading = ref(false);
const activeFilter = ref("all");

let containersRefreshInterval = null;
const loadErrorState = {
  containers: false,
  volumeBrowsers: false,
};

function notifyLoadErrorOnce(key, message) {
  if (loadErrorState[key]) return;
  loadErrorState[key] = true;
  toast.error(message);
}

function clearLoadError(key) {
  loadErrorState[key] = false;
}

// Container Grouping
const volumeContainers = computed(() => volumeBrowsers.value);

const yantrContainers = computed(() => {
  return containers.value.filter((c) => c.appLabels?.app);
});

const otherContainers = computed(() => {
  return containers.value.filter((c) => !c.appLabels?.app);
});

// Filter visibility computed properties
const showYantrApps = computed(() => activeFilter.value === "all" || activeFilter.value === "yantr");
const showDockerApps = computed(() => activeFilter.value === "all" || activeFilter.value === "docker");
const showVolumeBrowsers = computed(() => activeFilter.value === "all" || activeFilter.value === "volumes");
const showMetrics = computed(() => activeFilter.value === "all" || activeFilter.value === "metrics");

async function fetchContainers() {
  try {
    const response = await fetch(`${apiUrl.value}/api/containers`);
    const data = await expectApiSuccess(response, "Failed to load containers");
    containers.value = Array.isArray(data.containers) ? data.containers : [];
    clearLoadError("containers");
  } catch (error) {
    notifyLoadErrorOnce("containers", error.message || "Failed to load containers");
  }
}

async function fetchCatalogApps() {
  try {
    const response = await fetch(`${apiUrl.value}/api/apps`);
    const data = await expectApiSuccess(response, "Failed to load apps");
    catalogApps.value = Array.isArray(data.apps) ? data.apps : [];
  } catch {
    catalogApps.value = [];
  }
}

async function fetchVolumeBrowsers() {
  try {
    const response = await fetch(`${apiUrl.value}/api/volumes/browsers`);
    const data = await response.json().catch(() => []);
    volumeBrowsers.value = Array.isArray(data) ? data : [];
    clearLoadError("volumeBrowsers");
  } catch (error) {
    notifyLoadErrorOnce("volumeBrowsers", error.message || "Failed to load volume browsers");
  }
}

async function stopBrowser(volumeName) {
  try {
    const response = await fetch(`${apiUrl.value}/api/volumes/${volumeName}/browse`, { method: "DELETE" });
    await expectApiSuccess(response, t("volumes.failedToStopBrowser"));
    await fetchVolumeBrowsers();
    toast.success(t("volumes.browserStopped"));
  } catch (error) {
    toast.error(error.message || t("volumes.failedToStopBrowser"));
  }
}

function viewContainerDetail(container) {
  router.push(`/containers/${container.id}`);
}

function openKit(kitId) {
  router.push({ path: "/apps", query: { kit: kitId } });
}

async function refreshAll() {
  await Promise.all([fetchContainers(), fetchVolumeBrowsers()]);
}

onMounted(async () => {
  loading.value = true;
  await Promise.all([refreshAll(), fetchCatalogApps()]);
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
          <div v-if="containers.length > 0" class="flex flex-wrap items-center gap-2 overflow-x-auto pb-4 mb-4 scrollbar-hide">
            <button
              @click="activeFilter = 'all'"
              :class="[
                'px-4 py-2 rounded-lg text-xs font-bold uppercase tracking-wider whitespace-nowrap transition-all flex items-center gap-2',
                activeFilter === 'all'
                  ? 'bg-gray-900 text-white dark:bg-white dark:text-gray-900 smooth-shadow'
                  : 'bg-(--surface-muted) text-(--text-secondary) hover:text-(--text-primary) hover:smooth-shadow',
              ]"
            >
              <LayoutGrid :size="14" />
              <span>{{ t("home.all") }}</span>
            </button>
            <button
              v-if="yantrContainers.length > 0"
              @click="activeFilter = 'yantr'"
              :class="[
                'px-4 py-2 rounded-lg text-xs font-bold uppercase tracking-wider whitespace-nowrap transition-all flex items-center gap-2',
                activeFilter === 'yantr'
                  ? 'bg-gray-900 text-white dark:bg-white dark:text-gray-900 smooth-shadow'
                  : 'bg-(--surface-muted) text-(--text-secondary) hover:text-(--text-primary) hover:smooth-shadow',
              ]"
            >
              <PackageCheck :size="14" />
              <span>{{ t("home.yantrApps") }}</span>
            </button>
            <button
              v-if="otherContainers.length > 0"
              @click="activeFilter = 'docker'"
              :class="[
                'px-4 py-2 rounded-lg text-xs font-bold uppercase tracking-wider whitespace-nowrap transition-all flex items-center gap-2',
                activeFilter === 'docker'
                  ? 'bg-gray-900 text-white dark:bg-white dark:text-gray-900 smooth-shadow'
                  : 'bg-(--surface-muted) text-(--text-secondary) hover:text-(--text-primary) hover:smooth-shadow',
              ]"
            >
              <Container :size="14" />
              <span>{{ t("home.dockerApps") }}</span>
            </button>
            <button
              v-if="volumeContainers.length > 0"
              @click="activeFilter = 'volumes'"
              :class="[
                'px-4 py-2 rounded-lg text-xs font-bold uppercase tracking-wider whitespace-nowrap transition-all flex items-center gap-2',
                activeFilter === 'volumes'
                  ? 'bg-gray-900 text-white dark:bg-white dark:text-gray-900 smooth-shadow'
                  : 'bg-(--surface-muted) text-(--text-secondary) hover:text-(--text-primary) hover:smooth-shadow',
              ]"
            >
              <FolderOpen :size="14" />
              <span>{{ t("home.volumeBrowsers") }}</span>
            </button>
            <button
              @click="activeFilter = 'metrics'"
              :class="[
                'px-4 py-2 rounded-lg text-xs font-bold uppercase tracking-wider whitespace-nowrap transition-all flex items-center gap-2',
                activeFilter === 'metrics'
                  ? 'bg-gray-900 text-white dark:bg-white dark:text-gray-900 smooth-shadow'
                  : 'bg-(--surface-muted) text-(--text-secondary) hover:text-(--text-primary) hover:smooth-shadow',
              ]"
            >
              <Activity :size="14" />
              <span>{{ t("home.metrics") }}</span>
            </button>
          </div>

          <!-- Empty State — default homelab path -->
          <div
            v-if="containers.length === 0"
            class="mb-6 rounded-xl border border-gray-200 bg-white px-4 py-10 sm:px-8 dark:border-zinc-800 dark:bg-[#0A0A0A]"
          >
            <div class="mx-auto max-w-3xl text-center">
              <p class="text-[10px] font-black uppercase tracking-[0.24em] text-gray-400 dark:text-zinc-500">{{ t("kits.eyebrow") }}</p>
              <h3 class="mt-3 text-2xl font-semibold tracking-tight text-gray-900 dark:text-white">{{ t("kits.emptyTitle") }}</h3>
              <p class="mx-auto mt-3 max-w-lg text-sm font-medium leading-relaxed text-gray-500 dark:text-zinc-400">{{ t("kits.emptyDesc") }}</p>
            </div>
            <div class="mx-auto mt-8 max-w-5xl">
              <HomelabKits :apps="catalogApps" @select="openKit" />
            </div>
            <div class="mt-8 flex justify-center">
              <router-link
                to="/apps"
                class="inline-flex items-center gap-2 rounded-lg bg-gray-900 px-6 py-3 text-xs font-bold uppercase tracking-wider text-white shadow-sm transition-all hover:-translate-y-0.5 dark:bg-white dark:text-gray-900"
              >
                <Store :size="16" />
                <span>{{ t("home.browseAppStore") }}</span>
              </router-link>
            </div>
          </div>

          <!-- Unified grid: all cards are single-column on mobile, 2 columns on medium, 3 on large -->
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6 grid-flow-dense">
            <YantraContainersGrid v-if="showYantrApps && yantrContainers.length > 0" :containers="yantrContainers" :show-header="activeFilter !== 'all'" />
            <VolumeContainersGrid v-if="showVolumeBrowsers && volumeContainers.length > 0" :containers="volumeContainers" :show-header="activeFilter !== 'all'" @stop-browser="stopBrowser" />
            <OtherContainersGrid v-if="showDockerApps && otherContainers.length > 0" :containers="otherContainers" :show-header="activeFilter !== 'all'" @select="viewContainerDetail" />

            <template v-if="showMetrics">
              <!-- Section header only when metrics filter is active (not "all") -->
              <div v-if="activeFilter === 'metrics'" class="col-span-full flex items-center gap-2 pb-1">
                <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-(--text-secondary)">{{ t("home.metrics") }}</span>
              </div>
              <component
                v-for="(widget, i) in widgets"
                :key="i"
                :is="widget"
                class="h-full"
              />
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
