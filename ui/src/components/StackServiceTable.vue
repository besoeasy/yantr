<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ExternalLink, Network, Server } from '@lucide/vue'
import { containerDetail } from '../utils/routes.js'
import { appUrl, isNavigableProtocol } from '../utils/url'
import { groupStackServices } from '../utils/stackServices.js'

const props = defineProps({
  services: {
    type: Array,
    required: true
  },
  publishedPorts: {
    type: Array,
    default: () => []
  }
})

const router = useRouter()
const { t } = useI18n()

const groupedServices = computed(() => groupStackServices(props.services))

function serviceNames(group) {
  const names = new Set([group.serviceName])
  for (const svc of group.containers ?? []) {
    for (const name of [svc.service, svc.composeService, svc.name]) {
      if (name) names.add(name)
    }
  }
  return names
}

function portsForService(group) {
  const names = serviceNames(group)
  return props.publishedPorts.filter((port) => names.has(port.service))
}

function canOpenPort(port) {
  return port.show === true &&
    port.protocol?.toLowerCase() === 'tcp' &&
    port.hostPort &&
    isNavigableProtocol(port.displayProtocol)
}

function openPortsForService(group) {
  const names = serviceNames(group)
  return props.publishedPorts.filter((port) => names.has(port.service) && canOpenPort(port))
}

function portKey(port) {
  return `${port.hostPort}-${port.containerPort}-${port.protocol}-${port.service}`
}

function portLabel(port) {
  return `${port.label || port.displayProtocol || port.protocol} :${port.hostPort} → ${port.containerPort}`
}

function containerName(svc) {
  return svc.name || svc.id || ''
}

function containerShortId(svc) {
  return svc.id ? svc.id.substring(0, 12) : ''
}

function goToContainer(svcId) {
  if (!svcId) return
  router.push(containerDetail(svcId))
}

function goToService(group) {
  if (group.id) goToContainer(group.id)
}
</script>

