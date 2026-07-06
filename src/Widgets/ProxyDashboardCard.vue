<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { ShieldCheck, RefreshCw, ArrowUpRight } from '@lucide/vue'
import { useApiUrl } from '../composables/useApiUrl'
import { expectApiSuccess } from '../composables/useApiResponse'
import { useRouter } from 'vue-router'

const { apiUrl } = useApiUrl()
const router = useRouter()

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

async function reload(e) {
  if (e) e.stopPropagation()
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

function goToProxies() {
  router.push('/proxies')
}
</script>

<template>
  <div
    v-if="proxies.length > 0"
    @click="goToProxies"
    class="cursor-pointer relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] rounded-xl overflow-hidden transition-all duration-300 hover:-translate-y-0.5 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40"
  >
    <!-- top accent line -->
    <div class="absolute top-0 left-0 w-full h-0.5 bg-emerald-500 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>

    <div class="relative z-10 p-5 flex flex-col gap-4 h-full">
      <!-- header -->
      <div class="flex items-center justify-between gap-3">
        <div class="flex items-center gap-3 min-w-0">
          <div class="w-9 h-9 rounded-lg flex items-center justify-center shrink-0 transition-transform group-hover:scale-110">
            <ShieldCheck class="w-4.5 h-4.5 text-emerald-600 dark:text-emerald-400" />
          </div>
          <div class="min-w-0">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight leading-none">Active Proxies</h3>
            <div v-if="caddyRunning" class="flex items-center gap-1.5 mt-1.5">
              <div class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></div>
              <span class="text-[10px] font-bold uppercase tracking-wider text-emerald-600 dark:text-emerald-400">Live</span>
            </div>
            <div v-else class="text-[10px] font-medium text-gray-500 dark:text-zinc-400 uppercase tracking-wider mt-1.5">Stopped</div>
          </div>
        </div>
        <div class="flex items-center gap-2 shrink-0">
          <button
            @click="reload"
            :disabled="loading"
            class="p-1.5 rounded-lg text-gray-400 dark:text-zinc-500 hover:text-gray-700 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-800 transition-all z-20 relative"
            title="Reload Caddy config"
          >
            <RefreshCw class="w-3.5 h-3.5" :class="{ 'animate-spin': loading }" />
          </button>
        </div>
      </div>

      <div class="mt-auto pt-6 pb-2 flex items-baseline gap-2">
        <div class="text-6xl font-black tracking-tighter text-gray-900 dark:text-white group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors">
          {{ proxies.length }}
        </div>
        <div class="text-sm font-bold text-gray-400 dark:text-zinc-500 uppercase tracking-widest">
          Routes
        </div>
      </div>

      <!-- Arrow indicator -->
      <div class="absolute bottom-5 right-5 opacity-0 -translate-x-2 translate-y-2 group-hover:opacity-100 group-hover:translate-x-0 group-hover:translate-y-0 transition-all duration-300">
        <div class="w-8 h-8 rounded-full flex items-center justify-center">
           <ArrowUpRight class="w-4 h-4 text-gray-600 dark:text-zinc-400" />
        </div>
      </div>

    </div>
  </div>
</template>
