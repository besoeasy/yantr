<script>
// One column of the desktop grid. A single app a day wants a single column:
// the earlier full-width band existed to hold three apps, and with one pick
// that width became mostly empty space around a name.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { Sparkles, AlertCircle, ArrowUpRight } from "@lucide/vue";
import AppLogo from "../components/AppLogo.vue";
import { appDetail } from "../utils/routes.js";
import { appOfDay } from "../utils/spotlight";
import { useCatalog, useContainers } from "../composables/useDashboardData";

// One card, one app, one day. Nothing about the choice is server-driven or
// personalised: the catalog answers what exists, containers answer whether the
// pick is already running here, and the date picks which app it is. The same
// day reads the same on every host, and a reload never reshuffles it.

const { t } = useI18n();
const router = useRouter();

// Both lists are shared with the other dashboard cards that read them, so
// mounting costs no request of its own and the pick stays in step with the
// rest of the page.
const { apps, appsLoadFailed } = useCatalog();
const { containers, containersLoadFailed } = useContainers({ live: true });

const loadFailed = computed(() => appsLoadFailed.value || containersLoadFailed.value);

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

function handleSelect(appId) {
  if (!appId) return;
  router.push(appDetail(appId));
}

</script>

<template>
  <section
    v-if="dailyApp"
    class="group relative flex h-full w-full min-h-[280px] flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-[border-color,box-shadow] duration-200 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <!-- Header: what this card is, and whether today's app is already
         running on this host. -->
    <div class="flex w-full items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-sky-200 bg-sky-50 transition-transform duration-200 group-hover:scale-105 dark:border-sky-900/50 dark:bg-sky-900/20">
          <Sparkles class="h-4 w-4 text-sky-600 dark:text-sky-400" />
        </div>
        <h3 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500 dark:text-zinc-400">
          {{ t("home.dailyAppCard.featuredToday") }}
        </h3>
      </div>

      <div
        :class="[
          'flex shrink-0 items-center gap-2',
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

    <!-- The app. The whole body is the click target: at one column there is
         no room for the affordance to live anywhere but the bottom. -->
    <button
      type="button"
      @click="handleSelect(dailyApp.id)"
      class="group/app mt-6 flex flex-1 flex-col text-left"
    >
      <div class="flex h-14 w-14 shrink-0 items-center justify-center overflow-hidden rounded-xl border border-zinc-200 bg-zinc-50 transition-transform duration-200 group-hover/app:scale-105 dark:border-zinc-800 dark:bg-zinc-900">
        <AppLogo
          :logo="dailyApp?.logo"
          :name="dailyApp?.name"
          :seed="dailyApp?.id || dailyApp?.name"
          img-class="h-full w-full object-contain"
          icon-class="h-11 w-11 text-zinc-300 transition-colors duration-200 group-hover/app:text-zinc-400 dark:text-zinc-700 dark:group-hover/app:text-zinc-600"
        />
      </div>

      <div class="mt-4 text-3xl font-black leading-[1.1] tracking-tighter text-zinc-900 transition-transform duration-200 group-hover/app:-translate-y-[3px] dark:text-white">
        {{ dailyApp?.name }}
      </div>

      <div class="mt-2 text-[10px] font-semibold uppercase tracking-widest text-zinc-500 dark:text-zinc-400">
        {{ primaryTag || "App" }}
      </div>

      <p v-if="dailyApp?.short_description" class="mt-3 text-sm leading-relaxed text-zinc-500 line-clamp-3 dark:text-zinc-400">
        {{ dailyApp.short_description }}
      </p>

      <span class="mt-auto inline-flex items-center gap-2 pt-5 text-[11px] font-bold uppercase tracking-wider text-zinc-500 transition-colors group-hover/app:text-zinc-900 dark:group-hover/app:text-zinc-100">
        {{ t("home.dailyAppCard.viewApp") }}
        <ArrowUpRight :size="14" />
      </span>
    </button>
  </section>

  <!-- Error State -->
  <div
    v-else-if="loadFailed"
    class="flex h-full w-full min-h-[280px] flex-col overflow-hidden rounded-2xl border border-red-200 bg-white p-7 text-left dark:border-red-900/50 dark:bg-black"
  >
    <div class="flex items-center gap-3">
      <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-red-200 bg-red-50 dark:border-red-900/50 dark:bg-red-900/10">
        <AlertCircle class="h-4 w-4 text-red-600 dark:text-red-500" />
      </div>
      <h3 class="text-[10px] font-bold uppercase tracking-widest text-red-600 dark:text-red-400">
        {{ t("home.dailyAppCard.featuredToday") }}
      </h3>
    </div>

    <div class="mt-auto flex flex-col">
      <div class="text-[10px] font-semibold uppercase tracking-widest text-red-500 dark:text-red-400">
        Failed to load
      </div>
      <div class="mt-2 text-3xl font-black leading-[1.1] tracking-tighter text-red-950 dark:text-white">
        {{ t("common.error") }}
      </div>
    </div>
  </div>
</template>