<template>
  <section class="space-y-3 anim-enter">
    <div class="flex items-center gap-2">
      <div class="yantra-label">{{ t('stackView.services') }}</div>
      <span class="yantra-chip">{{ groupedServices.length }}</span>
    </div>

    <!-- Desktop: compact table. -->
    <div class="service-table overflow-x-auto rounded-xl border border-zinc-100 dark:border-zinc-800">
      <table class="w-full border-collapse text-left text-xs">
        <thead>
          <tr class="border-b border-zinc-100 dark:border-zinc-800">
            <th class="px-4 py-3 font-bold uppercase tracking-wider yantra-text-secondary">{{ t('stackView.service') }}</th>
            <th class="px-4 py-3 font-bold uppercase tracking-wider yantra-text-secondary">{{ t('stackView.image') }}</th>
            <th class="px-4 py-3 font-bold uppercase tracking-wider yantra-text-secondary">{{ t('stackView.state') }}</th>
            <th class="px-4 py-3 font-bold uppercase tracking-wider yantra-text-secondary">{{ t('stackView.ports') }}</th>
            <th class="px-4 py-3 font-bold uppercase tracking-wider yantra-text-secondary">{{ t('stackView.container') }}</th>
            <th class="px-4 py-3 font-bold uppercase tracking-wider yantra-text-secondary">{{ t('stackView.action') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="group in groupedServices"
            :key="group.key"
            class="cursor-pointer border-b border-zinc-100 hover:bg-gray-50 dark:border-zinc-800 dark:hover:bg-zinc-900/50"
            tabindex="0"
            @click="goToService(group)"
            @keydown.enter.prevent="goToService(group)"
            @keydown.space.prevent="goToService(group)"
          >
            <td class="min-w-44 px-4 py-3">
              <div class="flex items-center gap-2">
                <div class="yantra-tile relative h-8 w-8 shrink-0" :class="{ 'yantra-tile--ok': group.state === 'running' }">
                  <Server :size="14" />
                  <span v-if="group.state === 'running'" class="absolute -top-1 -right-1 flex h-2.5 w-2.5">
                    <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                    <span class="relative inline-flex rounded-full h-2.5 w-2.5 bg-green-500"></span>
                  </span>
                </div>
                <div class="min-w-0">
                  <div class="truncate text-sm font-bold yantra-text-primary">{{ group.serviceName }}</div>
                  <span v-if="group.hasYantrLabel" class="yantra-chip yantra-chip--accent mt-1">{{ t('stackView.primary') }}</span>
                </div>
              </div>
            </td>

            <td class="min-w-44 max-w-64 px-4 py-3">
              <div class="truncate font-mono yantra-text-secondary" :title="group.images.join(', ')">{{ group.image }}</div>
              <div v-if="group.images.length > 1" class="mt-1 text-[10px] yantra-text-secondary">+{{ group.images.length - 1 }} {{ t('stackView.image') }}</div>
            </td>

            <td class="px-4 py-3">
              <span class="yantra-chip" :class="{ 'yantra-chip--ok': group.state === 'running', 'yantra-chip--warn': group.state === 'partial' }">{{ group.state }}</span>
            </td>

            <td class="min-w-44 px-4 py-3">
              <div v-if="portsForService(group).length > 0" class="flex flex-wrap gap-1.5">
                <span
                  v-for="port in portsForService(group)"
                  :key="portKey(port)"
                  class="yantra-chip font-mono normal-case"
                  :class="{ 'yantra-chip--ok': port.show === true }"
                  :title="portLabel(port)"
                >
                  <Network :size="11" />:{{ port.hostPort }} → {{ port.containerPort }}
                </span>
              </div>
              <span v-else class="yantra-text-secondary">{{ t('containerDetail.internalOnly') }}</span>
            </td>

            <td class="min-w-44 px-4 py-3">
              <div v-if="group.containers.length === 1" class="truncate font-mono yantra-text-secondary" :title="containerName(group.containers[0])">
                {{ containerName(group.containers[0]) }}
              </div>
              <div v-if="group.containers.length === 1 && containerShortId(group.containers[0])" class="truncate font-mono text-[10px] yantra-text-secondary">
                {{ containerShortId(group.containers[0]) }}
              </div>
              <div v-else-if="group.containers.length > 1" class="font-mono yantra-text-secondary">
                {{ t('stackView.containersCount', { count: group.containers.length }) }}
              </div>
              <ul v-if="group.containers.length > 1" class="mt-1 space-y-1">
                <li v-for="svc in group.containers" :key="svc.id" class="truncate font-mono text-[10px] yantra-text-secondary" :title="containerName(svc)">
                  {{ containerName(svc) }}
                </li>
              </ul>
            </td>

            <td class="min-w-44 px-4 py-3" @click.stop @keydown.enter.stop @keydown.space.stop>
              <div class="flex flex-wrap gap-1.5">
                <a
                  v-for="port in openPortsForService(group)"
                  :key="`open-${portKey(port)}`"
                  :href="appUrl(port.hostPort, port.displayProtocol)"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="yantra-btn yantra-btn--primary"
                  @click.stop
                >
                  <ExternalLink :size="12" />
                  {{ t('containerDetail.openLabel', { label: port.label || t('containerDetail.networkPort') }) }}
                </a>

                <template v-if="group.containers.length === 1">
                  <button type="button" class="yantra-btn" @click.stop="goToContainer(group.containers[0].id)">
                    <ExternalLink :size="12" />{{ t('stackView.serviceDetails') }}
                  </button>
                </template>
                <ul v-else class="space-y-1">
                  <li v-for="svc in group.containers" :key="svc.id">
                    <button type="button" class="yantra-btn" @click.stop="goToContainer(svc.id)">
                      <ExternalLink :size="12" />{{ t('stackView.serviceDetails') }}
                    </button>
                  </li>
                </ul>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Mobile: compact service cards. -->
    <div class="service-cards">
      <article
        v-for="group in groupedServices"
        :key="`card-${group.key}`"
        class="yantra-surface p-4"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="flex min-w-0 items-center gap-2">
            <div class="yantra-tile relative h-8 w-8 shrink-0" :class="{ 'yantra-tile--ok': group.state === 'running' }">
              <Server :size="14" />
              <span v-if="group.state === 'running'" class="absolute -top-1 -right-1 flex h-2.5 w-2.5">
                <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                <span class="relative inline-flex rounded-full h-2.5 w-2.5 bg-green-500"></span>
              </span>
            </div>
            <div class="min-w-0">
              <div class="truncate text-sm font-bold yantra-text-primary">{{ group.serviceName }}</div>
              <span v-if="group.hasYantrLabel" class="yantra-chip yantra-chip--accent mt-1">{{ t('stackView.primary') }}</span>
            </div>
          </div>
          <span class="yantra-chip shrink-0" :class="{ 'yantra-chip--ok': group.state === 'running', 'yantra-chip--warn': group.state === 'partial' }">{{ group.state }}</span>
        </div>

        <div class="mt-3 truncate font-mono text-[11px] yantra-text-secondary" :title="group.images.join(', ')">{{ group.image }}</div>
        <div v-if="group.images.length > 1" class="text-[10px] yantra-text-secondary">+{{ group.images.length - 1 }} {{ t('stackView.image') }}</div>

        <div class="mt-3 flex flex-wrap gap-1.5">
          <template v-if="portsForService(group).length > 0">
            <span
              v-for="port in portsForService(group)"
              :key="`card-port-${portKey(port)}`"
              class="yantra-chip font-mono normal-case"
              :class="{ 'yantra-chip--ok': port.show === true }"
              :title="portLabel(port)"
            >
              <Network :size="11" />:{{ port.hostPort }} → {{ port.containerPort }}
            </span>
          </template>
          <span v-else class="yantra-text-secondary">{{ t('containerDetail.internalOnly') }}</span>
        </div>

        <div class="mt-3 font-mono text-[10px] yantra-text-secondary">
          <span v-if="group.containers.length === 1" class="truncate" :title="containerName(group.containers[0])">{{ containerName(group.containers[0]) }}</span>
          <span v-else>{{ t('stackView.containersCount', { count: group.containers.length }) }}</span>
        </div>

        <div class="mt-3 flex flex-wrap gap-1.5">
          <a
            v-for="port in openPortsForService(group)"
            :key="`card-open-${portKey(port)}`"
            :href="appUrl(port.hostPort, port.displayProtocol)"
            target="_blank"
            rel="noopener noreferrer"
            class="yantra-btn yantra-btn--primary"
          >
            <ExternalLink :size="12" />
            {{ t('containerDetail.openLabel', { label: port.label || t('containerDetail.networkPort') }) }}
          </a>

          <button
            v-if="group.containers.length === 1"
            type="button"
            class="yantra-btn"
            @click="goToContainer(group.containers[0].id)"
          >
            <ExternalLink :size="12" />{{ t('stackView.serviceDetails') }}
          </button>
        </div>

        <ul v-if="group.containers.length > 1" class="mt-3 space-y-1">
          <li v-for="svc in group.containers" :key="svc.id">
            <button type="button" class="yantra-btn w-full" @click="goToContainer(svc.id)">
              <ExternalLink :size="12" />
              <span class="truncate">{{ containerName(svc) }}</span>
            </button>
          </li>
        </ul>
      </article>
    </div>
  </section>
</template>

<style scoped>
.service-table {
  display: none;
}

.service-cards {
  display: grid;
  gap: 0.75rem;
}

@media (min-width: 40rem) {
  .service-table {
    display: block;
  }

  .service-cards {
    display: none;
  }
}
</style>
