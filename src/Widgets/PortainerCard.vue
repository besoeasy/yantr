<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { Container, Activity, ArrowRight, LayoutGrid } from '@lucide/vue'
import { useApiUrl } from '../composables/useApiUrl'
import { expectApiSuccess } from '../composables/useApiResponse'
import { useCurrentTime } from '../composables/useCurrentTime'

const router = useRouter()
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

const portainerContainer = computed(() => {
  const list = Array.isArray(containers.value) ? containers.value : []
  const matches = list.filter((c) => {
    const name = (c?.name || '').toLowerCase()
    return name === 'portainer' || name.includes('portainer')
  })
  if (!matches.length) return null
  return matches.find((c) => c?.state === 'running') || matches[0]
})

const isRunning = computed(() => portainerContainer.value?.state === 'running')

const uptimeMs = computed(() => {
  const c = portainerContainer.value
  if (!c || c.state !== 'running' || !c.created) return null
  const createdMs = Number(c.created) * 1000
  if (!Number.isFinite(createdMs)) return null
  return Math.max(0, currentTime.value - createdMs)
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
  <div class="group relative flex h-full w-full min-h-[320px] cursor-pointer flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-all duration-300 hover:-translate-y-1 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]" @click="router.push('/apps/portainer')">

    <div class="relative z-10 flex h-full flex-col gap-4">
      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Minimal Icon Block -->
          <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 transition-transform duration-300 group-hover:scale-105 dark:border-zinc-800 dark:bg-zinc-900">
            <Container class="h-5 w-5 text-zinc-900 dark:text-zinc-100" />
          </div>
          <div>
            <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors group-hover:text-black dark:text-white dark:group-hover:text-white">Portainer</h3>
            <p class="mt-2 text-[10px] font-semibold uppercase tracking-widest text-zinc-500">Docker Environment</p>
          </div>
        </div>
        
        <!-- Status Pill -->
        <div class="flex items-center gap-2 rounded-full border px-3 py-1.5 transition-colors duration-300" :class="isRunning ? 'border-emerald-200 bg-emerald-50 dark:border-emerald-900/30 dark:bg-emerald-900/20' : 'border-zinc-200 bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-900/50'">
           <div class="relative h-2 w-2 rounded-full">
              <div class="absolute inset-0 rounded-full animate-ping opacity-75" :class="isRunning ? 'bg-emerald-400' : 'bg-zinc-400'"></div>
              <div class="relative h-full w-full rounded-full" :class="isRunning ? 'bg-emerald-500' : 'bg-zinc-500'"></div>
           </div>
           <span class="text-[9px] font-bold uppercase tracking-widest" :class="isRunning ? 'text-emerald-700 dark:text-emerald-400' : 'text-zinc-500 dark:text-zinc-400'">
             {{ isRunning ? 'Online' : 'Offline' }}
           </span>
        </div>
      </div>

      <!-- Main Body -->
      <div class="relative z-10 mt-auto flex flex-col pt-6">
        
        <div class="mb-1 text-[10px] font-semibold uppercase tracking-widest text-zinc-500">
          {{ isRunning ? 'Current Uptime' : 'App Status' }}
        </div>
        <div class="mb-8 text-4xl font-black tracking-tighter text-zinc-900 transition-transform duration-500 group-hover:-translate-y-1 sm:text-5xl dark:text-white">
          {{ isRunning ? formatUptime(uptimeMs) : 'Ready' }}
        </div>

        <!-- Nested Containers for details -->
        <div class="grid grid-cols-2 gap-2">
          <div class="flex flex-col gap-1.5 rounded-xl border border-zinc-200 bg-zinc-50 p-3.5 transition-colors duration-300 hover:bg-zinc-100 dark:border-zinc-800 dark:bg-zinc-900/50 dark:hover:bg-zinc-900">
            <div class="flex items-center gap-2 text-zinc-500">
              <Activity class="h-3.5 w-3.5" :class="isRunning ? 'text-emerald-500' : 'text-zinc-400'" />
              <span class="text-[9px] font-semibold uppercase tracking-widest">State</span>
            </div>
            <div class="tabular-nums text-sm font-black tracking-tight text-zinc-900 dark:text-white">
               {{ isRunning ? 'Running' : 'Stopped' }}
            </div>
          </div>
          
          <div class="group/btn relative flex items-center justify-center overflow-hidden rounded-xl border border-zinc-200 bg-zinc-900 p-3.5 transition-colors duration-300 hover:bg-black dark:border-zinc-800 dark:bg-zinc-100 dark:hover:bg-white">
            <div class="relative z-10 flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-widest text-white transition-colors dark:text-zinc-900">
               {{ isRunning ? 'Manage' : 'Install' }}
               <ArrowRight class="h-4 w-4 transition-transform group-hover/btn:translate-x-1" />
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
