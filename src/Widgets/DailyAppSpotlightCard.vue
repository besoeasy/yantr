<script setup>
import { ref, computed, onMounted } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { AlertCircle, ArrowRight, CalendarDays, Layers, Sparkles } from "lucide-vue-next";
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
    class="group flex h-full min-h-72 w-full flex-col rounded-xl p-5 text-left smooth-shadow transition-all duration-300 hover:-translate-y-0.5 hover:smooth-shadow-lg sm:p-6"
    style="background: var(--surface); color: var(--text-primary)"
  >
    <div class="flex h-full flex-col gap-5">
      <div class="flex items-center justify-between gap-3">
        <div class="flex min-w-0 items-center gap-2 text-[10px] font-bold uppercase tracking-[0.18em]" style="color: var(--text-secondary)">
          <Sparkles class="h-3.5 w-3.5 shrink-0 text-amber-500" />
          <span class="truncate">{{ t("home.dailyAppCard.featuredToday") }}</span>
        </div>

        <div
          :class="[
            'inline-flex shrink-0 items-center gap-1.5 text-[10px] font-bold uppercase tracking-[0.16em]',
            appState === 'running'
              ? 'text-emerald-600 dark:text-emerald-400'
              : appState === 'installed'
                ? 'text-blue-600 dark:text-blue-400'
                : 'text-amber-600 dark:text-amber-400'
          ]"
        >
          <span class="h-1.5 w-1.5 rounded-full bg-current"></span>
          <span>{{ stateLabel }}</span>
        </div>
      </div>

      <div class="flex items-start gap-4">
        <div class="flex h-16 w-16 shrink-0 items-center justify-center rounded-xl p-3 transition-transform duration-300 group-hover:scale-105" style="background: var(--surface-muted)">
          <AppLogo
            :logo="dailyApp?.logo"
            :name="dailyApp?.name"
            :seed="dailyApp?.id || dailyApp?.name"
            img-class="h-full w-full object-contain"
            icon-class="h-7 w-7 text-[var(--text-secondary)]"
          />
        </div>

        <div class="min-w-0 flex-1">
          <h3 class="text-2xl font-semibold tracking-tight transition-colors duration-300 group-hover:text-amber-600 dark:group-hover:text-amber-300">
            {{ dailyApp?.name }}
          </h3>
          <p class="mt-1 text-xs font-medium leading-relaxed" style="color: var(--text-secondary)">
            {{ t("home.dailyAppCard.subtitle") }}
          </p>
        </div>
      </div>

      <p class="line-clamp-3 text-sm font-medium leading-relaxed" style="color: var(--text-secondary)">
        {{ dailyApp?.description || t("home.dailyAppCard.noDescription") }}
      </p>

      <div class="mt-auto grid gap-3 text-xs font-medium" style="color: var(--text-secondary)">
        <div
          v-if="primaryTag"
          class="flex min-w-0 items-center gap-2"
        >
          <Layers class="h-3.5 w-3.5 shrink-0 text-blue-500" />
          <span class="truncate">{{ primaryTag }}</span>
        </div>
        <div class="flex min-w-0 items-center gap-2">
          <CalendarDays class="h-3.5 w-3.5 shrink-0 text-amber-500" />
          <span class="truncate">{{ t("home.dailyAppCard.rotatesDaily") }}</span>
        </div>
      </div>

      <div class="flex items-center justify-between gap-3 pt-1">
        <span class="text-sm font-semibold transition-colors duration-300 group-hover:text-amber-600 dark:group-hover:text-amber-300">
          {{ actionLabel }}
        </span>
        <ArrowRight class="h-4 w-4 shrink-0 text-amber-500 transition-transform duration-300 group-hover:translate-x-1" />
      </div>
    </div>
  </button>

  <div
    v-else-if="loadFailed"
    class="flex h-full min-h-72 w-full flex-col rounded-xl p-5 text-left smooth-shadow sm:p-6"
    style="background: var(--surface); color: var(--text-primary)"
  >
    <div class="flex h-full flex-col gap-5">
      <div class="flex items-center gap-2 text-[10px] font-bold uppercase tracking-[0.18em]" style="color: var(--text-secondary)">
        <Sparkles class="h-3.5 w-3.5 shrink-0 text-amber-500" />
        <span>{{ t("home.dailyAppCard.featuredToday") }}</span>
      </div>

      <div class="flex items-start gap-4">
        <div class="flex h-16 w-16 shrink-0 items-center justify-center rounded-xl" style="background: var(--surface-muted)">
          <AlertCircle class="h-7 w-7 text-amber-500" />
        </div>

        <div class="min-w-0 flex-1">
          <h3 class="text-2xl font-semibold tracking-tight">
            {{ t("common.error") }}
          </h3>
          <p class="mt-1 text-sm font-medium leading-relaxed" style="color: var(--text-secondary)">
            Failed to load featured app.
          </p>
        </div>
      </div>
    </div>
  </div>
</template>