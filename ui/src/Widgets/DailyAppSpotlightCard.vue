<script>
// Two columns: the logo becomes a poster rail on the left and the app's
// story (name, description, tags, CTA) sits to its right. The earlier
// single-column band existed to hold one pick, and at one column most of
// the 320px height was empty space around a name.
export const meta = { cols: 2 };
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

// One card, one app, one day. Nothing about the choice is server-driven or
// personalised: the catalog answers what exists, containers answer whether the
// pick is already running here, and the date picks which app it is. The same
// day reads the same on every host, and a reload never reshuffles it.

const { t } = useI18n();
const router = useRouter();

// Both lists are shared with the other dashboard cards that read them, so
// mounting costs no request of its own and the pick stays in step with the
// rest of the page.
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

const instanceCount = computed(() => dailyApp.value?.instanceCount ?? 0);

const appState = computed(() => {
  if (instanceCount.value > 0) return "running";
  if (dailyApp.value?.isInstalled) return "installed";
  return "available";
});

const ctaLabel = computed(() => {
  // The whole card still routes to the app page (where a running stack is
  // actually opened), but the label reflects the state so an already-running
  // pick reads as an action rather than a catalog link.
  if (appState.value === "running") return t("home.dailyAppCard.openOverview");
  return t("home.dailyAppCard.viewApp");
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
  <WidgetSkeleton v-if="!appsLoaded && !loadFailed" />

  <section
    v-else-if="dailyApp"
    class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow] duration-200 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <!-- Header: what this card is, and whether today's app is already
         running on this host. -->
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

    <!-- The app. The whole body is the click target: at two columns the
         logo becomes a poster rail and the story (name, description, tags,
         CTA) sits to its right. -->
    <button
      type="button"
      @click="handleSelect(dailyApp.id)"
      class="group/app mt-6 flex flex-1 flex-row items-start gap-6 text-left"
    >
      <!-- Poster rail: logo with the app's tags stacked beneath it. -->
      <div class="flex w-48 shrink-0 flex-col items-center gap-3">
        <div class="flex h-32 w-32 shrink-0 items-center justify-center overflow-hidden rounded-xl bg-zinc-50 transition-transform duration-200 group-hover/app:scale-105 dark:border-zinc-800 dark:bg-zinc-900/50">
          <AppLogo
            :logo="dailyApp?.logo"
            :name="dailyApp?.name"
            :seed="dailyApp?.id || dailyApp?.name"
            img-class="h-full w-full object-contain"
            icon-class="h-20 w-20 text-zinc-300 transition-colors duration-200 group-hover/app:text-zinc-400 dark:text-zinc-700 dark:group-hover/app:text-zinc-600"
          />
        </div>

        <div class="flex flex-col items-center gap-2">
          <span
            v-for="tag in (dailyApp?.tags || []).slice(0, 3)"
            :key="tag"
            class="rounded-full border border-zinc-200 bg-zinc-50 px-3 py-1 text-xs font-medium text-zinc-700 dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-300"
          >
            {{ tag }}
          </span>
          <span
            v-if="(dailyApp?.tags?.length || 0) > 3"
            class="text-[11px] font-medium yantra-text-secondary"
          >
            +{{ (dailyApp?.tags?.length || 0) - 3 }} more
          </span>
        </div>
      </div>

      <!-- Story rail: name, description, and the affordance. -->
      <div class="flex min-w-0 flex-1 flex-col">
        <div class="text-2xl font-black leading-[1.1] tracking-tighter yantra-text-primary transition-transform duration-200 group-hover/app:-translate-y-[3px]">
          {{ dailyApp?.name }}
        </div>

        <p v-if="dailyApp?.short_description" class="mt-3 text-sm leading-relaxed yantra-text-secondary line-clamp-3">
          {{ dailyApp.short_description }}
        </p>

        <span class="mt-auto inline-flex items-center gap-2 pt-5 text-[11px] font-bold uppercase tracking-wider yantra-text-secondary transition-colors group-hover/app:text-zinc-900 dark:group-hover/app:text-zinc-100">
          {{ ctaLabel }}
          <ArrowUpRight :size="14" />
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