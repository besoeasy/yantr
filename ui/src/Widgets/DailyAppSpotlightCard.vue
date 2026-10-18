<script>
// Full-width band: all four of the desktop grid's four columns. The width that
// used to hold the single featured app now holds three — today's pick sized as
// the headline, plus the rotation's next two — so the app store promotion
// reads as a shelf rather than a solo card.
export const meta = { cols: 4 };
</script>
<script setup>
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { Sparkles, AlertCircle, ArrowUpRight } from "@lucide/vue";
import AppLogo from "../components/AppLogo.vue";
import { useApiUrl } from "../composables/useApiUrl";
import { expectApiSuccess } from "../composables/useApiResponse";
import { appDetail } from "../utils/routes.js";
import { appOfDay, upNextApps } from "../utils/spotlight";

// One card, one row of the dashboard. Nothing about the choice is
// server-driven or personalised: the catalog answers what exists, containers
// answer whether they are running, and the date picks which app leads. The
// two "up next" apps are tomorrow's and the day-after's literal picks — showcasing
// them means the card never says something the catalog rotation will not keep.

const { t } = useI18n();
const router = useRouter();
const { apiUrl } = useApiUrl();

const apps = ref([]);
const containers = ref([]);
const loadFailed = ref(false);

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
  const featured = appOfDay(apps.value);
  if (!featured) return null;
  return {
    ...featured,
    isInstalled: installedAppIds.value.has(featured.id),
    instanceCount: runningAppInstanceCounts.value[featured.id] || 0,
  };
});

// The rotation the featured app leads. Each entry carries only catalog facts:
// running state is a today-question and the "up next" shelves belong to days
// the containers of today say nothing about.
const upNext = computed(() => upNextApps(apps.value, dailyApp.value));

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
  } catch {
    loadFailed.value = true;
  }
}

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

// The "up next" caption per offset: tomorrow gets its own word, later days
// read as a countdown. The offset is the entry's position after the featured
// day — at least 1.
const SHOWN_UP_NEXT = 2;
function nextLabel(offset) {
  if (offset <= 1) return t("home.dailyAppCard.tomorrow");
  return t("home.dailyAppCard.inDays", { count: offset });
}

function handleSelect(appId) {
  if (!appId) return;
  router.push(appDetail(appId));
}

onMounted(fetchData);
</script>

<template>
  <section
    v-if="dailyApp"
    class="group relative flex h-full w-full min-h-[280px] flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-[border-color,box-shadow] duration-200 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <!-- Header. One row across the whole band: what this is, and whether the
         featured app is already yours. -->
    <div class="flex w-full items-start justify-between gap-4">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-sky-200 bg-sky-50 transition-transform duration-200 group-hover:scale-105 dark:border-sky-900/50 dark:bg-sky-900/20">
          <Sparkles class="h-5 w-5 text-sky-600 dark:text-sky-400" />
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

      <!-- Featured app's logo, kept top-right of the band -->
      <div class="flex h-12 shrink-0 items-center justify-center transition-transform duration-200 group-hover:scale-110 sm:h-14 sm:w-14">
        <AppLogo
          :logo="dailyApp?.logo"
          :name="dailyApp?.name"
          :seed="dailyApp?.id || dailyApp?.name"
          img-class="h-full w-full object-contain"
          icon-class="h-12 w-12 text-zinc-200 transition-colors duration-200 group-hover:text-zinc-300 dark:text-zinc-800 dark:group-hover:text-zinc-700"
        />
      </div>
    </div>

    <!-- Three shelves: the headline app, then the rotation's next two.
         Hairline dividers on desktop keep the sections from reading as one
         blob; on narrow screens they stack like the other bands. -->
    <div class="flex flex-1 flex-col gap-8 lg:flex-row lg:gap-0 lg:pt-2">
      <!-- Featured -->
      <button
        type="button"
        @click="handleSelect(dailyApp.id)"
        class="group/featured flex flex-1 flex-col text-left lg:pr-8"
      >
        <div class="mb-2 text-[10px] font-semibold uppercase tracking-widest text-zinc-500 dark:text-zinc-400">
          {{ primaryTag || "App" }}
        </div>
        <div class="mb-3 text-4xl font-black leading-[1.1] tracking-tighter text-zinc-900 transition-transform duration-200 group-hover/featured:-translate-y-[3px] sm:text-5xl dark:text-white">
          {{ dailyApp?.name }}
        </div>
        <p v-if="dailyApp?.short_description" class="max-w-[90%] text-sm leading-relaxed text-zinc-500 line-clamp-2 dark:text-zinc-400">
          {{ dailyApp.short_description }}
        </p>

        <span class="mt-auto inline-flex w-fit items-center gap-2 pt-4 text-[11px] font-bold uppercase tracking-wider text-zinc-500 transition-colors group-hover/featured:text-zinc-900 dark:group-hover/featured:text-zinc-100">
          {{ t("home.dailyAppCard.viewApp") }}
          <ArrowUpRight :size="14" />
        </span>
      </button>

      <!-- Up next -->
      <div
        v-for="(app, i) in upNext.slice(0, SHOWN_UP_NEXT)"
        :key="app.id"
        class="flex flex-1 flex-col lg:border-l lg:border-zinc-200 lg:pl-8 dark:lg:border-zinc-800"
      >
        <button
          type="button"
          @click="handleSelect(app.id)"
          class="group/next flex flex-1 flex-col text-left"
        >
          <span class="mb-2 flex items-center gap-2 text-[10px] font-semibold uppercase tracking-widest text-zinc-400 dark:text-zinc-500">
            {{ t("home.dailyAppCard.upNext") }}
            <span class="h-1 w-1 rounded-full bg-zinc-300 dark:bg-zinc-700"></span>
            <span class="text-zinc-500 dark:text-zinc-400">{{ nextLabel(i + 1) }}</span>
          </span>

          <span class="mb-3 flex items-center gap-3">
            <span class="flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-lg">
              <AppLogo
                :logo="app.logo"
                :name="app.name"
                :seed="app.id || app.name"
                img-class="h-full w-full object-contain"
                icon-class="h-7 w-7 text-zinc-200 transition-colors duration-200 group-hover/next:text-zinc-300 dark:text-zinc-800 dark:group-hover/next:text-zinc-700"
              />
            </span>
            <span class="text-xl font-bold tracking-tight text-zinc-900 transition-transform duration-200 group-hover/next:-translate-y-[3px] dark:text-white">
              {{ app.name }}
            </span>
          </span>

          <span v-if="app.short_description" class="line-clamp-2 text-[13px] leading-relaxed text-zinc-500 dark:text-zinc-400">
            {{ app.short_description }}
          </span>
          <span class="mt-auto inline-flex w-fit items-center gap-2 pt-4 text-[11px] font-bold uppercase tracking-wider text-zinc-500 transition-colors group-hover/next:text-zinc-900 dark:group-hover/next:text-zinc-100">
            {{ t("home.dailyAppCard.viewApp") }}
            <ArrowUpRight :size="14" />
          </span>
        </button>
      </div>
    </div>
  </section>

  <!-- Error State -->
  <div
    v-else-if="loadFailed"
    class="flex h-full w-full min-h-[280px] flex-col overflow-hidden rounded-2xl border border-red-200 bg-white p-7 text-left dark:border-red-900/50 dark:bg-black"
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
