<script setup>
import { ArrowRight } from "lucide-vue-next";

const { containers } = defineProps({
  containers: { type: Array, default: () => [] },
});

const emit = defineEmits(["select"]);
</script>

<template>
  <div style="display: contents">
    <div
      v-for="(container, index) in containers"
      :key="`other-${container.id}`"
      :style="{ animationDelay: `${index * 50}ms` }"
      @click="emit('select', container)"
      @keydown.enter.prevent="emit('select', container)"
      @keydown.space.prevent="emit('select', container)"
      role="button"
      tabindex="0"
      class="group relative overflow-hidden bg-white dark:bg-gray-900 rounded-3xl p-6 border border-slate-200 dark:border-gray-700 hover:shadow-2xl hover:-translate-y-2 transition-all duration-300 cursor-pointer animate-fadeIn flex flex-col h-full focus:outline-none focus:ring-4 focus:ring-slate-400"
    >
      <!-- Gradient Glow (Extra.vue card theme) -->
      <div
        class="absolute top-0 right-0 -mt-8 -mr-8 w-40 h-40 rounded-full blur-3xl opacity-0 group-hover:opacity-70 transition-opacity duration-500 bg-slate-200 dark:bg-slate-500/20"
      ></div>

      <div class="relative z-10 flex flex-col h-full">
      <div class="flex items-start gap-4 mb-4">
        <div class="w-14 h-14 bg-slate-900/5 dark:bg-slate-800/60 rounded-2xl flex items-center justify-center text-2xl grayscale opacity-70">🐳</div>
        <div class="overflow-hidden">
          <h3 class="font-bold text-lg text-slate-900 dark:text-white truncate mb-1" :title="container.name">
            {{ container.name.replace(/^\//, "") }}
          </h3>
          <span
            class="text-xs font-semibold px-2 py-0.5 rounded-md border"
            :class="
              container.state === 'running'
                ? 'bg-slate-900 text-white border-slate-900 dark:bg-white dark:text-slate-900 dark:border-white'
                : 'bg-slate-50 text-slate-600 border-slate-200 dark:bg-gray-800/40 dark:text-slate-300 dark:border-gray-700'
            "
          >
            {{ container.state }}
          </span>
        </div>
      </div>

      <div class="mt-auto pt-4 border-t border-slate-200 dark:border-gray-700">
        <div class="flex items-center justify-between text-xs text-slate-500 dark:text-slate-400">
          <span class="font-mono truncate max-w-37.5">{{ container.image.split(":")[0] }}</span>
          <ArrowRight :size="14" class="group-hover:translate-x-1 transition-transform" />
        </div>
      </div>
      </div>
    </div>
  </div>
</template>
