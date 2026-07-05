<script setup>
import { ref, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Server, ChevronRight, ChevronDown, Network, ExternalLink, Activity, Terminal } from '@lucide/vue'
import { formatDuration } from '../utils/metrics'
import { useCurrentTime } from '../composables/useCurrentTime'
import { useApiUrl } from '../composables/useApiUrl'
import { expectApiSuccess } from '../composables/useApiResponse'
import ContainerLogs from './ContainerLogs.vue'
import ContainerResources from './ContainerResources.vue'

const props = defineProps({
  services: {
    type: Array,
    required: true
  }
})

const router = useRouter()
const { t } = useI18n()
const { currentTime } = useCurrentTime()
const { apiUrl } = useApiUrl()

function formatUptime(service) {
  if (service.state !== 'running' || !service.created) return null
  const uptime = currentTime.value - service.created * 1000
  if (uptime <= 0) return t('stackView.justStarted')
  return formatDuration(uptime)
}

const expandedServiceId = ref(null)
const activeTab = ref('resources')
const containerStats = ref(null)
const containerLogs = ref([])
const refreshingLogs = ref(false)
const autoScrollLogs = ref(true)
let statsInterval = null

async function toggleService(svcId) {
  if (expandedServiceId.value === svcId) {
    expandedServiceId.value = null
    clearStatsInterval()
    return
  }
  
  expandedServiceId.value = svcId
  activeTab.value = 'resources'
  containerStats.value = null
  containerLogs.value = []
  
  await fetchContainerStats(svcId)
  startStatsInterval(svcId)
}

function selectTab(svcId, tab) {
  activeTab.value = tab
  if (tab === 'output') {
    fetchContainerLogs(svcId)
  }
}

async function fetchContainerStats(svcId) {
  try {
    const res = await fetch(`${apiUrl.value}/api/containers/${svcId}/stats`)
    const data = await expectApiSuccess(res)
    if (expandedServiceId.value === svcId) {
      containerStats.value = data.stats
    }
  } catch (err) {
    console.error(err)
  }
}

async function fetchContainerLogs(svcId = expandedServiceId.value) {
  if (!svcId) return
  refreshingLogs.value = true
  try {
    const res = await fetch(`${apiUrl.value}/api/containers/${svcId}/logs?tail=200`)
    const data = await expectApiSuccess(res)
    if (expandedServiceId.value === svcId) {
      containerLogs.value = Array.isArray(data.logs) ? data.logs : []
    }
  } catch (err) {
    console.error(err)
  } finally {
    refreshingLogs.value = false
  }
}

function startStatsInterval(svcId) {
  clearStatsInterval()
  statsInterval = setInterval(() => {
    if (expandedServiceId.value === svcId && activeTab.value === 'resources') {
      fetchContainerStats(svcId)
    }
  }, 2000)
}

function clearStatsInterval() {
  if (statsInterval) {
    clearInterval(statsInterval)
    statsInterval = null
  }
}

onUnmounted(() => {
  clearStatsInterval()
})

function goToContainer(svcId) {
  router.push(`/containers/${svcId}`)
}
</script>

