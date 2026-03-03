<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Wifi, WifiOff, Shield, Lock, Globe, Clock } from 'lucide-vue-next'

const { t } = useI18n()
const props = defineProps({
  containers: { type: Array, default: () => [] },
  currentTime: { type: Number, default: () => Date.now() },
})

const tailscaleContainer = computed(() => {
  const list = Array.isArray(props.containers) ? props.containers : []
  const matches = list.filter((c) => {
    const name = (c?.name || '').toLowerCase()
    const names = Array.isArray(c?.Names) ? c.Names : []
    return name.includes('tailscale') || names.some((n) => (n || '').toLowerCase().includes('tailscale'))
  })
  if (!matches.length) return null
  return matches.find((c) => c?.state === 'running') || matches[0]
})

const isRunning = computed(() => tailscaleContainer.value?.state === 'running')

const uptimeMs = computed(() => {
  const c = tailscaleContainer.value
  if (!c || c.state !== 'running' || !c.created) return null
  const createdMs = Number(c.created) * 1000
  if (!Number.isFinite(createdMs)) return null
  return Math.max(0, props.currentTime - createdMs)
})

const imageVersion = computed(() => {
  const image = tailscaleContainer.value?.image || ''
  // Extract tag portion after ':'
  const tag = image.split(':')[1] || ''
  if (!tag || tag === 'latest') return 'latest'
  // Trim long sha-style tags
  return tag.length > 12 ? tag.slice(0, 12) + '…' : tag
})

const containerName = computed(() => {
  const c = tailscaleContainer.value
  if (!c) return '—'
  return c.name || '—'
})

const exposedPorts = computed(() => {
  const ports = tailscaleContainer.value?.ports
  if (!Array.isArray(ports) || !ports.length) return null
  // Collect unique public-facing ports
  const pub = [...new Set(
    ports.filter(p => p.PublicPort).map(p => p.PublicPort)
  )]
  if (!pub.length) return null
  return pub.slice(0, 3).join(', ')
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
</script>

<template>
  <div class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl overflow-hidden transition-all duration-300 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:border-gray-300 dark:hover:border-zinc-600">

    <!-- Hover accent line — green when running, red when stopped -->
    <div
      class="absolute top-0 left-0 w-full h-[2px] opacity-0 group-hover:opacity-100 transition-opacity duration-500"
      :class="isRunning
        ? 'bg-gradient-to-r from-transparent via-green-500 to-transparent'
        : 'bg-gradient-to-r from-transparent via-red-500 to-transparent'"
    ></div>

    <!-- Dot-grid pattern -->
    <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none [mask-image:linear-gradient(to_bottom,white,transparent)]"></div>

    <div class="relative z-10 p-6 flex flex-col h-full gap-5">

      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="min-w-0 pr-2">
          <div class="flex items-center gap-2 mb-1">
            <Shield class="w-3.5 h-3.5 text-violet-500" />
            <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-400 dark:text-zinc-500">{{ t('tailscaleStatusCard.meshVpn') }}</span>
          </div>
          <div
            class="text-base font-semibold tracking-tight truncate transition-colors"
            :class="isRunning
              ? 'text-gray-900 dark:text-white group-hover:text-green-600 dark:group-hover:text-green-400'
              : 'text-gray-900 dark:text-white group-hover:text-red-500 dark:group-hover:text-red-400'"
            :title="containerName"
          >
            Tailscale
          </div>
        </div>

        <!-- Live status badge -->
        <div class="flex items-center gap-1.5 shrink-0 mt-0.5">
          <div
            class="w-1.5 h-1.5 rounded-full"
            :class="isRunning ? 'bg-green-500 animate-pulse' : 'bg-red-500'"
          ></div>
          <span
            class="text-[10px] font-bold uppercase tracking-wider"
            :class="isRunning ? 'text-green-600 dark:text-green-500' : 'text-red-600 dark:text-red-400'"
          >
            {{ isRunning ? t('tailscaleStatusCard.active') : t('tailscaleStatusCard.down') }}
          </span>
        </div>
      </div>

      <!-- Primary metric — uptime -->
      <div>
        <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-400 dark:text-zinc-500 mb-1.5">
          {{ isRunning ? t('tailscaleStatusCard.uptime') : t('tailscaleStatusCard.status') }}
        </div>
        <div class="text-4xl font-bold tabular-nums tracking-tighter text-gray-900 dark:text-white leading-none">
          {{ isRunning ? formatUptime(uptimeMs) : t('tailscaleStatusCard.offline') }}
        </div>
        <div class="mt-2 text-[11px] font-medium text-gray-500 dark:text-zinc-400">
          {{ isRunning ? t('tailscaleStatusCard.wireGuardMeshActive') : t('tailscaleStatusCard.remoteAccessUnavailable') }}
        </div>
      </div>

      <!-- Secondary stats grid -->
      <div class="grid grid-cols-2 gap-3 mt-auto">

        <!-- Encryption -->
        <div class="flex flex-col p-3 rounded-lg bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50 group-hover:border-gray-200 dark:group-hover:border-zinc-700/50 transition-colors">
          <div class="flex items-center gap-1.5 mb-2 text-gray-500 dark:text-zinc-400">
            <Lock class="w-3 h-3" />
            <span class="text-[9px] font-bold uppercase tracking-widest">{{ t('tailscaleStatusCard.encryption') }}</span>
          </div>
          <div class="mt-auto text-sm font-semibold text-gray-800 dark:text-zinc-200 tracking-tight">
            WireGuard
          </div>
        </div>

        <!-- Ports / Version -->
        <div class="flex flex-col p-3 rounded-lg bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50 group-hover:border-gray-200 dark:group-hover:border-zinc-700/50 transition-colors">
          <div class="flex items-center gap-1.5 mb-2 text-gray-500 dark:text-zinc-400">
            <Globe class="w-3 h-3" />
            <span class="text-[9px] font-bold uppercase tracking-widest">
              {{ exposedPorts ? t('tailscaleStatusCard.ports') : t('tailscaleStatusCard.version') }}
            </span>
          </div>
          <div class="mt-auto text-sm font-semibold text-gray-800 dark:text-zinc-200 tracking-tight font-mono truncate">
            {{ exposedPorts ? exposedPorts : imageVersion }}
          </div>
        </div>

      </div>

      <!-- Container name footer -->
      <div class="flex items-center gap-2 pt-3 border-t border-gray-100 dark:border-zinc-800/50">
        <Wifi v-if="isRunning" class="w-3 h-3 text-green-500 shrink-0" />
        <WifiOff v-else class="w-3 h-3 text-red-400 shrink-0" />
        <span class="text-[11px] text-gray-400 dark:text-zinc-500 font-mono truncate">{{ containerName }}</span>
        <span
          v-if="isRunning"
          class="ml-auto shrink-0 inline-flex items-center gap-1 text-[9px] font-bold uppercase tracking-widest px-2 py-0.5 rounded-full bg-green-50 dark:bg-green-500/10 text-green-700 dark:text-green-400 border border-green-100 dark:border-green-500/20"
        >
          <Clock class="w-2.5 h-2.5" />
          {{ t('tailscaleStatusCard.connected') }}
        </span>
        <span
          v-else
          class="ml-auto shrink-0 inline-flex items-center gap-1 text-[9px] font-bold uppercase tracking-widest px-2 py-0.5 rounded-full bg-red-50 dark:bg-red-500/10 text-red-700 dark:text-red-400 border border-red-100 dark:border-red-500/20"
        >
          {{ t('tailscaleStatusCard.stopped') }}
        </span>
      </div>

    </div>
  </div>
</template>
