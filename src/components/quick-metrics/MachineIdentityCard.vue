<script setup>
import { computed, onMounted, onUnmounted, ref } from "vue";
import { Globe, MapPin, Wifi } from "lucide-vue-next";

const props = defineProps({
  refreshMs: { type: Number, default: 5 * 60_000 },
});

const loading = ref(true);
const error = ref(null);
const identity = ref(null);
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
  return v?.isp || v?.org || "—";
});
</script>

<template>
  <div class="relative h-full overflow-hidden group rounded-2xl transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)] hover:-translate-y-1">
    <div class="absolute inset-0 bg-white dark:bg-gray-900">
      <div class="absolute inset-0 bg-linear-to-br from-emerald-200/60 via-slate-200/30 to-white/80 dark:from-emerald-600/20 dark:via-slate-600/10 dark:to-gray-900 z-10"></div>
      <div
        class="absolute top-0 right-0 w-64 h-64 bg-emerald-300/35 dark:bg-emerald-500/20 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2 group-hover:bg-emerald-400/45 dark:group-hover:bg-emerald-500/30 transition-colors duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"
      ></div>
      <div
        class="absolute bottom-0 left-0 w-48 h-48 bg-cyan-300/30 dark:bg-cyan-500/15 rounded-full blur-3xl translate-y-1/2 -translate-x-1/2 group-hover:bg-cyan-400/40 dark:group-hover:bg-cyan-500/25 transition-colors duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"
      ></div>
    </div>

    <div
      class="relative z-20 h-full p-6 flex flex-col border border-slate-200/80 dark:border-slate-700/60 rounded-2xl backdrop-blur-sm group-hover:border-emerald-300/60 dark:group-hover:border-emerald-500/30 transition-none"
    >
      <div class="flex items-start justify-between gap-4">
        <div class="flex items-center gap-4">
          <div class="relative">
            <div class="absolute inset-0 bg-emerald-400/25 dark:bg-emerald-500/20 rounded-xl blur-lg group-hover:blur-xl transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"></div>
            <div
              class="relative w-12 h-12 bg-linear-to-br from-emerald-500 to-cyan-600 rounded-xl flex items-center justify-center shadow-lg group-hover:scale-110 transition-transform duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"
            >
              <Globe class="w-6 h-6 text-white" />
            </div>
          </div>

          <div>
            <h3 class="text-lg font-bold text-slate-900 dark:text-white mb-1 group-hover:text-emerald-700 dark:group-hover:text-emerald-200 transition-colors">Network</h3>
            <p class="text-sm font-medium text-slate-600 dark:text-gray-400 group-hover:text-slate-700 dark:group-hover:text-gray-300 transition-colors">Machine public identity</p>
          </div>
        </div>

        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-xl bg-white/70 dark:bg-white/5 border border-slate-200/70 dark:border-slate-700/60 px-3 py-1.5 text-xs font-semibold text-slate-600 dark:text-gray-200 hover:bg-white transition-colors"
          @click="loadIdentity({ force: true })"
        >
          Refresh
        </button>
      </div>

      <div class="mt-5 flex-1">
        <div v-if="loading" class="rounded-2xl p-4">
          <div class="h-4 w-40 bg-slate-200/70 dark:bg-white/10 rounded animate-pulse"></div>
          <div class="mt-3 h-3 w-56 bg-slate-200/70 dark:bg-white/10 rounded animate-pulse"></div>
          <div class="mt-3 h-3 w-44 bg-slate-200/70 dark:bg-white/10 rounded animate-pulse"></div>
        </div>

        <div v-else-if="error" class="rounded-2xl border border-red-200/60 dark:border-red-500/20 bg-red-50 dark:bg-red-500/10 p-4">
          <div class="text-sm font-semibold text-red-600 dark:text-red-200">Couldn’t fetch IP details</div>
          <div class="mt-1 text-xs text-red-500 dark:text-red-200/80">{{ error }}</div>
          <div class="mt-3 text-xs text-slate-500 dark:text-gray-300">Tip: this needs outbound internet access from the daemon.</div>
        </div>

        <div v-else class="rounded-2xl p-4 bg-white/60 dark:bg-transparent">
          <div class="flex items-start justify-between gap-4">
            <div class="min-w-0">
              <div class="text-xs font-semibold text-slate-500 dark:text-gray-400">Public IP</div>
              <div class="mt-1 text-xl font-extrabold text-slate-900 dark:text-white tabular-nums break-all">
                {{ identity?.ip || "—" }}
              </div>
              <div class="mt-1 text-[11px] text-slate-500 dark:text-gray-400">
                Source: {{ identity?.source || "daemon" }}
                <span v-if="identity?.fetchedAt">• {{ new Date(identity.fetchedAt).toLocaleString() }}</span>
              </div>
            </div>

            <div class="shrink-0 inline-flex items-center gap-2 rounded-xl bg-white/70 dark:bg-white/5 border border-slate-200/70 dark:border-slate-700/60 px-3 py-1.5">
              <Wifi class="w-4 h-4 text-emerald-500 dark:text-emerald-300" />
              <span class="text-xs font-semibold text-slate-600 dark:text-gray-200">ISP</span>
            </div>
          </div>

          <div class="mt-4 space-y-3">
            <div class="flex items-center justify-between gap-3 text-sm py-2 border-b border-slate-200/70 dark:border-slate-700/60 border-dashed">
              <div class="inline-flex items-center gap-2 min-w-0">
                <MapPin class="w-4 h-4 text-cyan-500 dark:text-cyan-300" />
                <span class="text-slate-600 dark:text-gray-400 font-medium">Location</span>
              </div>
              <span class="text-slate-800 dark:text-gray-200 font-semibold text-right truncate" :title="locationText">{{ locationText || "—" }}</span>
            </div>

            <div class="flex items-center justify-between gap-3 text-sm py-2 border-b border-slate-200/70 dark:border-slate-700/60 border-dashed">
              <div class="inline-flex items-center gap-2 min-w-0">
                <Wifi class="w-4 h-4 text-emerald-500 dark:text-emerald-300" />
                <span class="text-slate-600 dark:text-gray-400 font-medium">ISP</span>
              </div>
              <span class="text-slate-800 dark:text-gray-200 font-semibold text-right truncate" :title="ispText">{{ ispText }}</span>
            </div>

            <div class="flex items-center justify-between gap-3 text-sm py-2">
              <div class="inline-flex items-center gap-2 min-w-0">
                <Globe class="w-4 h-4 text-emerald-500 dark:text-emerald-300" />
                <span class="text-slate-600 dark:text-gray-400 font-medium">ASN</span>
              </div>
              <span class="text-slate-800 dark:text-gray-200 font-semibold tabular-nums">{{ identity?.asn || "—" }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
