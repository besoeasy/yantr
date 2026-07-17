<script setup>
import { ref, computed, onMounted, onUnmounted} from 'vue'
import { useI18n } from 'vue-i18n'
import { Wifi, WifiOff, Shield, ArrowRight, Key, CheckCircle, AlertCircle, Loader, Globe } from '@lucide/vue'
import { useApiUrl } from '../composables/useApiUrl'
import { expectApiSuccess } from '../composables/useApiResponse'
import { useCurrentTime } from '../composables/useCurrentTime'
import { formatUptime } from '../utils/metrics'

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
  <div class="group relative flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-all duration-300 hover:-translate-y-1 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">

    <!-- Setup state: no tailscale container found -->
    <div v-if="!tailscaleContainer" class="relative z-10 flex h-full flex-col gap-4">
      <transition
        enter-active-class="transition-all duration-300 ease-out"
        enter-from-class="opacity-0 scale-95"
        enter-to-class="opacity-100 scale-100"
      >
        <div v-if="deploySuccess" class="absolute inset-0 z-20 flex flex-col items-center justify-center gap-3 rounded-2xl bg-white/95 dark:bg-black/95">
          <div class="flex h-12 w-12 items-center justify-center rounded-full bg-emerald-50 dark:bg-emerald-900/20">
            <CheckCircle class="h-6 w-6 text-emerald-600 dark:text-emerald-500" />
          </div>
          <div class="text-center">
            <p class="text-sm font-bold tracking-tight text-zinc-900 dark:text-white">{{ t('tailscaleSetupCard.tailscaleDeployed') }}</p>
            <p class="mt-1 text-[11px] font-semibold uppercase tracking-widest text-zinc-500">{{ t('tailscaleSetupCard.containerStarting') }}</p>
          </div>
        </div>
      </transition>

      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Minimal Icon Block -->
          <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 transition-transform duration-300 group-hover:scale-105 dark:border-zinc-800 dark:bg-zinc-900">
            <Shield class="h-5 w-5 text-zinc-900 dark:text-zinc-100" />
          </div>
          <div>
            <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors group-hover:text-black dark:text-white dark:group-hover:text-white">Tailscale</h3>
            <div class="mt-2 flex items-center gap-1.5">
              <span class="text-[10px] font-semibold uppercase tracking-widest text-zinc-500">{{ t('tailscaleSetupCard.meshVpn') }}</span>
            </div>
          </div>
        </div>
        <div class="flex items-center gap-2 rounded-full border border-zinc-200 bg-zinc-50 px-3 py-1.5 dark:border-zinc-800 dark:bg-zinc-900/50">
           <div class="relative h-1.5 w-1.5 animate-pulse rounded-full bg-amber-500"></div>
           <span class="text-[9px] font-bold uppercase tracking-widest text-zinc-500">{{ t('tailscaleSetupCard.notInstalled') }}</span>
        </div>
      </div>

      <!-- Main content -->
      <div class="relative z-10 mt-auto flex flex-col pt-6">
        <div class="mb-2 line-clamp-1 text-4xl font-black leading-[1.1] tracking-tighter text-zinc-900 transition-transform duration-500 group-hover:-translate-y-1 dark:text-white">
          {{ t('tailscaleCard.privateAccess') }}
        </div>
        <div class="mb-5 text-xs font-medium text-zinc-500 dark:text-zinc-400">
           {{ t('tailscaleCard.description') }}
        </div>
        
        <div>
          <div class="relative">
            <input
              v-model="authKey"
              type="text"
              :placeholder="t('tailscaleSetupCard.authKeyPlaceholder')"
              autocomplete="off"
              spellcheck="false"
              class="w-full rounded-xl border border-zinc-200 bg-white px-4 py-3 pr-10 font-mono text-sm text-zinc-900 outline-none transition-all duration-300 placeholder:text-zinc-400 focus:border-zinc-400 focus:ring-1 focus:ring-zinc-400/20 dark:border-zinc-800 dark:bg-black dark:text-white dark:placeholder:text-zinc-600 dark:focus:border-zinc-600 dark:focus:ring-zinc-600/20"
              :class="{
                '': tokenState === 'empty',
                '!border-emerald-500 focus:!border-emerald-500 focus:!ring-emerald-500/20': tokenState === 'valid',
                '!border-red-500 focus:!border-red-500 focus:!ring-red-500/20': tokenState === 'invalid',
              }"
            />
            <div class="absolute right-3.5 top-1/2 -translate-y-1/2">
              <CheckCircle v-if="tokenState === 'valid'" class="h-4 w-4 text-emerald-500" />
              <AlertCircle v-else-if="tokenState === 'invalid'" class="h-4 w-4 text-red-500" />
            </div>
          </div>
          <div class="overflow-hidden">
            <transition
              enter-active-class="transition-all duration-200 ease-out"
              enter-from-class="-translate-y-2 max-h-0 opacity-0"
              enter-to-class="translate-y-0 max-h-6 opacity-100"
              leave-active-class="transition-all duration-150 ease-in"
              leave-from-class="translate-y-0 max-h-6 opacity-100"
              leave-to-class="-translate-y-2 max-h-0 opacity-0"
            >
              <p v-if="tokenState === 'invalid'" class="mt-1.5 px-2 text-[11px] font-medium text-red-500 dark:text-red-400">
                {{ t('tailscaleSetupCard.mustStartWith') }} <span class="font-mono">tskey-</span> {{ t('tailscaleSetupCard.andBeChars') }}
              </p>
              <p v-else-if="tokenState === 'valid'" class="mt-1.5 px-2 text-[11px] font-medium text-emerald-600 dark:text-emerald-400">
                {{ t('tailscaleSetupCard.keyLooksValid') }}
              </p>
            </transition>
          </div>
        </div>

        <transition
          enter-active-class="transition-all duration-300 ease-out"
          enter-from-class="translate-y-1 opacity-0"
          enter-to-class="translate-y-0 opacity-100"
          leave-active-class="transition-all duration-150 ease-in"
          leave-from-class="translate-y-0 opacity-100"
          leave-to-class="translate-y-1 opacity-0"
        >
          <div v-if="deployError" class="mt-3 flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 p-3 dark:border-red-900/50 dark:bg-red-900/20">
            <AlertCircle class="h-4 w-4 shrink-0 text-red-500" />
            <p class="text-xs font-medium text-red-600 dark:text-red-400">{{ deployError }}</p>
          </div>
        </transition>

        <div class="mt-4 grid grid-cols-1 gap-2.5 sm:grid-cols-2">
          <a
            href="https://login.tailscale.com/admin/settings/keys"
            target="_blank"
            rel="noopener noreferrer"
            class="group/link flex h-11 items-center justify-center gap-2 rounded-xl border border-zinc-200 bg-zinc-50 text-zinc-500 transition-all duration-300 hover:bg-zinc-100 hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900/50 dark:hover:bg-zinc-800 dark:hover:text-white"
          >
            <Key class="h-3.5 w-3.5" />
            <span class="text-[11px] font-bold uppercase tracking-widest">{{ t('tailscaleSetupCard.getAuthKey') }}</span>
            <ArrowRight class="h-3.5 w-3.5 opacity-60 transition-transform duration-300 group-hover/link:translate-x-0.5" />
          </a>

          <button
            @click="deploy"
            :disabled="!isValidToken || deploying"
            class="group/cta relative flex h-11 w-full items-center justify-center gap-2 overflow-hidden rounded-xl border transition-all duration-300"
            :class="isValidToken && !deploying
              ? 'border-zinc-900 bg-zinc-900 text-white hover:bg-black active:scale-[0.98] dark:border-zinc-100 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-white'
              : 'cursor-not-allowed border-zinc-200 bg-zinc-100 text-zinc-400 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-600'"
          >
            <div class="relative z-10 flex items-center gap-2">
              <Loader v-if="deploying" class="h-4 w-4 animate-spin" />
              <Shield v-else class="h-4 w-4" />
              <span class="text-[11px] font-bold uppercase tracking-widest">{{ deploying ? t('tailscaleSetupCard.deploying') : t('tailscaleSetupCard.deployTailscale') }}</span>
            </div>
          </button>
        </div>
      </div>
    </div>

    <!-- Status state: tailscale container exists -->
    <div v-else class="relative z-10 flex h-full flex-col gap-4">
      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Minimal Icon Block -->
          <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 transition-transform duration-300 group-hover:scale-105 dark:border-zinc-800 dark:bg-zinc-900">
            <Shield class="h-5 w-5 text-zinc-900 dark:text-zinc-100" />
          </div>
          <div>
            <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors group-hover:text-black dark:text-white dark:group-hover:text-white">Tailscale</h3>
            <p class="mt-2 text-[10px] font-semibold uppercase tracking-widest text-zinc-500">{{ t('quickMetrics.tailscaleStatusCard.meshVpn') }}</p>
          </div>
        </div>
        
        <!-- Status Pill -->
        <div class="flex items-center gap-2 rounded-full border px-3 py-1.5 transition-colors duration-300" :class="isRunning ? 'border-emerald-200 bg-emerald-50 dark:border-emerald-900/30 dark:bg-emerald-900/20' : 'border-red-200 bg-red-50 dark:border-red-900/30 dark:bg-red-900/20'">
           <div class="relative h-2 w-2 rounded-full">
              <div class="absolute inset-0 rounded-full animate-ping opacity-75" :class="isRunning ? 'bg-emerald-400' : 'bg-red-400'"></div>
              <div class="relative h-full w-full rounded-full" :class="isRunning ? 'bg-emerald-500' : 'bg-red-500'"></div>
           </div>
           <span class="text-[9px] font-bold uppercase tracking-widest" :class="isRunning ? 'text-emerald-700 dark:text-emerald-400' : 'text-red-700 dark:text-red-400'">
             {{ isRunning ? t('tailscaleCard.online') : t('tailscaleCard.offline') }}
           </span>
        </div>
      </div>

      <!-- Main Body -->
      <div class="relative z-10 mt-auto flex flex-col pt-6">
        
        <div class="mb-1 text-[10px] font-semibold uppercase tracking-widest text-zinc-500">
          {{ t('tailscaleCard.connectionStatus') }}
        </div>
        <div class="mb-6 text-4xl font-black tracking-tighter text-zinc-900 transition-transform duration-500 group-hover:-translate-y-1 dark:text-white">
          {{ t('tailscaleCard.privateAccess') }}
        </div>

        <!-- Nested Containers for details -->
        <div class="grid grid-cols-2 gap-2">
          <div class="flex flex-col gap-1.5 rounded-xl border border-zinc-200 bg-zinc-50 p-3.5 transition-colors duration-300 hover:bg-zinc-100 dark:border-zinc-800 dark:bg-zinc-900/50 dark:hover:bg-zinc-900">
            <div class="flex items-center gap-2 text-zinc-500">
              <Wifi v-if="isRunning" class="h-3.5 w-3.5 text-emerald-500" />
              <WifiOff v-else class="h-3.5 w-3.5 text-red-500" />
              <span class="text-[9px] font-semibold uppercase tracking-widest">{{ isRunning ? t('quickMetrics.tailscaleStatusCard.uptime') : t('quickMetrics.tailscaleStatusCard.status') }}</span>
            </div>
            <div class="tabular-nums text-sm font-black tracking-tight text-zinc-900 dark:text-white">
               {{ isRunning ? formatUptime(uptimeMs) : t('quickMetrics.tailscaleStatusCard.offline') }}
            </div>
          </div>
          
          <div class="flex flex-col gap-1.5 rounded-xl border border-zinc-200 bg-zinc-50 p-3.5 transition-colors duration-300 hover:bg-zinc-100 dark:border-zinc-800 dark:bg-zinc-900/50 dark:hover:bg-zinc-900">
            <div class="flex items-center gap-2 text-zinc-500">
              <Globe class="h-3.5 w-3.5" />
              <span class="text-[9px] font-semibold uppercase tracking-widest">{{ exposedPorts ? t('quickMetrics.tailscaleStatusCard.ports') : t('quickMetrics.tailscaleStatusCard.version') }}</span>
            </div>
            <div class="truncate font-mono text-sm font-black tracking-tight text-zinc-900 dark:text-white">
               {{ exposedPorts ? exposedPorts : imageVersion }}
            </div>
          </div>
        </div>

      </div>
    </div>
  </div>
</template>
