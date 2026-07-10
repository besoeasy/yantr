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
    class="group relative flex h-full w-full min-h-[320px] cursor-pointer flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-all duration-300 hover:-translate-y-1 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <div class="flex w-full items-start justify-between">
      <div class="flex items-center gap-4">
        <!-- Minimal Icon Block -->
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 transition-transform duration-300 group-hover:scale-105 dark:border-zinc-800 dark:bg-zinc-900">
          <Sparkles class="h-5 w-5 text-zinc-900 dark:text-zinc-100" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors group-hover:text-black dark:text-white dark:group-hover:text-white">
            {{ t("home.dailyAppCard.featuredToday") }}
          </h3>
          
          <div
            :class="[
              'mt-2 flex items-center gap-2',
              appState === 'running'
                ? 'text-zinc-900 dark:text-white'
                : appState === 'installed'
                  ? 'text-zinc-700 dark:text-zinc-300'
                  : 'text-zinc-500 dark:text-zinc-400'
            ]"
          >
            <span v-if="appState === 'running'" class="relative flex h-2 w-2">
              <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
              <span class="relative inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
            </span>
            <span v-else-if="appState === 'installed'" class="h-2 w-2 rounded-full bg-zinc-300 dark:bg-zinc-700"></span>
            <span v-else class="h-2 w-2 rounded-full bg-zinc-200 dark:bg-zinc-800"></span>
            <span class="text-[10px] font-semibold uppercase tracking-widest">{{ stateLabel }}</span>
          </div>
        </div>
      </div>
      
      <!-- App Logo (top right) -->
      <div class="flex shrink-0 items-center gap-2">
         <div class="flex h-12 w-12 items-center justify-center transition-transform duration-300 group-hover:scale-110">
            <AppLogo
              :logo="dailyApp?.logo"
              :name="dailyApp?.name"
              :seed="dailyApp?.id || dailyApp?.name"
              img-class="h-full w-full object-contain"
              icon-class="h-10 w-10 text-zinc-200 transition-colors duration-300 group-hover:text-zinc-300 dark:text-zinc-800 dark:group-hover:text-zinc-700"
            />
         </div>
      </div>
    </div>

    <!-- Main Body -->
    <div class="relative z-10 mt-auto flex flex-col pt-6">
      <div class="mb-2 text-[10px] font-semibold uppercase tracking-widest text-zinc-500 dark:text-zinc-400">
        {{ primaryTag || "App" }}
      </div>
      <div class="mb-3 text-4xl font-black leading-[1.1] tracking-tighter text-zinc-900 transition-colors duration-500 group-hover:-translate-y-1 sm:text-5xl dark:text-white">
        {{ dailyApp?.name }}
      </div>
      <p v-if="dailyApp?.short_description" class="max-w-[90%] text-sm leading-relaxed text-zinc-500 line-clamp-2 dark:text-zinc-400">
        {{ dailyApp.short_description }}
      </p>
      
      <!-- Arrow indicator at bottom right -->
      <div class="absolute bottom-0 right-0 -translate-x-4 translate-y-4 opacity-0 transition-all duration-500 group-hover:translate-x-0 group-hover:translate-y-0 group-hover:opacity-100">
        <div class="flex h-10 w-10 items-center justify-center rounded-full border border-zinc-200 bg-white text-zinc-900 shadow-sm transition-colors duration-300 hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-800 dark:text-white dark:hover:bg-zinc-700">
           <ArrowUpRight class="h-5 w-5" />
        </div>
      </div>
    </div>
  </button>

  <!-- Error State -->
  <div
    v-else-if="loadFailed"
    class="flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl border border-red-200 bg-white p-7 text-left dark:border-red-900/50 dark:bg-black"
  >
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-red-200 bg-red-50 dark:border-red-900/50 dark:bg-red-900/10">
          <AlertCircle class="h-5 w-5 text-red-600 dark:text-red-500" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight text-red-950 dark:text-red-50">{{ t("home.dailyAppCard.featuredToday") }}</h3>
          <p class="mt-2 text-[10px] font-semibold uppercase tracking-widest text-red-600 dark:text-red-400">Error</p>
        </div>
      </div>
    </div>

    <div class="relative z-10 mt-auto flex flex-col pt-6">
      <div class="mb-2 text-[10px] font-semibold uppercase tracking-widest text-red-500 dark:text-red-400">
        Failed to load
      </div>
      <div class="mb-2 text-4xl font-black tracking-tighter text-red-950 dark:text-white">
        {{ t("common.error") }}
      </div>
    </div>
  </div>
</template>