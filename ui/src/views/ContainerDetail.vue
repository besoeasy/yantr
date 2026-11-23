<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { usePolling } from '../composables/usePolling'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useNotification } from '../composables/useNotification'
import { useApiUrl } from '../composables/useApiUrl'
import { expectApiSuccess, getApiErrorMessage, readJsonResponse } from '../composables/useApiResponse'
import { useYantrAuth } from '../composables/useYantrAuth'
import { useCurrentTime } from '../composables/useCurrentTime'
import { ExternalLink, Trash2, Network, Globe, HardDrive, Play, Square, RotateCcw, Wrench, Terminal, Copy, MoreVertical } from '@lucide/vue'
import AppLogo from '../components/AppLogo.vue'
import ContainerResources from '../components/ContainerResources.vue'
import ContainerLogs from '../components/ContainerLogs.vue'
import ContainerEnv from '../components/ContainerEnv.vue'
import AppLayout from '../components/AppLayout.vue'
import { appUrl, isNavigableProtocol } from '../utils/url'
import { formatUptime } from '../utils/metrics'
import { dashboard, appDetail, stackDetail } from '../utils/routes.js'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const toast = useNotification()
const { apiUrl } = useApiUrl()
const { openVolumeBrowser } = useYantrAuth()
const { currentTime } = useCurrentTime()

const selectedContainer = ref(null)
const shellPath = ref('')
const shellLoading = ref(false)
const shellError = ref('')
const shellCommand = computed(() => shellPath.value && selectedContainer.value
  ? `podman exec -it ${selectedContainer.value.id} ${shellPath.value}`
  : '')
const containerStats = ref(null)
const containerLogs = ref([])
const deleting = ref(false)
const refreshingLogs = ref(false)
const browsingVolume = ref({})
const showVolumeMenu = ref({})
const autoScrollLogs = ref(true)
const activeTab = ref('resources')
// "Access" shows only ports the app marked show: true; "All" also lists work
// ports — ports that exist for protocol plumbing and have no UI surface.
const showAccessPorts = ref(true)
// The actions that are not the everyday one live behind the kebab. It is a
// <details>, so it opens and closes with no state to keep in sync; this ref
// only lets an action close it again.
const menuEl = ref(null)
const loadErrorState = {
  stats: false,
  logs: false,
}

function notifyLoadErrorOnce(key, message) {
  if (loadErrorState[key]) return
  loadErrorState[key] = true
  toast.error(message)
}

function clearLoadError(key) {
  loadErrorState[key] = false
}

function closeMenu() {
  if (menuEl.value) menuEl.value.open = false
}

const containerVolumes = computed(() => {
  if (!selectedContainer.value?.mounts) return []
  
  return selectedContainer.value.mounts
    .filter(mount => mount.Type === 'volume')
    .map(mount => ({
      name: mount.Name,
      destination: mount.Destination,
      rw: mount.RW
    }))
})

