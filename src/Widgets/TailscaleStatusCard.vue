<script setup>
import { ref, computed, onMounted, onUnmounted} from 'vue'
import { useI18n } from 'vue-i18n'
import { Wifi, WifiOff, Shield, ArrowRight, Key, CheckCircle, AlertCircle, Loader, Globe } from '@lucide/vue'
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

const tailscaleContainer = computed(() => {
  const list = Array.isArray(containers.value) ? containers.value : []
  const matches = list.filter((c) => {
    const name = (c?.name || '').toLowerCase()
    const names = Array.isArray(c?.Names) ? c.Names : []
    return name.includes('tailscale') || names.some((n) => (n || '').toLowerCase().includes('tailscale'))
  })
  if (!matches.length) return null
  return matches.find((c) => c?.state === 'running') || matches[0]
})

// --- Status ---
const isRunning = computed(() => tailscaleContainer.value?.state === 'running')

const uptimeMs = computed(() => {
  const c = tailscaleContainer.value
  if (!c || c.state !== 'running' || !c.created) return null
  const createdMs = Number(c.created) * 1000
  if (!Number.isFinite(createdMs)) return null
  return Math.max(0, currentTime.value - createdMs)
})

const imageVersion = computed(() => {
  const image = tailscaleContainer.value?.image || ''
  const tag = image.split(':')[1] || ''
  if (!tag || tag === 'latest') return 'latest'
  return tag.length > 12 ? tag.slice(0, 12) + '…' : tag
})

const containerName = computed(() => tailscaleContainer.value?.name || '—')

const exposedPorts = computed(() => {
  const ports = tailscaleContainer.value?.ports
  if (!Array.isArray(ports) || !ports.length) return null
  const pub = [...new Set(ports.filter(p => p.PublicPort).map(p => p.PublicPort))]
  return pub.length ? pub.slice(0, 3).join(', ') : null
})

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
const authKey = ref('')
const deploying = ref(false)
const deployError = ref('')
const deploySuccess = ref(false)

const isValidToken = computed(() => {
  const k = authKey.value.trim()
  return k.startsWith('tskey-') && k.length >= 30 && !/\s/.test(k)
})

const tokenState = computed(() => {
  const k = authKey.value.trim()
  if (!k) return 'empty'
  return isValidToken.value ? 'valid' : 'invalid'
})

async function deploy() {
  if (!isValidToken.value || deploying.value) return
  deploying.value = true
  deployError.value = ''
  try {
    const res = await fetch(`${apiUrl.value}/api/deploy`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        appId: 'tailscale',
        environment: { TAILSCALE_AUTH_KEY: authKey.value.trim() },
      }),
    })
    await expectApiSuccess(res, t('tailscaleSetupCard.deploymentFailed'))
    deploySuccess.value = true
    setTimeout(fetchContainers, 3000)
  } catch (e) {
    deployError.value = e.message || t('tailscaleSetupCard.networkError')
  } finally {
    deploying.value = false
  }
}
</script>

