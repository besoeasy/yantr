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
    class="group relative flex h-full w-full min-h-[320px] cursor-pointer flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-all duration-300 hover:-translate-y-1 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <div class="relative z-10 flex h-full flex-col gap-4">
      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Minimal Icon Block -->
          <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 transition-transform duration-300 group-hover:scale-105 dark:border-zinc-800 dark:bg-zinc-900">
            <ShieldCheck class="h-5 w-5 text-zinc-900 dark:text-zinc-100" />
          </div>
          <div>
            <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors group-hover:text-black dark:text-white dark:group-hover:text-white">Active Proxies</h3>
            <div v-if="caddyRunning" class="mt-2 flex items-center gap-1.5">
              <div class="h-1.5 w-1.5 animate-pulse rounded-full bg-emerald-500"></div>
              <span class="text-[10px] font-semibold uppercase tracking-widest text-emerald-600 dark:text-emerald-500">Live</span>
            </div>
            <div v-else class="mt-2 text-[10px] font-semibold uppercase tracking-widest text-zinc-500">Stopped</div>
          </div>
        </div>
        
        <!-- Reload Button -->
        <button
            @click.stop="reload"
            :disabled="loading"
            class="group/btn z-20 flex cursor-pointer items-center gap-2 rounded-full border border-zinc-200 bg-zinc-50 px-3 py-1.5 transition-colors duration-300 hover:bg-zinc-100 dark:border-zinc-800 dark:bg-zinc-900/50 dark:hover:bg-zinc-800"
            title="Reload Caddy config"
        >
           <RefreshCw class="h-3.5 w-3.5 text-zinc-600 transition-transform duration-500 group-hover/btn:rotate-180 dark:text-zinc-400" :class="{ 'animate-spin': loading }" />
           <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-700 dark:text-zinc-300">Reload</span>
        </button>
      </div>

      <!-- Main Body -->
      <div class="relative z-10 mt-auto flex flex-col pt-6">
        <div class="mb-1 text-[10px] font-semibold uppercase tracking-widest text-zinc-500">
          Configured Routes
        </div>
        <div class="flex items-baseline gap-2">
          <div class="mb-2 text-6xl font-black tracking-tighter text-zinc-900 transition-transform duration-500 group-hover:-translate-y-1 dark:text-white">
            {{ proxies.length }}
          </div>
        </div>

        <!-- Arrow indicator at bottom right -->
        <div class="absolute bottom-0 right-0 opacity-0 -translate-x-4 translate-y-4 transition-all duration-500 group-hover:translate-x-0 group-hover:translate-y-0 group-hover:opacity-100">
          <div class="flex h-10 w-10 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 transition-colors duration-300 hover:bg-zinc-100 dark:border-zinc-800 dark:bg-zinc-900 dark:hover:bg-zinc-800">
             <ArrowUpRight class="h-4 w-4 text-zinc-900 dark:text-white" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