const allPortMappings = computed(() => {
  if (!selectedContainer.value || !selectedContainer.value.ports) {
    return []
  }
  
  const portByPort = {}
  for (const p of (selectedContainer.value.app?.ports || [])) {
    if (p.port != null) {
      const key = String(p.port)
      if (!portByPort[key]) portByPort[key] = []
      portByPort[key].push({
        protocol: (p.protocol || '').toLowerCase(),
        label: p.label || null,
        service: p.service || null,
        show: p.show === true
      })
    }
  }
  const containerService = selectedContainer.value.app?.service || selectedContainer.value.name
  function lookupLabel(privatePort) {
    const cands = portByPort[String(privatePort)]
    if (!cands || cands.length === 0) return null
    if (containerService) {
      const exact = cands.find((c) => c.service === containerService)
      if (exact) return exact
    }
    return cands[0]
  }
  
  const mappings = []
  const portKeys = Object.keys(selectedContainer.value.ports)
  
  portKeys.forEach(key => {
    const [privatePort, type] = key.split('/')
    const bindings = selectedContainer.value.ports[key]
    
    if (bindings && bindings.length > 0) {
      const seenHostPorts = new Set()
      bindings.forEach(binding => {
        if (binding.HostPort && !seenHostPorts.has(binding.HostPort)) {
          seenHostPorts.add(binding.HostPort)
          const label = lookupLabel(privatePort) || lookupLabel(binding.HostPort)
          mappings.push({
            containerPort: privatePort,
            hostPort: binding.HostPort,
            hostIp: binding.HostIp || '0.0.0.0',
            protocol: type,
            label: label?.label || null,
            labeledProtocol: label?.protocol || null,
            // A published port with no catalog entry has no show flag and is
            // therefore not an access port — it only appears under "All".
            show: label?.show === true
          })
        }
      })
    } else {
      const label = lookupLabel(privatePort)
      mappings.push({
        containerPort: privatePort,
        hostPort: null,
        hostIp: null,
        protocol: type,
        label: label?.label || null,
        labeledProtocol: label?.protocol || null,
        show: label?.show === true
      })
    }
  })
  
  return mappings.sort((a, b) => {
    if (a.hostPort && b.hostPort) {
      return parseInt(a.hostPort) - parseInt(b.hostPort)
    }
    if (a.hostPort && !b.hostPort) return -1
    if (!a.hostPort && b.hostPort) return 1
    return parseInt(a.containerPort) - parseInt(b.containerPort)
  })
})

// Ports the user is meant to reach, per x-yantr.ports `show: true`.
const accessPortMappings = computed(() => allPortMappings.value.filter(m => m.show))

const hasAccessPorts = computed(() => accessPortMappings.value.length > 0)

// The one the hero offers to open: a published port the catalog says is a UI.
const primaryPort = computed(() => accessPortMappings.value.find(
  (m) => m.hostPort && m.protocol === 'tcp' && isNavigableProtocol(m.labeledProtocol)
) || null)

const filteredPortMappings = computed(() => {
  if (!showAccessPorts.value) {
    return allPortMappings.value
  }
  // Fall back to the catalogued ports if nothing is flagged show: true, so an
  // app that has not opted any port in yet does not render an empty grid.
  if (!hasAccessPorts.value) {
    const described = allPortMappings.value.filter(mapping => mapping.label)
    return described.length > 0 ? described : allPortMappings.value
  }
  return accessPortMappings.value
})

// The header facts. Uptime comes from the inspect timestamp the container
// reports, so it is the container's own clock rather than a browser's guess;
// a stopped container has no uptime to show and says so.
const startedAt = computed(() => {
  const raw = selectedContainer.value?.stateDetails?.StartedAt
  const ms = raw ? Date.parse(raw) : NaN
  return Number.isFinite(ms) && ms > 0 ? ms : null
})

const uptime = computed(() => {
  if (selectedContainer.value?.state !== 'running' || !startedAt.value) {
    return t('containerDetail.notRunning')
  }
  return formatUptime(currentTime.value - startedAt.value)
})

const stateChipClass = computed(() => {
  const state = selectedContainer.value?.state
  if (state === 'running') return 'yantra-chip--ok'
  if (state === 'restarting' || state === 'paused') return 'yantra-chip--warn'
  return ''
})

const appTags = computed(() => (selectedContainer.value?.app?.tags || []).slice(0, 4))

const projectId = computed(() => selectedContainer.value?.app?.projectId || '')
const appId = computed(() => selectedContainer.value?.app?.id || '')

async function fetchContainerDetail() {
  try {
    const response = await fetch(`${apiUrl.value}/api/containers/${route.params.containerId}`)
    const data = await expectApiSuccess(response, t('containerDetail.error.containerNotFound'))
    selectedContainer.value = data.container
    shellPath.value = ''
    shellError.value = ''
    if (data.container.state === 'running') fetchContainerShell(data.container.id)
  } catch (error) {
    toast.error(error.message || t('containerDetail.error.failedToLoadDetails'))
    router.push(dashboard())
  }
}

