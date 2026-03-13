<script setup>
import { computed } from 'vue'

const props = defineProps({
  label: { type: String, required: true },
  value: { type: [String, Number], required: true },
  unit: { type: String, default: '' },
  accent: { type: String, default: 'blue' },
  icon: { type: Object, default: null }
})

// Static strings so Tailwind keeps all the needed utility classes
const accentMap = {
  blue:        { gradient: 'via-blue-500',   icon: 'group-hover:text-blue-500'   },
  'blue-light':{ gradient: 'via-blue-400',   icon: 'group-hover:text-blue-400'   },
  green:       { gradient: 'via-green-500',  icon: 'group-hover:text-green-500'  },
  amber:       { gradient: 'via-amber-500',  icon: 'group-hover:text-amber-500'  },
  red:         { gradient: 'via-red-500',    icon: 'group-hover:text-red-500'    },
  purple:      { gradient: 'via-purple-500', icon: 'group-hover:text-purple-500' },
}

const ac = computed(() => accentMap[props.accent] ?? accentMap.blue)
</script>

<template>
  <div class="group relative overflow-hidden bg-white dark:bg-[#0A0A0A] p-5 rounded-xl border border-gray-200 dark:border-zinc-800 flex flex-col justify-between h-32 hover:border-gray-300 dark:hover:border-zinc-600 transition-all duration-300">
    <div :class="['absolute top-0 left-0 w-full h-0.5 bg-linear-to-r from-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500', ac.gradient]"></div>
    <div class="flex justify-between items-start z-10">
      <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-500 dark:text-zinc-500">{{ label }}</span>
      <component v-if="icon" :is="icon" :class="['w-4 h-4 text-gray-400 dark:text-zinc-500 transition-colors', ac.icon]" />
    </div>
    <div class="flex items-baseline gap-1 z-10">
      <span class="text-4xl font-bold tracking-tighter tabular-nums text-gray-900 dark:text-white">{{ value }}</span>
      <span v-if="unit" class="text-sm font-semibold text-gray-500 dark:text-zinc-500">{{ unit }}</span>
    </div>
  </div>
</template>
