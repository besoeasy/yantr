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
      class="relative group flex flex-col bg-white dark:bg-[#0A0A0A] rounded-xl overflow-hidden transition-all duration-300 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:-translate-y-0.5 border border-gray-100 dark:border-zinc-800 text-left w-full h-full min-h-72 animate-fadeIn"
    >
      <div class="absolute top-0 left-0 w-full h-0.5 bg-violet-500 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>

      <div class="relative z-10 p-5 flex flex-col h-full gap-4">
        <!-- header -->
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-3 min-w-0">
            <div class="w-9 h-9 rounded-lg bg-violet-50 dark:bg-violet-500/10 border border-violet-100 dark:border-violet-500/20 flex items-center justify-center shrink-0 transition-transform group-hover:scale-110">
              <FolderOpen class="w-4.5 h-4.5 text-violet-600 dark:text-violet-400 group-hover:text-violet-500 transition-colors" />
            </div>
            <div class="min-w-0">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight leading-none truncate">{{ t("home.volumeContainersGrid.volumeBrowser") }}</h3>
              <div class="flex items-center gap-1.5 mt-1.5 text-violet-600 dark:text-violet-400">
                <span class="text-[10px] font-bold uppercase tracking-wider">{{ t("home.volumeContainersGrid.active") }}</span>
              </div>
            </div>
          </div>
          <div class="flex items-center justify-center w-8 h-8 rounded-full bg-gray-50 dark:bg-zinc-900/50 opacity-0 group-hover:opacity-100 group-hover:translate-x-1 transition-all duration-300 shrink-0">
             <ArrowRight class="w-4 h-4 text-gray-400 dark:text-zinc-500 group-hover:text-violet-500" />
          </div>
        </div>

        <div class="mt-auto pt-6 flex flex-col relative z-10">
          <div class="text-3xl sm:text-4xl font-black tracking-tighter text-gray-900 dark:text-white transition-colors line-clamp-1 leading-[1.1] mb-4" :title="browser.volumeName">
            {{ browser.volumeName }}
          </div>

          <!-- Status Box -->
          <div class="mb-4">
            <div class="p-4 rounded-xl bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50 flex flex-col gap-1">
               <div v-if="isTemporary(browser)" class="flex flex-col gap-1">
                 <div class="flex items-center gap-1.5 text-gray-500 dark:text-zinc-400">
                   <FolderOpen class="w-3.5 h-3.5 text-amber-500 shrink-0" />
                   <span class="text-[10px] font-bold uppercase tracking-widest">{{ t("home.volumeContainersGrid.expiresIn") }}</span>
                 </div>
                 <div class="text-sm font-bold tracking-tight tabular-nums"
                   :class="[
                      getExpirationInfo(browser).isExpired
                        ? 'text-red-600 dark:text-red-500'
                        : getExpirationInfo(browser).isExpiringSoon
                          ? 'text-amber-600 dark:text-amber-500 animate-pulse'
                          : 'text-gray-900 dark:text-white',
                    ]"
                 >
                   {{ getExpirationInfo(browser).timeRemaining }}
                 </div>
               </div>
               <div v-else class="flex flex-col gap-1">
                 <div class="flex items-center gap-1.5 text-gray-500 dark:text-zinc-400">
                   <FolderOpen class="w-3.5 h-3.5 text-violet-500 shrink-0" />
                   <span class="text-[10px] font-bold uppercase tracking-widest">{{ t("home.volumeContainersGrid.status") }}</span>
                 </div>
                 <div class="text-sm font-bold text-gray-900 dark:text-white tracking-tight">{{ t("home.volumeContainersGrid.active") }}</div>
               </div>
            </div>
          </div>

          <div class="mt-2 grid grid-cols-2 gap-2.5">
            <button
              @click="openBrowser($event, browser)"
              class="group/link flex h-11 items-center justify-center gap-2 rounded-xl bg-gray-50 dark:bg-zinc-900/70 text-gray-600 dark:text-zinc-400 hover:bg-gray-100 dark:hover:bg-zinc-800 transition-all duration-300 hover:text-gray-900 dark:hover:text-white border border-transparent dark:hover:border-zinc-700/50"
            >
              <ExternalLink class="w-3.5 h-3.5" />
              <span class="text-[11px] font-bold uppercase tracking-widest">{{ t("home.volumeContainersGrid.browse") }}</span>
            </button>

            <button
              @click.stop="emit('stop-browser', browser.volumeName)"
              class="group/cta relative flex h-11 w-full items-center justify-center gap-2 rounded-xl transition-all duration-300 overflow-hidden text-white shadow-md bg-red-500 hover:bg-red-600 shadow-red-500/20 active:scale-[0.98]"
            >
              <div class="flex items-center gap-2 relative z-10">
                <EyeOff class="w-4 h-4" />
                <span class="text-[11px] font-bold uppercase tracking-widest">{{ t("home.volumeContainersGrid.stop") }}</span>
              </div>
              <div class="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent -translate-x-full group-hover/cta:translate-x-full transition-transform duration-700 ease-out"></div>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
