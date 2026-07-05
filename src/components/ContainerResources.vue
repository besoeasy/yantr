<script setup>
import { Cpu, Activity } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { formatBytes } from '../utils/metrics'

const props = defineProps({
  containerStats: {
    type: Object,
    default: null
  }
})

const { t } = useI18n()
</script>

<template>
  <div v-if="containerStats" class="grid grid-cols-1 md:grid-cols-2 gap-4">
    <div class="bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 p-5 rounded-xl">
      <div class="flex items-center gap-2 text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 mb-3">
        <Cpu :size="12" /> {{ t('containerDetail.cpu') }}
      </div>
      <div class="text-3xl font-mono font-bold tracking-tighter text-gray-900 dark:text-white">
        {{ containerStats.cpu.percent }}%
      </div>
    </div>
    
    <div class="bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 p-5 rounded-xl">
      <div class="flex items-center gap-2 text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 mb-3">
        <Activity :size="12" /> {{ t('containerDetail.ram') }}
      </div>
      <div class="text-3xl font-mono font-bold tracking-tighter text-gray-900 dark:text-white">
        {{ containerStats.memory.percent }}%
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
