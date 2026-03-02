<script setup>
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useI18n } from 'vue-i18n'
import { Globe, MapPin, Network, RefreshCw, Server, ShieldCheck, AlertCircle } from "lucide-vue-next";

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
  <div class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl p-6 overflow-hidden transition-all duration-400 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:border-gray-300 dark:hover:border-zinc-600">
    
    <!-- Hover Accents -->
    <div class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-blue-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
    <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none [mask-image:linear-gradient(to_bottom,white,transparent)]"></div>

    <!-- Header -->
    <div class="relative z-10 flex items-start justify-between mb-6">
      <div class="flex items-center gap-4">
        <!-- Icon Container -->
        <div class="w-10 h-10 rounded-lg bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center shrink-0 group-hover:scale-105 group-hover:border-zinc-700 transition-all duration-500">
           <Globe class="w-5 h-5 text-gray-400 dark:text-zinc-500 group-hover:text-blue-500 transition-colors" />
        </div>
        
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
            {{ t('quickMetrics.machineIdentity.title') }}
          </h3>
          <div class="flex items-center gap-2 mt-1">
             <div class="w-1.5 h-1.5 rounded-full"
                  :class="error ? 'bg-red-500' : loading ? 'bg-amber-500 animate-pulse' : 'bg-green-500'">
             </div>
             <span class="text-[11px] font-medium text-gray-500 dark:text-zinc-400 uppercase tracking-wider">
               {{ loading ? t('quickMetrics.machineIdentity.updating') : error ? t('quickMetrics.machineIdentity.offline') : t('quickMetrics.machineIdentity.connected') }}
             </span>
          </div>
        </div>
      </div>
      
      <button 
         @click="loadIdentity({ force: true })"
         class="p-2 rounded-md text-gray-400 hover:text-blue-600 dark:hover:text-blue-400 hover:bg-gray-50 dark:hover:bg-zinc-900/50 transition-colors"
      >
         <RefreshCw :size="14" :class="{ 'animate-spin': loading }" />
      </button>
    </div>

    <!-- Main Content -->
    <div class="relative z-10 flex-1 flex flex-col justify-end gap-5">
      
      <!-- Error State -->
      <div v-if="error" class="bg-red-50/50 dark:bg-red-900/10 border border-red-100 dark:border-red-900/20 rounded-lg p-3 flex items-start gap-3">
        <AlertCircle class="w-4 h-4 text-red-500 shrink-0 mt-0.5" />
        <div>
          <div class="text-[11px] font-bold text-red-600 dark:text-red-400 uppercase tracking-wider">{{ t('quickMetrics.machineIdentity.connectionFailed') }}</div>
          <div class="text-[11px] leading-tight text-red-500/80 dark:text-red-400/80 mt-1 line-clamp-2">{{ error }}</div>
        </div>
      </div>

      <!-- Success/Loading State -->
      <template v-else>
         <!-- IP Address -->
         <div class="group/ip" @mouseenter="isIpHovered = true" @mouseleave="isIpHovered = false">
           <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-400 dark:text-zinc-500 mb-2 flex items-center gap-2">
             {{ t('quickMetrics.machineIdentity.publicEndpoint') }}
             <ShieldCheck v-if="identity?.ip" class="w-3.5 h-3.5 text-green-500" />
           </div>
           
           <div v-if="loading && !identity" class="h-8 w-32 bg-gray-100 dark:bg-zinc-800 rounded animate-pulse"></div>
           <div v-else class="text-3xl font-bold font-mono tracking-tighter text-gray-900 dark:text-white break-all transition-colors duration-300 group-hover/ip:text-blue-600 dark:group-hover/ip:text-blue-400">
             {{ displayIp }}
           </div>
         </div>

         <!-- Details Grid -->
         <div class="grid grid-cols-1 gap-3 pt-4 border-t border-gray-100 dark:border-zinc-800/80">
           <div class="flex items-center gap-3 group/location" @mouseenter="isLocationHovered = true" @mouseleave="isLocationHovered = false">
              <div class="w-6 h-6 rounded flex items-center justify-center bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50 shrink-0 group-hover/location:border-gray-200 dark:group-hover/location:border-zinc-700/50 transition-colors">
                 <MapPin class="w-3 h-3 text-gray-400 dark:text-zinc-500 group-hover/location:text-blue-500 transition-colors" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center justify-between">
                   <span class="text-[9px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500">{{ t('quickMetrics.machineIdentity.location') }}</span>
                   <div v-if="loading && !identity" class="h-3 w-16 bg-gray-100 dark:bg-zinc-800 rounded animate-pulse"></div>
                   <span v-else class="text-xs font-semibold text-gray-700 dark:text-gray-300 truncate transition-colors duration-300 group-hover/location:text-gray-900 dark:group-hover/location:text-white" :title="locationText">{{ displayLocation }}</span>
                </div>
              </div>
           </div>
           
           <div class="flex items-center gap-3">
              <div class="w-6 h-6 rounded flex items-center justify-center bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50 shrink-0">
                 <Server class="w-3 h-3 text-gray-400 dark:text-zinc-500" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center justify-between">
                   <span class="text-[9px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500">{{ t('quickMetrics.machineIdentity.provider') }}</span>
                   <div v-if="loading && !identity" class="h-3 w-16 bg-gray-100 dark:bg-zinc-800 rounded animate-pulse"></div>
                   <span v-else class="text-xs font-semibold text-gray-700 dark:text-gray-300 truncate">{{ ispText }}</span>
                </div>
              </div>
           </div>
         </div>
      </template>
      
    </div>
  </div>
</template>
