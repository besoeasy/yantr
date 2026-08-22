<script setup>
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { Activity, Globe, MemoryStick, Monitor, RefreshCw } from "@lucide/vue";
import { useApiUrl } from "../composables/useApiUrl";
import { expectApiSuccess } from "../composables/useApiResponse";
import { useNotification } from "../composables/useNotification";
import FleetMap from "../components/FleetMap.vue";
import AppLogo from "../components/AppLogo.vue";
import StatCard from "../components/StatCard.vue";

const { t } = useI18n();
const { apiUrl } = useApiUrl();
const toast = useNotification();

const loading = ref(true);
const stats = ref(null);

const ramAvg = computed(() => {
  const avg = stats.value?.ram?.avg;
  if (avg == null || Number.isNaN(Number(avg))) return "—";
  return Number(avg).toFixed(1);
});

const maxCountry = computed(() =>
  stats.value?.countries?.reduce((max, item) => Math.max(max, item.count || 0), 0) || 1
);
const maxOs = computed(() =>
  stats.value?.os?.reduce((max, item) => Math.max(max, item.count || 0), 0) || 1
);
const maxApp = computed(() =>
  stats.value?.apps?.reduce((max, item) => Math.max(max, item.count || 0), 0) || 1
);

async function fetchStats(force = false) {
  loading.value = true;
  try {
    const qs = force ? "?refresh=1" : "";
    const response = await fetch(`${apiUrl.value}/api/telemetry/stats${qs}`);
    const data = await expectApiSuccess(response, t("telemetry.loadFailed"));
    stats.value = data.stats || null;
  } catch (error) {
    toast.error(error.message || t("telemetry.loadFailed"));
  } finally {
    loading.value = false;
  }
}

onMounted(() => fetchStats(false));
</script>

