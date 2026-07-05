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
  <!-- Setup state: no cloudflared container found -->
  <div v-if="!cloudflaredContainer" class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] rounded-xl overflow-hidden transition-all duration-300 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:-translate-y-0.5 border border-gray-100 dark:border-zinc-800 text-left w-full min-h-72">
    <div class="absolute top-0 left-0 w-full h-0.5 bg-orange-500 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>

    <transition
      enter-active-class="transition-all duration-500 ease-out"
      enter-from-class="opacity-0 scale-95"
      enter-to-class="opacity-100 scale-100"
    >
      <div v-if="deploySuccess" class="absolute inset-0 z-20 flex flex-col items-center justify-center bg-white/95 dark:bg-[#0A0A0A]/95 rounded-xl gap-3">
        <div class="w-12 h-12 rounded-full bg-green-50 dark:bg-green-500/10 border border-green-200 dark:border-green-500/20 flex items-center justify-center">
          <CheckCircle class="w-6 h-6 text-green-600 dark:text-green-500" />
        </div>
        <div class="text-center">
          <p class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight">Tunnel Deployed</p>
          <p class="text-[11px] text-gray-500 dark:text-zinc-400 mt-1 uppercase tracking-widest font-medium">Container starting…</p>
        </div>
      </div>
    </transition>

    <div class="relative z-10 p-5 flex flex-col h-full gap-4">
      <!-- header -->
      <div class="flex items-center justify-between gap-3">
        <div class="flex items-center gap-3 min-w-0">
          <div class="w-9 h-9 rounded-lg bg-orange-50 dark:bg-orange-500/10 border border-orange-100 dark:border-orange-500/20 flex items-center justify-center shrink-0 transition-transform group-hover:scale-110">
            <Cloud class="w-4.5 h-4.5 text-orange-600 dark:text-orange-400 group-hover:text-orange-500 transition-colors" />
          </div>
          <div class="min-w-0">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight leading-none truncate">Cloudflared</h3>
            <div class="flex items-center gap-1.5 mt-1.5 text-orange-600 dark:text-orange-400">
              <span class="text-[10px] font-bold uppercase tracking-wider">Secure Tunnel</span>
            </div>
          </div>
        </div>
        <div class="flex items-center gap-1.5 shrink-0">
          <div class="w-1.5 h-1.5 rounded-full bg-amber-500 animate-pulse"></div>
          <span class="text-[10px] font-bold uppercase tracking-wider text-amber-600 dark:text-amber-500">Not Installed</span>
        </div>
      </div>

      <!-- Main content -->
      <div class="mt-auto pt-6 flex flex-col relative z-10">
        <div class="text-3xl sm:text-4xl font-black tracking-tighter text-gray-900 dark:text-white transition-colors line-clamp-1 leading-[1.1] mb-2">
          Secure Tunnel
        </div>
        <div class="text-xs text-gray-500 dark:text-zinc-400 font-medium mb-5">
           Connect your Yantr apps securely without exposing any ports.
        </div>
        
        <div>
          <label class="block text-[10px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500 mb-2">Tunnel Token</label>
          <div class="relative">
            <input
              v-model="tunnelToken"
              type="text"
              placeholder="eyJhIjoiY…"
              autocomplete="off"
              spellcheck="false"
              class="w-full bg-gray-50 dark:bg-zinc-900/50 border rounded-xl px-4 py-3 text-sm font-mono text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-zinc-600 outline-none transition-all duration-300 pr-10"
              :class="tunnelToken.trim() === ''
                ? 'border-gray-200 dark:border-zinc-800 focus:border-orange-400 dark:focus:border-orange-500 focus:ring-1 focus:ring-orange-400/20'
                : isValidToken
                  ? 'border-green-300 dark:border-green-600 focus:border-green-400 focus:ring-1 focus:ring-green-400/20'
                  : 'border-red-300 dark:border-red-700 focus:border-red-400 focus:ring-1 focus:ring-red-400/20'"
            />
            <div class="absolute right-3.5 top-1/2 -translate-y-1/2">
              <CheckCircle v-if="isValidToken" class="w-4 h-4 text-green-500" />
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
          <div v-if="deployError" class="mt-3 flex items-center gap-2 p-3 rounded-xl bg-red-50 dark:bg-red-500/10 border border-red-100 dark:border-red-500/20">
            <AlertCircle class="w-4 h-4 text-red-500 shrink-0" />
            <p class="text-xs text-red-600 dark:text-red-400 font-medium">{{ deployError }}</p>
          </div>
        </transition>

        <div class="mt-4 grid grid-cols-1 gap-2.5 sm:grid-cols-2">
          <a
            href="https://one.dash.cloudflare.com/"
            target="_blank"
            rel="noopener noreferrer"
            class="group/link flex h-11 items-center justify-center gap-2 rounded-xl bg-gray-50 dark:bg-zinc-900/70 text-gray-600 dark:text-zinc-400 hover:bg-gray-100 dark:hover:bg-zinc-800 transition-all duration-300 hover:text-gray-900 dark:hover:text-white border border-transparent dark:hover:border-zinc-700/50"
          >
            <Key class="w-3.5 h-3.5" />
            <span class="text-[11px] font-bold uppercase tracking-widest">Get Token</span>
            <ArrowRight class="w-3.5 h-3.5 opacity-60 group-hover/link:translate-x-0.5 transition-transform duration-300" />
          </a>

          <button
            @click="deploy"
            :disabled="!isValidToken || deploying"
            class="group/cta relative flex h-11 w-full items-center justify-center gap-2 rounded-xl transition-all duration-300 overflow-hidden text-white shadow-md"
            :class="isValidToken && !deploying
              ? 'bg-orange-500 hover:bg-orange-600 shadow-orange-500/20 active:scale-[0.98]'
              : 'bg-gray-300 dark:bg-zinc-700 text-gray-500 dark:text-zinc-500 shadow-none cursor-not-allowed'"
          >
            <div class="flex items-center gap-2 relative z-10">
              <Loader v-if="deploying" class="w-4 h-4 animate-spin" />
              <Cloud v-else class="w-4 h-4" />
              <span class="text-[11px] font-bold uppercase tracking-widest">{{ deploying ? 'Deploying…' : 'Deploy Tunnel' }}</span>
            </div>
            <div v-if="isValidToken && !deploying" class="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent -translate-x-full group-hover/cta:translate-x-full transition-transform duration-700 ease-out"></div>
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- Status state: cloudflared container exists -->
  <div v-else class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] rounded-xl overflow-hidden transition-all duration-300 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:-translate-y-0.5 border border-gray-100 dark:border-zinc-800 text-left w-full min-h-72">
    <div class="absolute top-0 left-0 w-full h-0.5 opacity-0 group-hover:opacity-100 transition-opacity duration-500" :class="isRunning ? 'bg-emerald-500' : 'bg-red-500'"></div>

    <div class="relative z-10 p-5 flex flex-col h-full gap-4">
      <!-- header -->
      <div class="flex items-center justify-between gap-3">
        <div class="flex items-center gap-3 min-w-0">
          <div class="w-9 h-9 rounded-lg bg-orange-50 dark:bg-orange-500/10 border border-orange-100 dark:border-orange-500/20 flex items-center justify-center shrink-0 transition-transform group-hover:scale-110">
            <Cloud class="w-4.5 h-4.5 text-orange-600 dark:text-orange-400 group-hover:text-orange-500 transition-colors" />
          </div>
          <div class="min-w-0">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight leading-none truncate">Cloudflared</h3>
            <div class="flex items-center gap-1.5 mt-1.5 text-orange-600 dark:text-orange-400">
              <span class="text-[10px] font-bold uppercase tracking-wider">Secure Tunnel</span>
            </div>
          </div>
        </div>
        <div class="flex items-center justify-center w-8 h-8 rounded-full bg-gray-50 dark:bg-zinc-900/50 opacity-0 group-hover:opacity-100 group-hover:translate-x-1 transition-all duration-300 shrink-0">
           <ArrowRight class="w-4 h-4 text-gray-400 dark:text-zinc-500 group-hover:text-orange-500" />
        </div>
      </div>

      <div class="mt-auto pt-6 flex flex-col relative z-10">
        <div class="text-3xl sm:text-4xl font-black tracking-tighter text-gray-900 dark:text-white transition-colors line-clamp-1 leading-[1.1] mb-4">
          Secure Tunnel
        </div>

        <div class="grid grid-cols-2 gap-2 mb-4">
          <div class="p-4 rounded-xl bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50 flex flex-col gap-1">
            <div class="flex items-center gap-1.5 text-gray-500 dark:text-zinc-400">
              <Cloud v-if="isRunning" class="w-3.5 h-3.5 text-emerald-500 shrink-0" />
              <CloudOff v-else class="w-3.5 h-3.5 text-red-500 shrink-0" />
              <span class="text-[10px] font-bold uppercase tracking-widest">{{ isRunning ? 'Uptime' : 'Status' }}</span>
            </div>
            <div class="text-sm font-bold text-gray-900 dark:text-white tabular-nums tracking-tight">
              {{ isRunning ? formatUptime(uptimeMs) : 'Offline' }}
            </div>
          </div>

          <div class="p-4 rounded-xl bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50 flex flex-col gap-1">
            <div class="flex items-center gap-1.5 text-gray-500 dark:text-zinc-400">
              <Globe class="w-3.5 h-3.5" />
              <span class="text-[10px] font-bold uppercase tracking-widest">Version</span>
            </div>
            <div class="text-sm font-bold text-gray-900 dark:text-white font-mono truncate tracking-tight">{{ imageVersion }}</div>
          </div>
        </div>

        <div class="flex items-center justify-between pt-4 border-t border-gray-100 dark:border-zinc-800/60">
           <div class="flex items-center gap-2">
             <Shield class="w-3.5 h-3.5" :class="isRunning ? 'text-emerald-500 dark:text-emerald-400' : 'text-red-500 dark:text-red-400'" />
             <span class="text-[10px] font-bold uppercase tracking-widest" :class="isRunning ? 'text-emerald-600 dark:text-emerald-500' : 'text-red-600 dark:text-red-500'">
               {{ isRunning ? 'Active — No Ports Exposed' : 'Tunnel Unavailable' }}
             </span>
           </div>
           <span class="text-[10px] text-gray-400 dark:text-zinc-500 font-mono truncate max-w-24 text-right">{{ containerName }}</span>
        </div>

      </div>
    </div>
  </div>
</template>
