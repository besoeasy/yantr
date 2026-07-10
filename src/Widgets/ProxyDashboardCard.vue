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
    class="relative group h-full flex flex-col rounded-[24px] overflow-hidden transition-all duration-500 hover:-translate-y-2 hover:shadow-[0_20px_40px_-15px_rgba(16,185,129,0.4)] dark:hover:shadow-[0_20px_40px_-15px_rgba(16,185,129,0.3)] text-left w-full min-h-[320px] cursor-pointer border border-emerald-100/50 dark:border-white/5 bg-gradient-to-br from-emerald-50/50 to-teal-50/50 dark:from-[#111827] dark:to-[#09090b]"
  >
    
    <!-- Animated Gradient Glow -->
    <div class="absolute -top-[50%] -left-[50%] w-[200%] h-[200%] bg-[radial-gradient(circle_at_50%_50%,_rgba(16,185,129,0.15)_0%,_transparent_50%)] dark:bg-[radial-gradient(circle_at_50%_50%,_rgba(16,185,129,0.12)_0%,_transparent_40%)] opacity-0 group-hover:opacity-100 transition-opacity duration-1000 pointer-events-none group-hover:animate-[spin_20s_linear_infinite]"></div>

    <!-- Massive Watermark Icon -->
    <ShieldCheck class="absolute -bottom-8 -right-8 w-64 h-64 text-emerald-500/10 dark:text-emerald-400/[0.03] rotate-[-20deg] group-hover:rotate-0 group-hover:scale-110 transition-all duration-700 ease-out pointer-events-none" />

    <div class="relative z-10 p-7 flex flex-col h-full gap-4">
      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Gorgeous Icon Block -->
          <div class="relative w-14 h-14 rounded-2xl flex items-center justify-center shrink-0 bg-gradient-to-br from-emerald-400 to-teal-600 shadow-xl shadow-emerald-500/30 group-hover:shadow-emerald-500/50 transition-all duration-500 group-hover:scale-105">
            <div class="absolute inset-0 bg-white/30 dark:bg-white/20 rounded-2xl rounded-bl-none opacity-50 dark:opacity-40 mix-blend-overlay"></div>
            <ShieldCheck class="w-7 h-7 text-white drop-shadow-md z-10" />
          </div>
          <div>
            <h3 class="text-xl font-black text-emerald-950 dark:text-white tracking-tight leading-none group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors">Active Proxies</h3>
            <div v-if="caddyRunning" class="flex items-center gap-1.5 mt-2">
              <div class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></div>
              <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-emerald-600/70 dark:text-emerald-500">Live</span>
            </div>
            <div v-else class="text-[10px] font-bold uppercase tracking-[0.2em] text-emerald-600/70 dark:text-zinc-500 mt-2">Stopped</div>
          </div>
        </div>
        
        <!-- Reload Button in Glass Pill -->
        <button
            @click.stop="reload"
            :disabled="loading"
            class="flex items-center gap-2 px-3 py-1.5 rounded-full border backdrop-blur-md shadow-[0_4px_12px_rgba(0,0,0,0.02)] dark:shadow-none transition-colors duration-500 bg-white/60 border-white dark:bg-zinc-800/80 dark:border-white/5 hover:bg-emerald-50 dark:hover:bg-zinc-700 z-20 cursor-pointer group/btn"
            title="Reload Caddy config"
        >
           <RefreshCw class="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400 group-hover/btn:rotate-180 transition-transform duration-500" :class="{ 'animate-spin': loading }" />
           <span class="text-[9px] font-black uppercase tracking-widest text-emerald-700 dark:text-emerald-300">Reload</span>
        </button>
      </div>

      <!-- Main Body -->
      <div class="mt-auto pt-6 flex flex-col relative z-10">
        <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-emerald-700/60 dark:text-zinc-500 mb-1 drop-shadow-sm">
          Configured Routes
        </div>
        <div class="flex items-baseline gap-2">
          <div class="text-6xl font-black tracking-tighter text-emerald-950 dark:text-white mb-2 drop-shadow-sm group-hover:-translate-y-1 transition-transform duration-500">
            {{ proxies.length }}
          </div>
        </div>

        <!-- Arrow indicator at bottom right -->
        <div class="absolute bottom-0 right-0 opacity-0 -translate-x-4 translate-y-4 group-hover:opacity-100 group-hover:translate-x-0 group-hover:translate-y-0 transition-all duration-500">
          <div class="w-12 h-12 rounded-full flex items-center justify-center bg-white/60 dark:bg-black/40 backdrop-blur-xl border border-white dark:border-white/5 shadow-sm hover:bg-emerald-50 hover:border-emerald-200 hover:text-emerald-600 transition-colors duration-300">
             <ArrowUpRight class="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
          </div>
        </div>

      </div>
    </div>
  </div>
</template>
