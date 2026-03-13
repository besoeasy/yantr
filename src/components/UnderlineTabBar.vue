<script setup>
defineProps({
  tabs: { type: Array, required: true },
  // each tab: { key: string, label: string, count?: number }
  modelValue: { type: String, required: true }
})

defineEmits(['update:modelValue'])
</script>

<template>
  <div class="flex items-center justify-between border-b border-gray-200 dark:border-zinc-800 pb-1">
    <div class="flex gap-6">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        @click="$emit('update:modelValue', tab.key)"
        :class="modelValue === tab.key
          ? 'text-gray-900 dark:text-white border-gray-900 dark:border-white'
          : 'text-gray-500 dark:text-zinc-500 border-transparent hover:text-gray-700 dark:hover:text-zinc-300'"
        class="pb-3 text-sm font-semibold tracking-tight border-b-2 transition-colors flex items-center gap-2"
      >
        {{ tab.label }}
        <span v-if="tab.count !== undefined" class="bg-gray-100 dark:bg-zinc-800 text-gray-600 dark:text-zinc-400 text-xs px-2 py-0.5 rounded-full font-bold tabular-nums">{{ tab.count }}</span>
      </button>
    </div>
    <slot name="action" />
  </div>
</template>
