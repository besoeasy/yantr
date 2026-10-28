<script>
// Dashboard engine: one column — the daily pick reads as a compact tile in
// the grid, with the logo on top and the story stacked beneath it.
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
import WidgetSkeleton from "../components/WidgetSkeleton.vue";

const { t } = useI18n();
const router = useRouter();

const { apps, appsLoaded, appsLoadFailed } = useCatalog();
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

const appState = computed(() => {
  const count = dailyApp.value?.instanceCount ?? 0;
  if (count > 0) return "running";
  if (dailyApp.value?.isInstalled) return "installed";
  return "available";
});

const stateLabel = computed(() => {
  if (appState.value === "running") {
    return t("home.dailyAppCard.running", { count: dailyApp.value?.instanceCount ?? 0 });
  }
  if (appState.value === "installed") return t("home.dailyAppCard.installed");
  return t("home.dailyAppCard.available");
});

const ctaLabel = computed(() => {
  if (appState.value === "running") return t("home.dailyAppCard.openOverview");
  return t("home.dailyAppCard.viewApp");
});

function handleSelect(appId) {
  if (!appId) return;
  router.push(appDetail(appId));
}
</script>

<template>
  <WidgetSkeleton v-if="!appsLoaded && !loadFailed" />

  <section
    v-else-if="dailyApp"
    class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow] duration-200 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <!-- Header -->
    <div class="flex w-full items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border-sky-200 bg-sky-50 transition-transform duration-200 group-hover:scale-105 dark:border-sky-900/50 dark:bg-sky-900/20">
          <Sparkles class="h-4 w-4 text-sky-600 dark:text-sky-400" />
        </div>
        <h3 class="text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
          {{ t("home.dailyAppCard.featuredToday") }}
        </h3>
      </div>

      <div
        :class="[
          'flex shrink-0 items-center gap-2',
          appState === 'running'
            ? 'yantra-text-primary'
            : appState === 'installed'
              ? 'text-zinc-700 dark:text-zinc-300'
              : 'yantra-text-secondary'
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

    <!-- Logo (centered, larger) -->
    <button
      type="button"
      @click="handleSelect(dailyApp.id)"
      class="group/app mt-5 flex flex-1 flex-col items-center text-center"
    >
      <div class="flex h-28 w-28 shrink-0 items-center justify-center overflow-hidden rounded-2xl bg-zinc-50 transition-transform duration-200 group-hover/app:scale-105 dark:border-zinc-800 dark:bg-zinc-900/50">
        <AppLogo
          :logo="dailyApp?.logo"
          :name="dailyApp?.name"
          :seed="dailyApp?.id || dailyApp?.name"
          img-class="h-full w-full object-contain"
          icon-class="h-16 w-16 text-zinc-300 transition-colors duration-200 group-hover/app:text-zinc-400 dark:text-zinc-700 dark:group-hover/app:text-zinc-600"
        />
      </div>

      <!-- Story -->
      <div class="mt-4 flex w-full flex-col">
        <div class="text-xl font-black leading-[1.1] tracking-tighter yantra-text-primary transition-transform duration-200 group-hover/app:-translate-y-[3px]">
          {{ dailyApp?.name }}
        </div>

        <p v-if="dailyApp?.short_description" class="mt-2 text-xs leading-relaxed yantra-text-secondary line-clamp-3">
          {{ dailyApp.short_description }}
        </p>

        <div class="mt-3 flex flex-wrap justify-center gap-2">
          <span
            v-for="tag in (dailyApp?.tags || []).slice(0, 3)"
            :key="tag"
            class="rounded-full border border-zinc-200 bg-zinc-50 px-2.5 py-0.5 text-[10px] font-medium text-zinc-700 dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-300"
          >
            {{ tag }}
          </span>
          <span
            v-if="(dailyApp?.tags?.length || 0) > 3"
            class="text-[10px] font-medium yantra-text-secondary"
          >
            +{{ (dailyApp?.tags?.length || 0) - 3 }}
          </span>
        </div>

        <span class="mt-4 inline-flex items-center justify-center gap-2 text-[10px] font-bold uppercase tracking-wider yantra-text-secondary transition-colors group-hover/app:text-zinc-900 dark:group-hover/app:text-zinc-100">
          {{ ctaLabel }}
          <ArrowUpRight :size="12" />
        </span>
      </div>
    </button>
  </section>

  <!-- Error State -->
  <div
    v-else-if="loadFailed"
    class="yantra-card yantra-card-min flex h-full w-full flex-col overflow-hidden border-red-200 p-7 text-left dark:border-red-900/50"
  >
    <div class="flex items-center gap-3">
      <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border-red-200 bg-red-50 dark:border-red-900/50 dark:bg-red-900/10">
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
      <div class="mt-2 text-3xl font-black leading-[1.1] tracking-tighter text-red-950 yantra-text-primary">
        {{ t("common.error") }}
      </div>
    </div>
  </div>
</template>