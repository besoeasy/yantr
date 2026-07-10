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
    class="relative group h-full flex flex-col rounded-[24px] overflow-hidden transition-all duration-500 hover:-translate-y-2 text-left w-full min-h-[320px] border dark:border-white/5"
    :class="[
      isCritical
        ? 'hover:shadow-[0_20px_40px_-15px_rgba(239,68,68,0.4)] dark:hover:shadow-[0_20px_40px_-15px_rgba(239,68,68,0.3)] border-red-100/50 bg-gradient-to-br from-red-50/50 to-rose-50/50 dark:from-[#1a0f12] dark:to-[#0f090a]'
        : 'hover:shadow-[0_20px_40px_-15px_rgba(245,158,11,0.4)] dark:hover:shadow-[0_20px_40px_-15px_rgba(245,158,11,0.3)] border-amber-100/50 bg-gradient-to-br from-amber-50/50 to-orange-50/50 dark:from-[#1a150f] dark:to-[#0f0a09]'
    ]"
  >
    
    <!-- Animated Gradient Glow -->
    <div class="absolute -top-[50%] -left-[50%] w-[200%] h-[200%] opacity-0 group-hover:opacity-100 transition-opacity duration-1000 pointer-events-none group-hover:animate-[spin_20s_linear_infinite]"
         :class="isCritical ? 'bg-[radial-gradient(circle_at_50%_50%,_rgba(239,68,68,0.15)_0%,_transparent_50%)] dark:bg-[radial-gradient(circle_at_50%_50%,_rgba(239,68,68,0.12)_0%,_transparent_40%)]' : 'bg-[radial-gradient(circle_at_50%_50%,_rgba(245,158,11,0.15)_0%,_transparent_50%)] dark:bg-[radial-gradient(circle_at_50%_50%,_rgba(245,158,11,0.12)_0%,_transparent_40%)]'"
    ></div>

    <!-- Massive Watermark Icon -->
    <Timer class="absolute -bottom-8 -right-8 w-64 h-64 rotate-[-20deg] group-hover:rotate-0 group-hover:scale-110 transition-all duration-700 ease-out pointer-events-none" 
           :class="isCritical ? 'text-red-500/10 dark:text-red-400/[0.03]' : 'text-amber-500/10 dark:text-amber-400/[0.03]'"
    />

    <!-- Header -->
    <div class="relative z-10 p-7 flex flex-col h-full gap-4">
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-4">
          <!-- Gorgeous Icon Block -->
          <div class="relative w-14 h-14 rounded-2xl flex items-center justify-center shrink-0 shadow-xl transition-all duration-500 group-hover:scale-105"
               :class="isCritical ? 'bg-gradient-to-br from-red-400 to-rose-600 shadow-red-500/30 group-hover:shadow-red-500/50' : 'bg-gradient-to-br from-amber-400 to-orange-600 shadow-amber-500/30 group-hover:shadow-amber-500/50'">
            <div class="absolute inset-0 bg-white/30 dark:bg-white/20 rounded-2xl rounded-bl-none opacity-50 dark:opacity-40 mix-blend-overlay"></div>
            <Timer class="w-7 h-7 text-white drop-shadow-md z-10" />
          </div>
          <div>
            <h3 class="text-xl font-black tracking-tight leading-none transition-colors"
                :class="isCritical ? 'text-red-950 dark:text-white group-hover:text-red-600 dark:group-hover:text-red-400' : 'text-amber-950 dark:text-white group-hover:text-amber-600 dark:group-hover:text-amber-400'">
              {{ t('quickMetrics.expiringContainers.title') }}
            </h3>
            <div class="flex items-center gap-1.5 mt-2">
               <div class="w-1.5 h-1.5 rounded-full"
                    :class="isCritical ? 'bg-red-500 animate-pulse' : 'bg-amber-500'">
               </div>
               <span class="text-[10px] font-bold uppercase tracking-[0.2em]"
                     :class="isCritical ? 'text-red-600/70 dark:text-red-500' : 'text-amber-600/70 dark:text-amber-500'">
                 {{ urgencyLabel }}
               </span>
            </div>
          </div>
        </div>
        
        <div class="flex flex-col items-end">
           <span class="text-xl font-black" :class="isCritical ? 'text-red-900 dark:text-white' : 'text-amber-900 dark:text-white'">{{ stats.count }}</span>
           <span class="text-[9px] font-bold uppercase tracking-widest" :class="isCritical ? 'text-red-600/60 dark:text-red-500/60' : 'text-amber-600/60 dark:text-amber-500/60'">{{ t('quickMetrics.expiringContainers.tracking') }}</span>
        </div>
      </div>

      <!-- Main Content -->
      <div class="mt-auto pt-6 flex flex-col relative z-10">
        
        <div class="text-[10px] font-bold uppercase tracking-[0.2em] mb-2 drop-shadow-sm"
             :class="isCritical ? 'text-red-700/60 dark:text-red-500/60' : 'text-amber-700/60 dark:text-amber-500/60'">
          {{ stats.next?.isExpired ? t('quickMetrics.expiringContainers.expiredFor') : t('quickMetrics.expiringContainers.expiresIn') }}
        </div>
        <div class="text-5xl font-black tabular-nums tracking-tighter mb-4 drop-shadow-sm group-hover:-translate-y-1 transition-transform duration-500"
             :class="[
               isCritical ? 'text-red-950 dark:text-white' : 'text-amber-950 dark:text-white',
               {'animate-pulse': isCritical}
             ]">
          {{ stats.next?.formatted.replace(' ago', '') }}
        </div>

        <div class="flex items-center gap-1.5 text-xs font-bold uppercase tracking-widest mb-6"
             :class="isCritical ? 'text-red-600 dark:text-red-400' : 'text-amber-600 dark:text-amber-400'">
          <Hourglass class="w-4 h-4" />
          <span class="truncate">{{ t('quickMetrics.expiringContainers.next') }} <span class="text-black dark:text-white ml-1">{{ stats.next?.name }}</span></span>
        </div>

        <!-- Mini List Glass Container -->
        <div class="p-4 rounded-[16px] bg-white/60 dark:bg-black/40 backdrop-blur-xl border border-white dark:border-white/5 shadow-[0_4px_12px_rgba(0,0,0,0.02)] dark:shadow-sm">
           <div class="flex flex-col gap-2">
              <div v-for="item in stats.items.slice(0, 3)" :key="item.id" class="flex items-center justify-between text-xs font-bold group/item">
                 <span class="truncate pr-4 transition-colors duration-300" :class="isCritical ? 'text-red-900/60 dark:text-zinc-400 group-hover/item:text-red-900 dark:group-hover/item:text-white' : 'text-amber-900/60 dark:text-zinc-400 group-hover/item:text-amber-900 dark:group-hover/item:text-white'">
                   {{ item.name }}
                 </span>
                 <span class="tabular-nums whitespace-nowrap"
                       :class="item.isUrgent || item.isExpired ? 'text-red-500 dark:text-red-400' : 'text-gray-500 dark:text-zinc-500'">
                   {{ item.formatted }}
                 </span>
              </div>
           </div>
           
           <div v-if="stats.count > 3" class="text-[9px] text-right font-bold uppercase tracking-widest mt-3"
                :class="isCritical ? 'text-red-600/50 dark:text-red-500/50' : 'text-amber-600/50 dark:text-amber-500/50'">
             {{ t('quickMetrics.expiringContainers.more', { count: stats.count - 3 }) }}
           </div>
        </div>

      </div>
    </div>
  </div>
</template>
