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
  <div class="relative group h-full flex flex-col rounded-[24px] overflow-hidden transition-all duration-500 hover:-translate-y-2 hover:shadow-[0_20px_40px_-15px_rgba(6,182,212,0.4)] dark:hover:shadow-[0_20px_40px_-15px_rgba(6,182,212,0.3)] text-left w-full min-h-[320px] cursor-pointer border border-cyan-100/50 dark:border-white/5 bg-gradient-to-br from-cyan-50/50 to-blue-50/50 dark:from-[#111827] dark:to-[#09090b]" @click="router.push('/apps/portainer')">
    
    <!-- Animated Gradient Glow -->
    <div class="absolute -top-[50%] -left-[50%] w-[200%] h-[200%] bg-[radial-gradient(circle_at_50%_50%,_rgba(6,182,212,0.15)_0%,_transparent_50%)] dark:bg-[radial-gradient(circle_at_50%_50%,_rgba(6,182,212,0.12)_0%,_transparent_40%)] opacity-0 group-hover:opacity-100 transition-opacity duration-1000 pointer-events-none group-hover:animate-[spin_20s_linear_infinite]"></div>

    <!-- Massive Watermark Icon -->
    <LayoutGrid class="absolute -bottom-8 -right-8 w-64 h-64 text-cyan-500/10 dark:text-cyan-400/[0.03] rotate-[-20deg] group-hover:rotate-0 group-hover:scale-110 transition-all duration-700 ease-out pointer-events-none" />

    <div class="relative z-10 p-7 flex flex-col h-full gap-4">
      <!-- Header -->
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Gorgeous Icon Block -->
          <div class="relative w-14 h-14 rounded-2xl flex items-center justify-center shrink-0 bg-gradient-to-br from-cyan-400 to-blue-600 shadow-xl shadow-cyan-500/30 group-hover:shadow-cyan-500/50 transition-all duration-500 group-hover:scale-105">
            <div class="absolute inset-0 bg-white/30 dark:bg-white/20 rounded-2xl rounded-bl-none opacity-50 dark:opacity-40 mix-blend-overlay"></div>
            <Container class="w-7 h-7 text-white drop-shadow-md z-10" />
          </div>
          <div>
            <h3 class="text-xl font-black text-gray-900 dark:text-white tracking-tight leading-none group-hover:text-cyan-600 dark:group-hover:text-cyan-400 transition-colors">Portainer</h3>
            <p class="text-[10px] font-bold uppercase tracking-[0.2em] text-cyan-600/70 dark:text-zinc-500 mt-2">Docker Environment</p>
          </div>
        </div>
        
        <!-- Premium Status Pill -->
        <div class="flex items-center gap-2 px-3 py-1.5 rounded-full border backdrop-blur-md shadow-sm transition-colors duration-500" :class="isRunning ? 'bg-white/80 border-cyan-100 dark:bg-cyan-900/30 dark:border-cyan-500/30' : 'bg-white/80 border-gray-200 dark:bg-zinc-800/80 dark:border-zinc-700'">
           <div class="w-2 h-2 rounded-full relative">
              <div class="absolute inset-0 rounded-full animate-ping opacity-75" :class="isRunning ? 'bg-cyan-500' : 'bg-gray-400'"></div>
              <div class="relative w-full h-full rounded-full" :class="isRunning ? 'bg-cyan-500' : 'bg-gray-400'"></div>
           </div>
           <span class="text-[9px] font-black uppercase tracking-widest" :class="isRunning ? 'text-cyan-700 dark:text-cyan-300' : 'text-gray-500 dark:text-gray-400'">
             {{ isRunning ? 'Online' : 'Offline' }}
           </span>
        </div>
      </div>

      <!-- Main Body -->
      <div class="mt-auto pt-6 flex flex-col relative z-10">
        
        <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-cyan-700/60 dark:text-zinc-500 mb-1 drop-shadow-sm">
          {{ isRunning ? 'Current Uptime' : 'App Status' }}
        </div>
        <div class="text-5xl font-black tracking-tighter text-cyan-950 dark:text-white mb-8 drop-shadow-sm group-hover:-translate-y-1 transition-transform duration-500">
          {{ isRunning ? formatUptime(uptimeMs) : 'Ready' }}
        </div>

        <!-- Nested Glass Container for details -->
        <div class="grid grid-cols-2 gap-3">
          <div class="p-4 rounded-[16px] bg-white/60 dark:bg-black/40 backdrop-blur-xl border border-white dark:border-white/5 flex flex-col gap-1.5 group-hover:bg-white/90 dark:group-hover:bg-white/[0.03] transition-all duration-300 shadow-[0_4px_12px_rgba(0,0,0,0.02)] dark:shadow-sm">
            <div class="flex items-center gap-2 text-cyan-800/60 dark:text-zinc-400">
              <Activity class="w-3.5 h-3.5" :class="isRunning ? 'text-cyan-500' : 'text-gray-400'" />
              <span class="text-[9px] font-bold uppercase tracking-widest">State</span>
            </div>
            <div class="text-sm font-black text-cyan-950 dark:text-white tabular-nums tracking-tight">
               {{ isRunning ? 'Running' : 'Stopped' }}
            </div>
          </div>
          
          <div class="p-4 rounded-[16px] bg-gradient-to-r from-cyan-100/50 to-blue-100/50 dark:from-cyan-950/40 dark:to-blue-950/40 backdrop-blur-xl border border-white dark:border-cyan-900/50 flex flex-col justify-center items-center group-hover:shadow-md transition-all duration-300 group/btn relative overflow-hidden shadow-[0_4px_12px_rgba(6,182,212,0.05)]">
            <div class="absolute inset-0 bg-cyan-100 dark:bg-cyan-900/50 translate-y-[100%] group-hover/btn:translate-y-0 transition-transform duration-300 ease-out"></div>
            <div class="text-[11px] font-black text-cyan-700 dark:text-cyan-400 uppercase tracking-widest flex items-center gap-1.5 relative z-10 group-hover/btn:text-cyan-800 dark:group-hover/btn:text-cyan-300 transition-colors">
               {{ isRunning ? 'Manage' : 'Install' }}
               <ArrowRight class="w-4 h-4 group-hover/btn:translate-x-1 transition-transform" />
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
