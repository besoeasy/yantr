<script setup>
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useI18n } from 'vue-i18n'
import { Globe, MapPin, Network, RefreshCw, Server, ShieldCheck, AlertCircle } from "@lucide/vue";

const { t } = useI18n()
const props = defineProps({
  refreshMs: { type: Number, default: 5 * 60_000 },
});

const loading = ref(true);
const error = ref(null);
const identity = ref(null);
const isIpHovered = ref(false);
const isLocationHovered = ref(false);
let refreshHandle = null;

async function loadIdentity({ force } = { force: false }) {
  try {
    loading.value = true;
    error.value = null;

    const url = force ? "/api/network/identity?force=true" : "/api/network/identity";
    const res = await fetch(url);
    const data = await res.json();

    if (!res.ok || !data?.success) {
      throw new Error(data?.error || "Failed to load network identity");
    }

    identity.value = data.identity || null;
  } catch (e) {
    error.value = e?.message || String(e);
    identity.value = null;
  } finally {
    loading.value = false;
  }
}

onMounted(async () => {
  await loadIdentity();
  refreshHandle = setInterval(() => {
    loadIdentity();
  }, Math.max(30_000, Number(props.refreshMs) || 300_000));
});

onUnmounted(() => {
  if (refreshHandle) {
    clearInterval(refreshHandle);
    refreshHandle = null;
  }
});

const locationText = computed(() => {
  const v = identity.value;
  const parts = [v?.city, v?.region, v?.country].filter(Boolean);
  return parts.join(", ");
});

const ispText = computed(() => {
  const v = identity.value;
  return v?.isp || v?.org || "N/A";
});

const displayIp = computed(() => {
  if (!identity.value?.ip) return "—";
  if (isIpHovered.value) return identity.value.ip;
  return "XXX.XX.XXX.XX"; 
});

const displayLocation = computed(() => {
  if (!locationText.value) return "—";
  if (isLocationHovered.value) return locationText.value;
  return "XXX, XX";
});
</script>

