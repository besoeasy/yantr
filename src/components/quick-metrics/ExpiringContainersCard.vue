<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Timer, Hourglass, Zap } from 'lucide-vue-next'
import { formatDuration } from '../../utils/metrics.js'

const { t } = useI18n()
const props = defineProps({
  containers: { type: Array, default: () => [] },
  currentTime: { type: Number, default: () => Date.now() }
})

const stats = computed(() => {
  const tempContainers = props.containers
    .filter(c => c?.labels && c.labels['yantr.expireAt'])
    .map(c => {
      const expireAt = parseInt(c.labels['yantr.expireAt'], 10) * 1000
      const remainingMs = expireAt - props.currentTime
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
    class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl p-6 overflow-hidden transition-all duration-400 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:border-gray-300 dark:hover:border-zinc-600"
  >
    <!-- Hover Accents -->
    <div class="absolute top-0 left-0 w-full h-[2px] opacity-0 group-hover:opacity-100 transition-opacity duration-500"
         :class="isCritical ? 'bg-gradient-to-r from-transparent via-red-500 to-transparent' : 'bg-gradient-to-r from-transparent via-amber-500 to-transparent'">
    </div>
    <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none [mask-image:linear-gradient(to_bottom,white,transparent)]"></div>

    <!-- Header -->
    <div class="relative z-10 flex items-center justify-between mb-6">
      <div class="flex items-center gap-4">
        <div class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0 group-hover:scale-105 transition-all duration-500"
             :class="isCritical ? 'bg-red-50/50 dark:bg-red-500/10 border border-red-100 dark:border-red-500/20' : 'bg-amber-50/50 dark:bg-amber-500/10 border border-amber-100 dark:border-amber-500/20'">
          <Timer class="w-5 h-5" :class="isCritical ? 'text-red-600 dark:text-red-500' : 'text-amber-600 dark:text-amber-500'" />
        </div>
        
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight group-hover:text-amber-600 dark:group-hover:text-amber-500 transition-colors"
              :class="{'group-hover:text-red-600 dark:group-hover:text-red-500': isCritical}">
            {{ t('quickMetrics.expiringContainers.title') }}
          </h3>
          <div class="flex items-center gap-2 mt-1">
            <div class="w-1.5 h-1.5 rounded-full"
                 :class="[isCritical ? 'bg-red-500 animate-pulse' : 'bg-amber-500']">
            </div>
            <span class="text-[11px] font-medium text-gray-500 dark:text-zinc-400 uppercase tracking-wider">
              {{ urgencyLabel }}
            </span>
          </div>
        </div>
      </div>
      
      <div class="px-2 py-1 rounded-md bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 flex items-center gap-1.5">
         <span class="text-[11px] font-bold text-gray-900 dark:text-white">{{ stats.count }}</span>
         <span class="text-[10px] font-medium text-gray-500 dark:text-zinc-500 uppercase tracking-wider">{{ t('quickMetrics.expiringContainers.tracking') }}</span>
      </div>
    </div>

    <!-- Main Content -->
    <div class="relative z-10 flex-1 flex items-end justify-between gap-6 mt-2">
      <!-- Big Metric (Next to expire) -->
      <div class="flex-1 min-w-0">
         <div class="text-[10px] font-bold uppercase tracking-[0.2em] mb-2"
              :class="isCritical ? 'text-red-500/80 dark:text-red-500/60' : 'text-gray-400 dark:text-zinc-500'">
           {{ stats.next?.isExpired ? t('quickMetrics.expiringContainers.expiredFor') : t('quickMetrics.expiringContainers.expiresIn') }}
         </div>
         
         <div class="text-4xl font-bold tabular-nums tracking-tighter leading-none" 
              :class="[
                isCritical ? 'text-red-600 dark:text-red-500' : 'text-gray-900 dark:text-white',
                {'animate-pulse': isCritical}
              ]">
           {{ stats.next?.formatted.replace(' ago', '') }}
         </div>
         
         <div class="mt-3 flex items-center gap-1.5 text-[11px] font-medium text-gray-500 dark:text-zinc-400 truncate">
           <Hourglass class="w-3.5 h-3.5" :class="isCritical ? 'text-red-500' : 'text-amber-500'" />
           <span class="truncate">{{ t('quickMetrics.expiringContainers.next') }} <span class="font-semibold" :class="isCritical ? 'text-red-600 dark:text-red-400' : 'text-amber-600 dark:text-amber-400'">{{ stats.next?.name }}</span></span>
         </div>
      </div>

      <!-- Mini List (Next 3) -->
      <div class="shrink-0 w-32 flex flex-col gap-1.5">
         <div v-for="item in stats.items.slice(0, 3)" :key="item.id" 
              class="relative flex items-center justify-between gap-2 text-xs py-1 border-b border-gray-100 dark:border-zinc-800 last:border-0 group/item">
            <div class="truncate text-[11px] text-gray-600 dark:text-zinc-400 font-medium max-w-[60%] transition-colors group-hover/item:text-gray-900 dark:group-hover/item:text-white">
              {{ item.name }}
            </div>
            <div class="text-[11px] font-bold tabular-nums" 
                 :class="item.isUrgent || item.isExpired ? 'text-red-500 dark:text-red-400' : 'text-gray-500 dark:text-zinc-500'">
              {{ item.formatted.split(' ')[0] }}{{ item.formatted.split(' ')[1]?.charAt(0) || '' }}
            </div>
         </div>
         
         <div v-if="stats.count > 3" class="text-[10px] text-right font-medium text-gray-400 dark:text-zinc-600 pt-1">
           {{ t('quickMetrics.expiringContainers.more', { count: stats.count - 3 }) }}
         </div>
      </div>
    </div>
  </div>
</template>
