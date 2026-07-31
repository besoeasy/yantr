<script setup>
import { ref, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { ClipboardList, HardDrive, Layers, Wrench } from "@lucide/vue";
import { useApiUrl } from "../composables/useApiUrl";
import { expectApiSuccess } from "../composables/useApiResponse";

const router = useRouter();
const { t } = useI18n();
const { apiUrl } = useApiUrl();

const imageCount = ref(null);
const volumeCount = ref(null);
let interval = null;

async function fetchCounts() {
  try {
    const [imgRes, volRes] = await Promise.all([
      fetch(`${apiUrl.value}/api/images`),
      fetch(`${apiUrl.value}/api/volumes`),
    ]);
    const [imgData, volData] = await Promise.all([
      expectApiSuccess(imgRes, "images"),
      expectApiSuccess(volRes, "volumes"),
    ]);
    imageCount.value = Array.isArray(imgData.images) ? imgData.images.length : null;
    volumeCount.value = Array.isArray(volData.volumes) ? volData.volumes.length : null;
  } catch {
    // counts stay null — rows still navigate fine
  }
}

onMounted(() => {
  fetchCounts();
  interval = setInterval(fetchCounts, 30000);
});

onUnmounted(() => {
  if (interval) clearInterval(interval);
});
</script>

<template>
  <div class="group relative flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-all duration-300 hover:-translate-y-1 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">

    <div class="pointer-events-none relative z-10 flex h-full flex-col gap-4">
      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Minimal Icon Block -->
          <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-sky-200 bg-sky-50 transition-transform duration-300 group-hover:scale-105 dark:border-sky-900/50 dark:bg-sky-900/20">
            <Wrench class="h-5 w-5 text-sky-600 dark:text-sky-400" />
          </div>
          <div>
            <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors group-hover:text-black dark:text-white dark:group-hover:text-white">
              {{ t("home.toolsNavCard.systemTools") }}
            </h3>
            <p class="mt-2 text-[10px] font-semibold uppercase tracking-widest text-sky-600 dark:text-sky-500">
              Management
            </p>
          </div>
        </div>
      </div>

      <!-- Main Body -->
      <div class="relative z-10 mt-auto flex flex-col pt-6 pr-[180px]">
        
        <div class="mb-1 text-[10px] font-semibold uppercase tracking-widest text-zinc-500">
          System Overview
        </div>
        
        <div class="mb-2 text-4xl font-black tracking-tighter text-zinc-900 transition-transform duration-500 group-hover:-translate-y-1 sm:text-5xl dark:text-white">
          Active
        </div>
        <div class="mt-1 line-clamp-3 text-xs font-medium leading-relaxed text-zinc-500 dark:text-zinc-400">
          Manage system resources, monitor live logs, and clean up environments.
        </div>
      </div>
      
      <!-- Tool Cluster in Right Corner -->
      <div class="pointer-events-auto absolute bottom-6 right-6 z-50 flex flex-col items-end gap-2">
         <!-- Top row: 1 item -->
         <div class="flex gap-2">
            <!-- Logs -->
            <button type="button" @click.prevent.stop="router.push('/logs')" :title="t('home.toolsNavCard.logs')" class="group/btn relative flex h-16 w-16 cursor-pointer items-center justify-center overflow-hidden rounded-xl border border-sky-200/80 bg-sky-50 text-sky-700 transition-all duration-300 hover:-translate-y-1 hover:border-sky-600 hover:bg-sky-600 hover:text-white dark:border-sky-900/50 dark:bg-sky-900/20 dark:text-sky-400 dark:hover:border-sky-400 dark:hover:bg-sky-400 dark:hover:text-zinc-900">
               <ClipboardList class="h-6 w-6 transition-all duration-300 group-hover/btn:-translate-y-10 group-hover/btn:opacity-0" />
               <span class="absolute inset-0 flex translate-y-10 items-center justify-center text-[10px] font-bold uppercase tracking-widest opacity-0 transition-all duration-300 group-hover/btn:translate-y-0 group-hover/btn:opacity-100">Live</span>
            </button>
         </div>
         <!-- Bottom row: 2 items -->
         <div class="flex gap-2">
            <!-- Images -->
            <button type="button" @click.prevent.stop="router.push('/images')" :title="t('home.toolsNavCard.images')" class="group/btn relative flex h-16 w-16 cursor-pointer items-center justify-center overflow-hidden rounded-xl border border-sky-200/80 bg-sky-50 text-sky-700 transition-all duration-300 hover:-translate-y-1 hover:border-sky-600 hover:bg-sky-600 hover:text-white dark:border-sky-900/50 dark:bg-sky-900/20 dark:text-sky-400 dark:hover:border-sky-400 dark:hover:bg-sky-400 dark:hover:text-zinc-900">
               <Layers class="h-6 w-6 transition-all duration-300 group-hover/btn:-translate-y-10 group-hover/btn:opacity-0" />
               <span class="absolute inset-0 flex translate-y-10 items-center justify-center text-xl font-black opacity-0 transition-all duration-300 group-hover/btn:translate-y-0 group-hover/btn:opacity-100">{{ imageCount !== null ? imageCount : '-' }}</span>
            </button>
            <!-- Volumes -->
            <button type="button" @click.prevent.stop="router.push('/volumes')" :title="t('home.toolsNavCard.volumes')" class="group/btn relative flex h-16 w-16 cursor-pointer items-center justify-center overflow-hidden rounded-xl border border-sky-200/80 bg-sky-50 text-sky-700 transition-all duration-300 hover:-translate-y-1 hover:border-sky-600 hover:bg-sky-600 hover:text-white dark:border-sky-900/50 dark:bg-sky-900/20 dark:text-sky-400 dark:hover:border-sky-400 dark:hover:bg-sky-400 dark:hover:text-zinc-900">
               <HardDrive class="h-6 w-6 transition-all duration-300 group-hover/btn:-translate-y-10 group-hover/btn:opacity-0" />
               <span class="absolute inset-0 flex translate-y-10 items-center justify-center text-xl font-black opacity-0 transition-all duration-300 group-hover/btn:translate-y-0 group-hover/btn:opacity-100">{{ volumeCount !== null ? volumeCount : '-' }}</span>
            </button>
         </div>
      </div>
    </div>
  </div>
</template>
