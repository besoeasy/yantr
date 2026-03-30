<script setup>
import { useI18n } from "vue-i18n";
import { useCurrentTime } from "../composables/useCurrentTime";
import { FolderOpen, ExternalLink, EyeOff } from "lucide-vue-next";

const { t } = useI18n();

const emit = defineEmits(['stop-browser'])

const { containers, showHeader } = defineProps({
  containers: { type: Array, default: () => [] },
  showHeader: { type: Boolean, default: false },
});

const { currentTime } = useCurrentTime();

function openBrowser(e, browser) {
  e.stopPropagation();
  window.open(`/browse/${browser.volumeName}/`, "_blank");
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
      class="group relative h-full flex flex-col bg-[var(--surface)] border border-gray-200 dark:border-zinc-800 rounded-xl overflow-hidden transition-all duration-300 smooth-shadow hover:smooth-shadow-lg hover:-translate-y-0.5 animate-fadeIn"
    >

      <div class="relative z-10 flex flex-col h-full p-5">
        <div class="flex items-start justify-between mb-4">
          <div class="min-w-0 pr-4">
             <h3 class="font-semibold text-base text-gray-900 dark:text-white line-clamp-1 mb-2 tracking-tight group-hover:text-violet-600 dark:group-hover:text-violet-400 transition-colors" :title="browser.volumeName">
               {{ browser.volumeName }}
             </h3>
         
             <div class="flex items-center gap-2 flex-wrap">
                <button
                  @click="openBrowser($event, browser)"
                  class="inline-flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-md bg-violet-50/50 text-violet-600 dark:bg-violet-500/10 dark:text-violet-400 hover:bg-violet-100/50 dark:hover:bg-violet-500/20 transition-colors"
                >
                  <ExternalLink :size="10" />
                  {{ t("home.volumeContainersGrid.browse") }}
                </button>
                <button
                  @click.stop="emit('stop-browser', browser.volumeName)"
                  class="inline-flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-md bg-red-50/50 text-red-500 dark:bg-red-500/10 dark:text-red-400 hover:bg-red-100/50 dark:hover:bg-red-500/20 transition-colors"
                >
                  <EyeOff :size="10" />
                  {{ t("home.volumeContainersGrid.stop") }}
                </button>
                
                <span v-if="isTemporary(browser)" 
                      class="text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-md bg-amber-50/50 text-amber-600 dark:bg-amber-500/10 dark:text-amber-500">
                  {{ t("home.volumeContainersGrid.temp") }}
                </span>
             </div>
          </div>
          
          <!-- Logo Container -->
          <div class="w-12 h-12 rounded-lg bg-[var(--surface-muted)] flex items-center justify-center shrink-0 group-hover:scale-105 transition-all duration-300 relative">
             <FolderOpen class="w-6 h-6 text-[var(--text-secondary)] group-hover:text-violet-500 transition-colors" />
             
             <!-- Status Dot -->
             <div class="absolute -top-1 -right-1 w-2 h-2 rounded-full bg-green-500"></div>
          </div>
        </div>

        <div v-if="isTemporary(browser)" class="mb-4">
           <div class="flex items-baseline gap-1.5">
             <span class="text-[10px] font-bold uppercase tracking-[0.15em] text-gray-400 dark:text-zinc-500">{{ t("home.volumeContainersGrid.expiresIn") }}</span>
             <span class="font-mono text-sm font-semibold tracking-tighter"
                :class="[
                    getExpirationInfo(browser).isExpired
                      ? 'text-red-600 dark:text-red-500'
                      : getExpirationInfo(browser).isExpiringSoon
                        ? 'text-amber-600 dark:text-amber-500 animate-pulse'
                        : 'text-gray-700 dark:text-gray-300',
                  ]"
              >{{ getExpirationInfo(browser).timeRemaining }}</span>
           </div>
        </div>
        <div v-else class="mb-4">
           <div class="flex items-baseline gap-1.5">
             <span class="text-[10px] font-bold uppercase tracking-[0.15em] text-gray-400 dark:text-zinc-500">{{ t("home.volumeContainersGrid.status") }}</span>
             <span class="font-mono text-sm font-semibold tracking-tighter text-gray-700 dark:text-gray-300">{{ t("home.volumeContainersGrid.active") }}</span>
           </div>
        </div>

        <div class="mt-auto pt-4 flex items-center justify-between overflow-hidden">
          <div class="flex items-center gap-1.5 text-gray-400 dark:text-zinc-500">
            <span class="text-[10px] font-semibold uppercase tracking-[0.15em]">{{ t("home.volumeContainersGrid.volumeBrowser") }}</span>
          </div>
          <span class="font-mono text-xs text-violet-500 dark:text-violet-400">/browse/{{ browser.volumeName }}/</span>
        </div>
      </div>
    </div>
  </div>
</template>
