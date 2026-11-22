<script>
// Dashboard engine: two columns. The daily featured app is a poster on the
// left — logo, name and its lead category on a tinted plate — and a reading
// column on the right: description, tag pills, CTA. Below `md` the card is a
// single dashboard column, so the two halves stack; the side-by-side form only
// appears once the card itself spans two columns (AGENTS.md §8).
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
    class="yantra-card yantra-card-min group relative flex h-full w-full flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <div
      aria-hidden="true"
      class="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-amber-500/30 to-transparent dark:via-amber-500/40"
    ></div>

    <div class="flex flex-1 flex-col gap-5">
      <!-- Eyebrow row: title and status -->
      <div class="flex w-full items-center justify-between gap-3">
        <div class="flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-widest text-amber-600 dark:text-amber-400">
          <Sparkles class="h-3.5 w-3.5 text-amber-500" />
          {{ t("home.dailyAppCard.featuredToday") }}
        </div>

        <div class="flex shrink-0 items-center gap-2">
          <div
            v-if="appState === 'running'"
            class="flex items-center gap-1.5 rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-0.5 text-[9px] font-semibold uppercase tracking-wider text-emerald-700 dark:border-emerald-900/50 dark:bg-emerald-900/20 dark:text-emerald-400"
          >
            <span class="relative flex h-1.5 w-1.5">
              <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
              <span class="relative inline-flex h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
            </span>
            <span>{{ stateLabel }}</span>
          </div>

          <div
            v-else-if="appState === 'installed'"
            class="flex items-center gap-1.5 rounded-full border border-zinc-200 bg-zinc-50 px-2.5 py-0.5 text-[9px] font-semibold uppercase tracking-wider text-zinc-600 dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-400"
          >
            <span class="h-1.5 w-1.5 rounded-full bg-zinc-300 dark:bg-zinc-700"></span>
            <span>{{ stateLabel }}</span>
          </div>

          <div
            v-else
            class="flex items-center gap-1.5 text-[9px] font-semibold uppercase tracking-widest yantra-text-secondary"
          >
            <span class="h-1.5 w-1.5 rounded-full bg-zinc-300 dark:bg-zinc-700"></span>
            <span>{{ stateLabel }}</span>
          </div>
        </div>
      </div>

      <!-- Body: poster plate on the left, reading column on the right -->
      <div class="grid flex-1 grid-cols-1 gap-5 md:grid-cols-2">
        <!-- Poster: logo, name, lead category -->
        <button
          type="button"
          @click="handleSelect(dailyApp.id)"
          class="flex min-w-0 flex-col justify-center gap-5 rounded-2xl border border-amber-200 bg-amber-50/50 p-5 text-left transition-colors duration-200 hover:bg-amber-100 dark:border-amber-900/40 dark:bg-amber-900/20 dark:hover:bg-amber-900/30"
        >
          <div class="flex h-16 w-16 shrink-0 items-center justify-center overflow-hidden rounded-2xl border border-amber-200 bg-white shadow-sm transition-transform duration-200 group-hover:scale-105 dark:border-amber-900/50 dark:bg-zinc-900">
            <AppLogo
              :logo="dailyApp?.logo"
              :name="dailyApp?.name"
              :seed="dailyApp?.id || dailyApp?.name"
              img-class="h-full w-full object-contain p-1.5"
              icon-class="h-7 w-7 text-amber-600 dark:text-amber-400"
            />
          </div>

          <div class="min-w-0">
            <span class="block max-w-full truncate text-2xl font-bold leading-tight tracking-tight yantra-text-primary transition-opacity hover:opacity-70">
              {{ dailyApp?.name }}
            </span>

            <div v-if="dailyApp?.tags?.length" class="mt-1 flex items-center gap-1.5 text-[11px] yantra-text-secondary">
              <span class="truncate font-medium text-amber-700 dark:text-amber-400">
                #{{ dailyApp.tags[0] }}
              </span>
              <span v-if="dailyApp.tags.length > 1" class="text-zinc-400 dark:text-zinc-600">
                +{{ dailyApp.tags.length - 1 }}
              </span>
            </div>
          </div>
        </button>

        <!-- Reading column: description, tags, CTA -->
        <div class="flex min-w-0 flex-col gap-4">
          <p class="line-clamp-3 text-sm leading-relaxed yantra-text-secondary">
            {{ dailyApp?.short_description || dailyApp?.description || t('appCard.noDescription') }}
          </p>

          <div v-if="dailyApp?.tags?.length" class="flex flex-wrap items-center gap-1.5">
            <span
              v-for="tag in dailyApp.tags.slice(0, 3)"
              :key="tag"
              class="rounded-full border border-zinc-200 bg-zinc-50 px-2.5 py-0.5 text-[10px] font-medium text-zinc-600 dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-400"
            >
              {{ tag }}
            </span>
            <span
              v-if="dailyApp.tags.length > 3"
              class="text-[10px] font-medium yantra-text-secondary"
            >
              +{{ dailyApp.tags.length - 3 }}
            </span>
          </div>

          <div class="mt-auto pt-1">
            <button
              type="button"
              @click="handleSelect(dailyApp.id)"
              class="group/cta flex h-10 w-full items-center justify-center gap-1.5 rounded-xl border border-amber-200 bg-amber-50 text-[10px] font-bold uppercase tracking-widest text-amber-700 transition-colors duration-200 hover:bg-amber-100 dark:border-amber-900/50 dark:bg-amber-900/20 dark:text-amber-400 dark:hover:bg-amber-900/30 active:scale-[0.98]"
            >
              <span>{{ ctaLabel }}</span>
              <ArrowUpRight :size="13" class="transition-transform duration-200 group-hover/cta:-translate-y-1 group-hover/cta:translate-x-1" />
            </button>
          </div>
        </div>
      </div>
    </div>
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
