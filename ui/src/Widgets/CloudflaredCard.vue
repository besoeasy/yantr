<script>
// Dashboard engine: two of the desktop grid's four columns — the tunnel
// status keeps its reading on the left while the token form earns a column
// rather than a squeezed stack. See Home.vue for how meta.cols is consumed.
export const meta = { cols: 2 };
</script>
<script setup>
import { ref, computed } from "vue";
import { useI18n } from 'vue-i18n'
import { Cloud, CloudOff, ArrowRight, Key, CheckCircle, AlertCircle, Loader, Globe } from '@lucide/vue'
import { useApiUrl } from '../composables/useApiUrl'
import { expectApiSuccess } from '../composables/useApiResponse'
import { useCurrentTime } from '../composables/useCurrentTime'
import { formatUptime } from '../utils/metrics'
import { useContainers } from '../composables/useDashboardData'

const { t } = useI18n()
const { apiUrl } = useApiUrl()
const { currentTime } = useCurrentTime()

// --- Container list ---
// Shared with the other dashboard cards that read it: one request at mount,
// one ticker for the whole page, and one agreed answer about failure.
const { containers, refreshContainers } = useContainers({ live: true })

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


// --- Setup / Deploy ---
const tunnelToken = ref('')
const deploying = ref(false)
const deployError = ref('')
const deploySuccess = ref(false)

// Cloudflare tunnel tokens are base64url-encoded JSON:
// {"a":"<account-id>","t":"<tunnel-id>","s":"<secret>"}
function parseTunnelToken(raw) {
  const trimmed = (raw || '').trim()
  if (!trimmed || /\s/.test(trimmed)) return null
  try {
    const padded = trimmed + '='.repeat((4 - (trimmed.length % 4)) % 4)
    const json = atob(padded.replace(/-/g, '+').replace(/_/g, '/'))
    const payload = JSON.parse(json)
    if (!payload || typeof payload !== 'object') return null
    const { a, t, s } = payload
    if (typeof a !== 'string' || !a || typeof t !== 'string' || !t || typeof s !== 'string' || !s) return null
    return { accountId: a, tunnelId: t }
  } catch {
    return null
  }
}

const parsedToken = computed(() => parseTunnelToken(tunnelToken.value))
const isValidToken = computed(() => parsedToken.value !== null)

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
    setTimeout(refreshContainers, 3000)
  } catch (e) {
    deployError.value = e.message || 'Network error'
  } finally {
    deploying.value = false
  }
}
</script>

