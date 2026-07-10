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
  <div class="relative group h-full flex flex-col rounded-[24px] overflow-hidden transition-all duration-500 hover:-translate-y-2 hover:shadow-[0_20px_40px_-15px_rgba(99,102,241,0.4)] dark:hover:shadow-[0_20px_40px_-15px_rgba(99,102,241,0.3)] text-left w-full min-h-[320px] border border-indigo-100/50 dark:border-white/5 bg-gradient-to-br from-indigo-50/50 to-blue-50/50 dark:from-[#111827] dark:to-[#09090b]">
    
    <!-- Animated Gradient Glow -->
    <div class="absolute -top-[50%] -left-[50%] w-[200%] h-[200%] bg-[radial-gradient(circle_at_50%_50%,_rgba(99,102,241,0.15)_0%,_transparent_50%)] dark:bg-[radial-gradient(circle_at_50%_50%,_rgba(99,102,241,0.12)_0%,_transparent_40%)] opacity-0 group-hover:opacity-100 transition-opacity duration-1000 pointer-events-none group-hover:animate-[spin_20s_linear_infinite]"></div>

    <!-- Massive Watermark Icon -->
    <Wrench class="absolute -bottom-8 -right-8 w-64 h-64 text-indigo-500/10 dark:text-indigo-400/[0.03] rotate-[-20deg] group-hover:rotate-0 group-hover:scale-110 transition-all duration-700 ease-out pointer-events-none" />

    <div class="relative z-10 p-7 flex flex-col h-full gap-4 pointer-events-none">
      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Gorgeous Icon Block -->
          <div class="relative w-14 h-14 rounded-2xl flex items-center justify-center shrink-0 bg-gradient-to-br from-indigo-400 to-blue-600 shadow-xl shadow-indigo-500/30 group-hover:shadow-indigo-500/50 transition-all duration-500 group-hover:scale-105">
            <div class="absolute inset-0 bg-white/30 dark:bg-white/20 rounded-2xl rounded-bl-none opacity-50 dark:opacity-40 mix-blend-overlay"></div>
            <Wrench class="w-7 h-7 text-white drop-shadow-md z-10" />
          </div>
          <div>
            <h3 class="text-xl font-black text-indigo-950 dark:text-white tracking-tight leading-none group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors">
              {{ t("home.toolsNavCard.systemTools") }}
            </h3>
            <p class="text-[10px] font-bold uppercase tracking-[0.2em] text-indigo-600/70 dark:text-zinc-500 mt-2">
              Management
            </p>
          </div>
        </div>
      </div>

      <!-- Main Body -->
      <div class="mt-auto pt-6 flex flex-col relative z-10 pr-[180px]">
        
        <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-indigo-700/60 dark:text-zinc-500 mb-1 drop-shadow-sm">
          System Overview
        </div>
        
        <div class="text-5xl font-black tracking-tighter text-indigo-950 dark:text-white mb-2 drop-shadow-sm group-hover:-translate-y-1 transition-transform duration-500">
          Active
        </div>
        <div class="text-[11px] font-medium text-indigo-800/60 dark:text-zinc-400 mt-1 leading-relaxed line-clamp-3">
          Manage system resources, monitor live logs, and clean up environments.
        </div>
      </div>
      
      <!-- Tool Cluster in Right Corner -->
      <div class="absolute bottom-6 right-6 flex flex-col items-end gap-3 z-50 pointer-events-auto">
         <!-- Top row: 1 item -->
         <div class="flex gap-3">
            <!-- Logs -->
            <button type="button" @click.prevent.stop="router.push('/logs')" :title="t('home.toolsNavCard.logs')" class="group/btn relative w-[72px] h-[72px] rounded-[20px] bg-white/60 dark:bg-black/40 backdrop-blur-xl border border-white dark:border-white/5 shadow-[0_4px_12px_rgba(0,0,0,0.02)] dark:shadow-none flex items-center justify-center transition-all duration-300 hover:bg-violet-500 hover:border-violet-400 hover:shadow-violet-500/30 text-violet-600 dark:text-violet-400 overflow-hidden cursor-pointer hover:-translate-y-1">
               <ClipboardList class="w-8 h-8 transition-all duration-300 group-hover/btn:-translate-y-16 group-hover/btn:opacity-0 pointer-events-none drop-shadow-sm" />
               <span class="absolute inset-0 flex items-center justify-center font-black text-xs translate-y-16 group-hover/btn:translate-y-0 transition-all duration-300 uppercase tracking-widest text-white opacity-0 group-hover/btn:opacity-100 pointer-events-none drop-shadow-md">Live</span>
            </button>
         </div>
         <!-- Bottom row: 2 items -->
         <div class="flex gap-3">
            <!-- Images -->
            <button type="button" @click.prevent.stop="router.push('/images')" :title="t('home.toolsNavCard.images')" class="group/btn relative w-[72px] h-[72px] rounded-[20px] bg-white/60 dark:bg-black/40 backdrop-blur-xl border border-white dark:border-white/5 shadow-[0_4px_12px_rgba(0,0,0,0.02)] dark:shadow-none flex items-center justify-center transition-all duration-300 hover:bg-blue-500 hover:border-blue-400 hover:shadow-blue-500/30 text-blue-600 dark:text-blue-400 overflow-hidden cursor-pointer hover:-translate-y-1">
               <Layers class="w-8 h-8 transition-all duration-300 group-hover/btn:-translate-y-16 group-hover/btn:opacity-0 pointer-events-none drop-shadow-sm" />
               <span class="absolute inset-0 flex items-center justify-center font-black text-3xl translate-y-16 group-hover/btn:translate-y-0 transition-all duration-300 text-white opacity-0 group-hover/btn:opacity-100 pointer-events-none drop-shadow-md">{{ imageCount !== null ? imageCount : '-' }}</span>
            </button>
            <!-- Volumes -->
            <button type="button" @click.prevent.stop="router.push('/volumes')" :title="t('home.toolsNavCard.volumes')" class="group/btn relative w-[72px] h-[72px] rounded-[20px] bg-white/60 dark:bg-black/40 backdrop-blur-xl border border-white dark:border-white/5 shadow-[0_4px_12px_rgba(0,0,0,0.02)] dark:shadow-none flex items-center justify-center transition-all duration-300 hover:bg-emerald-500 hover:border-emerald-400 hover:shadow-emerald-500/30 text-emerald-600 dark:text-emerald-400 overflow-hidden cursor-pointer hover:-translate-y-1">
               <HardDrive class="w-8 h-8 transition-all duration-300 group-hover/btn:-translate-y-16 group-hover/btn:opacity-0 pointer-events-none drop-shadow-sm" />
               <span class="absolute inset-0 flex items-center justify-center font-black text-3xl translate-y-16 group-hover/btn:translate-y-0 transition-all duration-300 text-white opacity-0 group-hover/btn:opacity-100 pointer-events-none drop-shadow-md">{{ volumeCount !== null ? volumeCount : '-' }}</span>
            </button>
         </div>
      </div>
    </div>
  </div>
</template>
