<script setup>
import { ArrowRight } from "lucide-vue-next";

const { containers, isTemporary, getExpirationInfo } = defineProps({
  containers: { type: Array, default: () => [] },
  isTemporary: { type: Function, required: true },
  getExpirationInfo: { type: Function, required: true },
});

const emit = defineEmits(["select"]);
</script>

<template>
  <div style="display: contents">
    <div
      v-for="(container, index) in containers"
      :key="container.id"
      :style="{ animationDelay: `${index * 50}ms` }"
      @click="emit('select', container)"
      @keydown.enter.prevent="emit('select', container)"
      @keydown.space.prevent="emit('select', container)"
      role="button"
      tabindex="0"
      class="group relative h-full overflow-hidden bg-white dark:bg-gray-900 rounded-3xl p-6 border border-slate-200 dark:border-gray-700 hover:shadow-2xl hover:-translate-y-2 transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)] cursor-pointer animate-fadeIn focus:outline-none focus:ring-4 focus:ring-amber-500"
    >
      <!-- Gradient Glow (Extra.vue card theme) -->
      <div
        class="absolute top-0 right-0 -mt-8 -mr-8 w-40 h-40 rounded-full blur-3xl opacity-0 group-hover:opacity-70 transition-opacity duration-300 ease-[cubic-bezier(0.16,1,0.3,1)] bg-amber-200 dark:bg-amber-500/20"
      ></div>

      <div class="relative z-10 h-full flex flex-col justify-between">
        <div class="flex items-center gap-4 mb-4">
          <div class="relative shrink-0">
            <div
              class="relative w-16 h-16 rounded-2xl flex items-center justify-center shadow-sm group-hover:scale-110 transition-transform duration-300 ease-[cubic-bezier(0.16,1,0.3,1)] bg-amber-900/10 dark:bg-amber-900/30"
            >
              <span class="text-2xl">📂</span>
            </div>
          </div>

          <div class="min-w-0">
            <h3 class="font-bold text-lg text-slate-900 dark:text-white line-clamp-1 mb-1 group-hover:text-black dark:group-hover:text-white transition-colors">
              {{ container.labels?.["yantra.volume-browser"] || container.name }}
            </h3>
            <span
              class="text-xs font-semibold px-2.5 py-1 rounded-lg border bg-slate-900 text-white border-slate-900 dark:bg-white dark:text-slate-900 dark:border-white"
            >
              Active
            </span>
          </div>
        </div>

        <div v-if="isTemporary(container)" class="mb-4 px-1">
          <div class="flex items-center justify-between text-xs py-1.5 px-3 rounded-lg border border-slate-200 bg-slate-50 dark:border-gray-700 dark:bg-gray-800/40">
            <span class="font-medium text-slate-600 dark:text-slate-300">Expires in</span>
            <span
              :class="[
                getExpirationInfo(container).timeRemaining === 'Expired'
                  ? 'text-rose-600 dark:text-rose-400 font-bold'
                  : getExpirationInfo(container).isExpiringSoon
                    ? 'text-amber-600 dark:text-amber-400 animate-pulse underline underline-offset-2 font-bold'
                    : 'text-slate-900 dark:text-white font-semibold',
              ]"
              class="font-mono tabular-nums"
            >
              {{ getExpirationInfo(container).timeRemaining }}
            </span>
          </div>
        </div>

        <div class="mt-auto pt-4 flex items-center justify-between text-sm border-t border-slate-200 dark:border-gray-700">
          <span class="text-slate-500 dark:text-gray-400 font-medium group-hover:text-slate-900 dark:group-hover:text-white transition-colors">Manage Files</span>
          <ArrowRight
            :size="16"
            class="text-slate-700 dark:text-slate-200 group-hover:text-slate-900 dark:group-hover:text-white transform group-hover:translate-x-0.5 transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"
          />
        </div>
      </div>
    </div>
  </div>
</template>
