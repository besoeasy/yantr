<script setup>
import { ref, computed, onMounted, onUnmounted} from 'vue'
import { useI18n } from 'vue-i18n'
import { Cloud, CloudOff, Shield, ArrowRight, Key, CheckCircle, AlertCircle, Loader, Globe, Wifi } from 'lucide-vue-next'
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
  <div v-if="!cloudflaredContainer" class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] rounded-xl overflow-hidden transition-all duration-300 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:-translate-y-0.5">
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

    <div class="relative z-10 p-6 flex flex-col h-full gap-5">
      <div class="flex items-center justify-between gap-4">
        <div class="flex items-center gap-3 min-w-0">
          <div class="w-10 h-10 rounded-lg bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center shrink-0 group-hover:scale-105 transition-all duration-500">
            <Cloud class="w-5 h-5 text-orange-500" />
          </div>
          <div class="min-w-0">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight">Cloudflared</h3>
            <div class="text-[11px] font-medium text-gray-500 dark:text-zinc-400 uppercase tracking-wider mt-1">Secure Tunnel</div>
          </div>
        </div>
        <div class="shrink-0 flex items-center gap-1.5">
          <div class="w-1.5 h-1.5 rounded-full bg-amber-500 animate-pulse"></div>
          <span class="text-[10px] font-bold uppercase tracking-wider text-amber-600 dark:text-amber-400">Not installed</span>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-2 sm:grid-cols-3">
        <div
          v-for="feature in features"
          :key="feature.label"
          class="flex items-center gap-2 rounded-lg border border-gray-100 bg-gray-50 px-3 py-2.5 text-gray-600 dark:border-zinc-800/50 dark:bg-zinc-900/50 dark:text-zinc-300"
        >
          <component :is="feature.icon" class="h-3.5 w-3.5 shrink-0 text-orange-500" />
          <span class="text-[11px] font-medium leading-tight">{{ feature.label }}</span>
        </div>
      </div>

      <div class="flex flex-col gap-3 flex-1">
        <div>
          <label class="block text-[10px] font-bold uppercase tracking-[0.2em] text-gray-400 dark:text-zinc-500 mb-2">Tunnel Token</label>
          <div class="relative">
            <input
              v-model="tunnelToken"
              type="text"
              placeholder="eyJhIjoiY…"
              autocomplete="off"
              spellcheck="false"
              class="w-full bg-gray-50 dark:bg-zinc-900 border rounded-lg px-3 py-2.5 text-xs font-mono text-gray-900 dark:text-white placeholder-gray-400 dark:placeholder-zinc-600 outline-none transition-all duration-200 pr-8"
              :class="tunnelToken.trim() === ''
                ? 'border-gray-200 dark:border-zinc-800 focus:border-orange-400 dark:focus:border-orange-500 focus:ring-1 focus:ring-orange-400/20'
                : isValidToken
                  ? 'border-green-300 dark:border-green-600 focus:border-green-400'
                  : 'border-red-300 dark:border-red-700 focus:border-red-400'"
            />
            <div class="absolute right-2.5 top-1/2 -translate-y-1/2">
              <CheckCircle v-if="isValidToken" class="w-3.5 h-3.5 text-green-500" />
              <AlertCircle v-else-if="tunnelToken.trim()" class="w-3.5 h-3.5 text-red-400" />
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
          <div v-if="deployError" class="flex items-center gap-2 p-2.5 rounded-lg bg-red-50 dark:bg-red-500/10 border border-red-100 dark:border-red-500/20">
            <AlertCircle class="w-3.5 h-3.5 text-red-500 shrink-0" />
            <p class="text-[11px] text-red-600 dark:text-red-400 font-medium">{{ deployError }}</p>
          </div>
        </transition>

        <div class="mt-auto grid grid-cols-1 gap-3 sm:grid-cols-2">
          <a
            href="https://one.dash.cloudflare.com/"
            target="_blank"
            rel="noopener noreferrer"
            class="flex items-center justify-center gap-2 px-4 py-3 rounded-lg border border-gray-200 dark:border-zinc-700 bg-gray-50 dark:bg-zinc-900/70 text-[11px] font-bold uppercase tracking-wider text-gray-700 dark:text-zinc-200 transition-all duration-200 hover:-translate-y-0.5 hover:border-orange-300 dark:hover:border-orange-500/40 hover:text-orange-600 dark:hover:text-orange-400"
          >
            <Key class="w-3.5 h-3.5" />
            <span>Get Token</span>
            <ArrowRight class="w-3.5 h-3.5" />
          </a>

          <button
            @click="deploy"
            :disabled="!isValidToken || deploying"
            class="w-full flex items-center justify-center gap-2 px-4 py-3 rounded-lg text-[11px] font-bold uppercase tracking-wider transition-all duration-200"
            :class="isValidToken && !deploying
              ? 'bg-gray-900 dark:bg-white text-white dark:text-gray-900 hover:-translate-y-0.5 hover:shadow-lg active:translate-y-0 cursor-pointer'
              : 'bg-gray-100 dark:bg-zinc-800 text-gray-400 dark:text-zinc-500 cursor-not-allowed'"
          >
            <Loader v-if="deploying" class="w-3.5 h-3.5 animate-spin" />
            <Cloud v-else class="w-3.5 h-3.5" />
            <span>{{ deploying ? 'Deploying…' : 'Deploy Tunnel' }}</span>
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- Status state: cloudflared container exists -->
  <div v-else class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] rounded-xl overflow-hidden transition-all duration-300 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40">
    <div
      class="absolute top-0 left-0 w-full h-0.5 opacity-0 group-hover:opacity-100 transition-opacity duration-500"
      :class="isRunning
        ? 'bg-orange-500'
        : 'bg-red-500'"
    ></div>

    <div class="relative z-10 p-6 flex flex-col h-full">
      <div class="flex items-center justify-between gap-4 mb-6">
        <div class="flex items-center gap-3 min-w-0">
          <div class="w-10 h-10 rounded-lg bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center shrink-0 group-hover:scale-105 transition-all duration-500">
            <Cloud class="w-5 h-5 text-orange-500" />
          </div>
          <div class="min-w-0">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight">Cloudflared</h3>
            <div class="text-[11px] font-medium text-gray-500 dark:text-zinc-400 uppercase tracking-wider mt-1">Secure Tunnel</div>
          </div>
        </div>
        <div class="flex items-center gap-1.5 shrink-0">
          <div class="w-1.5 h-1.5 rounded-full" :class="isRunning ? 'bg-green-500 animate-pulse' : 'bg-red-500'"></div>
          <span class="text-[10px] font-bold uppercase tracking-wider" :class="isRunning ? 'text-green-600 dark:text-green-500' : 'text-red-600 dark:text-red-400'">
            {{ isRunning ? 'Active' : 'Down' }}
          </span>
        </div>
      </div>

      <div class="space-y-3 mt-auto">
        <div class="flex items-center justify-between p-3 rounded-lg bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50">
          <div class="flex items-center gap-2 text-gray-500 dark:text-zinc-400">
            <Cloud v-if="isRunning" class="w-3.5 h-3.5 text-orange-500 shrink-0" />
            <CloudOff v-else class="w-3.5 h-3.5 text-red-400 shrink-0" />
            <span class="text-[10px] font-bold uppercase tracking-wider">{{ isRunning ? 'Uptime' : 'Status' }}</span>
          </div>
          <span class="text-sm font-semibold text-gray-900 dark:text-white tabular-nums">
            {{ isRunning ? formatUptime(uptimeMs) : 'Offline' }}
          </span>
        </div>

        <div class="grid grid-cols-2 gap-2">
          <div class="p-3 rounded-lg bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50">
            <div class="flex items-center gap-1.5 mb-2 text-gray-500 dark:text-zinc-400">
              <Shield class="w-3 h-3" />
              <span class="text-[9px] font-bold uppercase tracking-widest">Protocol</span>
            </div>
            <div class="text-sm font-semibold text-gray-800 dark:text-zinc-200 tracking-tight">HTTP/2</div>
          </div>

          <div class="p-3 rounded-lg bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50">
            <div class="flex items-center gap-1.5 mb-2 text-gray-500 dark:text-zinc-400">
              <Globe class="w-3 h-3" />
              <span class="text-[9px] font-bold uppercase tracking-widest">Version</span>
            </div>
            <div class="text-sm font-semibold text-gray-800 dark:text-zinc-200 tracking-tight font-mono truncate">{{ imageVersion }}</div>
          </div>
        </div>

        <div class="pt-3 border-t border-gray-100 dark:border-zinc-800/50 flex items-center justify-between gap-3">
          <span class="text-[11px] text-gray-400 dark:text-zinc-500 font-mono truncate">{{ containerName }}</span>
          <span class="text-[11px] font-medium text-gray-500 dark:text-zinc-400 truncate text-right">
            {{ isRunning ? 'Tunnel active — no ports exposed' : 'Tunnel unavailable' }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>