async function fetchContainerShell(id) {
  shellLoading.value = true
  try {
    const response = await fetch(`${apiUrl.value}/api/containers/${id}/shell`)
    const data = await expectApiSuccess(response, t('containerDetail.shellDetectionFailed'))
    if (selectedContainer.value?.id === id && selectedContainer.value.state === 'running') {
      shellPath.value = data.shell || ''
    }
  } catch {
    if (selectedContainer.value?.id === id && selectedContainer.value.state === 'running') {
      shellError.value = t('containerDetail.shellDetectionFailed')
    }
  } finally {
    shellLoading.value = false
  }
}

async function copyShellCommand() {
  try {
    await navigator.clipboard.writeText(shellCommand.value)
    toast.success(t('containerDetail.shellCopied'))
  } catch {
    toast.error(t('containerDetail.shellCopyFailed'))
  }
}

async function fetchContainerStats() {
  if (!selectedContainer.value) return
  
  try {
    const response = await fetch(`${apiUrl.value}/api/containers/${selectedContainer.value.id}/stats`)
    const data = await expectApiSuccess(response, 'Failed to load container stats')
    containerStats.value = data.stats
    clearLoadError('stats')
  } catch (error) {
    notifyLoadErrorOnce('stats', error.message || 'Failed to load container stats')
  }
}

async function fetchContainerLogs() {
  if (!selectedContainer.value) return
  
  refreshingLogs.value = true
  try {
    const response = await fetch(`${apiUrl.value}/api/containers/${selectedContainer.value.id}/logs?tail=200`)
    const data = await expectApiSuccess(response, 'Failed to load container logs')
    containerLogs.value = Array.isArray(data.logs) ? data.logs : []
    clearLoadError('logs')
    if (autoScrollLogs.value) {
      scrollToBottom()
    }
  } catch (error) {
    notifyLoadErrorOnce('logs', error.message || 'Failed to load container logs')
  } finally {
    setTimeout(() => {
      refreshingLogs.value = false
    }, 300)
  }
}

const scrollToBottom = () => {
    setTimeout(() => {
        const el = document.getElementById('terminal-logs')
        if (el) el.scrollTop = el.scrollHeight
    }, 100)
}

// start / stop / restart for a single container.
//
// These three routes were registered in core/main.go with no UI caller, so the
// whole `container.*` i18n block existed with nothing rendering it and users
// could not stop or restart anything.
//
// The server now records the intent per compose service (see #87), so stopping
// one service of a multi-service stack no longer disables crash recovery for
// its siblings, and a service stopped here stays stopped across a reboot.
const busyAction = ref(null)

async function lifecycleAction(action) {
  if (busyAction.value) return;
  const id = route.params.containerId;
  if (!id) return;

  busyAction.value = action;
  const labels = {
    start: ['container.starting', 'container.startFailed'],
    stop: ['container.stopping', 'container.stopFailed'],
    restart: ['container.restarting', 'container.restartFailed'],
  };
  const [okKey, errKey] = labels[action] || labels.restart;

  try {
    const res = await fetch(`${apiUrl.value}/api/containers/${id}/${action}`, { method: 'POST' });
    if (!res.ok) {
      const body = await readJsonResponse(res);
      throw new Error(getApiErrorMessage(body, t(errKey)));
    }
    toast.success(t(okKey));
    await fetchContainerDetail();
  } catch (e) {
    toast.error(t(errKey, { error: e.message }));
  } finally {
    busyAction.value = null;
  }
}

async function deleteContainer() {
  closeMenu()
  if (!confirm(`${t('containerDetail.terminateContainer')} ${selectedContainer.value.name}?\n\n${t('containerDetail.warning')}`)) return

  deleting.value = true
  try {
    const response = await fetch(`${apiUrl.value}/api/containers/${selectedContainer.value.id}`, {
      method: 'DELETE'
    })
    const data = await readJsonResponse(response)

    if (response.ok && data.success) {
      // No volumesRemoved handling: the backend has never sent that field, so
      // the branch was unreachable. It was a leftover from a removed
      // "remove volumes" option.
      toast.success(t('containerDetail.success.deletedSuccessfully', { name: selectedContainer.value.name }))
      router.push(dashboard())
    } else {
      throw new Error(getApiErrorMessage(data, t('containerDetail.error.deletionFailed', { error: t('common.error') })))
    }
  } catch (error) {
    toast.error(t('containerDetail.error.deletionFailed', { error: error.message }))
  } finally {
    deleting.value = false
  }
}

