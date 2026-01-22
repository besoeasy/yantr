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
      class="group bg-white dark:bg-slate-900/70 rounded-3xl p-6 shadow-sm border border-gray-200 dark:border-slate-800 hover:shadow-lg transition-all duration-300 cursor-pointer animate-fadeIn relative overflow-hidden flex flex-col h-full hover:-translate-y-1"
    >
      <div class="flex items-start gap-4 mb-4">
        <div class="w-12 h-12 bg-gray-100 dark:bg-slate-800 rounded-xl flex items-center justify-center text-2xl grayscale opacity-70">🐳</div>
        <div class="overflow-hidden">
          <h3 class="font-bold text-lg text-gray-700 dark:text-slate-200 truncate mb-1" :title="container.name">
            {{ container.name.replace(/^\//, "") }}
          </h3>
          <span
            class="text-xs font-semibold px-2 py-0.5 rounded-md"
            :class="
              container.state === 'running'
                ? 'bg-green-100 text-green-800 dark:bg-emerald-500/15 dark:text-emerald-200'
                : 'bg-gray-100 text-gray-600 dark:bg-slate-800 dark:text-slate-300'
            "
          >
            {{ container.state }}
          </span>
        </div>
      </div>

      <div class="mt-auto pt-4 border-t border-gray-100 dark:border-slate-800">
        <div class="flex items-center justify-between text-xs text-gray-500 dark:text-slate-400">
          <span class="font-mono truncate max-w-37.5">{{ container.image.split(":")[0] }}</span>
          <ArrowRight :size="14" class="group-hover:translate-x-1 transition-transform" />
        </div>
      </div>
    </div>
  </div>
</template>
