<script setup>
import { useI18n } from "vue-i18n";
import { useCurrentTime } from "../composables/useCurrentTime";
import { useYantrAuth } from "../composables/useYantrAuth";
import { FolderOpen, ExternalLink, EyeOff, ArrowRight } from "@lucide/vue";

const { t } = useI18n();
const { openVolumeBrowser } = useYantrAuth();

const emit = defineEmits(['stop-browser'])

const { containers, showHeader } = defineProps({
  containers: { type: Array, default: () => [] },
  showHeader: { type: Boolean, default: false },
});

const { currentTime } = useCurrentTime();

function openBrowser(e, browser) {
  e.stopPropagation();
  openVolumeBrowser(browser.volumeName);
}

function isTemporary(browser) {
  return !!browser.expireAt;
}

function formatTimeRemaining(expireAt) {
  const remaining = parseInt(expireAt, 10) * 1000 - currentTime.value;
  if (remaining <= 0) return 'Expired';
  const totalSeconds = Math.floor(remaining / 1000);
  const totalMinutes = Math.floor(totalSeconds / 60);
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  const seconds = totalSeconds % 60;
  if (hours > 0) return `${hours}h ${minutes}m`;
  if (minutes > 0) return `${minutes}m ${seconds}s`;
  return `${seconds}s`;
}

function getExpirationInfo(browser) {
  if (!browser.expireAt) return null;
  const remaining = parseInt(browser.expireAt, 10) * 1000 - currentTime.value;
  return {
    expireAt: browser.expireAt,
    timeRemaining: formatTimeRemaining(browser.expireAt),
    isExpired: remaining <= 0,
    isExpiringSoon: remaining > 0 && remaining < 60 * 60 * 1000,
  };
}
</script>

<template>
  <div style="display: contents">
    <!-- Section Header -->
    <div v-if="showHeader" class="col-span-full flex items-center gap-2 pt-2 pb-1">
      <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-[var(--text-secondary)]">{{ t("home.volumeBrowsers") }}</span>
      <span class="text-[10px] font-bold text-[var(--text-secondary)] opacity-50">{{ containers.length }}</span>
    </div>

    <div
      v-for="(browser, index) in containers"
      :key="browser.volumeName"
      :style="{ animationDelay: `${index * 50}ms` }"
      class="group relative flex h-full w-full min-h-72 flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-all duration-300 hover:-translate-y-1 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] animate-fadeIn dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
    >
      <div class="relative z-10 flex h-full flex-col gap-4">
        <!-- header -->
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-4">
            <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 transition-transform duration-300 group-hover:scale-105 dark:border-zinc-800 dark:bg-zinc-900">
              <FolderOpen class="h-5 w-5 text-zinc-900 dark:text-zinc-100" />
            </div>
            <div>
              <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors group-hover:text-black dark:text-white dark:group-hover:text-white">{{ t("home.volumeContainersGrid.volumeBrowser") }}</h3>
              <div class="mt-2 flex items-center gap-1.5">
                <span class="text-[10px] font-semibold uppercase tracking-widest text-zinc-500">{{ t("home.volumeContainersGrid.active") }}</span>
              </div>
            </div>
          </div>
          <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-zinc-50 opacity-0 transition-all duration-300 group-hover:translate-x-1 group-hover:opacity-100 dark:bg-zinc-900/50">
             <ArrowRight class="h-4 w-4 text-zinc-400 group-hover:text-zinc-900 dark:text-zinc-500 dark:group-hover:text-white" />
          </div>
        </div>

        <div class="relative z-10 mt-auto flex flex-col pt-6">
          <div class="mb-4 line-clamp-1 text-3xl font-black leading-[1.1] tracking-tighter text-zinc-900 transition-colors sm:text-4xl dark:text-white" :title="browser.volumeName">
            {{ browser.volumeName }}
          </div>

          <!-- Status Box -->
          <div class="mb-4">
            <div class="flex flex-col gap-1 rounded-xl border border-zinc-200 bg-zinc-50 p-4 dark:border-zinc-800/50 dark:bg-zinc-900/50">
               <div v-if="isTemporary(browser)" class="flex flex-col gap-1">
                 <div class="flex items-center gap-1.5 text-zinc-500 dark:text-zinc-400">
                   <FolderOpen class="h-3.5 w-3.5 shrink-0 text-amber-500" />
                   <span class="text-[10px] font-bold uppercase tracking-widest">{{ t("home.volumeContainersGrid.expiresIn") }}</span>
                 </div>
                 <div class="tabular-nums text-sm font-bold tracking-tight"
                   :class="[
                      getExpirationInfo(browser).isExpired
                        ? 'text-red-600 dark:text-red-500'
                        : getExpirationInfo(browser).isExpiringSoon
                          ? 'animate-pulse text-amber-600 dark:text-amber-500'
                          : 'text-zinc-900 dark:text-white',
                    ]"
                 >
                   {{ getExpirationInfo(browser).timeRemaining }}
                 </div>
               </div>
               <div v-else class="flex flex-col gap-1">
                 <div class="flex items-center gap-1.5 text-zinc-500 dark:text-zinc-400">
                   <FolderOpen class="h-3.5 w-3.5 shrink-0 text-zinc-500" />
                   <span class="text-[10px] font-bold uppercase tracking-widest">{{ t("home.volumeContainersGrid.status") }}</span>
                 </div>
                 <div class="text-sm font-bold tracking-tight text-zinc-900 dark:text-white">{{ t("home.volumeContainersGrid.active") }}</div>
               </div>
            </div>
          </div>

          <div class="mt-2 grid grid-cols-2 gap-2.5">
            <button
              @click="openBrowser($event, browser)"
              class="group/link flex h-11 items-center justify-center gap-2 rounded-xl border border-zinc-200 bg-zinc-50 text-zinc-500 transition-all duration-300 hover:bg-zinc-100 hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900/50 dark:hover:bg-zinc-800 dark:hover:text-white"
            >
              <ExternalLink class="h-3.5 w-3.5" />
              <span class="text-[11px] font-bold uppercase tracking-widest">{{ t("home.volumeContainersGrid.browse") }}</span>
            </button>

            <button
              @click.stop="emit('stop-browser', browser.volumeName)"
              class="group/cta relative flex h-11 w-full items-center justify-center gap-2 overflow-hidden rounded-xl border border-red-200 bg-red-50 text-red-600 transition-all duration-300 hover:border-red-500 hover:bg-red-500 hover:text-white active:scale-[0.98] dark:border-red-900/30 dark:bg-red-900/20 dark:text-red-400 dark:hover:border-red-500 dark:hover:bg-red-500 dark:hover:text-white"
            >
              <div class="relative z-10 flex items-center gap-2">
                <EyeOff class="h-4 w-4" />
                <span class="text-[11px] font-bold uppercase tracking-widest">{{ t("home.volumeContainersGrid.stop") }}</span>
              </div>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