async function browseVolume(volumeName, expiryMinutes = 60) {
  browsingVolume.value[volumeName] = volumeName
  showVolumeMenu.value[volumeName] = false
  try {
    const response = await fetch(`${apiUrl.value}/api/volumes/${volumeName}/browse`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ expiryMinutes }),
    })
    await expectApiSuccess(response, t('containerDetail.error.failedToStartVolumeBrowser'))
    const expiryText = expiryMinutes > 0 ? ` (${expiryMinutes}m)` : ' (no expiry)'
    toast.success(t('containerDetail.success.volumeBrowserStarted', { expiry: expiryText }))
    openVolumeBrowser(volumeName)
  } catch (error) {
    toast.error(error.message || t('containerDetail.error.failedToStartVolumeBrowser'))
  } finally {
    delete browsingVolume.value[volumeName]
  }
}

// Stats are polled only while the Resources tab is actually on screen. Before
// this, a 2s /stats poll ran even on the Output and Env tabs, and each poll
// opened a podman stats stream on the host.
const { start: startStatsPolling, stop: stopStatsPolling } = usePolling(
  async () => {
    if (activeTab.value !== 'resources') return
    await fetchContainerStats()
  },
  2000
)

onMounted(async () => {
  await fetchContainerDetail()
  await Promise.all([
    fetchContainerStats(),
    fetchContainerLogs()
  ])
  startStatsPolling()
})

watch(activeTab, (tab) => {
  // Off Resources means the numbers are not displayed; stop spending a request
  // every 2s on them.
  if (tab === 'resources') startStatsPolling()
  else stopStatsPolling()
})
</script>

