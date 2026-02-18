<script setup>
import { computed } from 'vue'
import { Timer, Hourglass } from 'lucide-vue-next'
import { formatDuration } from '../../utils/metrics.js'

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

const theme = computed(() => {
  if (stats.value.next?.isUrgent || stats.value.next?.isExpired) {
    return {
      text: 'text-rose-600 dark:text-rose-400',
      bg: 'bg-rose-500/10 dark:bg-rose-500/20',
      border: 'group-hover:border-rose-500/30 dark:group-hover:border-rose-400/30',
      ring: 'text-rose-500 dark:text-rose-400',
      bar: 'bg-rose-500 dark:bg-rose-400'
    }
  }
  return {
    text: 'text-amber-600 dark:text-amber-400',
    bg: 'bg-amber-500/10 dark:bg-amber-500/20',
    border: 'group-hover:border-amber-500/30 dark:group-hover:border-amber-400/30',
    ring: 'text-amber-500 dark:text-amber-400',
    bar: 'bg-amber-500 dark:bg-amber-400'
  }
})

const urgencyLabel = computed(() => {
  const next = stats.value.next
  if (!next) return '✅ Safe'
  if (next.isExpired) return '💀 Expired'
  if (next.isUrgent) return '🚨 Critical'
  return '⏰ Upcoming'
})
</script>

