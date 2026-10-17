<script setup>
// The deployed stacks.
//
// There is no GET /api/stacks collection endpoint and none is needed:
// /api/containers already carries app.projectId and appLabels.app on every
// container, which is exactly what the grouping needs. Before this page
// existed, a stack was only discoverable by scrolling the dashboard.
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { Layers, Store } from "@lucide/vue";
import { useApiUrl } from "../composables/useApiUrl";
import { expectApiSuccess } from "../composables/useApiResponse";
import { useNotification } from "../composables/useNotification";
import { usePolling } from "../composables/usePolling";
import AppLayout from "../components/AppLayout.vue";
import YantraContainersGrid from "../components/YantraContainersGrid.vue";
import { apps as appsRoute } from "../utils/routes.js";

const { t } = useI18n();
const { apiUrl } = useApiUrl();
const toast = useNotification();

const containers = ref([]);
const loading = ref(true);
const loadFailed = ref(false);

// The same predicate the dashboard uses for its Yantr Apps section — and the
// same one the backend uses to decide a container belongs to a compose project
// this app manages.
const stacks = computed(() => containers.value.filter((c) => c.appLabels?.app));

const stackCount = computed(() => {
  const projects = new Set(stacks.value.map((c) => c.app?.projectId || c.id));
  return projects.size;
});

async function fetchContainers() {
  try {
    const response = await fetch(`${apiUrl.value}/api/containers`);
    const data = await expectApiSuccess(response, t("stacks.loadFailed"));
    containers.value = Array.isArray(data.containers) ? data.containers : [];
    loadFailed.value = false;
  } catch (error) {
    // Only the first failure is worth a toast. The 10s poll would otherwise
    // repeat the same message for as long as the host is unreachable.
    if (!loadFailed.value) toast.error(error.message || t("stacks.loadFailed"));
    loadFailed.value = true;
  }
}

const { start: startPolling } = usePolling(fetchContainers, 10000);

onMounted(async () => {
  await fetchContainers();
  loading.value = false;
  startPolling();
});
</script>

<template>
  <AppLayout width="wide">
    <div class="flex flex-wrap items-end justify-between gap-4">
      <div>
        <p class="text-[10px] font-black uppercase tracking-[0.24em] text-gray-400 dark:text-zinc-500">{{ t("stacks.eyebrow") }}</p>
        <h1 class="mt-2 text-3xl font-semibold tracking-tight">{{ t("stacks.title") }}</h1>
        <p class="mt-2 max-w-xl text-sm text-gray-500 dark:text-zinc-400">{{ t("stacks.subtitle") }}</p>
      </div>
      <p class="text-[11px] font-semibold tabular-nums text-gray-400 dark:text-zinc-500">
        {{ stackCount }} {{ stackCount === 1 ? t("apps.app") : t("apps.apps") }}
      </p>
    </div>

    <div v-if="loading" class="flex flex-col items-center justify-center py-32">
      <div class="mb-4 h-8 w-8 animate-spin rounded-full border-2 border-gray-200 border-t-blue-500 dark:border-zinc-800"></div>
      <p class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-500">{{ t("common.loading") }}</p>
    </div>

    <div
      v-else-if="stacks.length === 0"
      class="flex flex-col items-center rounded-xl border border-gray-200 py-32 text-center dark:border-zinc-800"
    >
      <div class="mb-6 flex h-20 w-20 items-center justify-center rounded-2xl border border-gray-200 bg-gray-50 dark:border-zinc-800 dark:bg-zinc-900/50">
        <Layers :size="32" class="text-gray-400 dark:text-zinc-500" />
      </div>
      <h3 class="mb-2 text-xl font-semibold tracking-tight">{{ t("stacks.empty") }}</h3>
      <p class="mx-auto mb-8 max-w-md text-sm font-medium text-gray-500 dark:text-zinc-400">{{ t("stacks.emptyHint") }}</p>
      <router-link
        :to="appsRoute()"
        class="inline-flex items-center gap-2 rounded-lg bg-gray-900 px-6 py-3 text-xs font-bold uppercase tracking-wider text-white shadow-sm transition-all hover:-translate-y-0.5 dark:bg-white dark:text-gray-900"
      >
        <Store :size="16" />
        <span>{{ t("stacks.browseCatalog") }}</span>
      </router-link>
    </div>

    <div v-else class="grid grid-cols-1 gap-4 sm:gap-6 md:grid-cols-2 lg:grid-cols-3">
      <YantraContainersGrid :containers="stacks" />
    </div>
  </AppLayout>
</template>