<template>
  <AppLayout width="reading" gap="none">
    <div v-if="!selectedContainer" class="flex justify-center py-32">
       <div class="h-8 w-8 animate-spin rounded-full border-[3px] border-zinc-200 border-t-zinc-900 dark:border-zinc-800 dark:border-t-white"></div>
    </div>

    <!-- The page's sections share one rhythm: hero, the facts it answers with,
         then the body. The old page stacked six bordered cards with no
         hierarchy, so the name, the shell and the terminate button all read at
         the same weight. -->
    <main v-else class="anim-enter space-y-8">

        <!-- Breadcrumb: where this container sits. The stack is the parent, so
             that is the hop that is actually useful. -->
        <nav class="flex flex-wrap items-center gap-2 text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
          <router-link :to="dashboard()" class="transition-colors hover:text-zinc-900 dark:hover:text-white">
            {{ t('nav.dashboard') }}
          </router-link>
          <span class="text-zinc-300 dark:text-zinc-700">/</span>
          <router-link
            v-if="projectId && projectId !== selectedContainer.name"
            :to="stackDetail(projectId)"
            class="transition-colors hover:text-zinc-900 dark:hover:text-white"
          >
            {{ projectId }}
          </router-link>
          <span v-if="projectId && projectId !== selectedContainer.name" class="text-zinc-300 dark:text-zinc-700">/</span>
          <span class="yantra-text-primary">{{ selectedContainer.name }}</span>
        </nav>

        <!-- Identity: what it is, what state it is in, and the four things you
             can do about it. The everyday action is the one strong button; the
             rest sit beside it, and the destructive one is behind the kebab. -->
        <section class="yantra-surface p-6 sm:p-8">
          <div class="flex flex-col gap-6 sm:flex-row">
            <div class="yantra-tile h-20 w-20 shrink-0 p-4">
              <AppLogo
                :logo="selectedContainer.app?.logo"
                :name="selectedContainer.app?.name || selectedContainer.name"
                :seed="selectedContainer.app?.id || selectedContainer.name"
                img-class="h-full w-full object-contain filter dark:brightness-90"
                icon-class="h-full w-full yantra-text-primary"
              />
            </div>

            <div class="min-w-0 flex-1 space-y-4">
              <div class="flex flex-wrap items-center gap-2">
                <h1 class="text-3xl font-black tracking-tighter yantra-text-primary">
                  {{ selectedContainer.name }}
                </h1>
                <span class="yantra-chip" :class="stateChipClass">
                  <span v-if="selectedContainer.state === 'running'" class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
                  {{ selectedContainer.state }}
                </span>
                <span class="yantra-chip font-mono normal-case tracking-normal">
                  {{ selectedContainer.id.substring(0, 12) }}
                </span>
              </div>

              <p class="max-w-2xl text-sm leading-relaxed yantra-text-secondary">
                {{ selectedContainer.app?.short_description || selectedContainer.app?.description || t('containerDetail.descriptionNotAvailable') }}
              </p>

              <div class="flex flex-wrap gap-1.5">
                <span class="yantra-chip font-mono normal-case tracking-normal" :title="selectedContainer.image">
                  {{ selectedContainer.image }}
                </span>
                <span v-for="tag in appTags" :key="tag" class="yantra-chip">{{ tag }}</span>
              </div>

              <div class="flex flex-wrap items-center gap-2 pt-1">
                <a
                  v-if="primaryPort"
                  :href="appUrl(primaryPort.hostPort, primaryPort.labeledProtocol)"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="yantra-btn yantra-btn--primary"
                >
                  <ExternalLink :size="13" />
                  {{ t('containerDetail.openLabel', { label: primaryPort.label || t('containerDetail.networkPort') }) }}
                </a>

                <button
                  v-else-if="selectedContainer.state !== 'running'"
                  @click="lifecycleAction('start')"
                  :disabled="busyAction !== null"
                  class="yantra-btn yantra-btn--primary"
                >
                  <Play :size="13" />{{ busyAction === 'start' ? t('container.starting') : t('container.start') }}
                </button>

                <button
                  @click="lifecycleAction('restart')"
                  :disabled="busyAction !== null"
                  class="yantra-btn"
                >
                  <RotateCcw :size="13" :class="busyAction === 'restart' ? 'animate-spin' : ''" />
                  {{ busyAction === 'restart' ? t('container.restarting') : t('container.restart') }}
                </button>

                <button
                  v-if="selectedContainer.state === 'running'"
                  @click="lifecycleAction('stop')"
                  :disabled="busyAction !== null"
                  class="yantra-btn"
                >
                  <Square :size="13" />{{ busyAction === 'stop' ? t('container.stopping') : t('container.stop') }}
                </button>
                <button
                  v-else
                  @click="lifecycleAction('start')"
                  :disabled="busyAction !== null"
                  class="yantra-btn"
                >
                  <Play :size="13" />{{ busyAction === 'start' ? t('container.starting') : t('container.start') }}
                </button>

                <!-- The rest: links out, and the one action that cannot be
                     undone. -->
                <details ref="menuEl" class="kebab relative sm:ml-auto">
                  <summary class="yantra-btn cursor-pointer" :aria-label="t('containerDetail.moreActions')" aria-haspopup="menu">
                    <MoreVertical :size="15" />
                  </summary>
                  <div class="absolute right-0 z-20 mt-2 w-48 overflow-hidden rounded-lg border border-zinc-200 bg-white p-1.5 shadow-lg dark:border-zinc-800 dark:bg-zinc-900">
                    <router-link
                      v-if="appId"
                      :to="appDetail(appId)"
                      class="flex items-center gap-2 px-2.5 py-2 text-[11px] font-bold uppercase tracking-wider yantra-text-secondary transition-colors hover:bg-zinc-50 hover:text-zinc-900 dark:hover:bg-zinc-800 dark:hover:text-white"
                    >
                      <ExternalLink :size="13" />{{ t('stackView.appPage') }}
                    </router-link>
                    <router-link
                      v-if="projectId"
                      :to="stackDetail(projectId)"
                      class="flex items-center gap-2 px-2.5 py-2 text-[11px] font-bold uppercase tracking-wider yantra-text-secondary transition-colors hover:bg-zinc-50 hover:text-zinc-900 dark:hover:bg-zinc-800 dark:hover:text-white"
                    >
                      <Network :size="13" />{{ t('containerDetail.stackPage') }}
                    </router-link>
                    <div class="mt-1 mb-1 h-px bg-zinc-100 dark:bg-zinc-800"></div>
                    <button
                      @click="deleteContainer"
                      :disabled="deleting"
                      class="flex w-full items-center gap-2 px-2.5 py-2 text-[11px] font-bold uppercase tracking-wider text-red-600 transition-colors hover:bg-red-50 dark:text-red-500 dark:hover:bg-red-500/10"
                    >
                      <Trash2 :size="13" />{{ deleting ? t('containerDetail.terminating') : t('containerDetail.terminateContainer') }}
                    </button>
                  </div>
                </details>
              </div>
            </div>
          </div>
        </section>

        <!-- The three numbers the page is asked for first. -->
        <div class="yantra-facts">
          <div class="p-5">
            <div class="yantra-label">{{ t('containerDetail.uptime') }}</div>
            <div class="mt-2 font-mono text-2xl font-black tracking-tighter yantra-text-primary">{{ uptime }}</div>
            <div class="mt-1 text-[10px] yantra-text-secondary">{{ t('containerDetail.uptimeHint') }}</div>
          </div>
          <div class="p-5">
            <div class="yantra-label">{{ t('containerDetail.accessPorts') }}</div>
            <div class="mt-2 font-mono text-2xl font-black tracking-tighter yantra-text-primary">
              {{ accessPortMappings.length }} / {{ allPortMappings.length }}
            </div>
            <div class="mt-1 text-[10px] yantra-text-secondary">
              {{ t('containerDetail.portsPublished', { count: accessPortMappings.length }) }}
            </div>
          </div>
          <div class="p-5">
            <div class="yantra-label">{{ t('containerDetail.volumes') }}</div>
            <div class="mt-2 font-mono text-2xl font-black tracking-tighter yantra-text-primary">{{ containerVolumes.length }}</div>
            <div class="mt-1 text-[10px] yantra-text-secondary">{{ t('containerDetail.namedVolume') }}</div>
          </div>
        </div>

        <!-- Network and shell: the two ways into a container, each a full row.
             Side by side they were narrow — a port card lost its panels and a
             command line wrapped — and the taller of the two left a gap. -->
        <div class="space-y-8">
          <section class="space-y-3">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div class="flex items-center gap-2">
                <h2 class="yantra-label">{{ t('containerDetail.networkAccess') }}</h2>
                <span v-if="allPortMappings.length > 0" class="yantra-chip">
                  {{ showAccessPorts && accessPortMappings.length < allPortMappings.length ? `${accessPortMappings.length}/${allPortMappings.length}` : filteredPortMappings.length }}
                </span>
              </div>
              <div v-if="hasAccessPorts" class="yantra-segment">
                <button
                  type="button"
                  class="yantra-chip"
                  :class="{ 'yantra-chip--active': !showAccessPorts }"
                  @click="showAccessPorts = false"
                >{{ t('containerDetail.allPorts') }}</button>
                <button
                  type="button"
                  class="yantra-chip"
                  :class="{ 'yantra-chip--active': showAccessPorts }"
                  @click="showAccessPorts = true"
                >{{ t('containerDetail.access') }}</button>
              </div>
            </div>

            <div v-if="filteredPortMappings.length > 0" class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
              <div v-for="(p, i) in filteredPortMappings" :key="i" class="yantra-surface flex h-full flex-col p-5">
                <div class="mb-4 flex items-start gap-4">
                  <div class="yantra-tile h-10 w-10">
                    <Globe v-if="p.labeledProtocol === 'http' || p.labeledProtocol === 'https'" :size="17" />
                    <Network v-else :size="17" />
                  </div>
                  <div class="min-w-0 flex-1">
                    <div class="mb-1 flex items-center gap-2">
                      <span class="font-mono text-[10px] font-bold uppercase yantra-text-primary">{{ p.protocol }}</span>
                      <span v-if="p.labeledProtocol" class="yantra-chip yantra-chip--accent">{{ p.labeledProtocol }}</span>
                    </div>
                    <div class="truncate text-[11px] yantra-text-secondary" :title="p.label || p.service">
                      {{ p.label || t('containerDetail.networkPort') }}
                    </div>
                  </div>
                </div>

                <div class="yantra-panel mb-4">
                  <div class="yantra-fact">
                    <span class="yantra-fact__label">{{ t('containerDetail.hostPort') }}</span>
                    <span v-if="p.hostPort" class="yantra-fact__value">:{{ p.hostPort }}</span>
                    <span v-else class="yantra-text-secondary italic">{{ t('containerDetail.internal') }}</span>
                  </div>
                  <div class="yantra-fact">
                    <span class="yantra-fact__label">{{ t('containerDetail.containerPort') }}</span>
                    <span class="yantra-fact__value">{{ p.containerPort }}</span>
                  </div>
                </div>

                <div class="mt-auto">
                  <a
                    v-if="p.show && p.protocol === 'tcp' && p.hostPort && isNavigableProtocol(p.labeledProtocol)"
                    :href="appUrl(p.hostPort, p.labeledProtocol)"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="yantra-btn yantra-btn--primary w-full"
                  >
                    <ExternalLink :size="12" />{{ t('containerDetail.open') }}
                  </a>
                  <!-- A work port is protocol plumbing with nothing to open, so
                       it says so rather than offering a link that would fail. -->
                  <div
                    v-else-if="!p.show"
                    class="yantra-btn w-full cursor-default justify-between"
                    :title="t('containerDetail.workPortHint')"
                  >
                    <span class="inline-flex items-center gap-2"><Wrench :size="12" />{{ t('containerDetail.workPort') }}</span>
                  </div>
                  <div v-else class="yantra-btn w-full cursor-default justify-between">
                    <span>{{ t('containerDetail.internalOnly') }}</span>
                  </div>
                </div>
              </div>
            </div>

            <div v-else class="yantra-empty">
              <Network :size="24" />
              <span>{{ t('containerDetail.noPorts') }}</span>
            </div>
          </section>

          <section class="space-y-3">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div class="flex items-center gap-2">
                <h2 class="yantra-label">{{ t('containerDetail.shellAccess') }}</h2>
                <span v-if="shellPath" class="yantra-chip">{{ shellPath }}</span>
              </div>
            </div>

            <div class="yantra-surface flex h-full flex-col p-5">
              <div class="mb-4 flex items-start gap-4">
                <div class="yantra-tile h-10 w-10">
                  <Terminal :size="17" />
                </div>
                <div class="min-w-0 flex-1">
                  <div class="text-sm font-bold yantra-text-primary">{{ t('containerDetail.shellTitle') }}</div>
                  <div class="mt-1 text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
                    {{ t('containerDetail.shellSubtitle') }}
                  </div>
                </div>
              </div>

              <p v-if="selectedContainer.state !== 'running'" class="text-xs yantra-text-secondary">
                {{ t('containerDetail.shellStopped') }}
              </p>
              <p v-else-if="shellLoading" class="text-xs yantra-text-secondary">
                {{ t('containerDetail.detectingShell') }}
              </p>
              <p v-else-if="shellError || !shellCommand" class="text-xs yantra-text-secondary">
                {{ shellError || t('containerDetail.noShell') }}
              </p>
              <template v-else>
                <div class="flex items-center gap-2 rounded-lg border border-zinc-200 bg-zinc-50 p-2 dark:border-zinc-800 dark:bg-zinc-900/50">
                  <code class="min-w-0 flex-1 overflow-x-auto whitespace-nowrap px-1 font-mono text-[11px] yantra-text-primary">{{ shellCommand }}</code>
                  <button
                    type="button"
                    @click="copyShellCommand"
                    :aria-label="t('containerDetail.copyShellCommand')"
                    class="yantra-btn shrink-0 px-2"
                  >
                    <Copy :size="13" />
                  </button>
                </div>
                <p class="mt-3 text-[10px] leading-relaxed yantra-text-secondary">
                  {{ t('containerDetail.shellHint') }}
                </p>
              </template>
            </div>
          </section>
        </div>

        <!-- Storage: a named volume carries actions, so it is a card. -->
        <section v-if="containerVolumes.length > 0" class="space-y-3">
          <div class="flex items-center gap-2">
            <h2 class="yantra-label">{{ t('containerDetail.storageVolumes') }}</h2>
            <span class="yantra-chip">{{ containerVolumes.length }}</span>
          </div>

          <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            <div v-for="volume in containerVolumes" :key="volume.name" class="yantra-surface flex h-full flex-col p-5">
              <div class="mb-4 flex items-start gap-4">
                <div class="yantra-tile h-10 w-10">
                  <HardDrive :size="17" />
                </div>
                <div class="min-w-0 flex-1">
                  <div class="truncate text-sm font-bold tracking-tight yantra-text-primary" :title="volume.name">{{ volume.name }}</div>
                  <div class="mt-1 truncate font-mono text-[11px] yantra-text-secondary" :title="volume.destination">{{ volume.destination }}</div>
                </div>
                <span class="yantra-chip shrink-0">{{ t('containerDetail.named') }}</span>
              </div>

              <div class="mt-auto flex flex-wrap items-center gap-2 border-t border-zinc-100 pt-4 dark:border-zinc-800">
                <div v-if="browsingVolume[volume.name]" class="yantra-btn cursor-default">
                  {{ t('containerDetail.startingWebDAV') }}
                </div>
                <template v-else-if="!showVolumeMenu[volume.name]">
                  <button @click="showVolumeMenu[volume.name] = true" class="yantra-btn">
                    {{ t('containerDetail.browseFiles') }}
                  </button>
                </template>
                <template v-else>
                  <button
                    @click="browseVolume(volume.name, 60)"
                    class="yantra-btn"
                    :title="t('containerDetail.oneHourAccess')"
                  >1H</button>
                  <button
                    @click="browseVolume(volume.name, 0)"
                    class="yantra-btn"
                    :title="t('containerDetail.permanentAccess')"
                  >{{ t('containerDetail.permanentAccess') }}</button>
                </template>
              </div>
            </div>
          </div>
        </section>

        <!-- Diagnostics: three tabs over one surface, not three nested cards. -->
        <section class="space-y-3">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <h2 class="yantra-label">{{ t('containerDetail.systemDiagnostics') }}</h2>
            <div class="yantra-segment">
              <button
                type="button"
                class="yantra-chip"
                :class="{ 'yantra-chip--active': activeTab === 'resources' }"
                @click="activeTab = 'resources'"
              >{{ t('containerDetail.resources') }}</button>
              <button
                type="button"
                class="yantra-chip"
                :class="{ 'yantra-chip--active': activeTab === 'output' }"
                @click="activeTab = 'output'"
              >{{ t('containerDetail.output') }}</button>
              <button
                type="button"
                class="yantra-chip"
                :class="{ 'yantra-chip--active': activeTab === 'env' }"
                @click="activeTab = 'env'"
              >{{ t('containerDetail.env') }}</button>
            </div>
          </div>

          <div class="yantra-surface p-6">
            <ContainerResources v-if="activeTab === 'resources'" :container-stats="containerStats" />

            <ContainerLogs
              v-else-if="activeTab === 'output'"
              :logs="containerLogs"
              v-model:autoScroll="autoScrollLogs"
              :refreshing="refreshingLogs"
              @refresh="fetchContainerLogs"
            />

            <ContainerEnv v-else :env="selectedContainer.env" />
          </div>
        </section>

    </main>
  </AppLayout>
</template>

<style scoped>
.scrollbar-thin::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}
.scrollbar-thin::-webkit-scrollbar-track {
  background: transparent;
}
.scrollbar-thin::-webkit-scrollbar-thumb {
  background: #424242;
  border-radius: 3px;
}
.scrollbar-thin::-webkit-scrollbar-thumb:hover {
  background: #4f4f4f;
}

/* The kebab is a native <details>, so it needs its marker hidden by hand. */
.kebab > summary {
  list-style: none;
}
.kebab > summary::-webkit-details-marker {
  display: none;
}
</style>
