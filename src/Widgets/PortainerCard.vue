<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { Container, Activity, Box, ArrowRight } from '@lucide/vue'
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

// --- Status ---
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
  <div class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] rounded-xl overflow-hidden transition-all duration-300 hover:shadow-2xl hover:shadow-cyan-500/20 dark:hover:shadow-cyan-500/20 hover:-translate-y-0.5 text-left w-full min-h-72 cursor-pointer" @click="router.push('/apps/portainer')">
    <div class="absolute top-0 left-0 w-full h-0.5 opacity-0 group-hover:opacity-100 transition-opacity duration-500" :class="isRunning ? 'bg-cyan-500' : 'bg-gray-500'"></div>

    <div class="relative z-10 p-5 flex flex-col h-full gap-4">
      <!-- header -->
      <div class="flex items-center justify-between gap-3">
        <div class="flex items-center gap-3 min-w-0">
          <div class="w-9 h-9 rounded-lg flex items-center justify-center shrink-0 transition-transform group-hover:scale-110">
            <Container class="w-4.5 h-4.5 text-cyan-600 dark:text-cyan-400 group-hover:text-cyan-500 transition-colors" />
          </div>
          <div class="min-w-0">
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight leading-none truncate">Portainer</h3>
            <div class="flex items-center gap-1.5 mt-1.5 text-cyan-600 dark:text-cyan-400">
              <span class="text-[10px] font-bold uppercase tracking-widest">Advanced Manager</span>
            </div>
          </div>
        </div>
        <div class="flex items-center justify-center w-8 h-8 rounded-full opacity-0 group-hover:opacity-100 group-hover:translate-x-1 transition-all duration-300 shrink-0">
           <ArrowRight class="w-4 h-4 text-gray-400 dark:text-zinc-500 group-hover:text-cyan-500" />
        </div>
      </div>

      <div class="mt-auto pt-6 flex flex-col relative z-10">
        <div class="text-4xl font-black tracking-tighter text-gray-900 dark:text-white transition-colors line-clamp-1 leading-[1.1] mb-4">
          Advanced GUI
        </div>

        <div class="grid grid-cols-2 gap-2 mb-4">
          <div class="p-4 rounded-xl flex flex-col gap-1">
            <div class="flex items-center gap-1.5 text-gray-500 dark:text-zinc-400">
              <Activity v-if="isRunning" class="w-3.5 h-3.5 text-cyan-500 shrink-0" />
              <Box v-else class="w-3.5 h-3.5 text-gray-400 shrink-0" />
              <span class="text-[10px] font-bold uppercase tracking-widest">{{ isRunning ? 'Uptime' : 'Status' }}</span>
            </div>
            <div class="text-sm font-bold text-gray-900 dark:text-white tabular-nums tracking-tight">
              {{ isRunning ? formatUptime(uptimeMs) : 'Offline' }}
            </div>
          </div>
          
          <div class="p-4 rounded-xl flex flex-col gap-1 justify-center">
            <div v-if="!isRunning" class="text-xs font-semibold text-cyan-600 dark:text-cyan-400 uppercase tracking-widest group-hover:underline">
               Install App →
            </div>
            <div v-else class="text-xs font-semibold text-cyan-600 dark:text-cyan-400 uppercase tracking-widest group-hover:underline">
               Manage App →
            </div>
          </div>
        </div>

        <div class="flex items-center justify-between pt-4">
           <div class="flex items-center gap-2">
             <Container class="w-3.5 h-3.5 text-cyan-500 dark:text-cyan-400" />
             <span class="text-[10px] font-bold uppercase tracking-widest text-cyan-600 dark:text-cyan-500">
               Manage Containers, Networks & Volumes
             </span>
           </div>
        </div>

      </div>
    </div>
  </div>
</template>
