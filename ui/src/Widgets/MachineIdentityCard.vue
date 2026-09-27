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
  <div class="group relative flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-all duration-300 hover:-translate-y-1 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="relative z-10 flex flex-col gap-4">
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Minimal Icon Block -->
          <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-sky-200 bg-sky-50 transition-transform duration-300 group-hover:scale-105 dark:border-sky-900/50 dark:bg-sky-900/20">
            <Globe class="h-5 w-5 text-sky-600 dark:text-sky-400" />
          </div>
          <div>
            <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors group-hover:text-black dark:text-white dark:group-hover:text-white">
              {{ t('quickMetrics.machineIdentity.title') }}
            </h3>
            <div class="mt-2 flex items-center gap-2">
               <span v-if="loading" class="relative flex h-2 w-2">
                  <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-amber-400 opacity-75"></span>
                  <span class="relative inline-flex h-2 w-2 rounded-full bg-amber-500"></span>
               </span>
               <span v-else-if="error" class="h-2 w-2 rounded-full bg-red-500"></span>
               <span v-else class="relative flex h-2 w-2">
                  <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
                  <span class="relative inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
               </span>
               <span class="text-[10px] font-semibold uppercase tracking-widest" :class="error ? 'text-red-600 dark:text-red-500' : loading ? 'text-amber-600 dark:text-amber-500' : 'text-emerald-600 dark:text-emerald-500'">
                 {{ loading ? t('quickMetrics.machineIdentity.updating') : error ? t('quickMetrics.machineIdentity.offline') : t('quickMetrics.machineIdentity.connected') }}
               </span>
            </div>
          </div>
        </div>
        
        <button 
           @click="loadIdentity({ force: true })"
           class="group/btn z-20 flex cursor-pointer items-center gap-2 rounded-full border border-zinc-200 bg-zinc-50 px-3 py-1.5 shadow-[0_4px_12px_rgba(0,0,0,0.02)] transition-colors duration-500 hover:bg-zinc-100 dark:border-zinc-800 dark:bg-zinc-900/50 dark:hover:bg-zinc-800"
        >
           <RefreshCw class="h-3.5 w-3.5 text-zinc-500 transition-transform duration-500 group-hover/btn:rotate-180 dark:text-zinc-400" :class="{ 'animate-spin': loading }" />
        </button>
      </div>
    </div>

    <!-- Main Content -->
    <div class="relative z-10 mt-auto flex flex-col pt-4">
      <!-- Error State -->
      <div v-if="error" class="flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-900/50 dark:bg-red-900/20">
        <AlertCircle class="mt-0.5 h-5 w-5 shrink-0 text-red-500" />
        <div>
          <div class="mb-1 text-[10px] font-semibold uppercase tracking-widest text-red-600 dark:text-red-500">{{ t('quickMetrics.machineIdentity.connectionFailed') }}</div>
          <div class="line-clamp-2 text-[11px] font-medium leading-tight text-red-600 dark:text-red-400">{{ error }}</div>
        </div>
      </div>

      <!-- Success/Loading State -->
      <template v-else>
         <!-- IP Address -->
         <div class="group/ip mb-6" @mouseenter="isIpHovered = true" @mouseleave="isIpHovered = false">
           <div class="mb-2 flex items-center gap-2 text-[10px] font-semibold uppercase tracking-widest text-zinc-500 dark:text-zinc-400">
             {{ t('quickMetrics.machineIdentity.publicEndpoint') }}
             <ShieldCheck v-if="identity?.ip" class="h-3.5 w-3.5 text-emerald-500" />
           </div>
           
           <div v-if="loading && !identity" class="h-10 w-48 animate-pulse rounded bg-zinc-200 dark:bg-zinc-800"></div>
           <div v-else class="font-mono text-4xl font-black tracking-tighter text-zinc-900 transition-colors duration-300 group-hover/ip:text-black sm:text-5xl dark:text-white dark:group-hover/ip:text-zinc-200 break-all">
             {{ displayIp }}
           </div>
         </div>

         <!-- Details Grid -->
         <div class="grid grid-cols-2 gap-2">
           <!-- Location Card -->
           <div class="group/location flex flex-col gap-2 rounded-xl border border-zinc-200 bg-zinc-50 p-3.5 transition-all duration-300 dark:border-zinc-800 dark:bg-zinc-900/50" @mouseenter="isLocationHovered = true" @mouseleave="isLocationHovered = false">
              <div class="flex items-center justify-between gap-2">
                 <div class="flex items-center gap-1.5 text-zinc-500 dark:text-zinc-400">
                   <MapPin class="h-3.5 w-3.5" />
                   <span class="text-[9px] font-semibold uppercase tracking-widest">{{ t('quickMetrics.machineIdentity.location') }}</span>
                 </div>
              </div>
              <div v-if="loading && !identity" class="h-4 w-20 animate-pulse rounded bg-zinc-200 dark:bg-zinc-800"></div>
              <div v-else class="truncate text-sm font-black text-zinc-900 transition-colors duration-300 dark:text-white" :title="locationText">
                {{ displayLocation }}
              </div>
           </div>
           
           <!-- Provider Card -->
           <div class="flex flex-col gap-2 rounded-xl border border-zinc-200 bg-zinc-50 p-3.5 transition-all duration-300 dark:border-zinc-800 dark:bg-zinc-900/50">
              <div class="flex items-center gap-1.5 text-zinc-500 dark:text-zinc-400">
                <Server class="h-3.5 w-3.5" />
                <span class="text-[9px] font-semibold uppercase tracking-widest">{{ t('quickMetrics.machineIdentity.provider') }}</span>
              </div>
              <div v-if="loading && !identity" class="h-4 w-24 animate-pulse rounded bg-zinc-200 dark:bg-zinc-800"></div>
              <div v-else class="truncate text-sm font-black text-zinc-900 dark:text-white" :title="ispText">
                {{ ispText }}
              </div>
           </div>
         </div>
      </template>
    </div>
  </div>
</template>
