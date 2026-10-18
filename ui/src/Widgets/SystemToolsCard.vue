<script>
// Dashboard engine: two of the desktop grid's four columns — the tools no
// longer huddle in one corner; the card splits into an explanation and a
// three-button row. See Home.vue for how meta.cols is consumed.
export const meta = { cols: 2 };
</script>
<script setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { ClipboardList, HardDrive, Layers, Wrench } from "@lucide/vue";
import { useStorageCounts } from "../composables/useDashboardData"
import WidgetSkeleton from "../components/WidgetSkeleton.vue";
import { logs, storageTab } from '../utils/routes.js'

const router = useRouter();
const { t } = useI18n();

// The counts pair two endpoints; the shared resource reads both as one
// logical fetch and keeps them fresh with one ticker for the dashboard.
const { counts, countsLoaded, countsLoadFailed } = useStorageCounts({ live: true });
const imageCount = computed(() => counts.value?.imageCount ?? null);
const volumeCount = computed(() => counts.value?.volumeCount ?? null);
</script>

<template>
  <WidgetSkeleton v-if="!countsLoaded && !countsLoadFailed" />
  <div v-else class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden rounded-2xl p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">

    <div class="pointer-events-none relative z-10 flex h-full flex-col gap-4">
      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Minimal Icon Block -->
          <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-sky-200 bg-sky-50 transition-transform duration-200 group-hover:scale-105 dark:border-sky-900/50 dark:bg-sky-900/20">
            <Wrench class="h-5 w-5 text-sky-600 dark:text-sky-400" />
          </div>
          <div>
            <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
              {{ t("home.toolsNavCard.systemTools") }}
            </h3>
            <p class="mt-2 text-[10px] font-semibold uppercase tracking-widest text-sky-600 dark:text-sky-500">
              {{ t('home.systemToolsCard.management') }}
            </p>
          </div>
        </div>
      </div>

      <!-- Half-width body: the explanation stays off the buttons instead of the
           old corner cluster needing a right-padding hack on the text. -->
      <div class="mt-auto flex flex-col gap-6 pt-6 sm:flex-row sm:items-end">
        <div class="min-w-0 flex-1">
          <div class="mb-1 text-[10px] font-semibold uppercase tracking-widest yantra-text-secondary">
            {{ t('home.systemToolsCard.systemOverview') }}
          </div>

          <div class="mb-2 text-4xl font-black tracking-tighter yantra-text-primary transition-transform duration-200 group-hover:-translate-y-[3px] sm:text-5xl yantra-text-primary">
            {{ t('home.systemToolsCard.active') }}
          </div>
          <div class="line-clamp-3 text-xs font-medium leading-relaxed yantra-text-secondary">
            Manage system resources, monitor live logs, and clean up environments.
          </div>
        </div>

        <!-- Tools: the three shortcuts share one row rather than a corner cluster. -->
        <div class="pointer-events-auto flex shrink-0 flex-none items-end gap-2">
          <!-- Logs -->
          <button type="button" @click.prevent.stop="router.push(logs())" :title="t('home.toolsNavCard.logs')" class="group/btn relative flex h-16 w-16 cursor-pointer items-center justify-center overflow-hidden rounded-xl border-sky-200/80 bg-sky-50 text-sky-700 transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:border-sky-600 hover:bg-sky-600 hover:text-white dark:border-sky-900/50 dark:bg-sky-900/20 dark:text-sky-400 dark:hover:border-sky-400 dark:hover:bg-sky-400 dark:hover:text-zinc-900">
            <ClipboardList class="h-6 w-6 transition-[border-color,box-shadow,transform] duration-200 group-hover/btn:-translate-y-10 group-hover/btn:opacity-0" />
            <span class="absolute inset-0 flex translate-y-10 items-center justify-center text-[10px] font-bold uppercase tracking-widest opacity-0 transition-[border-color,box-shadow,transform] duration-200 group-hover/btn:translate-y-0 group-hover/btn:opacity-100">{{ t('home.systemToolsCard.live') }}</span>
          </button>
          <!-- Images -->
          <button type="button" @click.prevent.stop="router.push(storageTab('images-active'))" :title="t('home.toolsNavCard.images')" class="group/btn relative flex h-16 w-16 cursor-pointer items-center justify-center overflow-hidden rounded-xl border-sky-200/80 bg-sky-50 text-sky-700 transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:border-sky-600 hover:bg-sky-600 hover:text-white dark:border-sky-900/50 dark:bg-sky-900/20 dark:text-sky-400 dark:hover:border-sky-400 dark:hover:bg-sky-400 dark:hover:text-zinc-900">
            <Layers class="h-6 w-6 transition-[border-color,box-shadow,transform] duration-200 group-hover/btn:-translate-y-10 group-hover/btn:opacity-0" />
            <span class="absolute inset-0 flex translate-y-10 items-center justify-center text-xl font-black opacity-0 transition-[border-color,box-shadow,transform] duration-200 group-hover/btn:translate-y-0 group-hover/btn:opacity-100">{{ imageCount !== null ? imageCount : '-' }}</span>
          </button>
          <!-- Volumes -->
          <button type="button" @click.prevent.stop="router.push(storageTab('volumes-active'))" :title="t('home.toolsNavCard.volumes')" class="group/btn relative flex h-16 w-16 cursor-pointer items-center justify-center overflow-hidden rounded-xl border-sky-200/80 bg-sky-50 text-sky-700 transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:border-sky-600 hover:bg-sky-600 hover:text-white dark:border-sky-900/50 dark:bg-sky-900/20 dark:text-sky-400 dark:hover:border-sky-400 dark:hover:bg-sky-400 dark:hover:text-zinc-900">
            <HardDrive class="h-6 w-6 transition-[border-color,box-shadow,transform] duration-200 group-hover/btn:-translate-y-10 group-hover/btn:opacity-0" />
            <span class="absolute inset-0 flex translate-y-10 items-center justify-center text-xl font-black opacity-0 transition-[border-color,box-shadow,transform] duration-200 group-hover/btn:translate-y-0 group-hover/btn:opacity-100">{{ volumeCount !== null ? volumeCount : '-' }}</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
