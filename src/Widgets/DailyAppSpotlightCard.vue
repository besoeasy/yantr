<script setup>
import { ref, computed, onMounted } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { ArrowRight, Layers, Sparkles } from "lucide-vue-next";
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

const actionLabel = computed(() => {
  if (appState.value === "running") return t("home.dailyAppCard.openOverview");
  return t("home.dailyAppCard.viewApp");
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
    class="relative group h-full w-full flex flex-col overflow-hidden rounded-xl bg-white dark:bg-[#0A0A0A] text-left transition-all duration-400 hover:-translate-y-0.5 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40"
  >
    <div class="absolute inset-x-0 top-0 h-0.5 bg-amber-500 opacity-0 transition-opacity duration-500 group-hover:opacity-100"></div>

    <div class="relative z-10 flex h-full flex-col p-6">
      <div class="flex items-start justify-between gap-3 mb-5">
        <div>
          <div class="inline-flex items-center gap-2 rounded-full border border-amber-200 bg-amber-50 px-3 py-1 text-[10px] font-bold uppercase tracking-[0.18em] text-amber-700 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-300">
            <Sparkles class="h-3.5 w-3.5" />
            <span>{{ t("home.dailyAppCard.featuredToday") }}</span>
          </div>
          <h3 class="mt-4 text-xl font-semibold tracking-tight text-gray-900 transition-colors group-hover:text-amber-600 dark:text-white dark:group-hover:text-amber-300">
            {{ dailyApp?.name }}
          </h3>
          <p class="mt-1 text-[11px] font-medium uppercase tracking-[0.18em] text-gray-400 dark:text-zinc-500">
            {{ t("home.dailyAppCard.subtitle") }}
          </p>
        </div>

        <div class="flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl border border-gray-100 bg-gray-50 p-3 dark:border-zinc-800 dark:bg-zinc-900">
          <AppLogo
            :logo="dailyApp?.logo"
            :name="dailyApp?.name"
            :seed="dailyApp?.id || dailyApp?.name"
            img-class="h-full w-full object-contain"
            icon-class="h-6 w-6 text-gray-400 dark:text-zinc-500"
          />
        </div>
      </div>

      <p class="text-sm font-medium leading-relaxed text-gray-500 dark:text-zinc-400 line-clamp-3">
        {{ dailyApp?.description || t("home.dailyAppCard.noDescription") }}
      </p>

      <div class="mt-5 flex flex-wrap gap-2">
        <div
          v-if="primaryTag"
          class="inline-flex items-center gap-2 rounded-md border border-gray-200 bg-gray-50 px-3 py-1.5 dark:border-zinc-800 dark:bg-zinc-900/70"
        >
          <Layers class="h-3.5 w-3.5 text-gray-400 dark:text-zinc-500" />
          <span class="text-[11px] font-semibold uppercase tracking-wider text-gray-600 dark:text-zinc-300">{{ primaryTag }}</span>
        </div>
        <div
          :class="[
            'inline-flex items-center gap-2 rounded-md border px-3 py-1.5',
            appState === 'running'
              ? 'border-emerald-200 bg-emerald-50 dark:border-emerald-500/20 dark:bg-emerald-500/10'
              : appState === 'installed'
                ? 'border-blue-200 bg-blue-50 dark:border-blue-500/20 dark:bg-blue-500/10'
                : 'border-gray-200 bg-gray-50 dark:border-zinc-800 dark:bg-zinc-900/70'
          ]"
        >
          <span
            :class="[
              'h-2 w-2 rounded-full',
              appState === 'running'
                ? 'bg-emerald-500'
                : appState === 'installed'
                  ? 'bg-blue-500'
                  : 'bg-gray-400 dark:bg-zinc-500'
            ]"
          ></span>
          <span
            :class="[
              'text-[11px] font-semibold uppercase tracking-wider',
              appState === 'running'
                ? 'text-emerald-700 dark:text-emerald-300'
                : appState === 'installed'
                  ? 'text-blue-700 dark:text-blue-300'
                  : 'text-gray-600 dark:text-zinc-300'
            ]"
          >
            {{ stateLabel }}
          </span>
        </div>
      </div>

      <div class="mt-auto pt-5 flex items-center justify-between gap-3 border-t border-gray-100 dark:border-zinc-800/80">
        <div>
          <p class="text-[10px] font-bold uppercase tracking-[0.18em] text-gray-400 dark:text-zinc-500">{{ t("home.dailyAppCard.rotatesDaily") }}</p>
          <p class="mt-1 text-xs font-semibold text-gray-600 dark:text-zinc-300">{{ actionLabel }}</p>
        </div>
        <div class="flex h-9 w-9 items-center justify-center rounded-full border border-gray-200 bg-white text-gray-700 transition-all group-hover:translate-x-0.5 group-hover:border-gray-300 group-hover:text-black dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-300 dark:group-hover:border-zinc-600 dark:group-hover:text-white">
          <ArrowRight class="h-4 w-4" />
        </div>
      </div>
    </div>
  </button>

  <div
    v-else-if="loadFailed"
    class="relative h-full w-full flex flex-col overflow-hidden rounded-xl bg-white dark:bg-[#0A0A0A] text-left"
  >
    <div class="relative z-10 flex h-full flex-col p-6">
      <div class="flex items-start justify-between gap-3 mb-5">
        <div>
          <div class="inline-flex items-center gap-2 rounded-full border border-amber-200 bg-amber-50 px-3 py-1 text-[10px] font-bold uppercase tracking-[0.18em] text-amber-700 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-300">
            <Sparkles class="h-3.5 w-3.5" />
            <span>{{ t("home.dailyAppCard.featuredToday") }}</span>
          </div>
          <h3 class="mt-4 text-xl font-semibold tracking-tight text-gray-900 dark:text-white">
            {{ t("common.error") }}
          </h3>
          <p class="mt-1 text-[11px] font-medium uppercase tracking-[0.18em] text-gray-400 dark:text-zinc-500">
            {{ t("home.dailyAppCard.subtitle") }}
          </p>
        </div>

        <div class="flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl border border-gray-100 bg-gray-50 p-3 dark:border-zinc-800 dark:bg-zinc-900">
          <Bot class="h-6 w-6 text-gray-400 dark:text-zinc-500" />
        </div>
      </div>

      <p class="text-sm font-medium leading-relaxed text-gray-500 dark:text-zinc-400 line-clamp-3">
        Failed to load featured app.
      </p>
    </div>
  </div>
</template>