<template>
  <div class="relative group h-full flex flex-col rounded-[24px] overflow-hidden transition-all duration-500 hover:-translate-y-2 hover:shadow-[0_20px_40px_-15px_rgba(59,130,246,0.4)] dark:hover:shadow-[0_20px_40px_-15px_rgba(59,130,246,0.3)] text-left w-full min-h-[320px] border border-blue-100/50 dark:border-white/5 bg-gradient-to-br from-blue-50/50 to-indigo-50/50 dark:from-[#111827] dark:to-[#09090b]">
    
    <!-- Animated Gradient Glow -->
    <div class="absolute -top-[50%] -left-[50%] w-[200%] h-[200%] bg-[radial-gradient(circle_at_50%_50%,_rgba(59,130,246,0.15)_0%,_transparent_50%)] dark:bg-[radial-gradient(circle_at_50%_50%,_rgba(59,130,246,0.12)_0%,_transparent_40%)] opacity-0 group-hover:opacity-100 transition-opacity duration-1000 pointer-events-none group-hover:animate-[spin_20s_linear_infinite]"></div>

    <!-- Massive Watermark Icon -->
    <Globe class="absolute -bottom-8 -right-8 w-64 h-64 text-blue-500/10 dark:text-blue-400/[0.03] rotate-[-20deg] group-hover:rotate-0 group-hover:scale-110 transition-all duration-700 ease-out pointer-events-none" />

    <!-- Header -->
    <div class="relative z-10 p-7 flex flex-col h-full gap-4">
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Gorgeous Icon Block -->
          <div class="relative w-14 h-14 rounded-2xl flex items-center justify-center shrink-0 bg-gradient-to-br from-blue-400 to-indigo-600 shadow-xl shadow-blue-500/30 group-hover:shadow-blue-500/50 transition-all duration-500 group-hover:scale-105">
            <div class="absolute inset-0 bg-white/30 dark:bg-white/20 rounded-2xl rounded-bl-none opacity-50 dark:opacity-40 mix-blend-overlay"></div>
            <Globe class="w-7 h-7 text-white drop-shadow-md z-10" />
          </div>
          <div>
            <h3 class="text-xl font-black text-blue-950 dark:text-white tracking-tight leading-none group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
              {{ t('quickMetrics.machineIdentity.title') }}
            </h3>
            <div class="flex items-center gap-1.5 mt-2">
               <div class="w-1.5 h-1.5 rounded-full"
                    :class="error ? 'bg-red-500' : loading ? 'bg-amber-500 animate-pulse' : 'bg-emerald-500'">
               </div>
               <span class="text-[10px] font-bold uppercase tracking-[0.2em]" :class="error ? 'text-red-600/70 dark:text-red-500' : loading ? 'text-amber-600/70 dark:text-amber-500' : 'text-emerald-600/70 dark:text-emerald-500'">
                 {{ loading ? t('quickMetrics.machineIdentity.updating') : error ? t('quickMetrics.machineIdentity.offline') : t('quickMetrics.machineIdentity.connected') }}
               </span>
            </div>
          </div>
        </div>
        
        <button 
           @click="loadIdentity({ force: true })"
           class="flex items-center gap-2 px-3 py-1.5 rounded-full border backdrop-blur-md shadow-[0_4px_12px_rgba(0,0,0,0.02)] dark:shadow-sm transition-colors duration-500 bg-white/60 border-white dark:bg-zinc-800/80 dark:border-white/5 hover:bg-blue-50 dark:hover:bg-zinc-700 z-20 cursor-pointer group/btn"
        >
           <RefreshCw class="w-3.5 h-3.5 text-blue-600 dark:text-blue-400 group-hover/btn:rotate-180 transition-transform duration-500" :class="{ 'animate-spin': loading }" />
        </button>
      </div>

      <!-- Main Content -->
      <div class="mt-auto pt-4 flex flex-col relative z-10">
        
        <!-- Error State -->
        <div v-if="error" class="bg-white/60 dark:bg-red-900/10 border border-white dark:border-red-900/50 rounded-[16px] p-4 flex items-start gap-3 backdrop-blur-xl">
          <AlertCircle class="w-5 h-5 text-red-500 shrink-0 mt-0.5" />
          <div>
            <div class="text-[10px] font-bold text-red-600 dark:text-red-400 uppercase tracking-[0.2em] mb-1">{{ t('quickMetrics.machineIdentity.connectionFailed') }}</div>
            <div class="text-[11px] leading-tight text-red-700/80 dark:text-red-400/80 line-clamp-2 font-medium">{{ error }}</div>
          </div>
        </div>

        <!-- Success/Loading State -->
        <template v-else>
           <!-- IP Address -->
           <div class="group/ip mb-6" @mouseenter="isIpHovered = true" @mouseleave="isIpHovered = false">
             <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-blue-700/60 dark:text-zinc-500 mb-2 flex items-center gap-2 drop-shadow-sm">
               {{ t('quickMetrics.machineIdentity.publicEndpoint') }}
               <ShieldCheck v-if="identity?.ip" class="w-3.5 h-3.5 text-emerald-500" />
             </div>
             
             <div v-if="loading && !identity" class="h-10 w-48 rounded bg-blue-100/50 dark:bg-zinc-800 animate-pulse"></div>
             <div v-else class="text-4xl sm:text-5xl font-black font-mono tracking-tighter text-blue-950 dark:text-white break-all transition-all duration-300 group-hover/ip:text-blue-600 dark:group-hover/ip:text-blue-400 drop-shadow-sm">
               {{ displayIp }}
             </div>
           </div>

           <!-- Details Grid -->
           <div class="grid grid-cols-2 gap-2">
             
             <!-- Location Card -->
             <div class="p-3.5 rounded-[16px] bg-white/60 dark:bg-black/40 backdrop-blur-xl border border-white dark:border-white/5 flex flex-col gap-1.5 shadow-[0_4px_12px_rgba(0,0,0,0.02)] dark:shadow-sm group/location" @mouseenter="isLocationHovered = true" @mouseleave="isLocationHovered = false">
                <div class="flex items-center justify-between gap-2">
                   <div class="flex items-center gap-1.5 text-blue-800/60 dark:text-zinc-400">
                     <MapPin class="w-3.5 h-3.5" />
                     <span class="text-[9px] font-bold uppercase tracking-widest">{{ t('quickMetrics.machineIdentity.location') }}</span>
                   </div>
                </div>
                <div v-if="loading && !identity" class="h-4 w-20 rounded bg-blue-100/50 dark:bg-zinc-800 animate-pulse"></div>
                <div v-else class="text-sm font-black text-blue-950 dark:text-white truncate transition-colors duration-300" :title="locationText">
                  {{ displayLocation }}
                </div>
             </div>
             
             <!-- Provider Card -->
             <div class="p-3.5 rounded-[16px] bg-white/60 dark:bg-black/40 backdrop-blur-xl border border-white dark:border-white/5 flex flex-col gap-1.5 shadow-[0_4px_12px_rgba(0,0,0,0.02)] dark:shadow-sm">
                <div class="flex items-center gap-1.5 text-blue-800/60 dark:text-zinc-400">
                  <Server class="w-3.5 h-3.5" />
                  <span class="text-[9px] font-bold uppercase tracking-widest">{{ t('quickMetrics.machineIdentity.provider') }}</span>
                </div>
                <div v-if="loading && !identity" class="h-4 w-24 rounded bg-blue-100/50 dark:bg-zinc-800 animate-pulse"></div>
                <div v-else class="text-sm font-black text-blue-950 dark:text-white truncate" :title="ispText">
                  {{ ispText }}
                </div>
             </div>
             
           </div>
        </template>
        
      </div>
    </div>
  </div>
</template>
