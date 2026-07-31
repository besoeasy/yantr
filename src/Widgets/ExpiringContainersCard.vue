<script setup>
import { ref, computed, onMounted, onUnmounted} from 'vue'
import { useI18n } from 'vue-i18n'
import { Timer, Hourglass, Zap } from '@lucide/vue'
import { formatDuration } from '../utils/metrics.js'
import { useApiUrl } from '../composables/useApiUrl'
import { expectApiSuccess } from '../composables/useApiResponse'
import { useCurrentTime } from '../composables/useCurrentTime'

const { t } = useI18n()
const { apiUrl } = useApiUrl()
const { currentTime } = useCurrentTime()

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

const stats = computed(() => {
  const tempContainers = containers.value
    .filter(c => c?.labels && c.labels['yantr.expireAt'])
    .map(c => {
      const expireAt = parseInt(c.labels['yantr.expireAt'], 10) * 1000
      const remainingMs = expireAt - currentTime.value
      return {
        id: c.id,
        name: c?.app?.name || c?.name || 'Unknown',
        expireAt,
        remainingMs,
        formatted: formatDuration(Math.abs(remainingMs)),
        isExpired: remainingMs <= 0,
        isUrgent: remainingMs > 0 && remainingMs < (60 * 60 * 1000) // < 1 hour
      }
    })
    .sort((a, b) => a.remainingMs - b.remainingMs)

  if (tempContainers.length === 0) {
    return { count: 0, items: [], next: null }
  }

  return {
    count: tempContainers.length,
    items: tempContainers,
    next: tempContainers[0]
  }
})

const isCritical = computed(() => stats.value.next?.isUrgent || stats.value.next?.isExpired)

const urgencyLabel = computed(() => {
  const next = stats.value.next
  if (!next) return t('quickMetrics.expiringContainers.safe')
  if (next.isExpired) return t('quickMetrics.expiringContainers.expired')
  if (next.isUrgent) return t('quickMetrics.expiringContainers.critical')
  return t('quickMetrics.expiringContainers.upcoming')
})
</script>

<template>
  <div
    v-if="stats.count > 0"
    class="group relative flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl border bg-white p-7 text-left transition-all duration-300 hover:-translate-y-1 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:bg-black dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
    :class="isCritical 
        ? 'border-red-200 hover:border-red-300 dark:border-red-900/50 dark:hover:border-red-800' 
        : 'border-zinc-200 hover:border-zinc-300 dark:border-zinc-800 dark:hover:border-zinc-700'"
  >
    <!-- Header -->
    <div class="relative z-10 flex flex-col gap-4">
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Minimal Icon Block -->
          <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border transition-transform duration-300 group-hover:scale-105"
               :class="isCritical ? 'border-red-200 bg-red-50 dark:border-red-900/50 dark:bg-red-900/20' : 'border-amber-200 bg-amber-50 dark:border-amber-900/50 dark:bg-amber-900/20'">
            <Timer class="h-5 w-5" :class="isCritical ? 'text-red-500' : 'text-amber-600 dark:text-amber-400'" />
          </div>
          <div>
            <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors group-hover:text-black dark:text-white dark:group-hover:text-white">
              {{ t('quickMetrics.expiringContainers.title') }}
            </h3>
            <div class="mt-2 flex items-center gap-1.5">
               <div v-if="isCritical" class="relative h-2 w-2">
                 <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-red-400 opacity-75"></span>
                 <span class="relative inline-flex h-2 w-2 rounded-full bg-red-500"></span>
               </div>
               <div v-else class="h-2 w-2 rounded-full bg-amber-500"></div>
               <span class="text-[10px] font-semibold uppercase tracking-widest"
                     :class="isCritical ? 'text-red-600 dark:text-red-500' : 'text-amber-600 dark:text-amber-500'">
                 {{ urgencyLabel }}
               </span>
            </div>
          </div>
        </div>
        
        <div class="flex flex-col items-end">
           <span class="text-lg font-black" :class="isCritical ? 'text-red-600 dark:text-red-500' : 'text-zinc-900 dark:text-white'">{{ stats.count }}</span>
           <span class="text-[9px] font-semibold uppercase tracking-widest text-zinc-500">{{ t('quickMetrics.expiringContainers.tracking') }}</span>
        </div>
      </div>
    </div>

    <!-- Main Content -->
    <div class="relative z-10 mt-auto flex flex-col pt-6">
      
      <div class="mb-1 text-[10px] font-semibold uppercase tracking-widest text-zinc-500">
        {{ stats.next?.isExpired ? t('quickMetrics.expiringContainers.expiredFor') : t('quickMetrics.expiringContainers.expiresIn') }}
      </div>
      <div class="mb-4 tabular-nums text-4xl font-black tracking-tighter transition-transform duration-500 group-hover:-translate-y-1 sm:text-5xl"
           :class="[
             isCritical ? 'text-red-500 dark:text-red-400' : 'text-zinc-900 dark:text-white',
             {'animate-pulse': isCritical}
           ]">
        {{ stats.next?.formatted.replace(' ago', '') }}
      </div>

      <div class="mb-6 flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-widest text-zinc-500">
        <Hourglass class="h-3.5 w-3.5" />
        <span class="truncate">{{ t('quickMetrics.expiringContainers.next') }} <span class="ml-1 text-zinc-900 dark:text-white">{{ stats.next?.name }}</span></span>
      </div>

      <!-- Mini List Container -->
      <div class="rounded-xl border border-zinc-200 bg-zinc-50 p-4 transition-colors duration-300 hover:bg-zinc-100 dark:border-zinc-800 dark:bg-zinc-900/50 dark:hover:bg-zinc-900">
         <div class="flex flex-col gap-2">
            <div v-for="item in stats.items.slice(0, 3)" :key="item.id" class="group/item flex items-center justify-between text-xs font-semibold">
               <span class="truncate pr-4 text-zinc-500 transition-colors duration-300 group-hover/item:text-zinc-900 dark:text-zinc-400 dark:group-hover/item:text-white">
                 {{ item.name }}
               </span>
               <span class="whitespace-nowrap tabular-nums font-bold"
                     :class="item.isUrgent || item.isExpired ? 'text-red-500 dark:text-red-400' : 'text-zinc-900 dark:text-white'">
                 {{ item.formatted }}
               </span>
            </div>
         </div>
         
         <div v-if="stats.count > 3" class="mt-3 text-right text-[9px] font-semibold uppercase tracking-widest text-zinc-400 dark:text-zinc-500">
           {{ t('quickMetrics.expiringContainers.more', { count: stats.count - 3 }) }}
         </div>
      </div>

    </div>
  </div>
</template>
