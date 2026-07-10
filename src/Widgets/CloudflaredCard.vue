<script setup>
import { ref, computed, onMounted, onUnmounted} from 'vue'
import { useI18n } from 'vue-i18n'
import { Cloud, CloudOff, Shield, ArrowRight, Key, CheckCircle, AlertCircle, Loader, Globe, Wifi } from '@lucide/vue'
import { useApiUrl } from '../composables/useApiUrl'
import { expectApiSuccess } from '../composables/useApiResponse'
import { useCurrentTime } from '../composables/useCurrentTime'

const { t } = useI18n()
const { apiUrl } = useApiUrl()
const { currentTime } = useCurrentTime()

// --- Container polling ---
const containers = ref([])
let refreshInterval = null

async function fetchContainers() {
  try {
    const response = await fetch(`${apiUrl.value}/api/containers`)
    const data = await expectApiSuccess(response, 'Failed to load containers')
    containers.value = Array.isArray(data.containers) ? data.containers : []
  } catch {}
}

onMounted(() => {
  fetchContainers()
  refreshInterval = setInterval(fetchContainers, 15000)
})

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval)
})

const cloudflaredContainer = computed(() => {
  const list = Array.isArray(containers.value) ? containers.value : []
  const matches = list.filter((c) => {
    const name = (c?.name || '').toLowerCase()
    const names = Array.isArray(c?.Names) ? c.Names : []
    return name.includes('cloudflared') || names.some((n) => (n || '').toLowerCase().includes('cloudflared'))
  })
  if (!matches.length) return null
  return matches.find((c) => c?.state === 'running') || matches[0]
})

// --- Status ---
const isRunning = computed(() => cloudflaredContainer.value?.state === 'running')

const uptimeMs = computed(() => {
  const c = cloudflaredContainer.value
  if (!c || c.state !== 'running' || !c.created) return null
  const createdMs = Number(c.created) * 1000
  if (!Number.isFinite(createdMs)) return null
  return Math.max(0, currentTime.value - createdMs)
})

const imageVersion = computed(() => {
  const image = cloudflaredContainer.value?.image || ''
  const tag = image.split(':')[1] || ''
  if (!tag || tag === 'latest') return 'latest'
  return tag.length > 12 ? tag.slice(0, 12) + '…' : tag
})

const containerName = computed(() => cloudflaredContainer.value?.name || '—')

