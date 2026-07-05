<script setup>
import { Terminal, Pause, Play, RefreshCw } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  logs: {
    type: Array,
    default: () => []
  },
  autoScroll: {
    type: Boolean,
    default: true
  },
  refreshing: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:autoScroll', 'refresh'])
const { t } = useI18n()
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2 text-[10px] font-bold text-gray-500 dark:text-zinc-500 uppercase tracking-widest">
        <Terminal :size="12" /> {{ t('containerDetail.outputConsole') }}
      </div>
      <div class="flex items-center gap-2">
        <button @click="emit('update:autoScroll', !autoScroll)" class="p-1.5 rounded-md hover:bg-gray-100 dark:hover:bg-zinc-800 text-gray-500 dark:text-zinc-400 transition-colors" :title="autoScroll ? t('containerDetail.pauseAutoScroll') : t('containerDetail.enableAutoScroll')">
          <component :is="autoScroll ? Pause : Play" :size="14" />
        </button>
        <button @click="emit('refresh')" class="p-1.5 rounded-md hover:bg-gray-100 dark:hover:bg-zinc-800 text-gray-500 dark:text-zinc-400 transition-colors" :title="refreshing ? t('containerDetail.refreshing') : t('common.refresh')">
          <RefreshCw :size="14" :class="{ 'animate-spin': refreshing }" />
        </button>
      </div>
    </div>
    
    <div 
      id="terminal-logs"
      class="h-96 overflow-y-auto p-4 font-mono text-[11px] leading-5 text-gray-800 dark:text-gray-300 bg-gray-50 dark:bg-[#111] border border-gray-200 dark:border-zinc-800 rounded-xl scrollbar-thin scrollbar-thumb-gray-300 dark:scrollbar-thumb-zinc-700 scrollbar-track-transparent"
    >
      <div v-if="logs.length === 0" class="flex flex-col items-center justify-center h-full text-gray-400 dark:text-zinc-600">
        <div class="mb-2 text-[10px] font-bold uppercase tracking-widest">{{ t('containerDetail.noOutputLogs') }}</div>
      </div>
      <div v-else class="space-y-0.5">
        <div v-for="(log, i) in logs" :key="i" class="break-all whitespace-pre-wrap hover:bg-gray-100 dark:hover:bg-zinc-900 px-1 -mx-1 rounded-sm">
          <span class="text-gray-400 dark:text-zinc-600 select-none mr-3 w-6 inline-block text-right">{{ i + 1 }}</span>{{ log }}
        </div>
      </div>
    </div>
  </div>
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
</style>
