<script setup>
import { ref, computed, onMounted } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { Sparkles, ArrowUpRight, AlertCircle, Wrench } from "@lucide/vue";
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

const customApps = computed(() => {
  return apps.value.filter((a) => a.customapp === true);
});

const featuredCustomApp = computed(() => {
  if (customApps.value.length === 0) return null;
  const sorted = [...customApps.value].sort((a, b) => a.id.localeCompare(b.id));
  const index = hashString(getDateDaySeed() + "-yantr-custom") % sorted.length;
  const item = sorted[index];
  return {
    ...item,
    isInstalled: installedAppIds.value.has(item.id),
    instanceCount: runningAppInstanceCounts.value[item.id] || 0,
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
  if (!featuredCustomApp.value?.id) return;
  router.push(`/apps/${featuredCustomApp.value.id}`);
}

onMounted(fetchData);

const instanceCount = computed(() => featuredCustomApp.value?.instanceCount ?? 0);

const appState = computed(() => {
  if (instanceCount.value > 0) return "running";
  if (featuredCustomApp.value?.isInstalled) return "installed";
  return "available";
});

const primaryTag = computed(() => {
  const tags = featuredCustomApp.value?.tags;
  if (!Array.isArray(tags) || tags.length === 0) return null;
  return tags[0];
});

const stateLabel = computed(() => {
  if (appState.value === "running") return `${instanceCount.value} running`;
  if (appState.value === "installed") return "Ready";
  return "Available";
});
</script>

<template>
  <button
    v-if="featuredCustomApp"
    type="button"
    @click="handleSelect"
    class="group relative flex h-full w-full min-h-[320px] cursor-pointer flex-col overflow-hidden rounded-2xl border border-amber-500/30 bg-white p-7 text-left transition-all duration-300 hover:-translate-y-1 hover:border-amber-500/80 hover:shadow-[0_8px_30px_rgba(245,158,11,0.12)] dark:border-amber-500/25 dark:bg-black dark:hover:border-amber-400/60 dark:hover:shadow-[0_8px_30px_rgba(245,158,11,0.15)]"
  >
    <!-- Background Ambient Glow -->
    <div class="pointer-events-none absolute -top-16 -right-16 h-40 w-40 rounded-full bg-amber-500/10 blur-3xl dark:bg-amber-500/15"></div>

    <div class="flex w-full items-start justify-between">
      <div class="flex items-center gap-4">
        <!-- Yantr Built Icon Badge -->
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-amber-500/30 bg-amber-500/10 transition-transform duration-300 group-hover:scale-105 dark:bg-amber-500/15">
          <Sparkles class="h-5 w-5 text-amber-600 dark:text-amber-400" />
        </div>
        <div>
          <div class="flex items-center gap-2">
            <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors dark:text-white">
              Yantr Built Spotlight
            </h3>
          </div>
          
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
            <span v-else-if="appState === 'installed'" class="h-2 w-2 rounded-full bg-amber-500/60"></span>
            <span v-else class="h-2 w-2 rounded-full bg-zinc-300 dark:bg-zinc-700"></span>
            <span class="text-[10px] font-semibold uppercase tracking-widest">{{ stateLabel }}</span>
          </div>
        </div>
      </div>
      
      <!-- App Logo (top right) -->
      <div class="flex shrink-0 items-center gap-2">
         <div class="flex h-12 w-12 items-center justify-center transition-transform duration-300 group-hover:scale-110">
            <AppLogo
              :logo="featuredCustomApp?.logo"
              :name="featuredCustomApp?.name"
              :seed="featuredCustomApp?.id || featuredCustomApp?.name"
              img-class="h-full w-full object-contain"
              icon-class="h-10 w-10 text-amber-500/80 transition-colors duration-300"
            />
         </div>
      </div>
    </div>

    <!-- Main Body -->
    <div class="relative z-10 mt-auto flex flex-col pt-6">
      <div class="flex items-center gap-2 mb-2">
        <span class="inline-flex items-center gap-1 rounded-md bg-amber-500/10 px-2 py-0.5 text-[9px] font-bold uppercase tracking-[0.1em] text-amber-700 dark:bg-amber-500/15 dark:text-amber-300 border border-amber-500/30">
          <Sparkles class="h-2.5 w-2.5 text-amber-500" />
          Yantr Built
        </span>
        <span class="text-[10px] font-semibold uppercase tracking-widest text-zinc-500 dark:text-zinc-400">
          • {{ primaryTag || "App" }}
        </span>
      </div>
      <div class="mb-3 text-4xl font-black leading-[1.1] tracking-tighter text-zinc-900 transition-colors duration-500 group-hover:-translate-y-1 sm:text-5xl dark:text-white">
        {{ featuredCustomApp?.name }}
      </div>
      <p v-if="featuredCustomApp?.short_description" class="max-w-[90%] text-sm leading-relaxed text-zinc-500 line-clamp-2 dark:text-zinc-400">
        {{ featuredCustomApp.short_description }}
      </p>
      
      <!-- Arrow indicator at bottom right -->
      <div class="absolute bottom-0 right-0 -translate-x-4 translate-y-4 opacity-0 transition-all duration-500 group-hover:translate-x-0 group-hover:translate-y-0 group-hover:opacity-100">
        <div class="flex h-10 w-10 items-center justify-center rounded-full border border-amber-500/40 bg-amber-500/10 text-amber-700 shadow-sm transition-colors duration-300 hover:bg-amber-500/20 dark:bg-amber-500/20 dark:text-amber-300">
           <ArrowUpRight class="h-5 w-5" />
        </div>
      </div>
    </div>
  </button>
</template>
