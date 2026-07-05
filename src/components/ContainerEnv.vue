<script setup>
import { Lock } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  env: {
    type: Array,
    default: () => []
  }
})

const { t } = useI18n()
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center gap-2 text-[10px] font-bold text-gray-500 dark:text-zinc-500 uppercase tracking-widest">
      <Lock :size="12" /> {{ t('containerDetail.environmentVariables') }}
    </div>
    <div v-if="env && env.length > 0" class="bg-gray-50 dark:bg-[#111] border border-gray-200 dark:border-zinc-800 rounded-xl p-5 max-h-80 overflow-y-auto custom-scrollbar">
      <div v-for="(envVar, i) in env" :key="i" class="font-mono text-[11px] mb-3 last:mb-0 break-all flex flex-col sm:flex-row gap-1 sm:gap-4">
        <div class="text-gray-500 dark:text-zinc-500 font-bold shrink-0 sm:w-1/3">{{ envVar.split('=')[0] }}</div>
        <div class="text-gray-900 dark:text-zinc-300 flex-1">{{ envVar.split('=').slice(1).join('=') }}</div>
      </div>
    </div>
    <div v-else class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t('containerDetail.noEnvVars') }}</div>
  </div>
</template>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 3px;
}
</style>
