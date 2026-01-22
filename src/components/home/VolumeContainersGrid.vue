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
      class="relative h-full overflow-hidden group rounded-2xl cursor-pointer transition-all duration-500 hover:-translate-y-1 animate-fadeIn"
    >
      <div class="absolute inset-0 bg-white dark:bg-gray-900">
        <div
          class="absolute inset-0 bg-linear-to-br from-indigo-200/60 via-purple-200/30 to-white/80 dark:from-indigo-600/25 dark:via-purple-600/10 dark:to-gray-900 z-10"
        ></div>
        <div
          class="absolute top-0 right-0 w-64 h-64 bg-indigo-300/35 dark:bg-indigo-500/15 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2 group-hover:bg-indigo-400/45 dark:group-hover:bg-indigo-500/25 transition-colors duration-700"
        ></div>
        <div
          class="absolute bottom-0 left-0 w-48 h-48 bg-purple-300/30 dark:bg-purple-600/15 rounded-full blur-3xl translate-y-1/2 -translate-x-1/2 group-hover:bg-purple-400/40 dark:group-hover:bg-purple-600/25 transition-colors duration-700"
        ></div>
      </div>

      <div
        class="relative z-20 h-full p-5 flex flex-col justify-between border border-slate-200/80 dark:border-slate-700/60 rounded-2xl backdrop-blur-sm group-hover:border-indigo-300/70 dark:group-hover:border-indigo-500/30 transition-none"
      >
        <div class="flex items-center gap-4 mb-4">
          <div class="relative shrink-0">
            <div class="absolute inset-0 bg-indigo-400/25 dark:bg-indigo-500/20 rounded-2xl blur-lg group-hover:blur-xl transition-all duration-500"></div>
            <div
              class="relative w-14 h-14 bg-linear-to-br from-indigo-600 to-purple-600 rounded-2xl flex items-center justify-center shadow-lg group-hover:scale-110 transition-transform duration-500"
            >
              <span class="text-2xl">📂</span>
            </div>
          </div>

          <div class="min-w-0">
            <h3 class="font-bold text-lg text-slate-900 dark:text-white line-clamp-1 mb-1 group-hover:text-indigo-700 dark:group-hover:text-indigo-200 transition-colors">
              {{ container.labels?.["yantra.volume-browser"] || container.name }}
            </h3>
            <span class="text-xs font-semibold px-2.5 py-1 rounded-lg border bg-emerald-100 text-emerald-700 border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-200 dark:border-emerald-500/20">
              Active
            </span>
          </div>
        </div>

        <div v-if="isTemporary(container)" class="mb-4 px-1">
          <div class="flex items-center justify-between text-xs py-1.5 px-3 rounded-lg border border-orange-200 bg-orange-100 dark:border-orange-500/20 dark:bg-orange-500/10">
            <span class="font-medium text-orange-700 dark:text-orange-200">Expires in</span>
            <span
              :class="getExpirationInfo(container).isExpiringSoon ? 'text-red-500 dark:text-red-300 animate-pulse font-bold' : 'text-orange-600 dark:text-orange-200 font-semibold'"
              class="font-mono tabular-nums"
            >
              {{ getExpirationInfo(container).timeRemaining }}
            </span>
          </div>
        </div>

        <div class="mt-auto pt-3 flex items-center justify-between text-sm border-t border-slate-200/70 dark:border-slate-700/60 group-hover:border-indigo-200/70 dark:group-hover:border-slate-600/70 transition-colors">
          <span class="text-slate-500 dark:text-gray-400 font-medium group-hover:text-indigo-600 dark:group-hover:text-indigo-200 transition-colors">Manage Files</span>
          <ArrowRight
            :size="16"
            class="text-slate-600 dark:text-white/70 group-hover:text-indigo-600 dark:group-hover:text-indigo-200 transform group-hover:translate-x-0.5 transition-all"
          />
        </div>
      </div>
    </div>
  </div>
</template>