<template>
  <div class="space-y-3 animate-fadeIn">
    <div class="flex items-center justify-between mb-1">
      <div class="text-xs font-bold uppercase tracking-widest" style="color: var(--text-secondary)">
        {{ t('stackView.containers') }}
      </div>
      <span class="text-xs font-mono font-bold px-2.5 py-1 rounded-lg" style="background: var(--surface-muted); color: var(--text-secondary)">
        {{ services.length }}
      </span>
    </div>

    <div class="grid gap-3">
      <div
        v-for="(svc, i) in services"
        :key="svc.id"
        class="group rounded-2xl border border-gray-100 dark:border-zinc-800 smooth-shadow transition-all duration-300 overflow-hidden"
        style="background: var(--surface)"
      >
        <!-- Header -->
        <div 
          @click="toggleService(svc.id)"
          class="p-5 cursor-pointer hover:bg-gray-50 dark:hover:bg-zinc-900/50 transition-colors flex flex-col"
        >
          <div class="flex items-start gap-4">
            <div
              class="w-11 h-11 rounded-xl flex items-center justify-center shrink-0 relative border transition-all group-hover:scale-110"
              :class="svc.state === 'running'
                ? 'bg-green-50 dark:bg-green-500/10 border-green-200 dark:border-green-500/20'
                : 'bg-gray-50 dark:bg-zinc-900 border-gray-200 dark:border-zinc-800'"
            >
              <Server :size="18" :class="svc.state === 'running' ? 'text-green-600 dark:text-green-400' : 'text-gray-400 dark:text-zinc-600'" />
              <span v-if="svc.state === 'running'" class="absolute -top-1 -right-1 flex h-3 w-3">
                <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                <span class="relative inline-flex rounded-full h-3 w-3 bg-green-500"></span>
              </span>
            </div>
            <div class="flex-1 min-w-0">
              <div class="flex items-center flex-wrap gap-2 mb-1">
                <span class="font-bold text-base transition-colors" style="color: var(--text-primary)">{{ svc.service }}</span>
                <span
                  class="text-[10px] font-bold uppercase tracking-widest px-2 py-0.5 rounded-md border"
                  :class="svc.state === 'running'
                    ? 'bg-green-50 dark:bg-green-500/10 text-green-700 dark:text-green-400 border-green-200 dark:border-green-500/20'
                    : 'bg-gray-100 dark:bg-zinc-800 text-gray-500 dark:text-zinc-400 border-gray-200 dark:border-zinc-700'"
                >{{ svc.state }}</span>
                <span v-if="svc.hasYantrLabel" class="text-[10px] font-bold uppercase tracking-widest px-2 py-0.5 rounded-md border bg-blue-50 dark:bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-200 dark:border-blue-500/20">{{ t('stackView.primary') }}</span>
              </div>
              <div class="font-mono text-xs truncate" style="color: var(--text-secondary)" :title="svc.image">{{ svc.image }}</div>
            </div>
            <div class="flex items-center gap-3 shrink-0">
              <div v-if="formatUptime(svc)" class="text-right hidden sm:block">
                <div class="text-[10px] font-bold uppercase tracking-widest mb-0.5" style="color: var(--text-secondary)">{{ t('stackView.uptime') }}</div>
                <div class="font-mono text-xs font-bold tabular-nums" style="color: var(--text-primary)">{{ formatUptime(svc) }}</div>
              </div>
              <div class="w-8 h-8 rounded-full flex items-center justify-center border border-gray-100 dark:border-zinc-800 transition-all" style="background: var(--surface-muted)">
                <ChevronDown v-if="expandedServiceId === svc.id" :size="15" class="text-gray-400 dark:text-zinc-500" />
                <ChevronRight v-else :size="15" class="text-gray-400 dark:text-zinc-500" />
              </div>
            </div>
          </div>
          <div v-if="svc.rawPorts && svc.rawPorts.length > 0" class="flex items-center gap-2 flex-wrap mt-3 pt-3 border-t border-gray-100 dark:border-zinc-800">
            <template v-if="svc.rawPorts.filter((p) => p.PublicPort).length > 0">
              <span
                v-for="p in [...new Map(svc.rawPorts.filter((rp) => rp.PublicPort).map((rp) => [`${rp.PublicPort}:${rp.PrivatePort}:${rp.Type}`, rp])).values()]"
                :key="`${p.PublicPort}-${p.Type}`"
                class="inline-flex items-center gap-1.5 text-[10px] font-mono font-bold px-2.5 py-1 rounded-lg bg-green-50 dark:bg-green-500/10 text-green-700 dark:text-green-400 border border-green-200 dark:border-green-500/20 hover:scale-105 transition-transform"
              >
                <Network :size="11" />:{{ p.PublicPort }} → {{ p.PrivatePort }}
              </span>
            </template>
            <template v-else>
              <span
                v-for="p in [...new Map(svc.rawPorts.map((rp) => [`${rp.PrivatePort}:${rp.Type}`, rp])).values()]"
                :key="`internal-${p.PrivatePort}-${p.Type}`"
                class="inline-flex items-center gap-1.5 text-[10px] font-mono font-bold px-2.5 py-1 rounded-lg border border-gray-200 dark:border-zinc-800 hover:scale-105 transition-transform"
                style="color: var(--text-secondary)"
              >
                <Network :size="11" />{{ p.PrivatePort }}/{{ p.Type }}
              </span>
            </template>
          </div>
        </div>
        
        <!-- Expanded Details -->
        <div v-if="expandedServiceId === svc.id" class="border-t border-gray-100 dark:border-zinc-800 p-5 bg-gray-50 dark:bg-[#0A0A0A]">
          <div class="flex items-center justify-between mb-4">
            <div class="flex gap-2">
              <button 
                @click="selectTab(svc.id, 'resources')"
                :class="activeTab === 'resources' ? 'bg-white dark:bg-zinc-800 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 hover:text-gray-700 dark:hover:text-zinc-300'"
                class="flex items-center gap-1.5 px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider rounded-lg transition-all"
              >
                <Activity :size="14" /> {{ t('containerDetail.resources') }}
              </button>
              <button 
                @click="selectTab(svc.id, 'output')"
                :class="activeTab === 'output' ? 'bg-white dark:bg-zinc-800 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 hover:text-gray-700 dark:hover:text-zinc-300'"
                class="flex items-center gap-1.5 px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider rounded-lg transition-all"
              >
                <Terminal :size="14" /> {{ t('containerDetail.output') }}
              </button>
            </div>
            
            <button 
              @click.stop="goToContainer(svc.id)"
              class="flex items-center gap-1.5 px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider bg-gray-900 dark:bg-zinc-100 text-white dark:text-gray-900 rounded-lg transition-all hover:opacity-90"
            >
              <ExternalLink :size="14" /> Details
            </button>
          </div>
          
          <div v-if="activeTab === 'resources'">
            <ContainerResources :container-stats="containerStats" />
          </div>
          <div v-else-if="activeTab === 'output'">
            <ContainerLogs 
              :logs="containerLogs" 
              v-model:autoScroll="autoScrollLogs" 
              :refreshing="refreshingLogs" 
              @refresh="() => fetchContainerLogs(svc.id)" 
            />
          </div>
        </div>

      </div>
    </div>
  </div>
</template>
