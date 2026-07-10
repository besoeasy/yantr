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
  <div class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] rounded-xl overflow-hidden transition-all duration-300 hover:-translate-y-0.5 hover:shadow-2xl hover:shadow-indigo-500/20 dark:hover:shadow-indigo-500/20 text-left w-full min-h-72">
    <!-- top accent line -->
    <div class="absolute top-0 left-0 w-full h-0.5 bg-indigo-500 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>

    <div class="relative z-10 p-5 flex flex-col h-full">
      <!-- header -->
      <div class="flex items-center justify-between gap-3">
        <div class="flex items-center gap-3 min-w-0">
          <div class="w-9 h-9 rounded-lg flex items-center justify-center shrink-0 transition-transform group-hover:scale-110">
            <Wrench class="w-4.5 h-4.5 text-indigo-600 dark:text-indigo-400" />
          </div>
          <div class="min-w-0">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight leading-none">{{ t("home.toolsNavCard.systemTools") }}</h3>
            <div class="flex items-center gap-1.5 mt-1.5 text-indigo-600 dark:text-indigo-400">
              <span class="text-[10px] font-bold uppercase tracking-widest">Management</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Content (Removed large text) -->
      <div class="mt-auto pt-6 pb-2 flex flex-col relative z-10 pr-[120px]">
      </div>

      <!-- Tool Cluster in Right Corner -->
      <div class="absolute bottom-5 right-5 flex flex-col items-end gap-3 z-50 pointer-events-auto">
         <!-- Top row: 1 item -->
         <div class="flex gap-3">
            <!-- Logs -->
            <button type="button" @click.prevent.stop="router.push('/logs')" :title="t('home.toolsNavCard.logs')" class="group/btn relative w-16 h-16 rounded-2xl flex items-center justify-center transition-all duration-300 hover:bg-violet-500 hover:scale-105 hover:shadow-xl hover:shadow-violet-500/30 text-violet-600 dark:text-violet-400 overflow-hidden cursor-pointer">
               <ClipboardList class="w-8 h-8 transition-all duration-300 group-hover/btn:-translate-y-16 group-hover/btn:opacity-0 pointer-events-none" />
               <span class="absolute inset-0 flex items-center justify-center font-black text-xs translate-y-16 group-hover/btn:translate-y-0 transition-all duration-300 uppercase tracking-widest text-white opacity-0 group-hover/btn:opacity-100 pointer-events-none">Live</span>
            </button>
         </div>
         <!-- Bottom row: 2 items -->
         <div class="flex gap-3">
            <!-- Images -->
            <button type="button" @click.prevent.stop="router.push('/images')" :title="t('home.toolsNavCard.images')" class="group/btn relative w-16 h-16 rounded-2xl flex items-center justify-center transition-all duration-300 hover:bg-blue-500 hover:scale-105 hover:shadow-xl hover:shadow-blue-500/30 text-blue-600 dark:text-blue-400 overflow-hidden cursor-pointer">
               <Layers class="w-8 h-8 transition-all duration-300 group-hover/btn:-translate-y-16 group-hover/btn:opacity-0 pointer-events-none" />
               <span class="absolute inset-0 flex items-center justify-center font-black text-2xl translate-y-16 group-hover/btn:translate-y-0 transition-all duration-300 text-white opacity-0 group-hover/btn:opacity-100 pointer-events-none">{{ imageCount !== null ? imageCount : '-' }}</span>
            </button>
            <!-- Volumes -->
            <button type="button" @click.prevent.stop="router.push('/volumes')" :title="t('home.toolsNavCard.volumes')" class="group/btn relative w-16 h-16 rounded-2xl flex items-center justify-center transition-all duration-300 hover:bg-emerald-500 hover:scale-105 hover:shadow-xl hover:shadow-emerald-500/30 text-emerald-600 dark:text-emerald-400 overflow-hidden cursor-pointer">
               <HardDrive class="w-8 h-8 transition-all duration-300 group-hover/btn:-translate-y-16 group-hover/btn:opacity-0 pointer-events-none" />
               <span class="absolute inset-0 flex items-center justify-center font-black text-2xl translate-y-16 group-hover/btn:translate-y-0 transition-all duration-300 text-white opacity-0 group-hover/btn:opacity-100 pointer-events-none">{{ volumeCount !== null ? volumeCount : '-' }}</span>
            </button>
         </div>
      </div>
    </div>
  </div>
</template>
