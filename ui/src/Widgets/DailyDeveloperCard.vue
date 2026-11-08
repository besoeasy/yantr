<script>
// Two columns, like every other wide widget on the dashboard. The card is a
// pointer into the developer page now rather than a support surface: identity,
// the apps credited to the account, and one CTA. The payment rails, the DM
// code and the follow link all live on that page, so they are not duplicated
// here at a size that would not scan anyway.
export const meta = { cols: 2 };
</script>
<script setup>
import { computed, onUnmounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import {
  AlertCircle,
  ArrowUpRight,
  AtSign,
  Check,
  Copy,
  Sparkles,
  Zap,
} from "@lucide/vue";
import AppLogo from "../components/AppLogo.vue";
import { useDeveloperProfile } from "../composables/useDeveloperProfile";
import { useCatalog, useContainers } from "../composables/useDashboardData";
import WidgetSkeleton from "../components/WidgetSkeleton.vue";
import { appDetail, developerDetail } from "../utils/routes.js";
import {
  DEVELOPER_LABEL,
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
const { containers } = useContainers({ live: true });

const avatarError = ref(false);
const copied = ref(false);
let copiedTimer = null;

const pickedKey = computed(() => {
  if (!appsLoaded.value || appsLoadFailed.value) return "";
  return dailyDeveloperPick(developerKeysFromApps(apps.value));
});

const developerApps = computed(() => appsByDeveloper(apps.value, pickedKey.value));

const runningAppIds = computed(() => {
  return new Set(
    containers.value
      .filter((c) => c.state === "running")
      .map((c) => c?.app?.id)
      .filter(Boolean)
  );
});

const activeAppsCount = computed(() => {
  return developerApps.value.filter((app) => runningAppIds.value.has(app.id)).length;
});

const SHOWCASE_APP_LIMIT = 4;
const showcaseApps = computed(() => developerApps.value.slice(0, SHOWCASE_APP_LIMIT));
const hiddenAppCount = computed(() => Math.max(0, developerApps.value.length - SHOWCASE_APP_LIMIT));

const hasCustomName = computed(() => Boolean(profile.value?.name?.trim()));
const npubShort = computed(() => developerShortKey(pickedKey.value));
const displayName = computed(() => profile.value?.name?.trim() || npubShort.value);
const initials = computed(() => developerInitials(profile.value?.name?.trim() || pickedKey.value));
const avatarUrl = computed(() => profile.value?.avatarUrl ?? "");

const hasZapSupport = computed(() => Boolean(profile.value?.lud16 || profile.value?.bitcoin));

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

async function copyNostrKey() {
  if (!pickedKey.value) return;
  try {
    await navigator.clipboard.writeText(pickedKey.value);
    copied.value = true;
    if (copiedTimer) clearTimeout(copiedTimer);
    copiedTimer = setTimeout(() => {
      copied.value = false;
    }, 2000);
  } catch {
    // Clipboard access might fail in insecure contexts
  }
}

watch(
  pickedKey,
  (next) => {
    avatarError.value = false;
    if (next) loadDeveloper(next);
  },
  { immediate: true }
);

onUnmounted(() => {
  if (copiedTimer) clearTimeout(copiedTimer);
});
</script>

<template>
  <WidgetSkeleton v-if="!appsLoaded && !appsLoadFailed" />

  <section
    v-else-if="pickedKey"
    class="yantra-card yantra-card-min group relative flex h-full w-full flex-col overflow-hidden p-7 text-left"
  >
    <div
      aria-hidden="true"
      class="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-fuchsia-500/30 to-transparent dark:via-fuchsia-500/40"
    ></div>

    <div class="flex flex-1 flex-col gap-5">
      <!-- Eyebrow row: title and stats -->
      <div class="flex w-full items-center justify-between gap-3">
        <div class="flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-widest text-fuchsia-600 dark:text-fuchsia-400">
          <Sparkles class="h-3.5 w-3.5" />
          {{ t("home.dailyDeveloper.title") }}
        </div>

        <div class="flex shrink-0 items-center gap-2">
          <div
            v-if="activeAppsCount > 0"
            class="flex items-center gap-1.5 rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-0.5 text-[9px] font-semibold uppercase tracking-wider text-emerald-700 dark:border-emerald-900/50 dark:bg-emerald-900/20 dark:text-emerald-400"
          >
            <span class="relative flex h-1.5 w-1.5">
              <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
              <span class="relative inline-flex h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
            </span>
            <span>{{ t("home.dailyAppCard.running", { count: activeAppsCount }) }}</span>
          </div>

          <div class="flex items-baseline gap-1 text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
            <span class="text-sm font-black tabular-nums yantra-text-primary">{{ developerApps.length }}</span>
            <span>{{ t("home.dailyDeveloper.appsCount") }}</span>
          </div>
        </div>
      </div>

      <!-- Identity -->
      <button
        type="button"
        @click="viewDeveloper"
        class="flex min-w-0 items-center gap-4 text-left"
      >
        <div class="flex h-12 w-12 shrink-0 items-center justify-center overflow-hidden rounded-xl border border-fuchsia-200 bg-fuchsia-50 transition-transform duration-200 group-hover:scale-105 dark:border-fuchsia-900/50 dark:bg-fuchsia-900/20">
          <img
            v-if="avatarUrl && !avatarError"
            :src="avatarUrl"
            :alt="displayName"
            referrerpolicy="no-referrer"
            @error="avatarError = true"
            class="h-full w-full object-cover"
          />
          <span v-else class="text-lg font-black tracking-tight text-fuchsia-600 dark:text-fuchsia-400">
            {{ initials }}
          </span>
        </div>

        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <span class="block max-w-full truncate text-xl font-bold tracking-tight yantra-text-primary transition-opacity hover:opacity-70">
              {{ displayName }}
            </span>
            <span class="inline-flex items-center rounded-md border border-fuchsia-200 bg-fuchsia-50 px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-wider text-fuchsia-700 dark:border-fuchsia-900/50 dark:bg-fuchsia-900/20 dark:text-fuchsia-400">
              {{ DEVELOPER_LABEL }}
            </span>
          </div>

          <div class="mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] yantra-text-secondary">
            <span v-if="profile?.nip05" class="inline-flex max-w-full items-center gap-1 truncate font-medium text-fuchsia-700 dark:text-fuchsia-400">
              <AtSign :size="12" class="shrink-0" />
              <span class="truncate">{{ profile.nip05 }}</span>
            </span>

            <span v-if="hasCustomName" :title="pickedKey" class="truncate font-mono">
              {{ npubShort }}
            </span>

            <span v-if="hasZapSupport" class="inline-flex items-center gap-0.5 font-medium text-amber-600 dark:text-amber-400">
              <Zap :size="11" class="text-amber-500 dark:text-amber-400" />
              <span>Zap</span>
            </span>
          </div>
        </div>
      </button>

      <!-- Pitch -->
      <p class="line-clamp-2 text-sm leading-relaxed yantra-text-secondary">
        {{ pitch }}
      </p>

      <!-- Apps -->
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-for="app in showcaseApps"
          :key="app.id"
          type="button"
          @click="viewApp(app.id)"
          class="inline-flex max-w-full items-center gap-2 rounded-full border border-zinc-200 bg-zinc-50 py-1 pl-1 pr-3 text-xs font-medium text-zinc-700 transition-colors hover:border-zinc-300 hover:bg-white hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-300 dark:hover:border-zinc-700 dark:hover:bg-zinc-900 dark:hover:text-zinc-100"
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
          <span
            v-if="runningAppIds.has(app.id)"
            class="h-1.5 w-1.5 shrink-0 rounded-full bg-emerald-500"
            :title="t('home.dailyAppCard.running', { count: 1 })"
          ></span>
        </button>

        <button
          v-if="hiddenAppCount > 0"
          type="button"
          @click="viewDeveloper"
          class="rounded-full border border-dashed border-zinc-200 px-3 py-1 text-xs font-medium yantra-text-secondary transition-colors hover:border-zinc-300 hover:text-zinc-900 dark:border-zinc-800 dark:hover:border-zinc-700 dark:hover:text-zinc-100"
        >
          {{ t("home.dailyDeveloper.appsMore", { count: hiddenAppCount }) }}
        </button>
      </div>

      <!-- CTA and Actions -->
      <div class="mt-auto flex items-center gap-2.5 pt-2">
        <button
          type="button"
          @click="viewDeveloper"
          class="group/cta flex h-10 flex-1 items-center justify-center gap-1.5 rounded-xl border border-fuchsia-200 bg-fuchsia-50 text-[10px] font-bold uppercase tracking-widest text-fuchsia-700 transition-colors duration-200 hover:bg-fuchsia-100 dark:border-fuchsia-900/50 dark:bg-fuchsia-900/20 dark:text-fuchsia-400 dark:hover:bg-fuchsia-900/30 active:scale-[0.98]"
        >
          <span>{{ t("home.dailyDeveloper.viewProfile") }}</span>
          <ArrowUpRight :size="13" class="transition-transform duration-200 group-hover/cta:-translate-y-1 group-hover/cta:translate-x-1" />
        </button>

        <button
          type="button"
          @click="copyNostrKey"
          :title="t('developer.nostrCopy')"
          class="flex h-10 shrink-0 items-center justify-center gap-1.5 rounded-xl border border-zinc-200 bg-zinc-50 px-3 text-[10px] font-bold uppercase tracking-wider yantra-text-secondary transition-colors hover:border-zinc-300 hover:bg-white hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900/50 dark:hover:border-zinc-700 dark:hover:bg-zinc-900 dark:hover:text-zinc-100 active:scale-[0.98]"
        >
          <Check v-if="copied" :size="13" class="text-emerald-600 dark:text-emerald-400" />
          <Copy v-else :size="13" />
          <span>{{ copied ? t("developer.nostrCopied") : t("developer.nostrCopy") }}</span>
        </button>
      </div>
    </div>
  </section>

  <!-- Error State -->
  <div
    v-else-if="appsLoadFailed"
    class="yantra-card yantra-card-min flex h-full w-full flex-col overflow-hidden border-red-200 p-7 text-left dark:border-red-900/50"
  >
    <div class="flex items-center gap-3">
      <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border-red-200 bg-red-50 dark:border-red-900/50 dark:bg-red-900/10">
        <AlertCircle class="h-4 w-4 text-red-600 dark:text-red-500" />
      </div>
      <h3 class="text-[10px] font-bold uppercase tracking-widest text-red-600 dark:text-red-400">
        {{ t("home.dailyDeveloper.title") }}
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
