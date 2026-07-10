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
    class="relative group h-full flex flex-col rounded-[24px] overflow-hidden transition-all duration-500 hover:-translate-y-2 hover:shadow-[0_20px_40px_-15px_rgba(245,158,11,0.4)] dark:hover:shadow-[0_20px_40px_-15px_rgba(245,158,11,0.3)] text-left w-full min-h-[320px] cursor-pointer border border-amber-100/50 dark:border-white/5 bg-gradient-to-br from-amber-50/50 to-orange-50/50 dark:from-[#1a150f] dark:to-[#0f0a09]"
  >
    <!-- Animated Gradient Glow -->
    <div class="absolute -top-[50%] -left-[50%] w-[200%] h-[200%] bg-[radial-gradient(circle_at_50%_50%,_rgba(245,158,11,0.15)_0%,_transparent_50%)] dark:bg-[radial-gradient(circle_at_50%_50%,_rgba(245,158,11,0.12)_0%,_transparent_40%)] opacity-0 group-hover:opacity-100 transition-opacity duration-1000 pointer-events-none group-hover:animate-[spin_20s_linear_infinite]"></div>

    <!-- Massive Watermark Icon -->
    <Sparkles class="absolute -bottom-8 -right-8 w-64 h-64 text-amber-500/10 dark:text-amber-400/[0.03] rotate-[-20deg] group-hover:rotate-0 group-hover:scale-110 transition-all duration-700 ease-out pointer-events-none" />

    <div class="relative z-10 p-7 flex flex-col h-full gap-4 w-full">
      <!-- header -->
      <div class="flex items-start justify-between w-full">
        <div class="flex items-center gap-4">
          <!-- Gorgeous Icon Block -->
          <div class="relative w-14 h-14 rounded-2xl flex items-center justify-center shrink-0 bg-gradient-to-br from-amber-400 to-orange-600 shadow-xl shadow-amber-500/30 group-hover:shadow-amber-500/50 transition-all duration-500 group-hover:scale-105">
            <div class="absolute inset-0 bg-white/30 dark:bg-white/20 rounded-2xl rounded-bl-none opacity-50 dark:opacity-40 mix-blend-overlay"></div>
            <Sparkles class="w-7 h-7 text-white drop-shadow-md z-10" />
          </div>
          <div>
            <h3 class="text-xl font-black text-amber-950 dark:text-white tracking-tight leading-none group-hover:text-amber-600 dark:group-hover:text-amber-400 transition-colors">{{ t("home.dailyAppCard.featuredToday") }}</h3>
            
            <div
              :class="[
                'flex items-center gap-1.5 mt-2',
                appState === 'running'
                  ? 'text-emerald-600 dark:text-emerald-500'
                  : appState === 'installed'
                    ? 'text-blue-600 dark:text-blue-500'
                    : 'text-amber-600/70 dark:text-amber-500'
              ]"
            >
              <div class="w-1.5 h-1.5 rounded-full bg-current" :class="{ 'animate-pulse': appState === 'running' }"></div>
              <span class="text-[10px] font-bold uppercase tracking-[0.2em]">{{ stateLabel }}</span>
            </div>
          </div>
        </div>
        
        <!-- App Logo (top right) -->
        <div class="flex items-center gap-2 shrink-0">
           <div class="w-12 h-12 flex items-center justify-center transition-transform duration-300 group-hover:scale-110 group-hover:rotate-3">
              <AppLogo
                :logo="dailyApp?.logo"
                :name="dailyApp?.name"
                :seed="dailyApp?.id || dailyApp?.name"
                img-class="h-full w-full object-contain filter drop-shadow-sm"
                icon-class="h-10 w-10 text-amber-900/10 dark:text-zinc-800 transition-colors duration-300 group-hover:text-amber-500/30"
              />
           </div>
        </div>
      </div>

      <!-- Main Body -->
      <div class="mt-auto pt-6 flex flex-col relative z-10">
        
        <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-amber-700/60 dark:text-zinc-500 mb-1 drop-shadow-sm">
          {{ primaryTag || "App" }}
        </div>
        <div class="text-4xl sm:text-5xl font-black tracking-tighter text-amber-950 dark:text-white transition-colors line-clamp-2 leading-[1.1] mb-2 drop-shadow-sm group-hover:-translate-y-1 duration-500">
          {{ dailyApp?.name }}
        </div>
        <p v-if="dailyApp?.short_description" class="text-sm font-medium text-amber-900/60 dark:text-zinc-400 line-clamp-2 leading-relaxed max-w-[90%]">
          {{ dailyApp.short_description }}
        </p>
        
        <!-- Arrow indicator at bottom right -->
        <div class="absolute bottom-0 right-0 opacity-0 -translate-x-4 translate-y-4 group-hover:opacity-100 group-hover:translate-x-0 group-hover:translate-y-0 transition-all duration-500">
          <div class="w-12 h-12 rounded-full flex items-center justify-center bg-white/60 dark:bg-black/40 backdrop-blur-xl border border-white dark:border-white/5 shadow-sm hover:bg-amber-50 hover:border-amber-200 hover:text-amber-600 transition-colors duration-300">
             <ArrowUpRight class="w-5 h-5 text-amber-600 dark:text-amber-400" />
          </div>
        </div>
      </div>
    </div>
  </button>

  <!-- Error State -->
  <div
    v-else-if="loadFailed"
    class="relative group h-full flex flex-col rounded-[24px] overflow-hidden transition-all duration-500 text-left w-full min-h-[320px] border border-red-100/50 dark:border-white/5 bg-gradient-to-br from-red-50/50 to-rose-50/50 dark:from-[#1a0f12] dark:to-[#0f090a]"
  >
    <AlertCircle class="absolute -bottom-8 -right-8 w-64 h-64 text-red-500/10 dark:text-red-400/[0.03] rotate-[-20deg] pointer-events-none" />

    <div class="relative z-10 p-7 flex flex-col h-full gap-4">
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <div class="relative w-14 h-14 rounded-2xl flex items-center justify-center shrink-0 bg-gradient-to-br from-red-400 to-rose-600 shadow-xl shadow-red-500/30">
            <div class="absolute inset-0 bg-white/30 dark:bg-white/20 rounded-2xl rounded-bl-none opacity-50 dark:opacity-40 mix-blend-overlay"></div>
            <AlertCircle class="w-7 h-7 text-white drop-shadow-md z-10" />
          </div>
          <div>
            <h3 class="text-xl font-black text-red-950 dark:text-white tracking-tight leading-none">{{ t("home.dailyAppCard.featuredToday") }}</h3>
            <p class="text-[10px] font-bold uppercase tracking-[0.2em] text-red-600/70 dark:text-red-400 mt-2">Error</p>
          </div>
        </div>
      </div>

      <div class="mt-auto pt-6 flex flex-col relative z-10">
        <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-red-700/60 dark:text-zinc-500 mb-1 drop-shadow-sm">
          Failed to load
        </div>
        <div class="text-4xl font-black tracking-tighter text-red-950 dark:text-white mb-2 drop-shadow-sm">
          {{ t("common.error") }}
        </div>
      </div>
    </div>
  </div>
</template>