<script setup>
import { ref, onMounted, onUnmounted} from 'vue'
import { ShieldCheck, Trash2, Loader, Globe, User, RefreshCw } from '@lucide/vue'
import { useApiUrl } from '../composables/useApiUrl'
import { expectApiSuccess } from '../composables/useApiResponse'

const { apiUrl } = useApiUrl()

const proxies = ref([])
const caddyRunning = ref(false)
const loading = ref(false)
let refreshInterval = null

async function fetchProxies() {
  try {
    const res = await fetch(`${apiUrl.value}/api/proxy`)
    const data = await expectApiSuccess(res, 'Failed to load proxies')
    proxies.value = Array.isArray(data.proxies) ? data.proxies : []
    caddyRunning.value = !!data.caddyRunning
  } catch {}
}

onMounted(() => {
  fetchProxies()
  refreshInterval = setInterval(fetchProxies, 10000)
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
})


async function reload() {
  loading.value = true
  try {
    const response = await fetch(`${apiUrl.value}/api/proxy/reload`, { method: 'POST' })
    await expectApiSuccess(response, 'Failed to reload proxy config')
    await fetchProxies()
  } catch {}
  finally {
    loading.value = false
  }
}
</script>

<template>
  <div
    v-if="proxies.length > 0"
    class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] rounded-xl overflow-hidden transition-all duration-300 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40"
  >
    <!-- top accent line -->
    <div class="absolute top-0 left-0 w-full h-0.5 bg-emerald-500 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>

    <div class="relative z-10 p-5 flex flex-col gap-4 h-full">
      <!-- header -->
      <div class="flex items-center justify-between gap-3">
        <div class="flex items-center gap-3 min-w-0">
          <div class="w-9 h-9 rounded-lg bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-100 dark:border-emerald-500/20 flex items-center justify-center shrink-0">
            <ShieldCheck class="w-4.5 h-4.5 text-emerald-600 dark:text-emerald-400" />
          </div>
          <div class="min-w-0">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight leading-none">Active Proxies</h3>
            <p class="text-[11px] font-medium text-gray-500 dark:text-zinc-400 uppercase tracking-wider mt-1">Caddy · {{ proxies.length }} running</p>
          </div>
        </div>
        <div class="flex items-center gap-2 shrink-0">
          <div v-if="caddyRunning" class="flex items-center gap-1.5">
            <div class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></div>
            <span class="text-[10px] font-bold uppercase tracking-wider text-emerald-600 dark:text-emerald-400">Live</span>
          </div>
          <button
            @click="reload"
            :disabled="loading"
            class="p-1.5 rounded-lg text-gray-400 dark:text-zinc-500 hover:text-gray-700 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-800 transition-all"
            title="Reload Caddy config"
          >
            <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': loading }" />
          </button>
        </div>
      </div>

      <!-- proxy list -->
      <div class="flex flex-col gap-2">
        <div
          v-for="p in proxies"
          :key="p.projectId"
          class="flex items-center gap-3 rounded-lg border border-gray-100 dark:border-zinc-800/80 bg-gray-50/50 dark:bg-zinc-900/40 px-3 py-2.5 group/row transition-colors hover:border-gray-200 dark:hover:border-zinc-700"
        >
          <!-- ports -->
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-1.5 flex-wrap">
              <span class="font-mono text-[12px] font-bold text-emerald-600 dark:text-emerald-400">:{{ p.servePort }}</span>
              <span class="text-[10px] text-gray-400 dark:text-zinc-500">→</span>
              <span class="font-mono text-[11px] text-gray-600 dark:text-zinc-300">:{{ p.targetPort }}</span>
            </div>
            <div class="flex items-center gap-2 mt-0.5 flex-wrap">
              <div class="flex items-center gap-1 text-[10px] text-gray-400 dark:text-zinc-500">
                <Globe class="w-3 h-3" />
                <span class="truncate max-w-[100px]">{{ p.containerName }}</span>
              </div>
              <div v-if="p.authUser" class="flex items-center gap-1 text-[10px] text-gray-400 dark:text-zinc-500">
                <User class="w-3 h-3" />
                <span>{{ p.authUser }}</span>
              </div>
            </div>
          </div>


        </div>
      </div>
    </div>
  </div>
</template>
