<script setup>
import { computed } from 'vue'
import { Sparkles, Activity } from 'lucide-vue-next'

const props = defineProps({
  runningApps: { type: Number, default: 0 },
  totalVolumes: { type: Number, default: 0 }
})

const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 12) return 'Good morning'
  if (hour < 18) return 'Good afternoon'
  return 'Good evening'
})
</script>

<template>
  <div class="relative h-full overflow-hidden group rounded-2xl transition-all duration-500 hover:-translate-y-1">
    <!-- Background Mesh -->
    <div class="absolute inset-0 bg-gray-900">
      <div class="absolute inset-0 bg-linear-to-br from-blue-600/25 via-purple-600/10 to-gray-900 z-10"></div>

      <!-- Orbs -->
      <div
        class="absolute top-0 right-0 w-64 h-64 bg-blue-500/20 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2 group-hover:bg-blue-500/30 transition-colors duration-700"
      ></div>
      <div
        class="absolute bottom-0 left-0 w-48 h-48 bg-purple-600/20 rounded-full blur-3xl translate-y-1/2 -translate-x-1/2 group-hover:bg-purple-600/30 transition-colors duration-700"
      ></div>
    </div>

    <!-- Content Container -->
    <div class="relative z-20 h-full p-6 flex flex-col justify-between border border-white/5 rounded-2xl backdrop-blur-sm group-hover:border-blue-500/30 transition-colors duration-500">
      <div class="flex items-start justify-between gap-4">
        <div class="flex items-center gap-4">
          <!-- Icon Container -->
          <div class="relative">
            <div class="absolute inset-0 bg-blue-500/20 rounded-xl blur-lg group-hover:blur-xl transition-all duration-500"></div>
            <div
              class="relative w-12 h-12 bg-linear-to-br from-blue-500 to-purple-600 rounded-xl flex items-center justify-center shadow-lg group-hover:scale-110 transition-transform duration-500"
            >
              <Sparkles class="w-6 h-6 text-white" />
            </div>
          </div>

          <div>
            <h1 class="text-lg sm:text-xl font-bold text-white mb-1 group-hover:text-blue-200 transition-colors">
              {{ greeting }}
            </h1>
            <p class="text-sm font-medium text-gray-400 group-hover:text-gray-300 transition-colors">
              Here's what's happening with your system today.
            </p>
          </div>
        </div>

        <!-- Status Dot -->
        <span class="relative flex h-3 w-3 mt-1">
          <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75"></span>
          <span class="relative inline-flex rounded-full h-3 w-3 bg-blue-500"></span>
        </span>
      </div>

      <div class="mt-5 grid grid-cols-2 gap-3">
        <div class="rounded-2xl border border-white/10 bg-white/5 px-4 py-3">
          <div class="text-3xl font-extrabold text-white leading-none tabular-nums">{{ props.runningApps }}</div>
          <div class="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mt-1">Running Apps</div>
        </div>
        <div class="rounded-2xl border border-white/10 bg-white/5 px-4 py-3">
          <div class="text-3xl font-extrabold text-white leading-none tabular-nums">{{ props.totalVolumes }}</div>
          <div class="text-[11px] font-semibold text-gray-400 uppercase tracking-wider mt-1">Volumes</div>
        </div>
      </div>

      <div class="mt-5 pt-4 border-t border-white/5 flex items-center justify-between group-hover:border-white/10 transition-colors">
        <div class="flex items-center gap-2 text-xs font-bold text-blue-300 uppercase tracking-wider">
          <Activity class="w-3.5 h-3.5" />
          <span>Overview</span>
        </div>
        <div class="text-sm font-bold text-white/90 group-hover:text-white transition-colors">
          Dashboard
        </div>
      </div>
    </div>
  </div>
</template>
