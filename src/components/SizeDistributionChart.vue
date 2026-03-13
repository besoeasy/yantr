<script setup>
defineProps({
  items: { type: Array, default: () => [] },
  title: { type: String, required: true },
  legend: { type: Array, default: () => [] },
  // legend: [{ color: '#10b981', label: 'In Use' }]
  unit: { type: String, default: 'MB' }
})
</script>

<template>
  <div v-if="items.length > 0" class="bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 p-6">
    <div class="flex items-center justify-between mb-5">
      <h3 class="text-sm font-semibold tracking-tight text-gray-900 dark:text-white">{{ title }}</h3>
      <div class="flex items-center gap-4 text-[10px] font-bold uppercase tracking-[0.15em] text-gray-500 dark:text-zinc-500">
        <span v-for="leg in legend" :key="leg.label" class="flex items-center gap-1.5">
          <span class="w-2 h-2 rounded-full inline-block" :style="{ backgroundColor: leg.color }"></span>
          {{ leg.label }}
        </span>
      </div>
    </div>
    <div class="space-y-2.5">
      <div v-for="item in items" :key="item.name" class="flex items-center gap-3">
        <div class="w-36 shrink-0 text-xs font-mono text-gray-500 dark:text-zinc-400 truncate text-right" :title="item.name">{{ item.name }}</div>
        <div class="flex-1 bg-gray-100 dark:bg-zinc-900 rounded-full h-1.5 overflow-hidden">
          <div class="h-full rounded-full transition-all duration-500" :style="{ width: item.pct + '%', backgroundColor: item.color }"></div>
        </div>
        <div class="w-16 shrink-0 text-xs tabular-nums text-gray-500 dark:text-zinc-500 text-right">{{ item.size }} {{ unit }}</div>
      </div>
    </div>
  </div>
</template>