<template>
  <div class="relative group h-full flex flex-col rounded-[24px] overflow-hidden transition-all duration-500 hover:-translate-y-2 hover:shadow-[0_20px_40px_-15px_rgba(59,130,246,0.4)] dark:hover:shadow-[0_20px_40px_-15px_rgba(59,130,246,0.3)] text-left w-full min-h-[320px] border border-blue-100/50 dark:border-white/5 bg-gradient-to-br from-blue-50/50 to-indigo-50/50 dark:from-[#111827] dark:to-[#09090b]">
    
    <!-- Animated Gradient Glow -->
    <div class="absolute -top-[50%] -left-[50%] w-[200%] h-[200%] bg-[radial-gradient(circle_at_50%_50%,_rgba(59,130,246,0.15)_0%,_transparent_50%)] dark:bg-[radial-gradient(circle_at_50%_50%,_rgba(59,130,246,0.12)_0%,_transparent_40%)] opacity-0 group-hover:opacity-100 transition-opacity duration-1000 pointer-events-none group-hover:animate-[spin_20s_linear_infinite]"></div>

    <!-- Massive Watermark Icon -->
    <Shield class="absolute -bottom-8 -right-8 w-64 h-64 text-blue-500/10 dark:text-blue-400/[0.03] rotate-[-20deg] group-hover:rotate-0 group-hover:scale-110 transition-all duration-700 ease-out pointer-events-none" />

    <!-- Setup state: no tailscale container found -->
    <div v-if="!tailscaleContainer" class="relative z-10 p-7 flex flex-col h-full gap-4">
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
            <p class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight">{{ t('tailscaleSetupCard.tailscaleDeployed') }}</p>
            <p class="text-[11px] text-gray-500 dark:text-zinc-400 mt-1 uppercase tracking-widest font-medium">{{ t('tailscaleSetupCard.containerStarting') }}</p>
          </div>
        </div>
      </transition>

      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Gorgeous Icon Block -->
          <div class="relative w-14 h-14 rounded-2xl flex items-center justify-center shrink-0 bg-gradient-to-br from-blue-400 to-indigo-600 shadow-xl shadow-blue-500/30 group-hover:shadow-blue-500/50 transition-all duration-500 group-hover:scale-105">
            <div class="absolute inset-0 bg-white/30 dark:bg-white/20 rounded-2xl rounded-bl-none opacity-50 dark:opacity-40 mix-blend-overlay"></div>
            <Shield class="w-7 h-7 text-white drop-shadow-md z-10" />
          </div>
          <div>
            <h3 class="text-xl font-black text-blue-950 dark:text-white tracking-tight leading-none group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">Tailscale</h3>
            <div class="flex items-center gap-1.5 mt-2">
              <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-blue-600/70 dark:text-zinc-500">{{ t('tailscaleSetupCard.meshVpn') }}</span>
            </div>
          </div>
        </div>
        <div class="flex items-center gap-2 px-3 py-1.5 rounded-full border backdrop-blur-md shadow-sm transition-colors duration-500 bg-white/80 border-gray-200 dark:bg-zinc-800/80 dark:border-zinc-700">
           <div class="w-1.5 h-1.5 rounded-full bg-amber-500 animate-pulse relative"></div>
           <span class="text-[9px] font-black uppercase tracking-widest text-gray-500 dark:text-gray-400">{{ t('tailscaleSetupCard.notInstalled') }}</span>
        </div>
      </div>

      <!-- Main content -->
      <div class="mt-auto pt-6 flex flex-col relative z-10">
        <div class="text-4xl font-black tracking-tighter text-blue-950 dark:text-white transition-colors line-clamp-1 leading-[1.1] mb-2 drop-shadow-sm group-hover:-translate-y-1 duration-500">
          Private Access
        </div>
        <div class="text-xs text-blue-800/70 dark:text-zinc-400 font-medium mb-5">
           Join this host to your Tailscale network and reach services securely.
        </div>
        
        <div>
          <div class="relative">
            <input
              v-model="authKey"
              type="text"
              :placeholder="t('tailscaleSetupCard.authKeyPlaceholder')"
              autocomplete="off"
              spellcheck="false"
              class="w-full rounded-[16px] px-4 py-3 text-sm font-mono text-gray-900 dark:text-white placeholder-blue-300 dark:placeholder-zinc-600 outline-none transition-all duration-300 pr-10 bg-white/60 dark:bg-black/40 backdrop-blur-xl border border-white dark:border-white/5 shadow-[0_4px_12px_rgba(0,0,0,0.02)] dark:shadow-sm"
              :class="{
                'focus:border-blue-400 dark:focus:border-blue-500 focus:ring-1 focus:ring-blue-400/20': tokenState === 'empty',
                'border-emerald-300 dark:border-emerald-600 focus:border-emerald-400 focus:ring-1 focus:ring-emerald-400/20': tokenState === 'valid',
                'border-red-300 dark:border-red-700 focus:border-red-400 focus:ring-1 focus:ring-red-400/20': tokenState === 'invalid',
              }"
            />
            <div class="absolute right-3.5 top-1/2 -translate-y-1/2">
              <CheckCircle v-if="tokenState === 'valid'" class="w-4 h-4 text-emerald-500" />
              <AlertCircle v-else-if="tokenState === 'invalid'" class="w-4 h-4 text-red-400" />
            </div>
          </div>
          <div class="overflow-hidden">
            <transition
              enter-active-class="transition-all duration-200 ease-out"
              enter-from-class="opacity-0 -translate-y-2 max-h-0"
              enter-to-class="opacity-100 translate-y-0 max-h-6"
              leave-active-class="transition-all duration-150 ease-in"
              leave-from-class="opacity-100 translate-y-0 max-h-6"
              leave-to-class="opacity-0 -translate-y-2 max-h-0"
            >
              <p v-if="tokenState === 'invalid'" class="mt-1.5 text-[11px] text-red-500 dark:text-red-400 font-medium px-2">
                {{ t('tailscaleSetupCard.mustStartWith') }} <span class="font-mono">tskey-</span> {{ t('tailscaleSetupCard.andBeChars') }}
              </p>
              <p v-else-if="tokenState === 'valid'" class="mt-1.5 text-[11px] text-emerald-600 dark:text-emerald-400 font-medium px-2">
                {{ t('tailscaleSetupCard.keyLooksValid') }}
              </p>
            </transition>
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
            href="https://login.tailscale.com/admin/settings/keys"
            target="_blank"
            rel="noopener noreferrer"
            class="group/link flex h-11 items-center justify-center gap-2 rounded-[16px] text-blue-700/60 dark:text-zinc-400 bg-white/40 dark:bg-white/5 backdrop-blur-xl border border-white dark:border-white/5 hover:bg-white/80 dark:hover:bg-zinc-800 transition-all duration-300 hover:text-blue-900 dark:hover:text-white shadow-[0_4px_12px_rgba(0,0,0,0.02)] dark:shadow-none"
          >
            <Key class="w-3.5 h-3.5" />
            <span class="text-[11px] font-bold uppercase tracking-widest">{{ t('tailscaleSetupCard.getAuthKey') }}</span>
            <ArrowRight class="w-3.5 h-3.5 opacity-60 group-hover/link:translate-x-0.5 transition-transform duration-300" />
          </a>

          <button
            @click="deploy"
            :disabled="!isValidToken || deploying"
            class="group/cta relative flex h-11 w-full items-center justify-center gap-2 rounded-[16px] transition-all duration-300 overflow-hidden text-white shadow-md border border-transparent"
            :class="isValidToken && !deploying
              ? 'bg-gradient-to-r from-blue-500 to-indigo-600 hover:from-blue-400 hover:to-indigo-500 shadow-blue-500/30 hover:shadow-blue-500/50 active:scale-[0.98]'
              : 'bg-gray-300 dark:bg-zinc-700 text-gray-500 dark:text-zinc-500 shadow-none cursor-not-allowed opacity-50'"
          >
            <div class="flex items-center gap-2 relative z-10">
              <Loader v-if="deploying" class="w-4 h-4 animate-spin" />
              <Shield v-else class="w-4 h-4 drop-shadow-sm" />
              <span class="text-[11px] font-bold uppercase tracking-widest drop-shadow-sm">{{ deploying ? t('tailscaleSetupCard.deploying') : t('tailscaleSetupCard.deployTailscale') }}</span>
            </div>
            <div v-if="isValidToken && !deploying" class="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent -translate-x-full group-hover/cta:translate-x-full transition-transform duration-700 ease-out mix-blend-overlay"></div>
          </button>
        </div>
      </div>
    </div>

    <!-- Status state: tailscale container exists -->
    <div v-else class="relative z-10 p-7 flex flex-col h-full gap-4">
      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Gorgeous Icon Block -->
          <div class="relative w-14 h-14 rounded-2xl flex items-center justify-center shrink-0 bg-gradient-to-br from-blue-400 to-indigo-600 shadow-xl shadow-blue-500/30 group-hover:shadow-blue-500/50 transition-all duration-500 group-hover:scale-105">
            <div class="absolute inset-0 bg-white/30 dark:bg-white/20 rounded-2xl rounded-bl-none opacity-50 dark:opacity-40 mix-blend-overlay"></div>
            <Shield class="w-7 h-7 text-white drop-shadow-md z-10" />
          </div>
          <div>
            <h3 class="text-xl font-black text-blue-950 dark:text-white tracking-tight leading-none group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">Tailscale</h3>
            <p class="text-[10px] font-bold uppercase tracking-[0.2em] text-blue-600/70 dark:text-zinc-500 mt-2">{{ t('quickMetrics.tailscaleStatusCard.meshVpn') }}</p>
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
        
        <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-blue-700/60 dark:text-zinc-500 mb-1 drop-shadow-sm">
          Connection Status
        </div>
        <div class="text-4xl font-black tracking-tighter text-blue-950 dark:text-white mb-6 drop-shadow-sm group-hover:-translate-y-1 transition-transform duration-500">
          Private Access
        </div>

        <!-- Nested Glass Container for details -->
        <div class="grid grid-cols-2 gap-3">
          <div class="p-3.5 rounded-[16px] bg-white/60 dark:bg-black/40 backdrop-blur-xl border border-white dark:border-white/5 flex flex-col gap-1.5 group-hover:bg-white/90 dark:group-hover:bg-white/[0.03] transition-all duration-300 shadow-[0_4px_12px_rgba(0,0,0,0.02)] dark:shadow-sm">
            <div class="flex items-center gap-2 text-blue-800/60 dark:text-zinc-400">
              <Wifi v-if="isRunning" class="w-3.5 h-3.5 text-emerald-500" />
              <WifiOff v-else class="w-3.5 h-3.5 text-red-500" />
              <span class="text-[9px] font-bold uppercase tracking-widest">{{ isRunning ? t('quickMetrics.tailscaleStatusCard.uptime') : t('quickMetrics.tailscaleStatusCard.status') }}</span>
            </div>
            <div class="text-sm font-black text-blue-950 dark:text-white tabular-nums tracking-tight">
               {{ isRunning ? formatUptime(uptimeMs) : t('quickMetrics.tailscaleStatusCard.offline') }}
            </div>
          </div>
          
          <div class="p-3.5 rounded-[16px] bg-white/60 dark:bg-black/40 backdrop-blur-xl border border-white dark:border-white/5 flex flex-col gap-1.5 group-hover:bg-white/90 dark:group-hover:bg-white/[0.03] transition-all duration-300 shadow-[0_4px_12px_rgba(0,0,0,0.02)] dark:shadow-sm">
            <div class="flex items-center gap-2 text-blue-800/60 dark:text-zinc-400">
              <Globe class="w-3.5 h-3.5" />
              <span class="text-[9px] font-bold uppercase tracking-widest">{{ exposedPorts ? t('quickMetrics.tailscaleStatusCard.ports') : t('quickMetrics.tailscaleStatusCard.version') }}</span>
            </div>
            <div class="text-sm font-black text-blue-950 dark:text-white font-mono truncate tracking-tight">
               {{ exposedPorts ? exposedPorts : imageVersion }}
            </div>
          </div>
        </div>

      </div>
    </div>
  </div>
</template>
