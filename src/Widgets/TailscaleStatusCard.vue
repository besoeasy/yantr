<script setup>
import { ref, computed, onMounted, onUnmounted} from 'vue'
import { useI18n } from 'vue-i18n'
import { Wifi, WifiOff, Shield, ArrowRight, Key, CheckCircle, AlertCircle, Loader } from 'lucide-vue-next'
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
  <!-- Setup state: no tailscale container found -->
  <div v-if="!tailscaleContainer" class="relative group h-full flex flex-col rounded-xl bg-white dark:bg-[#0A0A0A] overflow-hidden transition-all duration-400 hover:-translate-y-0.5 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40">
    <div class="absolute top-0 left-0 h-0.5 w-full bg-blue-500 opacity-0 transition-opacity duration-500 group-hover:opacity-100"></div>

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
          <p class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight">{{ t('tailscaleSetupCard.tailscaleDeployed') }}</p>
          <p class="text-[11px] text-gray-500 dark:text-zinc-400 mt-1 uppercase tracking-widest font-medium">{{ t('tailscaleSetupCard.containerStarting') }}</p>
        </div>
      </div>
    </transition>

    <div class="relative z-10 flex h-full flex-col p-6">
      <div class="flex items-start justify-between gap-4">
        <div class="flex items-center gap-3 min-w-0">
          <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-gray-100 bg-gray-50 transition-all duration-500 group-hover:border-zinc-700 dark:border-zinc-800 dark:bg-zinc-900">
            <Shield class="h-5 w-5 text-gray-400 transition-colors group-hover:text-blue-500 dark:text-zinc-500" />
          </div>
          <div class="min-w-0">
            <h3 class="text-sm font-semibold tracking-tight text-gray-900 transition-colors group-hover:text-blue-600 dark:text-white dark:group-hover:text-blue-400">{{ t('tailscaleSetupCard.tailscale') }}</h3>
            <div class="text-[11px] font-medium text-gray-500 dark:text-zinc-400 uppercase tracking-wider mt-1">{{ t('tailscaleSetupCard.meshVpn') }}</div>
          </div>
        </div>
        <div class="shrink-0 flex items-center gap-1.5">
          <div class="w-1.5 h-1.5 rounded-full bg-amber-500 animate-pulse"></div>
          <span class="text-[10px] font-bold uppercase tracking-wider text-amber-600 dark:text-amber-400">{{ t('tailscaleSetupCard.notInstalled') }}</span>
        </div>
      </div>

      <div class="mt-6">
        <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-400 dark:text-zinc-500">{{ t('tailscaleSetupCard.meshVpn') }}</div>
        <div class="mt-2 text-3xl font-semibold tracking-tight text-gray-900 dark:text-white">Private access</div>
        <p class="mt-2 max-w-xl text-sm leading-relaxed text-gray-500 dark:text-zinc-400">
          Join this host to your Tailscale network with an auth key. Deploy once, then reach services securely without opening public ports.
        </p>
      </div>

      <div class="mt-auto flex flex-col gap-3 pt-5">
        <div>
          <label class="block text-[10px] font-bold uppercase tracking-[0.2em] text-gray-400 dark:text-zinc-500 mb-2">{{ t('tailscaleSetupCard.authKey') }}</label>
          <div class="relative">
            <input
              v-model="authKey"
              type="text"
              :placeholder="t('tailscaleSetupCard.authKeyPlaceholder')"
              autocomplete="off"
              spellcheck="false"
              class="w-full rounded-lg border bg-gray-50 px-3 py-2.5 pr-8 text-xs font-mono text-gray-900 outline-none transition-all duration-200 placeholder-gray-400 dark:border-zinc-800 dark:bg-zinc-900 dark:text-white dark:placeholder-zinc-600"
              :class="{
                'border-gray-200 focus:border-blue-400 focus:ring-1 focus:ring-blue-400/20 dark:focus:border-blue-500': tokenState === 'empty',
                'border-green-300 dark:border-green-600 focus:border-green-400': tokenState === 'valid',
                'border-red-300 dark:border-red-700 focus:border-red-400': tokenState === 'invalid',
              }"
            />
            <div class="absolute right-2.5 top-1/2 -translate-y-1/2 transition-all duration-200">
              <CheckCircle v-if="tokenState === 'valid'" class="w-3.5 h-3.5 text-green-500" />
              <AlertCircle v-else-if="tokenState === 'invalid'" class="w-3.5 h-3.5 text-red-400" />
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
              <p v-if="tokenState === 'invalid'" class="mt-1.5 text-[11px] text-red-500 dark:text-red-400 font-medium">
                {{ t('tailscaleSetupCard.mustStartWith') }} <span class="font-mono">tskey-</span> {{ t('tailscaleSetupCard.andBeChars') }}
              </p>
              <p v-else-if="tokenState === 'valid'" class="mt-1.5 text-[11px] text-green-600 dark:text-green-400 font-medium">
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
          <div v-if="deployError" class="flex items-center gap-2 p-2.5 rounded-lg bg-red-50 dark:bg-red-500/10 border border-red-100 dark:border-red-500/20">
            <AlertCircle class="w-3.5 h-3.5 text-red-500 shrink-0" />
            <p class="text-[11px] text-red-600 dark:text-red-400 font-medium">{{ deployError }}</p>
          </div>
        </transition>

        <div class="mt-auto grid grid-cols-1 gap-3 sm:grid-cols-2">
          <a
            href="https://login.tailscale.com/admin/settings/keys"
            target="_blank"
            rel="noopener noreferrer"
            class="flex items-center justify-center gap-2 rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 text-[11px] font-bold uppercase tracking-wider text-gray-700 transition-all duration-200 hover:-translate-y-0.5 hover:border-blue-300 hover:text-blue-600 dark:border-zinc-700 dark:bg-zinc-900/70 dark:text-zinc-200 dark:hover:border-blue-500/40 dark:hover:text-blue-400"
          >
            <Key class="w-3.5 h-3.5" />
            <span>{{ t('tailscaleSetupCard.getAuthKey') }}</span>
            <ArrowRight class="w-3.5 h-3.5" />
          </a>

          <button
            @click="deploy"
            :disabled="!isValidToken || deploying"
            class="group/deploy w-full flex items-center justify-center gap-2 rounded-lg px-4 py-3 text-[11px] font-bold uppercase tracking-wider transition-all duration-200"
            :class="isValidToken && !deploying
              ? 'bg-gray-900 dark:bg-white text-white dark:text-gray-900 hover:-translate-y-0.5 hover:shadow-lg active:translate-y-0 cursor-pointer'
              : 'bg-gray-100 dark:bg-zinc-800 text-gray-400 dark:text-zinc-500 cursor-not-allowed'"
          >
            <Loader v-if="deploying" class="w-3.5 h-3.5 animate-spin" />
            <Shield v-else class="w-3.5 h-3.5 transition-transform duration-200" :class="isValidToken ? 'group-hover:scale-110' : ''" />
            <span>{{ deploying ? t('tailscaleSetupCard.deploying') : t('tailscaleSetupCard.deployTailscale') }}</span>
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- Status state: tailscale container exists -->
  <div v-else class="relative group h-full flex flex-col rounded-xl bg-white dark:bg-[#0A0A0A] overflow-hidden transition-all duration-400 hover:-translate-y-0.5 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40">
    <div
      class="absolute top-0 left-0 h-0.5 w-full opacity-0 transition-opacity duration-500 group-hover:opacity-100"
      :class="isRunning
        ? 'bg-green-500'
        : 'bg-red-500'"
    ></div>

    <div class="relative z-10 flex h-full flex-col p-6">
      <div class="flex items-start justify-between gap-4">
        <div class="flex items-center gap-3 min-w-0">
          <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-gray-100 bg-gray-50 transition-all duration-500 group-hover:border-zinc-700 dark:border-zinc-800 dark:bg-zinc-900">
            <Shield class="h-5 w-5 text-gray-400 transition-colors group-hover:text-blue-500 dark:text-zinc-500" />
          </div>
          <div class="min-w-0">
            <h3 class="truncate text-sm font-semibold tracking-tight text-gray-900 transition-colors group-hover:text-blue-600 dark:text-white dark:group-hover:text-blue-400">Tailscale</h3>
            <div class="text-[11px] font-medium text-gray-500 dark:text-zinc-400 uppercase tracking-wider mt-1">{{ t('quickMetrics.tailscaleStatusCard.meshVpn') }}</div>
          </div>
        </div>
        <div class="flex items-center gap-1.5 shrink-0">
          <div
            class="w-1.5 h-1.5 rounded-full"
            :class="isRunning ? 'bg-green-500 animate-pulse' : 'bg-red-500'"
          ></div>
          <span
            class="text-[10px] font-bold uppercase tracking-wider"
            :class="isRunning ? 'text-green-600 dark:text-green-500' : 'text-red-600 dark:text-red-400'"
          >
            {{ isRunning ? t('quickMetrics.tailscaleStatusCard.active') : t('quickMetrics.tailscaleStatusCard.down') }}
          </span>
        </div>
      </div>

      <div class="mt-6">
        <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-400 dark:text-zinc-500">
          {{ isRunning ? t('quickMetrics.tailscaleStatusCard.uptime') : t('quickMetrics.tailscaleStatusCard.status') }}
        </div>
        <div class="mt-2 flex items-end justify-between gap-3">
          <div class="flex items-end gap-3">
            <Wifi v-if="isRunning" class="h-5 w-5 shrink-0 text-green-500" />
            <WifiOff v-else class="h-5 w-5 shrink-0 text-red-400" />
            <div class="text-3xl font-semibold tracking-tight text-gray-900 dark:text-white tabular-nums">
              {{ isRunning ? formatUptime(uptimeMs) : t('quickMetrics.tailscaleStatusCard.offline') }}
            </div>
          </div>
          <div class="text-right text-[11px] font-medium text-gray-500 dark:text-zinc-400">
            {{ isRunning ? t('quickMetrics.tailscaleStatusCard.wireGuardMeshActive') : t('quickMetrics.tailscaleStatusCard.remoteAccessUnavailable') }}
          </div>
        </div>
      </div>

      <div class="mt-auto grid grid-cols-1 gap-3 pt-5 border-t border-gray-100 dark:border-zinc-800/80 sm:grid-cols-2">
        <div class="flex items-center gap-3 min-w-0">
          <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-gray-100 bg-gray-50 dark:border-zinc-800/50 dark:bg-zinc-900/50">
            <Lock class="h-3.5 w-3.5 text-gray-400 dark:text-zinc-500" />
          </div>
          <div class="min-w-0 flex-1">
            <div class="text-[9px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500">{{ t('quickMetrics.tailscaleStatusCard.encryption') }}</div>
            <div class="mt-1 truncate text-sm font-semibold tracking-tight text-gray-800 dark:text-zinc-200">{{ t('quickMetrics.tailscaleStatusCard.wireGuard') }}</div>
          </div>
        </div>

        <div class="flex items-center gap-3 min-w-0">
          <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-gray-100 bg-gray-50 dark:border-zinc-800/50 dark:bg-zinc-900/50">
            <Globe class="h-3.5 w-3.5 text-gray-400 dark:text-zinc-500" />
          </div>
          <div class="min-w-0 flex-1">
            <div class="text-[9px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500">{{ exposedPorts ? t('quickMetrics.tailscaleStatusCard.ports') : t('quickMetrics.tailscaleStatusCard.version') }}</div>
            <div class="mt-1 truncate font-mono text-sm font-semibold tracking-tight text-gray-800 dark:text-zinc-200">{{ exposedPorts ? exposedPorts : imageVersion }}</div>
          </div>
        </div>
      </div>

      <div class="mt-4 flex items-center justify-between gap-3 border-t border-gray-100 pt-4 dark:border-zinc-800/50">
        <span class="truncate font-mono text-[11px] text-gray-400 dark:text-zinc-500">{{ containerName }}</span>
        <span class="truncate text-right text-[11px] font-medium text-gray-500 dark:text-zinc-400">Container identity</span>
      </div>
    </div>
  </div>
</template>
