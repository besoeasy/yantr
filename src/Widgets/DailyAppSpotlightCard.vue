<script setup>
import { ref, computed, onMounted } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { Sparkles, ArrowUpRight, AlertCircle } from "@lucide/vue";
import AppLogo from "../components/AppLogo.vue";
import { useApiUrl } from "../composables/useApiUrl";
import { expectApiSuccess } from "../composables/useApiResponse";

const { t } = useI18n();
const router = useRouter();
const { apiUrl } = useApiUrl();

const apps = ref([]);
const containers = ref([]);
const loadFailed = ref(false);

function getDateDaySeed() {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function hashString(value) {
  let hash = 0;
  for (let i = 0; i < value.length; i++) {
    hash = (hash * 31 + value.charCodeAt(i)) >>> 0;
  }
  return hash;
}

const installedAppIds = computed(() => {
  return new Set(containers.value.map((c) => c?.app?.id).filter(Boolean));
});

const runningAppInstanceCounts = computed(() => {
  const projectsByApp = {};
  containers.value
    .filter((c) => c.state === "running")
    .forEach((c) => {
      const appId = c?.app?.id;
      const projectId = c?.app?.projectId;
      if (!appId || !projectId) return;
      if (!projectsByApp[appId]) projectsByApp[appId] = new Set();
      projectsByApp[appId].add(projectId);
    });
  const counts = {};
  for (const [appId, projects] of Object.entries(projectsByApp)) {
    counts[appId] = projects.size;
  }
  return counts;
});

const dailyApp = computed(() => {
  if (apps.value.length === 0) return null;
  const catalog = [...apps.value].sort((a, b) => a.id.localeCompare(b.id));
  const index = hashString(getDateDaySeed()) % catalog.length;
  const featured = catalog[index];
  return {
    ...featured,
    isInstalled: installedAppIds.value.has(featured.id),
    instanceCount: runningAppInstanceCounts.value[featured.id] || 0,
  };
});

async function fetchData() {
  try {
    const [appsRes, containersRes] = await Promise.all([
      fetch(`${apiUrl.value}/api/apps`),
      fetch(`${apiUrl.value}/api/containers`),
    ]);
    const [appsData, containersData] = await Promise.all([
      expectApiSuccess(appsRes, "Failed to load apps"),
      expectApiSuccess(containersRes, "Failed to load containers"),
    ]);

    apps.value = Array.isArray(appsData.apps) ? appsData.apps : [];
    containers.value = Array.isArray(containersData.containers) ? containersData.containers : [];
    loadFailed.value = false;
  } catch (error) {
    loadFailed.value = true;
  }
}

function handleSelect() {
  if (!dailyApp.value?.id) return;
  router.push(`/apps/${dailyApp.value.id}`);
}

onMounted(fetchData);

const instanceCount = computed(() => dailyApp.value?.instanceCount ?? 0);

const appState = computed(() => {
  if (instanceCount.value > 0) return "running";
  if (dailyApp.value?.isInstalled) return "installed";
  return "available";
});

const primaryTag = computed(() => {
  const tags = dailyApp.value?.tags;
  if (!Array.isArray(tags) || tags.length === 0) return null;
  return tags[0];
});

const stateLabel = computed(() => {
  if (appState.value === "running") return t("home.dailyAppCard.running", { count: instanceCount.value });
  if (appState.value === "installed") return t("home.dailyAppCard.installed");
  return t("home.dailyAppCard.available");
});
</script>

<template>
  <button
    v-if="dailyApp"
    type="button"
    @click="handleSelect"
    class="cursor-pointer relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] rounded-xl overflow-hidden transition-all duration-300 hover:-translate-y-0.5 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 text-left w-full"
  >
    <!-- top accent line -->
    <div class="absolute top-0 left-0 w-full h-0.5 bg-amber-500 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>

    <div class="relative z-10 p-5 flex flex-col gap-4 h-full min-h-72">
      <!-- header -->
      <div class="flex items-center justify-between gap-3">
        <div class="flex items-center gap-3 min-w-0">
          <div class="w-9 h-9 rounded-lg flex items-center justify-center shrink-0 transition-transform group-hover:scale-110">
            <Sparkles class="w-4.5 h-4.5 text-amber-600 dark:text-amber-400" />
          </div>
          <div class="min-w-0">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight leading-none">{{ t("home.dailyAppCard.featuredToday") }}</h3>
            
            <div
              :class="[
                'flex items-center gap-1.5 mt-1.5',
                appState === 'running'
                  ? 'text-emerald-600 dark:text-emerald-400'
                  : appState === 'installed'
                    ? 'text-blue-600 dark:text-blue-400'
                    : 'text-amber-600 dark:text-amber-400'
              ]"
            >
              <div class="w-1.5 h-1.5 rounded-full bg-current" :class="{ 'animate-pulse': appState === 'running' }"></div>
              <span class="text-[10px] font-bold uppercase tracking-wider">{{ stateLabel }}</span>
            </div>
          </div>
        </div>
        <div class="flex items-center gap-2 shrink-0">
           <div class="w-10 h-10 flex items-center justify-center transition-transform duration-300 group-hover:scale-110 group-hover:rotate-3 opacity-90">
              <AppLogo
                :logo="dailyApp?.logo"
                :name="dailyApp?.name"
                :seed="dailyApp?.id || dailyApp?.name"
                img-class="h-full w-full object-contain"
                icon-class="h-8 w-8 text-gray-200 dark:text-zinc-800 transition-colors duration-300 group-hover:text-amber-500/30"
              />
           </div>
        </div>
      </div>

      <div class="mt-auto pt-6 pb-2 flex flex-col gap-1.5 relative z-10 pr-6">
        <div class="text-3xl sm:text-4xl md:text-5xl font-black tracking-tighter text-gray-900 dark:text-white group-hover:text-amber-600 dark:group-hover:text-amber-400 transition-colors line-clamp-2 leading-[1.1]">
          {{ dailyApp?.name }}
        </div>
        <p v-if="dailyApp?.short_description" class="text-sm text-gray-500 dark:text-zinc-400 line-clamp-2 mt-1 leading-relaxed max-w-[90%]">
          {{ dailyApp.short_description }}
        </p>
        <div class="text-xs font-bold text-gray-400 dark:text-zinc-500 uppercase tracking-widest mt-2 truncate">
          {{ primaryTag || "App" }}
        </div>
      </div>

      <!-- Arrow indicator -->
      <div class="absolute bottom-5 right-5 opacity-0 -translate-x-2 translate-y-2 group-hover:opacity-100 group-hover:translate-x-0 group-hover:translate-y-0 transition-all duration-300">
        <div class="w-8 h-8 rounded-full flex items-center justify-center">
           <ArrowUpRight class="w-4 h-4 text-amber-600 dark:text-amber-400" />
        </div>
      </div>

    </div>
  </button>

  <div
    v-else-if="loadFailed"
    class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] rounded-xl overflow-hidden text-left w-full p-5 min-h-72"
  >
    <div class="flex h-full flex-col gap-5">
      <div class="flex items-center gap-3 min-w-0">
        <div class="w-9 h-9 rounded-lg flex items-center justify-center shrink-0">
          <AlertCircle class="w-4.5 h-4.5 text-red-600 dark:text-red-400" />
        </div>
        <div class="min-w-0">
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight leading-none">{{ t("home.dailyAppCard.featuredToday") }}</h3>
          <span class="text-[10px] font-bold uppercase tracking-wider text-red-600 dark:text-red-400 mt-1.5 block">Error</span>
        </div>
      </div>

      <div class="mt-auto pt-6 pb-2">
        <div class="text-3xl font-black tracking-tighter text-gray-900 dark:text-white leading-[1.1]">
          {{ t("common.error") }}
        </div>
        <div class="text-sm font-bold text-gray-400 dark:text-zinc-500 uppercase tracking-widest mt-2">
          Failed to load
        </div>
      </div>
    </div>
  </div>
</template>