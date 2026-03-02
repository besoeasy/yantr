<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { Wifi, WifiOff, ArrowRight, Shield } from 'lucide-vue-next'

const props = defineProps({
  containers: { type: Array, default: () => [] },
  currentTime: { type: Number, default: () => Date.now() },
})

const router = useRouter()

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
  <div
    @click="router.push('/apps/tailscale')"
    class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl p-6 overflow-hidden transition-all duration-400 cursor-pointer hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:-translate-y-1 hover:border-gray-300 dark:hover:border-zinc-600"
  >
    <!-- Hover accent line -->
    <div
      class="absolute top-0 left-0 w-full h-[2px] opacity-0 group-hover:opacity-100 transition-opacity duration-500"
      :class="isRunning ? 'bg-gradient-to-r from-transparent via-green-500 to-transparent' : 'bg-gradient-to-r from-transparent via-red-500 to-transparent'"
    ></div>
    <!-- Dot-grid -->
    <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none [mask-image:linear-gradient(to_bottom,white,transparent)]"></div>

    <!-- Header -->
    <div class="relative z-10 flex items-start justify-between mb-6">
      <div class="flex items-center gap-4">
        <div
          class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0 group-hover:scale-105 transition-all duration-500"
          :class="isRunning
            ? 'bg-green-50/50 dark:bg-green-500/10 border border-green-100 dark:border-green-500/20'
            : 'bg-red-50/50 dark:bg-red-500/10 border border-red-100 dark:border-red-500/20'"
        >
          <Wifi v-if="isRunning" class="w-5 h-5 text-green-600 dark:text-green-500" />
          <WifiOff v-else class="w-5 h-5 text-red-600 dark:text-red-500" />
        </div>
        <div>
          <h3
            class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight transition-colors"
            :class="isRunning ? 'group-hover:text-green-600 dark:group-hover:text-green-400' : 'group-hover:text-red-600 dark:group-hover:text-red-400'"
          >
            Tailscale
          </h3>
          <div class="flex items-center gap-2 mt-1">
            <div
              class="w-1.5 h-1.5 rounded-full"
              :class="isRunning ? 'bg-green-500 animate-pulse' : 'bg-red-500'"
            ></div>
            <span class="text-[11px] font-medium text-gray-500 dark:text-zinc-400 uppercase tracking-wider">
              {{ isRunning ? 'Connected' : 'Stopped' }}
            </span>
          </div>
        </div>
      </div>

      <div class="shrink-0">
        <Shield class="w-4 h-4 text-gray-300 dark:text-zinc-600 group-hover:text-green-500 dark:group-hover:text-green-500 transition-colors duration-300" />
      </div>
    </div>

    <!-- Main metric -->
    <div class="relative z-10 flex-1 flex items-end justify-between gap-4">
      <div>
        <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-400 dark:text-zinc-500 mb-2">
          {{ isRunning ? 'Uptime' : 'Status' }}
        </div>
        <div class="text-4xl font-bold tabular-nums tracking-tighter text-gray-900 dark:text-white leading-none">
          {{ isRunning ? formatUptime(uptimeMs) : 'Down' }}
        </div>
        <div class="mt-3 text-[11px] font-medium text-gray-500 dark:text-zinc-400">
          {{ isRunning ? 'Secure mesh VPN active' : 'Remote access unavailable' }}
        </div>
      </div>

      <!-- Manage arrow (slides in on hover) -->
      <div class="shrink-0 flex flex-col items-end justify-end gap-1 pb-0.5 translate-y-4 opacity-0 group-hover:translate-y-0 group-hover:opacity-100 transition-all duration-300">
        <span class="text-[10px] font-bold uppercase tracking-wider text-gray-400 dark:text-zinc-500">Manage</span>
        <ArrowRight class="w-4 h-4 text-gray-400 dark:text-zinc-500" />
      </div>
    </div>
  </div>
</template>
