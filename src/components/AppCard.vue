<script setup>
import { computed, toRefs } from "vue";
import { Check, Bot } from "lucide-vue-next";

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
</script>

<template>
  <div
    class="group relative flex items-start p-3 hover:bg-slate-100 dark:hover:bg-slate-800/50 rounded-xl transition-all duration-200 cursor-pointer"
    role="button"
    tabindex="0"
    :aria-label="`Open ${app?.name ?? 'app'} details`"
  >
    <!-- Logo -->
    <div class="w-12 h-12 shrink-0 mr-4">
        <div class="w-full h-full rounded-xl bg-white dark:bg-[#1a1a1c] border border-slate-200 dark:border-slate-800 flex items-center justify-center p-2 shadow-sm group-hover:shadow transition-all">
            <img
            v-if="app?.logo"
            :src="app.logo"
            :alt="app.name"
            class="w-full h-full object-contain"
            loading="lazy"
            />
            <Bot v-else :size="24" class="text-slate-400" />
        </div>
    </div>

    <!-- Content -->
    <div class="flex-1 min-w-0 pt-0.5">
      <div class="flex items-center gap-2 mb-1">
        <h3 class="text-sm font-bold text-slate-900 dark:text-slate-100 truncate group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
            {{ app?.name || 'Unknown App' }}
        </h3>
        <!-- Status Indicators -->
        <div v-if="app?.isInstalled" class="bg-emerald-100 dark:bg-emerald-500/20 text-emerald-600 dark:text-emerald-400 rounded-full p-0.5" title="Installed">
            <Check :size="10" stroke-width="3" />
        </div>
        <span v-if="instanceCount > 0" class="text-[10px] font-mono bg-slate-100 dark:bg-slate-800 px-1.5 py-0.5 rounded text-slate-500" title="Running Instances">{{ instanceCount }}</span>
      </div>
      
      <p class="text-xs text-slate-500 dark:text-slate-400 leading-relaxed line-clamp-2">
        {{ app?.description || 'No description available' }}
      </p>
    </div>
  </div>
</template>