<template>
  <div class="min-h-screen bg-white dark:bg-[#0A0A0A] text-gray-900 dark:text-white font-sans">
    <div class="p-3 sm:p-4 lg:p-8 max-w-400 mx-auto space-y-8">
      <div class="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p class="text-[10px] font-black uppercase tracking-[0.24em] text-gray-400 dark:text-zinc-500">{{ t("telemetry.eyebrow") }}</p>
          <h1 class="mt-2 text-3xl font-semibold tracking-tight">{{ t("telemetry.title") }}</h1>
          <p class="mt-2 max-w-xl text-sm text-gray-500 dark:text-zinc-400">{{ t("telemetry.subtitle") }}</p>
        </div>
        <button
          type="button"
          @click="fetchStats(true)"
          class="inline-flex items-center gap-2 rounded-lg border border-zinc-200 px-4 py-2 text-xs font-bold uppercase tracking-wider text-zinc-600 hover:bg-zinc-50 dark:border-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-900"
        >
          <RefreshCw :size="14" :class="loading ? 'animate-spin' : ''" />
          {{ t("common.refresh") }}
        </button>
      </div>

      <div v-if="loading && !stats" class="flex flex-col items-center justify-center py-32">
        <div class="mb-4 h-8 w-8 animate-spin rounded-full border-2 border-gray-200 border-t-blue-500 dark:border-zinc-800"></div>
        <p class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-500">{{ t("common.loading") }}</p>
      </div>

      <template v-else-if="stats">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard :label="t('telemetry.nodes')" :value="stats.nodes ?? 0" :icon="Globe" accent="blue" />
          <StatCard :label="t('telemetry.ramLow')" :value="stats.ram?.min || '—'" :unit="stats.ram?.unit" :icon="MemoryStick" accent="green" />
          <StatCard :label="t('telemetry.ramHigh')" :value="stats.ram?.max || '—'" :unit="stats.ram?.unit" :icon="MemoryStick" accent="amber" />
          <StatCard :label="t('telemetry.ramAvg')" :value="ramAvg" :unit="stats.ram?.unit" :icon="Activity" accent="purple" />
        </div>

        <section class="space-y-3">
          <h2 class="text-sm font-semibold tracking-tight">{{ t("telemetry.mapTitle") }}</h2>
          <FleetMap :countries="stats.countries || []" />
        </section>

        <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <section class="rounded-2xl border border-zinc-200 p-5 dark:border-zinc-800">
            <div class="mb-4 flex items-center gap-2">
              <Globe :size="16" class="text-zinc-400" />
              <h2 class="text-sm font-semibold">{{ t("telemetry.countries") }}</h2>
            </div>
            <div v-if="!stats.countries?.length" class="text-sm text-zinc-500">{{ t("telemetry.empty") }}</div>
            <ul v-else class="space-y-3">
              <li v-for="item in stats.countries" :key="item.code" class="space-y-1">
                <div class="flex items-baseline justify-between gap-3 text-sm">
                  <span class="truncate">{{ item.name }}</span>
                  <span class="tabular-nums text-zinc-500">{{ item.count }}</span>
                </div>
                <div class="h-1.5 overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-900">
                  <div class="h-full rounded-full bg-sky-500" :style="{ width: `${(item.count / maxCountry) * 100}%` }"></div>
                </div>
              </li>
            </ul>
          </section>

          <section class="rounded-2xl border border-zinc-200 p-5 dark:border-zinc-800">
            <div class="mb-4 flex items-center gap-2">
              <Monitor :size="16" class="text-zinc-400" />
              <h2 class="text-sm font-semibold">{{ t("telemetry.os") }}</h2>
            </div>
            <div v-if="!stats.os?.length" class="text-sm text-zinc-500">{{ t("telemetry.empty") }}</div>
            <ul v-else class="space-y-3">
              <li v-for="item in stats.os" :key="item.name" class="space-y-1">
                <div class="flex items-baseline justify-between gap-3 text-sm">
                  <span class="truncate">{{ item.name }}</span>
                  <span class="tabular-nums text-zinc-500">{{ item.count }}</span>
                </div>
                <div class="h-1.5 overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-900">
                  <div class="h-full rounded-full bg-emerald-500" :style="{ width: `${(item.count / maxOs) * 100}%` }"></div>
                </div>
              </li>
            </ul>
          </section>

          <section class="rounded-2xl border border-zinc-200 p-5 dark:border-zinc-800">
            <div class="mb-4 flex items-center gap-2">
              <Activity :size="16" class="text-zinc-400" />
              <h2 class="text-sm font-semibold">{{ t("telemetry.apps") }}</h2>
            </div>
            <div v-if="!stats.apps?.length" class="text-sm text-zinc-500">{{ t("telemetry.empty") }}</div>
            <ul v-else class="space-y-3">
              <li v-for="item in stats.apps" :key="item.id || item.name" class="flex items-center gap-3">
                <div class="h-8 w-8 shrink-0">
                  <AppLogo
                    :logo="item.id ? `/api/apps/${item.id}/logo` : null"
                    :name="item.name"
                    :seed="item.id || item.name"
                    img-class="h-full w-full object-contain"
                    icon-class="h-full w-full text-zinc-300 dark:text-zinc-700"
                  />
                </div>
                <div class="min-w-0 flex-1 space-y-1">
                  <div class="flex items-baseline justify-between gap-3 text-sm">
                    <span class="truncate">{{ item.name }}</span>
                    <span class="tabular-nums text-zinc-500">{{ item.count }}</span>
                  </div>
                  <div class="h-1.5 overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-900">
                    <div class="h-full rounded-full bg-violet-500" :style="{ width: `${(item.count / maxApp) * 100}%` }"></div>
                  </div>
                </div>
              </li>
            </ul>
          </section>
        </div>

        <p class="text-[11px] text-zinc-400 dark:text-zinc-600">
          {{ t("telemetry.footnote", { window: stats.window, events: stats.events }) }}
        </p>
      </template>
    </div>
  </div>
</template>