function formatUptime(ms) {
  if (ms === null) return '—'
  const s = Math.floor(ms / 1000)
  if (s < 60) return `${s}s`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h ${m % 60}m`
  const d = Math.floor(h / 24)
  return `${d}d ${h % 24}h`
}

// --- Setup / Deploy ---
const tunnelToken = ref('')
const deploying = ref(false)
const deployError = ref('')
const deploySuccess = ref(false)

const features = [
  { icon: Shield, label: 'No port forwarding' },
  { icon: Globe, label: 'Auto HTTPS' },
  { icon: Wifi, label: 'Zero-trust access' },
]

const isValidToken = computed(() => tunnelToken.value.trim().length > 20)

async function deploy() {
  if (!isValidToken.value || deploying.value) return
  deploying.value = true
  deployError.value = ''
  try {
    const res = await fetch(`${apiUrl.value}/api/deploy`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        appId: 'cloudflared',
        environment: { TUNNEL_TOKEN: tunnelToken.value.trim() },
      }),
    })
    await expectApiSuccess(res, 'Deployment failed')
    deploySuccess.value = true
    setTimeout(fetchContainers, 3000)
  } catch (e) {
    deployError.value = e.message || 'Network error'
  } finally {
    deploying.value = false
  }
}
</script>

<template>
  <div class="relative group h-full flex flex-col rounded-[24px] overflow-hidden transition-all duration-500 hover:-translate-y-2 hover:shadow-[0_20px_40px_-15px_rgba(249,115,22,0.4)] dark:hover:shadow-[0_20px_40px_-15px_rgba(249,115,22,0.3)] text-left w-full min-h-[320px] border border-orange-100/50 dark:border-white/5 bg-gradient-to-br from-orange-50/50 to-amber-50/50 dark:from-[#111827] dark:to-[#09090b]">
    
    <!-- Animated Gradient Glow -->
    <div class="absolute -top-[50%] -left-[50%] w-[200%] h-[200%] bg-[radial-gradient(circle_at_50%_50%,_rgba(249,115,22,0.15)_0%,_transparent_50%)] dark:bg-[radial-gradient(circle_at_50%_50%,_rgba(249,115,22,0.12)_0%,_transparent_40%)] opacity-0 group-hover:opacity-100 transition-opacity duration-1000 pointer-events-none group-hover:animate-[spin_20s_linear_infinite]"></div>

    <!-- Massive Watermark Icon -->
    <Cloud class="absolute -bottom-8 -right-8 w-64 h-64 text-orange-500/10 dark:text-orange-400/[0.03] rotate-[-20deg] group-hover:rotate-0 group-hover:scale-110 transition-all duration-700 ease-out pointer-events-none" />

    <!-- Setup state: no cloudflared container found -->
    <div v-if="!cloudflaredContainer" class="relative z-10 p-7 flex flex-col h-full gap-4">
      <transition
        enter-active-class="transition-all duration-500 ease-out"
        enter-from-class="opacity-0 scale-95"
        enter-to-class="opacity-100 scale-100"
      >
        <div v-if="deploySuccess" class="absolute inset-0 z-20 flex flex-col items-center justify-center bg-white/95 dark:bg-[#0A0A0A]/95 rounded-[24px] gap-3">
          <div class="w-12 h-12 rounded-full flex items-center justify-center">
            <CheckCircle class="w-6 h-6 text-green-600 dark:text-green-500" />
          </div>
          <div class="text-center">
            <p class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight">Tunnel Deployed</p>
            <p class="text-[11px] text-gray-500 dark:text-zinc-400 mt-1 uppercase tracking-widest font-medium">Container starting…</p>
          </div>
        </div>
      </transition>

      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Gorgeous Icon Block -->
          <div class="relative w-14 h-14 rounded-2xl flex items-center justify-center shrink-0 bg-gradient-to-br from-orange-400 to-amber-600 shadow-xl shadow-orange-500/30 group-hover:shadow-orange-500/50 transition-all duration-500 group-hover:scale-105">
            <div class="absolute inset-0 bg-white/30 dark:bg-white/20 rounded-2xl rounded-bl-none opacity-50 dark:opacity-40 mix-blend-overlay"></div>
            <Cloud class="w-7 h-7 text-white drop-shadow-md z-10" />
          </div>
          <div>
            <h3 class="text-xl font-black text-orange-950 dark:text-white tracking-tight leading-none group-hover:text-orange-600 dark:group-hover:text-orange-400 transition-colors">Cloudflared</h3>
            <div class="flex items-center gap-1.5 mt-2">
              <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-orange-600/70 dark:text-zinc-500">Secure Tunnel</span>
            </div>
          </div>
        </div>
        <div class="flex items-center gap-2 px-3 py-1.5 rounded-full border backdrop-blur-md shadow-sm transition-colors duration-500 bg-white/80 border-gray-200 dark:bg-zinc-800/80 dark:border-zinc-700">
           <div class="w-1.5 h-1.5 rounded-full bg-amber-500 animate-pulse relative"></div>
           <span class="text-[9px] font-black uppercase tracking-widest text-gray-500 dark:text-gray-400">Not Installed</span>
        </div>
      </div>

      <!-- Main content -->
      <div class="mt-auto pt-6 flex flex-col relative z-10">
        <div class="text-4xl font-black tracking-tighter text-orange-950 dark:text-white transition-colors line-clamp-1 leading-[1.1] mb-2 drop-shadow-sm group-hover:-translate-y-1 duration-500">
          Secure Tunnel
        </div>
        <div class="text-xs text-orange-800/70 dark:text-zinc-400 font-medium mb-5">
           Connect your Yantr apps securely without exposing any ports.
        </div>
        
        <div>
          <div class="relative">
            <input
              v-model="tunnelToken"
              type="text"
              placeholder="eyJhIjoiY…"
              autocomplete="off"
              spellcheck="false"
              class="w-full rounded-[16px] px-4 py-3 text-sm font-mono text-gray-900 dark:text-white placeholder-orange-300 dark:placeholder-zinc-600 outline-none transition-all duration-300 pr-10 bg-white/60 dark:bg-black/40 backdrop-blur-xl border border-white dark:border-white/5 shadow-[0_4px_12px_rgba(0,0,0,0.02)] dark:shadow-sm"
              :class="tunnelToken.trim() === ''
                ? 'focus:border-orange-400 dark:focus:border-orange-500 focus:ring-1 focus:ring-orange-400/20'
                : isValidToken
                  ? 'border-emerald-300 dark:border-emerald-600 focus:border-emerald-400 focus:ring-1 focus:ring-emerald-400/20'
                  : 'border-red-300 dark:border-red-700 focus:border-red-400 focus:ring-1 focus:ring-red-400/20'"
            />
            <div class="absolute right-3.5 top-1/2 -translate-y-1/2">
              <CheckCircle v-if="isValidToken" class="w-4 h-4 text-emerald-500" />
              <AlertCircle v-else-if="tunnelToken.trim()" class="w-4 h-4 text-red-400" />
            </div>
          </div>
        </div>

        <transition
          enter-active-class="transition-all duration-300 ease-out"
          enter-from-class="opacity-0 translate-y-1"
          enter-to-class="opacity-100 translate-y-0"
          leave-active-class="transition-all duration-150 ease-in"
          leave-from-class="opacity-100 translate-y-0"
          leave-to-class="opacity-0 translate-y-1"
        >
          <div v-if="deployError" class="mt-3 flex items-center gap-2 p-3 rounded-[12px] bg-red-50 dark:bg-red-950/30 border border-red-100 dark:border-red-900/50">
            <AlertCircle class="w-4 h-4 text-red-500 shrink-0" />
            <p class="text-xs text-red-600 dark:text-red-400 font-medium">{{ deployError }}</p>
          </div>
        </transition>

        <div class="mt-4 grid grid-cols-1 gap-2.5 sm:grid-cols-2">
          <a
            href="https://one.dash.cloudflare.com/"
            target="_blank"
            rel="noopener noreferrer"
            class="group/link flex h-11 items-center justify-center gap-2 rounded-[16px] text-orange-700/60 dark:text-zinc-400 bg-white/40 dark:bg-white/5 backdrop-blur-xl border border-white dark:border-white/5 hover:bg-white/80 dark:hover:bg-zinc-800 transition-all duration-300 hover:text-orange-900 dark:hover:text-white shadow-[0_4px_12px_rgba(0,0,0,0.02)] dark:shadow-none"
          >
            <Key class="w-3.5 h-3.5" />
            <span class="text-[11px] font-bold uppercase tracking-widest">Get Token</span>
            <ArrowRight class="w-3.5 h-3.5 opacity-60 group-hover/link:translate-x-0.5 transition-transform duration-300" />
          </a>

          <button
            @click="deploy"
            :disabled="!isValidToken || deploying"
            class="group/cta relative flex h-11 w-full items-center justify-center gap-2 rounded-[16px] transition-all duration-300 overflow-hidden text-white shadow-md border border-transparent"
            :class="isValidToken && !deploying
              ? 'bg-gradient-to-r from-orange-500 to-amber-600 hover:from-orange-400 hover:to-amber-500 shadow-orange-500/30 hover:shadow-orange-500/50 active:scale-[0.98]'
              : 'bg-gray-300 dark:bg-zinc-700 text-gray-500 dark:text-zinc-500 shadow-none cursor-not-allowed opacity-50'"
          >
            <div class="flex items-center gap-2 relative z-10">
              <Loader v-if="deploying" class="w-4 h-4 animate-spin" />
              <Cloud v-else class="w-4 h-4 drop-shadow-sm" />
              <span class="text-[11px] font-bold uppercase tracking-widest drop-shadow-sm">{{ deploying ? 'Deploying…' : 'Deploy Tunnel' }}</span>
            </div>
            <div v-if="isValidToken && !deploying" class="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent -translate-x-full group-hover/cta:translate-x-full transition-transform duration-700 ease-out mix-blend-overlay"></div>
          </button>
        </div>
      </div>
    </div>

    <!-- Status state: cloudflared container exists -->
    <div v-else class="relative z-10 p-7 flex flex-col h-full gap-4">
      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Gorgeous Icon Block -->
          <div class="relative w-14 h-14 rounded-2xl flex items-center justify-center shrink-0 bg-gradient-to-br from-orange-400 to-amber-600 shadow-xl shadow-orange-500/30 group-hover:shadow-orange-500/50 transition-all duration-500 group-hover:scale-105">
            <div class="absolute inset-0 bg-white/30 dark:bg-white/20 rounded-2xl rounded-bl-none opacity-50 dark:opacity-40 mix-blend-overlay"></div>
            <Cloud class="w-7 h-7 text-white drop-shadow-md z-10" />
          </div>
          <div>
            <h3 class="text-xl font-black text-orange-950 dark:text-white tracking-tight leading-none group-hover:text-orange-600 dark:group-hover:text-orange-400 transition-colors">Cloudflared</h3>
            <p class="text-[10px] font-bold uppercase tracking-[0.2em] text-orange-600/70 dark:text-zinc-500 mt-2">Secure Tunnel</p>
          </div>
        </div>
        
        <!-- Premium Status Pill -->
        <div class="flex items-center gap-2 px-3 py-1.5 rounded-full border backdrop-blur-md shadow-sm transition-colors duration-500" :class="isRunning ? 'bg-white/80 border-emerald-100 dark:bg-emerald-900/30 dark:border-emerald-500/30' : 'bg-white/80 border-red-100 dark:bg-red-900/30 dark:border-red-500/30'">
           <div class="w-2 h-2 rounded-full relative">
              <div class="absolute inset-0 rounded-full animate-ping opacity-75" :class="isRunning ? 'bg-emerald-500' : 'bg-red-500'"></div>
              <div class="relative w-full h-full rounded-full" :class="isRunning ? 'bg-emerald-500' : 'bg-red-500'"></div>
           </div>
           <span class="text-[9px] font-black uppercase tracking-widest" :class="isRunning ? 'text-emerald-700 dark:text-emerald-300' : 'text-red-700 dark:text-red-300'">
             {{ isRunning ? 'Online' : 'Offline' }}
           </span>
        </div>
      </div>

      <!-- Main Body -->
      <div class="mt-auto pt-6 flex flex-col relative z-10">
        
        <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-orange-700/60 dark:text-zinc-500 mb-1 drop-shadow-sm">
          Connection Status
        </div>
        <div class="text-4xl font-black tracking-tighter text-orange-950 dark:text-white mb-6 drop-shadow-sm group-hover:-translate-y-1 transition-transform duration-500">
          Secure Tunnel
        </div>

        <!-- Nested Glass Container for details -->
        <div class="grid grid-cols-2 gap-3">
          <div class="p-3.5 rounded-[16px] bg-white/60 dark:bg-black/40 backdrop-blur-xl border border-white dark:border-white/5 flex flex-col gap-1.5 group-hover:bg-white/90 dark:group-hover:bg-white/[0.03] transition-all duration-300 shadow-[0_4px_12px_rgba(0,0,0,0.02)] dark:shadow-sm">
            <div class="flex items-center gap-2 text-orange-800/60 dark:text-zinc-400">
              <Cloud v-if="isRunning" class="w-3.5 h-3.5 text-emerald-500" />
              <CloudOff v-else class="w-3.5 h-3.5 text-red-500" />
              <span class="text-[9px] font-bold uppercase tracking-widest">{{ isRunning ? 'Uptime' : 'Status' }}</span>
            </div>
            <div class="text-sm font-black text-orange-950 dark:text-white tabular-nums tracking-tight">
               {{ isRunning ? formatUptime(uptimeMs) : 'Offline' }}
            </div>
          </div>
          
          <div class="p-3.5 rounded-[16px] bg-white/60 dark:bg-black/40 backdrop-blur-xl border border-white dark:border-white/5 flex flex-col gap-1.5 group-hover:bg-white/90 dark:group-hover:bg-white/[0.03] transition-all duration-300 shadow-[0_4px_12px_rgba(0,0,0,0.02)] dark:shadow-sm">
            <div class="flex items-center gap-2 text-orange-800/60 dark:text-zinc-400">
              <Globe class="w-3.5 h-3.5" />
              <span class="text-[9px] font-bold uppercase tracking-widest">Version</span>
            </div>
            <div class="text-sm font-black text-orange-950 dark:text-white font-mono truncate tracking-tight">
               {{ imageVersion }}
            </div>
          </div>
        </div>

      </div>
    </div>
  </div>
</template>
