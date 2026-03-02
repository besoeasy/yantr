<script setup>
import { useI18n } from "vue-i18n";
import { useCurrentTime } from "../../composables/useCurrentTime";
import { FolderOpen, ExternalLink, EyeOff } from "lucide-vue-next";

const { t } = useI18n();

const emit = defineEmits(['stop-browser'])

const { containers } = defineProps({
  containers: { type: Array, default: () => [] },
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
    <div
      v-for="(browser, index) in containers"
      :key="browser.volumeName"
      :style="{ animationDelay: `${index * 50}ms` }"
      class="group relative h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl overflow-hidden transition-all duration-400 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:-translate-y-1 hover:border-gray-300 dark:hover:border-zinc-600 animate-fadeIn"
    >
      <!-- Hover Accents -->
      <div class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-violet-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
      <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none [mask-image:linear-gradient(to_bottom,white,transparent)]"></div>

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
          <div class="w-12 h-12 rounded-lg bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center shrink-0 group-hover:scale-105 group-hover:border-zinc-700 transition-all duration-500 relative">
             <FolderOpen class="w-6 h-6 text-gray-400 dark:text-zinc-500 group-hover:text-violet-500 transition-colors" />
             
             <!-- Status Dot -->
             <div class="absolute -top-1 -right-1 w-2.5 h-2.5 rounded-full border-2 border-white dark:border-[#0A0A0A] bg-green-500 animate-pulse"></div>
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

        <div class="mt-auto pt-4 border-t border-gray-100 dark:border-zinc-800/80 flex items-center justify-between overflow-hidden">
          <div class="flex items-center gap-1.5 text-gray-400 dark:text-zinc-500">
            <span class="text-[10px] font-semibold uppercase tracking-[0.15em]">{{ t("home.volumeContainersGrid.volumeBrowser") }}</span>
          </div>
          <span class="font-mono text-xs text-violet-500 dark:text-violet-400">/browse/{{ browser.volumeName }}/</span>
        </div>
      </div>
    </div>
  </div>
</template>