<template>
  <div class="group relative flex h-full w-full yantra-card-min flex-col overflow-hidden rounded-2xl p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">

    <!-- Setup state: no cloudflared container found -->
    <div v-if="!cloudflaredContainer" class="relative z-10 flex h-full flex-col gap-4">
      <transition
        enter-active-class="transition-[border-color,box-shadow,transform] duration-200 ease-out"
        enter-from-class="opacity-0 scale-95"
        enter-to-class="opacity-100 scale-100"
      >
        <div v-if="deploySuccess" class="absolute inset-0 z-20 flex flex-col items-center justify-center gap-3 rounded-2xl /95 /95">
          <div class="flex h-12 w-12 items-center justify-center rounded-full bg-emerald-50 dark:bg-emerald-900/20">
            <CheckCircle class="h-6 w-6 text-emerald-600 dark:text-emerald-500" />
          </div>
          <div class="text-center">
            <p class="text-sm font-bold tracking-tight yantra-text-primary">{{ t('cloudflaredCard.tunnelDeployed') }}</p>
            <p class="mt-1 text-[11px] font-semibold uppercase tracking-widest yantra-text-secondary">{{ t('cloudflaredCard.containerStarting') }}</p>
          </div>
        </div>
      </transition>

      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Minimal Icon Block -->
          <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-violet-200 bg-violet-50 transition-transform duration-200 group-hover:scale-105 dark:border-violet-900/50 dark:bg-violet-900/20">
            <Cloud class="h-5 w-5 text-violet-600 dark:text-violet-400" />
          </div>
          <div>
            <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">{{ t('cloudflaredCard.title') }}</h3>
            <div class="mt-2 flex items-center gap-1.5">
              <span class="text-[10px] font-semibold uppercase tracking-widest text-violet-600 dark:text-violet-500">{{ t('cloudflaredCard.secureTunnel') }}</span>
            </div>
          </div>
        </div>
        <div class="flex items-center gap-2 rounded-full bg-zinc-50 px-3 py-1.5 dark:border-zinc-800 dark:bg-zinc-900/50">
           <div class="relative h-1.5 w-1.5 animate-pulse rounded-full bg-amber-500"></div>
           <span class="text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">{{ t('cloudflaredCard.notInstalled') }}</span>
        </div>
      </div>

      <!-- Half-width content: context and token entry get their own columns. -->
      <div class="relative z-10 mt-auto grid gap-5 pt-6 sm:grid-cols-[minmax(0,1fr)_minmax(0,15rem)] sm:items-start">
        <div>
          <div class="mb-2 text-4xl font-black leading-[1.1] tracking-tighter yantra-text-primary transition-transform duration-200 group-hover:-translate-y-[3px] yantra-text-primary">
            {{ t('cloudflaredCard.secureTunnel') }}
          </div>
          <div class="text-xs font-medium leading-relaxed yantra-text-secondary">
            {{ t('cloudflaredCard.description') }}
          </div>

          <a
            href="https://one.dash.cloudflare.com/"
            target="_blank"
            rel="noopener noreferrer"
            class="group/link mt-4 flex h-10 w-full items-center justify-center gap-2 rounded-xl bg-zinc-50 yantra-text-secondary transition-[border-color,box-shadow,transform] duration-200 hover:bg-zinc-100 hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900/50 dark:hover:bg-zinc-800 dark:hover:text-white"
          >
            <Key class="h-3.5 w-3.5" />
            <span class="text-[11px] font-bold uppercase tracking-widest">{{ t('cloudflaredCard.getToken') }}</span>
            <ArrowRight class="h-3.5 w-3.5 opacity-60 transition-transform duration-200 group-hover/link:translate-x-0.5" />
          </a>
        </div>

        <div>
          <div class="relative">
            <input
              v-model="tunnelToken"
              type="text"
              :aria-label="t('cloudflaredCard.tunnelToken')"
              placeholder="eyJhIjoiY…"
              autocomplete="off"
              spellcheck="false"
              class="w-full rounded-xl px-4 py-3 pr-10 font-mono text-sm yantra-text-primary outline-none transition-[border-color,box-shadow,transform] duration-200 placeholder:text-zinc-400 focus:border-zinc-400 focus:ring-1 focus:ring-zinc-400/20 dark:border-zinc-800 yantra-text-primary dark:placeholder:text-zinc-600 dark:focus:border-zinc-600 dark:focus:ring-zinc-600/20"
              :class="tunnelToken.trim() === ''
                ? ''
                : isValidToken
                  ? '!border-emerald-500 focus:!border-emerald-500 focus:!ring-emerald-500/20'
                  : '!border-red-500 focus:!border-red-500 focus:!ring-red-500/20'"
            />
            <div class="absolute right-3.5 top-1/2 -translate-y-1/2">
              <CheckCircle v-if="isValidToken" class="h-4 w-4 text-emerald-500" />
              <AlertCircle v-else-if="tunnelToken.trim()" class="h-4 w-4 text-red-500" />
            </div>
          </div>
          <div v-if="parsedToken" class="mt-2 space-y-1 font-mono text-[11px] yantra-text-secondary">
            <div class="truncate" :title="parsedToken.accountId">Account: {{ parsedToken.accountId }}</div>
            <div class="truncate" :title="parsedToken.tunnelId">Tunnel: {{ parsedToken.tunnelId }}</div>
          </div>
          <p v-else-if="tunnelToken.trim()" class="mt-2 text-[11px] font-medium text-red-500">
            {{ t('cloudflaredCard.invalidToken') }}
          </p>

          <transition
            enter-active-class="transition-[border-color,box-shadow,transform] duration-200 ease-out"
            enter-from-class="translate-y-1 opacity-0"
            enter-to-class="translate-y-0 opacity-100"
            leave-active-class="transition-all duration-150 ease-in"
            leave-from-class="translate-y-0 opacity-100"
            leave-to-class="translate-y-1 opacity-0"
          >
            <div v-if="deployError" class="mt-3 flex items-center gap-2 rounded-lg border-red-200 bg-red-50 p-3 dark:border-red-900/50 dark:bg-red-900/20">
              <AlertCircle class="h-4 w-4 shrink-0 text-red-500" />
              <p class="text-xs font-medium text-red-600 dark:text-red-400">{{ deployError }}</p>
            </div>
          </transition>

          <button
            @click="deploy"
            :disabled="!isValidToken || deploying"
            class="group/cta relative mt-3 flex h-11 w-full items-center justify-center gap-2 overflow-hidden rounded-xl transition-[border-color,box-shadow,transform] duration-200"
            :class="isValidToken && !deploying
              ? 'border-zinc-900 bg-zinc-900 text-white hover:bg-black active:scale-[0.98] dark:border-zinc-100 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-white'
              : 'cursor-not-allowed border-zinc-200 bg-zinc-100 yantra-text-secondary dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-600'"
          >
            <div class="relative z-10 flex items-center gap-2">
              <Loader v-if="deploying" class="h-4 w-4 animate-spin" />
              <Cloud v-else class="h-4 w-4" />
              <span class="text-[11px] font-bold uppercase tracking-widest">{{ deploying ? t('cloudflaredCard.deploying') : t('cloudflaredCard.deployTunnel') }}</span>
            </div>
          </button>
        </div>
      </div>
    </div>

    <!-- Status state: cloudflared container exists -->
    <div v-else class="relative z-10 flex h-full flex-col gap-4">
      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Minimal Icon Block -->
          <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-violet-200 bg-violet-50 transition-transform duration-200 group-hover:scale-105 dark:border-violet-900/50 dark:bg-violet-900/20">
            <Cloud class="h-5 w-5 text-violet-600 dark:text-violet-400" />
          </div>
          <div>
            <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">{{ t('cloudflaredCard.title') }}</h3>
            <p class="mt-2 text-[10px] font-semibold uppercase tracking-widest text-violet-600 dark:text-violet-500">{{ t('cloudflaredCard.secureTunnel') }}</p>
          </div>
        </div>
        
        <!-- Status Pill -->
        <div class="flex items-center gap-2 rounded-full px-3 py-1.5 transition-colors duration-200" :class="isRunning ? 'border-emerald-200 bg-emerald-50 dark:border-emerald-900/30 dark:bg-emerald-900/20' : 'border-red-200 bg-red-50 dark:border-red-900/30 dark:bg-red-900/20'">
           <div class="relative h-2 w-2 rounded-full">
              <div class="absolute inset-0 rounded-full animate-ping opacity-75" :class="isRunning ? 'bg-emerald-400' : 'bg-red-400'"></div>
              <div class="relative h-full w-full rounded-full" :class="isRunning ? 'bg-emerald-500' : 'bg-red-500'"></div>
           </div>
           <span class="text-[9px] font-bold uppercase tracking-widest" :class="isRunning ? 'text-emerald-700 dark:text-emerald-400' : 'text-red-700 dark:text-red-400'">
             {{ isRunning ? t('cloudflaredCard.online') : t('cloudflaredCard.offline') }}
           </span>
        </div>
      </div>

      <!-- Half-width body: the reading stays left, the ops facts stack right. -->
      <div class="relative z-10 mt-auto grid gap-5 pt-6 sm:grid-cols-[minmax(0,1fr)_minmax(0,15rem)] sm:items-start">
        <div>
          <div class="mb-1 text-[10px] font-semibold uppercase tracking-widest yantra-text-secondary">
            {{ t('cloudflaredCard.connectionStatus') }}
          </div>
          <div class="text-4xl font-black tracking-tighter yantra-text-primary transition-transform duration-200 group-hover:-translate-y-[3px] yantra-text-primary">
            {{ t('cloudflaredCard.secureTunnel') }}
          </div>
        </div>

        <!-- Ops facts: uptime and image version share one column, stacked. -->
        <div class="flex flex-col gap-2">
          <div class="flex flex-col gap-1.5 rounded-xl bg-zinc-50 p-3.5 transition-colors duration-200 hover:bg-zinc-100 dark:border-zinc-800 dark:bg-zinc-900/50 dark:hover:bg-zinc-900">
            <div class="flex items-center gap-2 yantra-text-secondary">
              <Cloud v-if="isRunning" class="h-3.5 w-3.5 text-emerald-500" />
              <CloudOff v-else class="h-3.5 w-3.5 text-red-500" />
              <span class="text-[9px] font-semibold uppercase tracking-widest">{{ isRunning ? t('cloudflaredCard.uptime') : t('cloudflaredCard.status') }}</span>
            </div>
            <div class="tabular-nums text-sm font-black tracking-tight yantra-text-primary">
               {{ isRunning ? formatUptime(uptimeMs) : t('cloudflaredCard.offline') }}
            </div>
          </div>

          <div class="flex flex-col gap-1.5 rounded-xl bg-zinc-50 p-3.5 transition-colors duration-200 hover:bg-zinc-100 dark:border-zinc-800 dark:bg-zinc-900/50 dark:hover:bg-zinc-900">
            <div class="flex items-center gap-2 yantra-text-secondary">
              <Globe class="h-3.5 w-3.5" />
              <span class="text-[9px] font-semibold uppercase tracking-widest">{{ t('cloudflaredCard.version') }}</span>
            </div>
            <div class="truncate font-mono text-sm font-black tracking-tight yantra-text-primary">
               {{ imageVersion }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
