<script setup>
import { computed, toRefs } from "vue";
import { Check, ArrowRight, Box, Bot } from "lucide-vue-next";

const props = defineProps({
  app: {
    type: Object,
    required: true,
  },
  instanceCount: {
    type: Number,
    default: 0,
  },
});

const { app, instanceCount } = toRefs(props);

function hashStringToUint32(value) {
  // FNV-1a 32-bit
  let hash = 0x811c9dc5;
  for (let i = 0; i < value.length; i++) {
    hash ^= value.charCodeAt(i);
    hash = Math.imul(hash, 0x01000193);
  }
  return hash >>> 0;
}

const categories = computed(() => {
  const raw = app.value?.category ?? "";
  const parts = raw
    .split(",")
    .map((c) => c.trim())
    .filter(Boolean);

  if (parts.length <= 2) {
    return { shown: parts };
  }

  const seedSource = String(app.value?.id ?? app.value?.name ?? raw);
  const h1 = hashStringToUint32(seedSource);
  const h2 = hashStringToUint32(`${seedSource}::2`);

  const firstIndex = h1 % parts.length;
  const remainingIndices = [];
  for (let i = 0; i < parts.length; i++) {
    if (i !== firstIndex) remainingIndices.push(i);
  }
  const secondIndex = remainingIndices[h2 % remainingIndices.length];

  const shown = [parts[firstIndex], parts[secondIndex]];
  return { shown };
});
</script>

<template>
  <div
    class="group relative flex flex-col h-full bg-white dark:bg-[#0c0c0e] border border-slate-200 dark:border-slate-800 hover:border-blue-400 dark:hover:border-blue-600 rounded-lg p-5 transition-all duration-200 cursor-pointer"
    role="button"
    tabindex="0"
    :aria-label="`Open ${app?.name ?? 'app'} details`"
  >
    <!-- Header: Logo & Status -->
    <div class="flex justify-between items-start mb-4">
      <div class="w-12 h-12 rounded bg-slate-50 dark:bg-[#1a1a1c] border border-slate-100 dark:border-slate-800 flex items-center justify-center p-2.5 shrink-0">
        <img
          v-if="app?.logo"
          :src="app.logo"
          :alt="app.name"
          class="w-full h-full object-contain filter grayscale group-hover:grayscale-0 transition-all duration-300"
          loading="lazy"
        />
        <Bot v-else :size="20" class="text-slate-400" />
      </div>

      <div class="flex flex-col items-end gap-2">
         <span v-if="app?.isInstalled" class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider bg-emerald-50 dark:bg-emerald-950/20 text-emerald-600 dark:text-emerald-400 border border-emerald-100 dark:border-emerald-900/30">
            <Check :size="10" />
            Installed
         </span>
         <span v-if="instanceCount > 0" class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-mono text-slate-500 bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-700">
            <Box :size="10" />
            {{ instanceCount }}
         </span>
      </div>
    </div>

    <!-- Content -->
    <div class="flex-1 flex flex-col">
      <h3 class="text-base font-bold text-slate-900 dark:text-slate-100 mb-1 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
        {{ app?.name || 'Unknown App' }}
      </h3>
      
      <p class="text-sm text-slate-500 dark:text-slate-400 leading-relaxed mb-4 line-clamp-2">
        {{ app?.description || 'No description available' }}
      </p>

      <div class="mt-auto pt-4 border-t border-slate-50 dark:border-slate-800/50 flex items-center justify-between">
         <div class="flex flex-wrap gap-1.5">
            <span 
              v-for="cat in categories.shown" 
              :key="cat"
              class="text-[10px] font-mono px-1.5 py-0.5 bg-slate-50 dark:bg-[#1a1a1c] text-slate-500 dark:text-slate-400 border border-slate-200 dark:border-slate-800 rounded"
            >
              {{ cat }}
            </span>
         </div>
         
         <div class="text-slate-300 dark:text-slate-600 group-hover:text-slate-500 dark:group-hover:text-slate-400 transition-colors">
             <ArrowRight :size="16" />
         </div>
      </div>
    </div>
  </div>
</template>
