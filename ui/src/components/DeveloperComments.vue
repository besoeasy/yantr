<script setup>
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { RefreshCw, AlertCircle, Package } from "@lucide/vue";
import { useDeveloperComments } from "../composables/useDeveloperComments";
import { useApiUrl } from "../composables/useApiUrl";
import { developerInitials } from "../utils/developer";
import { appDetail } from "../utils/routes.js";

// One developer's comments across the whole catalog, shown under their apps on
// the developer page. The composable does the relay work; this component owns
// only the catalog mapping: each event carries the app id its author tagged
// (`t: yantr-<id>`), and the local catalog turns ids into names and links. A
// tag that names no app on this host renders as plain text rather than a dead
// link — the comment is still theirs, the app just is not ours.

const props = defineProps({
  developerKey: { type: String, required: true },
});

const { t, locale } = useI18n();
const { apiUrl } = useApiUrl();

const keyRef = computed(() => String(props.developerKey ?? ""));
const {
  comments,
  loading,
  loadFailed,
  relaysReached,
  relaysTotal,
  profile,
  load,
} = useDeveloperComments(keyRef);

// appId → app, filled from the catalog once the account turns out to have
// written anything — a page with no comments never pays for the request. A
// failed fetch leaves the map empty, which degrades every chip to plain text,
// never to a broken route.
const catalog = ref({});
const catalogFetched = ref(false);

async function fetchCatalog() {
  if (catalogFetched.value) return;
  catalogFetched.value = true;
  try {
    const res = await fetch(`${apiUrl.value}/api/apps`);
    const data = await res.json();
    const map = {};
    for (const app of Array.isArray(data.apps) ? data.apps : []) {
      if (app && app.id) map[String(app.id).trim().toLowerCase()] = app;
    }
    catalog.value = map;
  } catch {
    // Chips degrade to unknown-app text; the comments themselves stay up.
  }
}

watch(
  () => comments.value.length > 0,
  (hasComments) => {
    if (hasComments) fetchCatalog();
  },
  { immediate: true }
);

function appOf(appId) {
  return catalog.value[String(appId ?? "").trim().toLowerCase()] || null;
}

function appName(appId) {
  return appOf(appId)?.name || "";
}

// The meta line, as on the app thread: relative time to read, exact stamp in
// the title to keep.
const relativeTime = computed(() => {
  try {
    return new Intl.RelativeTimeFormat(locale.value, { numeric: "auto" });
  } catch {
    return null;
  }
});

const TIME_UNITS = [
  ["year", 31536000],
  ["month", 2592000],
  ["week", 604800],
  ["day", 86400],
  ["hour", 3600],
  ["minute", 60],
];

function formatTime(createdAt) {
  const seconds = Number(createdAt);
  if (!Number.isFinite(seconds) || seconds <= 0) return "";

  const formatter = relativeTime.value;
  if (!formatter) {
    try {
      return new Date(seconds * 1000).toLocaleDateString();
    } catch {
      return "";
    }
  }

  const delta = seconds - Date.now() / 1000;
  for (const [unit, size] of TIME_UNITS) {
    if (Math.abs(delta) >= size) return formatter.format(Math.round(delta / size), unit);
  }
  return formatter.format(Math.round(delta), "second");
}

function exactTime(createdAt) {
  try {
    return new Date(Number(createdAt) * 1000).toLocaleString();
  } catch {
    return "";
  }
}

onMounted(() => {
  if (keyRef.value) load();
});

// Navigating between two developer pages reuses this view, so the fetch cannot
// live only in onMounted.
watch(keyRef, (next) => {
  if (next) load();
});
</script>

