<script setup>
import { ref, watch } from 'vue'
import { Cpu, Activity } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { formatBytes } from '../utils/metrics'
import { createCpuRateCalculator } from '../utils/cpu'

const props = defineProps({
  containerStats: {
    type: Object,
    default: null
  }
})

const { t } = useI18n()

// CPU percent is derived here, from two consecutive samples, because the server
// cannot compute it: a single-shot podman stats request leaves precpu_stats all
// zero, so a server-side percentage would be a lifetime average, not a rate.
// See core/handlers_containers.go and ui/src/utils/cpu.js.
const rate = createCpuRateCalculator()
const cpuPercent = ref(null)

// The baseline is per-container: switching services must not difference two
// unrelated counters.
watch(
  () => props.containerStats,
  (stats) => {
    if (!stats) {
      rate.reset()
      cpuPercent.value = null
      return
    }
    cpuPercent.value = rate.push(stats.cpu)
  },
  { immediate: true }
)
</script>

<template>
  <div v-if="containerStats" class="grid grid-cols-1 md:grid-cols-2 gap-4">
    <div class="bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 p-5 rounded-xl">
      <div class="flex items-center gap-2 text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 mb-3">
        <Cpu :size="12" /> {{ t('containerDetail.cpu') }}
      </div>
      <div class="text-3xl font-mono font-bold tracking-tighter text-gray-900 dark:text-white">
        <template v-if="cpuPercent !== null">{{ cpuPercent }}%</template>
        <span v-else class="text-gray-400 dark:text-zinc-600" :title="t('containerDetail.cpuWarmingUp')">&mdash;</span>
      </div>
    </div>

    <div class="bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 p-5 rounded-xl">
      <div class="flex items-center gap-2 text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 mb-3">
        <Activity :size="12" /> {{ t('containerDetail.ram') }}
      </div>
      <div class="text-3xl font-mono font-bold tracking-tighter text-gray-900 dark:text-white">
        {{ formatBytes(containerStats.memory.usage) }}
      </div>
      <!-- A percentage is only meaningful against a limit the container is
           actually held to. When none is set, Podman reports host RAM as the
           limit, which pins every container at 0.00%. -->
      <div v-if="!containerStats.memory.unlimited" class="mt-1 text-[11px] text-gray-500 dark:text-zinc-400">
        {{ t('containerDetail.ramOfLimit', { percent: containerStats.memory.percent, limit: formatBytes(containerStats.memory.limit) }) }}
      </div>
    </div>

    <div class="md:col-span-2 bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 p-5 rounded-xl flex flex-col sm:flex-row sm:justify-between sm:items-center gap-4">
      <div>
        <div class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 mb-1.5">{{ t('containerDetail.networkIO') }}</div>
        <div class="text-sm font-mono font-semibold text-gray-900 dark:text-white">
          <span class="text-green-600 dark:text-green-500">↓ {{ formatBytes(containerStats.network.rx) }}</span>
          <span class="text-gray-300 dark:text-zinc-700 mx-3">|</span>
          <span class="text-blue-600 dark:text-blue-500">↑ {{ formatBytes(containerStats.network.tx) }}</span>
        </div>
      </div>
      <div class="sm:text-right">
        <div class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 mb-1.5">{{ t('containerDetail.blockIO') }}</div>
        <div class="text-sm font-mono font-semibold text-gray-900 dark:text-white">
          {{ formatBytes(containerStats.blockIO.read) }} / {{ formatBytes(containerStats.blockIO.write) }}
        </div>
      </div>
    </div>
  </div>
  <div v-else class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t('containerDetail.noResourceData') }}</div>
</template>
