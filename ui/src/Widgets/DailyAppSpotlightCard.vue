<script>
// Dashboard engine: one column — the daily pick as a "station" card: a soft
// sky-to-indigo wash behind an eyebrow + big name, a logo + story row, and a
// bordered CTA that spans the card.
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
    class="spotlight-card yantra-card yantra-card-min flex h-full w-full flex-col overflow-hidden p-7 text-left"
  >
    <!-- Eyebrow row: what this is, and whether the pick is live here. -->
    <div class="flex w-full items-center justify-between gap-3">
      <div class="flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
        <Sparkles class="h-3.5 w-3.5 text-sky-600 dark:text-sky-400" />
        {{ t("home.dailyAppCard.featuredToday") }}
      </div>

      <div
        :class="[
          'flex shrink-0 items-center gap-1.5',
          appState === 'running'
            ? 'text-emerald-600 dark:text-emerald-400'
            : 'yantra-text-secondary'
        ]"
      >
        <span v-if="appState === 'running'" class="relative flex h-2 w-2">
          <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
          <span class="relative inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
        </span>
        <span v-else class="h-2 w-2 rounded-full bg-zinc-300 dark:bg-zinc-700"></span>
        <span class="text-[9px] font-semibold uppercase tracking-widest">{{ stateLabel }}</span>
      </div>
    </div>

    <button
      type="button"
      @click="handleSelect(dailyApp.id)"
      class="group/app mt-4 flex flex-1 flex-col text-left"
    >
      <div class="text-2xl font-black leading-[1.05] tracking-tighter yantra-text-primary transition-transform duration-200 group-hover/app:-translate-y-[3px]">
        {{ dailyApp?.name }}
      </div>

      <div class="mt-3 flex items-start gap-3">
        <div class="flex h-14 w-14 shrink-0 items-center justify-center overflow-hidden rounded-xl border border-zinc-200 bg-white p-1.5 transition-transform duration-200 group-hover/app:scale-105 dark:border-zinc-800 dark:bg-zinc-900">
          <AppLogo
            :logo="dailyApp?.logo"
            :name="dailyApp?.name"
            :seed="dailyApp?.id || dailyApp?.name"
            img-class="h-full w-full object-contain"
            icon-class="h-8 w-8 text-zinc-300 transition-colors duration-200 group-hover/app:text-zinc-400 dark:text-zinc-700 dark:group-hover/app:text-zinc-600"
          />
        </div>

        <div class="flex min-w-0 flex-1 flex-col">
          <p v-if="dailyApp?.short_description" class="text-[11px] leading-relaxed yantra-text-secondary line-clamp-2">
            {{ dailyApp.short_description }}
          </p>
          <div class="mt-2 flex flex-wrap gap-1.5">
            <span
              v-for="tag in (dailyApp?.tags || []).slice(0, 2)"
              :key="tag"
              class="rounded-full border border-zinc-200 px-2 py-0.5 text-[9px] font-medium text-zinc-700 dark:border-zinc-800 dark:text-zinc-300"
            >
              {{ tag }}
            </span>
            <span
              v-if="(dailyApp?.tags?.length || 0) > 2"
              class="text-[9px] font-medium yantra-text-secondary"
            >
              +{{ (dailyApp?.tags?.length || 0) - 2 }}
            </span>
          </div>
        </div>
      </div>
    </button>

    <!-- CTA: full-width, bordered so it reads as an affordance over the wash. -->
    <button
      type="button"
      @click="handleSelect(dailyApp.id)"
      class="group/cta mt-5 flex h-10 w-full items-center justify-center gap-2 rounded-xl border border-zinc-200 bg-zinc-50 text-[10px] font-bold uppercase tracking-widest yantra-text-primary transition-colors duration-200 hover:bg-white dark:border-zinc-800 dark:bg-zinc-900/50 dark:hover:bg-zinc-900 active:scale-[0.98]"
    >
      {{ ctaLabel }}
      <ArrowUpRight :size="12" class="transition-transform duration-200 group-hover/cta:-translate-y-1 group-hover/cta:translate-x-1" />
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

<style scoped>
/* Instance styling: the wash behind this card is this card's own look —
   main.css keeps what a card *is*; the gradient is what this one is. */
.spotlight-card {
  --card-inset: var(--color-zinc-50);
  background:
    linear-gradient(180deg, rgb(2 132 199 / 0.10), rgb(99 102 241 / 0.05) 55%, transparent),
    var(--card-bg);
  border: 1px solid var(--card-border);
  border-radius: var(--card-radius);
  transition: box-shadow var(--yantra-dur) var(--yantra-ease);
}
.spotlight-card:hover {
  box-shadow: var(--card-shadow-hover);
}
:root.dark .spotlight-card {
  background:
    linear-gradient(180deg, rgb(2 132 199 / 0.16), rgb(99 102 241 / 0.08) 55%, transparent),
    var(--card-bg);
}
</style>