<template>
  <section class="mt-10">
    <div class="mb-4 flex items-end justify-between gap-4">
      <div>
        <h2 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">
          {{ t("developer.activity") }}
        </h2>
        <p class="mt-1.5 text-xs text-zinc-500">{{ t("developer.activityHint") }}</p>
      </div>
      <button
        @click="load"
        :disabled="loading"
        class="inline-flex shrink-0 items-center gap-1.5 rounded-lg px-2 py-1 text-[11px] font-medium text-zinc-500 transition-colors hover:bg-zinc-100 hover:text-zinc-900 disabled:opacity-50 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-white"
      >
        <RefreshCw :size="13" :class="{ 'animate-spin': loading }" />
        {{ t("appComments.refresh") }}
      </button>
    </div>

    <!-- Loading. Same two skeleton rows as the app thread, so a shared shape
         is kept between the two comment lists. -->
    <div v-if="loading" class="flex flex-col gap-6">
      <span class="a11y-visually-hidden" aria-live="polite">{{ t("appComments.loading") }}</span>
      <div v-for="i in 2" :key="i" class="flex animate-pulse gap-3" aria-hidden="true">
        <span class="h-7 w-7 shrink-0 rounded-full bg-zinc-100 dark:bg-zinc-800"></span>
        <span class="mt-0.5 flex-1 space-y-2">
          <span class="block h-2.5 w-2/5 rounded-full bg-zinc-100 dark:bg-zinc-800"></span>
          <span class="block h-2.5 w-full rounded-full bg-zinc-100 dark:bg-zinc-800"></span>
        </span>
      </div>
    </div>

    <template v-else>
      <p
        v-if="loadFailed"
        class="flex items-start gap-2 text-xs leading-relaxed text-zinc-500"
      >
        <AlertCircle :size="14" class="mt-0.5 shrink-0" />
        <span>{{ t("developer.activityLoadFailed") }}</span>
      </p>

      <p v-else-if="comments.length === 0" class="text-sm leading-relaxed text-zinc-500">
        {{ t("developer.activityEmpty") }}
      </p>

      <!-- Comment list. Same rendering rules as the app thread: content is
           pre-sanitised by the composable and renders via interpolation, never
           v-html. -->
      <ul v-else class="flex flex-col gap-6">
        <li v-for="c in comments" :key="c.id" class="flex gap-3">
          <span
            class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-zinc-100 text-[10px] font-semibold text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400"
          >
            {{ developerInitials(profile?.name || t("appComments.anonymous")) }}
          </span>
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <!-- One chip per tagged app. Only ids this host's catalog knows
                   link through; unknown ones stay honest as text. -->
              <template v-if="c.apps && c.apps.length > 0">
                <template v-for="appId in c.apps" :key="`${c.id}-${appId}`">
                  <RouterLink
                    v-if="appOf(appId)"
                    :to="appDetail(appId)"
                    class="inline-flex max-w-full items-center gap-1.5 rounded-full border border-zinc-200 bg-zinc-50 px-2.5 py-0.5 text-[11px] font-medium text-zinc-600 transition-colors hover:border-zinc-300 hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-300 dark:hover:border-zinc-700 dark:hover:text-zinc-100"
                  >
                    <Package :size="11" class="shrink-0 text-zinc-400" />
                    <span class="truncate">{{ appName(appId) }}</span>
                  </RouterLink>
                  <span
                    v-else
                    :title="t('developer.activityUnknownApp')"
                    class="inline-flex max-w-full items-center gap-1.5 rounded-full border border-dashed border-zinc-200 px-2.5 py-0.5 font-mono text-[10px] text-zinc-400 dark:border-zinc-800"
                  >
                    {{ appId }}
                  </span>
                </template>
              </template>
              <span class="shrink-0 text-[11px] tabular-nums text-zinc-400" :title="exactTime(c.created_at)">
                {{ formatTime(c.created_at) }}
              </span>
            </div>
            <p class="mt-1.5 whitespace-pre-wrap break-words text-sm leading-relaxed text-zinc-600 dark:text-zinc-400">
              {{ c.content }}
            </p>
          </div>
        </li>
      </ul>

      <!-- Relay coverage, the same fine print the app thread carries. -->
      <p
        v-if="comments.length > 0 && !loading && !loadFailed"
        class="mt-5 text-[11px] leading-relaxed text-zinc-400"
      >
        {{ t("appComments.relayStatus", { reached: relaysReached, total: relaysTotal }) }}
      </p>
    </template>
  </section>
</template>
