<script setup>
import { ref } from 'vue'
import { usePolling } from '../composables/usePolling'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Server, ChevronRight, ChevronDown, Network, ExternalLink, Activity, Terminal } from '@lucide/vue'
import { formatDuration } from '../utils/metrics'
import { useCurrentTime } from '../composables/useCurrentTime'
import { useApiUrl } from '../composables/useApiUrl'
import { expectApiSuccess } from '../composables/useApiResponse'
import ContainerLogs from './ContainerLogs.vue'
import ContainerResources from './ContainerResources.vue'
import { containerDetail } from '../utils/routes.js'

defineProps({
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

// Monotonic expansion token.
//
// Two races are fixed with this:
//  1. A slow fetch for service A resolved *after* the user had expanded B, and
//     its continuation then called startStatsInterval(A). That cleared B's
//     interval and installed one whose guard (expandedServiceId === A) could
//     never be true again, so B's panel silently froze on one-shot numbers.
//  2. Two in-flight fetches for the *same* service could resolve out of order,
//     letting the older response overwrite the newer numbers.
// Only the newest token may install a timer or write results.
let statsSeq = 0

async function toggleService(svcId) {
  if (expandedServiceId.value === svcId) {
    expandedServiceId.value = null
    clearStatsInterval()
    return
  }

  // Clear the outgoing service's timer *before* awaiting. Clearing it only in
  // the continuation is what allowed a late response to destroy the interval
  // belonging to the service the user is actually looking at.
  clearStatsInterval()
  statsSeq += 1
  const token = statsSeq

  expandedServiceId.value = svcId
  activeTab.value = 'resources'
  containerStats.value = null
  containerLogs.value = []

  await fetchContainerStats(svcId, token)
  if (token === statsSeq && expandedServiceId.value === svcId) {
    startStatsInterval(svcId)
  }
}

function selectTab(svcId, tab) {
  activeTab.value = tab
  if (tab === 'output') {
    fetchContainerLogs(svcId)
  }
}

async function fetchContainerStats(svcId, token = null) {
  const myToken = token === null ? statsSeq : token
  try {
    const res = await fetch(`${apiUrl.value}/api/containers/${svcId}/stats`)
    const data = await expectApiSuccess(res)
    if (myToken !== statsSeq) return
    if (expandedServiceId.value === svcId) {
      containerStats.value = data.stats
    }
  } catch {
    // Drop the numbers rather than leaving a panel showing values that are no
    // longer being refreshed — it reads as live but is not.
    if (myToken === statsSeq && expandedServiceId.value === svcId) {
      containerStats.value = null
    }
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
  } catch {
    // Intentionally swallowed: a failed log fetch keeps the previously fetched
    // lines visible. Clearing them would make a transient engine hiccup look
    // like "this container produced no output".
  } finally {
    refreshingLogs.value = false
  }
}

// Key-scoped poll: only the expanded service, and only while its Resources tab
// is showing. shouldRun reads the reactive guards, so a tick after a collapse
// is skipped before any request is made.
const { start: startStatsPolling, stop: stopStatsPolling } = usePolling(
  () => fetchContainerStats(expandedServiceId.value),
  2000,
  { shouldRun: () => expandedServiceId.value !== null && activeTab.value === 'resources' }
)

function startStatsInterval(svcId) {
  // A different service was expanded: re-key the poll. The token check in
  // toggleService already guarantees this only happens for the current one.
  stopStatsPolling()
  expandedServiceId.value = svcId
  startStatsPolling()
}

function clearStatsInterval() {
  stopStatsPolling()
}

function goToContainer(svcId) {
  router.push(containerDetail(svcId))
}
</script>

<template>
  <div class="space-y-3 anim-enter">
    <div class="flex items-center gap-2">
      <div class="yantra-label">{{ t('stackView.containers') }}</div>
      <span class="yantra-chip">{{ services.length }}</span>
    </div>

    <div class="grid gap-3">
      <div
        v-for="svc in services"
        :key="svc.id"
        class="yantra-surface overflow-hidden"
      >
        <!-- Header
             Keyboard reachable. The expanded panel holds the Resources/Output
             tabs and the Details link to the container page, so without
             role/tabindex/keydown a keyboard-only user could not read stats,
             read logs, or navigate onward from this page at all. The pattern
             (including the focus ring) is copied from YantraContainersGrid. -->
        <div
          @click="toggleService(svc.id)"
          @keydown.enter.prevent="toggleService(svc.id)"
          @keydown.space.prevent="toggleService(svc.id)"
          role="button"
          tabindex="0"
          :aria-expanded="expandedServiceId === svc.id"
          :aria-label="t('container.serviceHeader', { name: svc.service || svc.name })"
          class="p-5 cursor-pointer hover:bg-gray-50 dark:hover:bg-zinc-900/50 transition-colors flex flex-col focus-visible:outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-zinc-500"
        >
          <div class="flex items-start gap-4">
            <div
              class="yantra-tile relative h-11 w-11 shrink-0"
              :class="{ 'yantra-tile--ok': svc.state === 'running' }"
            >
              <Server :size="17" />
              <span v-if="svc.state === 'running'" class="absolute -top-1 -right-1 flex h-3 w-3">
                <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                <span class="relative inline-flex rounded-full h-3 w-3 bg-green-500"></span>
              </span>
            </div>
            <div class="flex-1 min-w-0">
              <div class="mb-1 flex flex-wrap items-center gap-2">
                <span class="truncate text-sm font-bold yantra-text-primary">{{ svc.service }}</span>
                <span class="yantra-chip" :class="{ 'yantra-chip--ok': svc.state === 'running' }">{{ svc.state }}</span>
                <span v-if="svc.hasYantrLabel" class="yantra-chip yantra-chip--accent">{{ t('stackView.primary') }}</span>
              </div>
              <div class="truncate font-mono text-[11px] yantra-text-secondary" :title="svc.image">{{ svc.image }}</div>
            </div>
            <div class="flex items-center gap-3 shrink-0">
              <div v-if="formatUptime(svc)" class="text-right hidden sm:block">
                <div class="yantra-fact__label mb-0.5">{{ t('stackView.uptime') }}</div>
                <div class="yantra-text-primary font-mono text-xs font-bold tabular-nums">{{ formatUptime(svc) }}</div>
              </div>
              <div class="yantra-tile h-8 w-8 rounded-full">
                <ChevronDown v-if="expandedServiceId === svc.id" :size="14" />
                <ChevronRight v-else :size="14" />
              </div>
            </div>
          </div>
          <div v-if="svc.rawPorts && svc.rawPorts.length > 0" class="mt-3 flex flex-wrap items-center gap-1.5 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <template v-if="svc.rawPorts.filter((p) => p.PublicPort).length > 0">
              <span
                v-for="p in [...new Map(svc.rawPorts.filter((rp) => rp.PublicPort).map((rp) => [`${rp.PublicPort}:${rp.PrivatePort}:${rp.Type}`, rp])).values()]"
                :key="`${p.PublicPort}-${p.Type}`"
                class="yantra-chip yantra-chip--ok font-mono normal-case"
              >
                <Network :size="11" />:{{ p.PublicPort }} → {{ p.PrivatePort }}
              </span>
            </template>
            <template v-else>
              <span
                v-for="p in [...new Map(svc.rawPorts.map((rp) => [`${rp.PrivatePort}:${rp.Type}`, rp])).values()]"
                :key="`internal-${p.PrivatePort}-${p.Type}`"
                class="yantra-chip font-mono normal-case"
              >
                <Network :size="11" />{{ p.PrivatePort }}/{{ p.Type }}
              </span>
            </template>
          </div>
        </div>
        
        <!-- Expanded Details -->
        <div v-if="expandedServiceId === svc.id" class="border-t border-zinc-100 bg-zinc-50 p-5 dark:border-zinc-800 dark:bg-zinc-900/30">
          <div class="mb-4 flex items-center justify-between">
            <div class="yantra-segment">
              <button
                type="button"
                class="yantra-chip"
                :class="{ 'yantra-chip--active': activeTab === 'resources' }"
                @click="selectTab(svc.id, 'resources')"
              >
                <Activity :size="13" /> {{ t('containerDetail.resources') }}
              </button>
              <button
                type="button"
                class="yantra-chip"
                :class="{ 'yantra-chip--active': activeTab === 'output' }"
                @click="selectTab(svc.id, 'output')"
              >
                <Terminal :size="13" /> {{ t('containerDetail.output') }}
              </button>
            </div>

            <button type="button" @click.stop="goToContainer(svc.id)" class="yantra-btn yantra-btn--primary">
              <ExternalLink :size="13" /> {{ t('stackView.serviceDetails') }}
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