<template>
  <div
    v-if="stats.count > 0"
    class="relative h-full overflow-hidden group rounded-none transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)] hover:-translate-y-1"
  >
    <!-- Animated Background Mesh -->
    <div class="absolute inset-0 bg-white dark:bg-gray-900">
      <div 
        class="absolute inset-0 z-10"
        :class="stats.next?.isUrgent || stats.next?.isExpired 
          ? 'bg-gradient-to-br from-orange-200/70 via-pink-200/40 to-white/80 dark:from-orange-600/25 dark:via-pink-600/15 dark:to-gray-900'
          : 'bg-gradient-to-br from-orange-200/60 via-yellow-200/35 to-white/80 dark:from-orange-600/20 dark:via-yellow-600/12 dark:to-gray-900'"
      ></div>
      
      <!-- Animated Orbs -->
      <div 
        class="absolute top-0 right-0 w-64 h-64 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2 transition-colors duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"
        :class="stats.next?.isUrgent || stats.next?.isExpired
          ? 'bg-orange-400/40 dark:bg-orange-500/25 group-hover:bg-orange-500/50 dark:group-hover:bg-orange-500/35'
          : 'bg-orange-300/35 dark:bg-orange-500/20 group-hover:bg-orange-400/45 dark:group-hover:bg-orange-500/30'"
      ></div>
      <div 
        class="absolute bottom-0 left-0 w-48 h-48 rounded-full blur-3xl translate-y-1/2 -translate-x-1/2 transition-colors duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"
        :class="stats.next?.isUrgent || stats.next?.isExpired
          ? 'bg-pink-400/35 dark:bg-pink-600/25 group-hover:bg-pink-500/45 dark:group-hover:bg-pink-600/35'
          : 'bg-yellow-300/30 dark:bg-yellow-600/20 group-hover:bg-yellow-400/40 dark:group-hover:bg-yellow-600/30'"
      ></div>
    </div>

    <!-- Animated Border Gradient -->
    <div class="absolute inset-0 z-10 opacity-0 group-hover:opacity-100 transition-opacity duration-500">
      <div 
        class="absolute inset-0 animate-spin-slow"
        :class="stats.next?.isUrgent || stats.next?.isExpired
          ? 'bg-gradient-to-r from-orange-500/0 via-pink-500/70 to-orange-500/0'
          : 'bg-gradient-to-r from-orange-500/0 via-orange-500/60 to-orange-500/0'"
        style="animation-duration: 3s;"
      ></div>
    </div>

    <div class="relative z-20 h-full p-6 flex flex-col justify-between border backdrop-blur-sm transition-all duration-300"
         :class="[
           'border-slate-200/80 dark:border-slate-700/60',
           stats.next?.isUrgent || stats.next?.isExpired 
             ? 'group-hover:border-pink-400/50 dark:group-hover:border-pink-500/40 group-hover:shadow-[0_0_20px_rgba(236,72,153,0.3)]'
             : 'group-hover:border-orange-400/50 dark:group-hover:border-orange-500/40 group-hover:shadow-[0_0_20px_rgba(251,146,60,0.3)]'
         ]">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="p-2.5 rounded-xl transition-colors duration-300" :class="theme.bg">
            <Timer class="w-5 h-5 transition-transform duration-700 ease-in-out group-hover:-rotate-12" :class="theme.text" />
          </div>
          <div>
            <h3 class="text-sm font-bold text-slate-900 dark:text-white uppercase tracking-wider">⏳ Expirations</h3>
            <div class="flex items-center gap-1.5 mt-0.5">
              <span class="relative flex h-2 w-2">
                <span v-if="stats.next?.isUrgent" class="animate-ping absolute inline-flex h-full w-full rounded-full opacity-75 bg-rose-400"></span>
                <span class="relative inline-flex rounded-full h-2 w-2" :class="stats.next?.isUrgent ? 'bg-rose-500' : 'bg-amber-500'"></span>
              </span>
              <span class="text-xs font-medium text-slate-500 dark:text-slate-400">{{ urgencyLabel }}</span>
            </div>
          </div>
        </div>
        
        <div class="px-2.5 py-1 rounded-lg bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-700">
           <span class="text-xs font-bold text-slate-700 dark:text-slate-300">{{ stats.count }}</span>
           <span class="text-[10px] text-slate-500 dark:text-slate-500 ml-1 font-medium">Tracking</span>
        </div>
      </div>

      <!-- Main Content -->
      <div class="flex items-end justify-between gap-6 mt-6">
        <!-- Big Metric (Next to expire) -->
        <div class="flex-1 min-w-0">
           <div class="text-xs font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-wide mb-1">
             {{ stats.next?.isExpired ? '💀 Expired For' : '⏰ Expires In' }}
           </div>
           
           <div class="text-3xl sm:text-4xl font-black tabular-nums tracking-tight leading-none" 
                :class="[
                  stats.next?.isUrgent || stats.next?.isExpired ? 'text-rose-600 dark:text-rose-400' : 'text-slate-900 dark:text-white',
                  {'animate-pulse': stats.next?.isUrgent || stats.next?.isExpired}
                ]">
             {{ stats.next?.formatted.replace(' ago', '') }}
           </div>
           
           <div class="mt-2 text-xs font-medium text-slate-500 dark:text-slate-400 flex items-center gap-1.5 truncate">
             <Hourglass class="w-3.5 h-3.5" />
             <span class="truncate">🎯 Next: <span :class="theme.text">{{ stats.next?.name }}</span></span>
           </div>
        </div>

        <!-- Mini List (Next 3) -->
        <div class="shrink-0 w-32 flex flex-col gap-2">
           <div v-for="item in stats.items.slice(0, 3)" :key="item.id" 
                class="relative group/item flex items-center justify-between gap-2 text-xs py-1 border-b border-slate-100 dark:border-slate-800 last:border-0">
              <div class="truncate text-slate-600 dark:text-slate-400 font-medium max-w-[60%] opacity-90 transition-opacity group-hover/item:opacity-100">
                {{ item.name }}
              </div>
              <div class="font-bold tabular-nums" 
                   :class="item.isUrgent || item.isExpired ? 'text-rose-500 dark:text-rose-400' : 'text-slate-500 dark:text-slate-500'">
                {{ item.formatted.split(' ')[0] }}{{ item.formatted.split(' ')[1]?.charAt(0) || '' }}
              </div>
              
              <!-- Hover tooltip indicator could go here -->
           </div>
           
           <div v-if="stats.count > 3" class="text-[10px] text-center text-slate-400 font-medium pt-1">
             +{{ stats.count - 3 }} others
           </div>
        </div>
      </div>
    </div>
  </div>
</template>
