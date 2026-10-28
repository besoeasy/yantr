<script>
// Two columns, like every other wide widget on the dashboard. The card is a
// pointer into the developer page now rather than a support surface: identity,
// the apps credited to the account, and one CTA. The payment rails, the DM
// code and the follow link all live on that page, so they are not duplicated
// here at a size that would not scan anyway.
export const meta = { cols: 2 };
</script>
<script setup>
import { computed, watch } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { ArrowUpRight } from "@lucide/vue";
import AppLogo from "../components/AppLogo.vue";
import { useDeveloperProfile } from "../composables/useDeveloperProfile";
import { useCatalog } from "../composables/useDashboardData";
import WidgetSkeleton from "../components/WidgetSkeleton.vue";
import { appDetail, developerDetail } from "../utils/routes.js";
import {
  appsByDeveloper,
  dailyDeveloperPick,
  developerInitials,
  developerKeysFromApps,
  developerShortKey,
} from "../utils/developer";

const { t } = useI18n();
const router = useRouter();

const { profile, load: loadDeveloper } = useDeveloperProfile();

const { apps, appsLoaded, appsLoadFailed } = useCatalog();

const pickedKey = computed(() => {
  if (!appsLoaded.value || appsLoadFailed.value) return "";
  return dailyDeveloperPick(developerKeysFromApps(apps.value));
});

const developerApps = computed(() => appsByDeveloper(apps.value, pickedKey.value));

const SHOWCASE_APP_LIMIT = 4;
const showcaseApps = computed(() => developerApps.value.slice(0, SHOWCASE_APP_LIMIT));
const hiddenAppCount = computed(() => Math.max(0, developerApps.value.length - SHOWCASE_APP_LIMIT));

const displayName = computed(() => profile.value?.name?.trim() || pickedKey.value);
const initials = computed(() => developerInitials(displayName.value));
const avatarUrl = computed(() => profile.value?.avatarUrl ?? "");

const npubShort = computed(() => developerShortKey(pickedKey.value));

// Prefer the account's own words when the relay answered; the canned pitch is
// the fallback for an account with no profile or an unreachable relay.
const pitch = computed(() => {
  const bio = String(profile.value?.bio ?? "").trim();
  return bio || t("home.dailyDeveloper.pitch");
});

function viewApp(appId) {
  router.push(appDetail(appId));
}

function viewDeveloper() {
  router.push(developerDetail(pickedKey.value));
}

watch(pickedKey, (next) => {
  if (next) loadDeveloper(next);
});
</script>

<template>
  <WidgetSkeleton v-if="!appsLoaded && !appsLoadFailed" />

  <section
    v-else-if="pickedKey"
    class="yantra-card yantra-card-min group relative flex h-full w-full flex-col overflow-hidden p-7 transition-[border-color,box-shadow] duration-200 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <div
      aria-hidden="true"
      class="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-fuchsia-500/30 to-transparent dark:via-fuchsia-500/40"
    ></div>

    <div class="flex flex-1 flex-col gap-5">
      <!-- Identity -->
      <div class="flex items-start justify-between gap-4">
        <button
          @click="viewDeveloper"
          class="flex min-w-0 items-center gap-4 text-left"
        >
          <div class="flex h-12 w-12 shrink-0 items-center justify-center overflow-hidden rounded-xl border-fuchsia-200 bg-fuchsia-50 transition-transform duration-200 group-hover:scale-105 dark:border-fuchsia-900/50 dark:bg-fuchsia-900/20">
            <img
              v-if="avatarUrl"
              :src="avatarUrl"
              :alt="displayName"
              referrerpolicy="no-referrer"
              class="h-full w-full object-cover"
            />
            <span v-else class="text-lg font-black tracking-tight text-fuchsia-600 dark:text-fuchsia-400">
              {{ initials }}
            </span>
          </div>
          <div class="min-w-0">
            <span class="block max-w-full truncate text-xl font-bold tracking-tight yantra-text-primary transition-opacity hover:opacity-70">
              {{ displayName }}
            </span>
            <p :title="pickedKey" class="mt-0.5 truncate font-mono text-[11px] yantra-text-secondary">
              {{ npubShort }}
            </p>
          </div>
        </button>

        <div class="shrink-0 text-right">
          <p class="text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
            {{ t("home.dailyDeveloper.title") }}
          </p>
          <div class="mt-1.5 flex items-baseline justify-end gap-1.5">
            <span class="text-xl font-bold tabular-nums yantra-text-primary">
              {{ developerApps.length }}
            </span>
            <span class="text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
              {{ t("home.dailyDeveloper.appsCount") }}
            </span>
          </div>
        </div>
      </div>

      <!-- Pitch -->
      <p class="line-clamp-3 text-sm leading-relaxed yantra-text-secondary">
        {{ pitch }}
      </p>

      <!-- Apps -->
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-for="app in showcaseApps"
          :key="app.id"
          @click="viewApp(app.id)"
          class="inline-flex max-w-full items-center gap-2 rounded-full bg-zinc-50 py-1 pl-1 pr-3 text-xs font-medium text-zinc-700 transition-colors hover:border-zinc-300 hover:bg-white hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-300 dark:hover:border-zinc-700 dark:hover:bg-zinc-900 dark:hover:text-zinc-100"
        >
          <span class="flex h-5 w-5 shrink-0 items-center justify-center overflow-hidden rounded-full">
            <AppLogo
              :logo="app.logo"
              :name="app.name"
              :seed="app.id"
              img-class="h-full w-full object-contain"
              icon-class="h-4 w-4 text-zinc-300 dark:text-zinc-700"
            />
          </span>
          <span class="truncate">{{ app.name }}</span>
        </button>
        <span
          v-if="hiddenAppCount > 0"
          class="rounded-full border border-dashed border-zinc-200 px-3 py-1 text-xs font-medium yantra-text-secondary dark:border-zinc-800"
        >
          {{ t("home.dailyDeveloper.appsMore", { count: hiddenAppCount }) }}
        </span>
      </div>

      <!-- CTA -->
      <button
        @click="viewDeveloper"
        class="mt-auto inline-flex w-full items-center justify-center gap-1.5 rounded-full border border-fuchsia-200 bg-fuchsia-50 px-3.5 py-2.5 text-[11px] font-semibold text-fuchsia-700 transition-colors hover:bg-fuchsia-100 dark:border-fuchsia-900/50 dark:bg-fuchsia-900/20 dark:text-fuchsia-400 dark:hover:bg-fuchsia-900/30"
      >
        {{ t("home.dailyDeveloper.viewProfile") }}
        <ArrowUpRight :size="13" />
      </button>
    </div>
  </section>
</template>