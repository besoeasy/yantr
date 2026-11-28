<script setup>
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ExternalLink, Network, Server } from '@lucide/vue'
import { containerDetail } from '../utils/routes.js'
import { appUrl, isNavigableProtocol } from '../utils/url'

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

function serviceName(svc) {
  return svc.service || svc.composeService || svc.name || ''
}

function containerName(svc) {
  return svc.name || svc.id || ''
}

function serviceNames(svc) {
  return new Set([svc.service, svc.composeService, svc.name].filter(Boolean))
}

function portsForService(svc) {
  const names = serviceNames(svc)
  return props.publishedPorts.filter((port) => names.has(port.service))
}

function canOpenPort(port) {
  return port.show === true &&
    port.protocol?.toLowerCase() === 'tcp' &&
    port.hostPort &&
    isNavigableProtocol(port.displayProtocol)
}

function openPortsForService(svc) {
  const names = serviceNames(svc)
  return props.publishedPorts.filter((port) => names.has(port.service) && canOpenPort(port))
}

function portKey(port) {
  return `${port.hostPort}-${port.containerPort}-${port.protocol}-${port.service}`
}

function portLabel(port) {
  return `${port.label || port.displayProtocol || port.protocol} :${port.hostPort} → ${port.containerPort}`
}

function goToContainer(svcId) {
  router.push(containerDetail(svcId))
}
</script>

<template>
  <section class="space-y-3 anim-enter">
    <div class="flex items-center gap-2">
      <div class="yantra-label">{{ t('stackView.services') }}</div>
      <span class="yantra-chip">{{ services.length }}</span>
    </div>

    <div class="overflow-x-auto rounded-xl border border-zinc-100 dark:border-zinc-800">
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
            v-for="svc in services"
            :key="svc.id"
            class="cursor-pointer border-b border-zinc-100 hover:bg-gray-50 dark:border-zinc-800 dark:hover:bg-zinc-900/50"
            tabindex="0"
            @click="goToContainer(svc.id)"
            @keydown.enter.prevent="goToContainer(svc.id)"
            @keydown.space.prevent="goToContainer(svc.id)"
          >
            <td class="min-w-44 px-4 py-3">
              <div class="flex items-center gap-2">
                <div class="yantra-tile relative h-8 w-8 shrink-0" :class="{ 'yantra-tile--ok': svc.state === 'running' }">
                  <Server :size="14" />
                  <span v-if="svc.state === 'running'" class="absolute -top-1 -right-1 flex h-2.5 w-2.5">
                    <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                    <span class="relative inline-flex rounded-full h-2.5 w-2.5 bg-green-500"></span>
                  </span>
                </div>
                <div class="min-w-0">
                  <div class="truncate text-sm font-bold yantra-text-primary">{{ serviceName(svc) }}</div>
                  <span v-if="svc.hasYantrLabel" class="yantra-chip yantra-chip--accent mt-1">{{ t('stackView.primary') }}</span>
                </div>
              </div>
            </td>

            <td class="min-w-44 max-w-64 px-4 py-3">
              <div class="truncate font-mono yantra-text-secondary" :title="svc.image">{{ svc.image }}</div>
            </td>

            <td class="px-4 py-3">
              <span class="yantra-chip" :class="{ 'yantra-chip--ok': svc.state === 'running' }">{{ svc.state }}</span>
            </td>

            <td class="min-w-44 px-4 py-3">
              <div v-if="portsForService(svc).length > 0" class="flex flex-wrap gap-1.5">
                <span
                  v-for="port in portsForService(svc)"
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
              <div class="truncate font-mono yantra-text-secondary" :title="containerName(svc)">{{ containerName(svc) }}</div>
              <div v-if="svc.id" class="truncate font-mono text-[10px] yantra-text-secondary" :title="svc.id">{{ svc.id }}</div>
            </td>

            <td class="min-w-44 px-4 py-3" @click.stop @keydown.enter.stop @keydown.space.stop>
              <div class="flex flex-wrap gap-1.5">
                <a
                  v-for="port in openPortsForService(svc)"
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

                <button type="button" class="yantra-btn" @click.stop="goToContainer(svc.id)">
                  <ExternalLink :size="12" />{{ t('stackView.serviceDetails') }}
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
