<script setup>
import { ArrowRight } from "lucide-vue-next";

const { containers, formatUptime, isTemporary, getExpirationInfo } = defineProps({
  containers: { type: Array, default: () => [] },
  formatUptime: { type: Function, required: true },
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
      class="group relative h-full overflow-hidden bg-white dark:bg-gray-900 rounded-3xl p-6 border border-slate-200 dark:border-gray-700 hover:shadow-2xl hover:-translate-y-2 transition-all duration-300 cursor-pointer animate-fadeIn focus:outline-none focus:ring-4 focus:ring-blue-500"
    >
      <!-- Gradient Glow (Extra.vue card theme) -->
      <div
        class="absolute top-0 right-0 -mt-8 -mr-8 w-40 h-40 rounded-full blur-3xl opacity-0 group-hover:opacity-70 transition-opacity duration-500 bg-blue-200 dark:bg-blue-500/20"
      ></div>

      <div class="relative z-10 h-full flex flex-col justify-between">
        <div class="flex items-start justify-between mb-4">
          <div class="flex items-center gap-4 min-w-0">
            <div class="relative shrink-0">
              <div
                class="relative w-16 h-16 rounded-2xl flex items-center justify-center overflow-hidden transition-all duration-300 group-hover:scale-110 bg-blue-900/10 dark:bg-blue-900/30"
              >
                <img
                  v-if="container.app && container.app.logo"
                  :src="container.app.logo"
                  :alt="container.name"
                  class="w-12 h-12 object-contain filter group-hover:brightness-110 transition-all"
                />
                <div v-else class="text-2xl">🐳</div>
              </div>

              <div class="absolute -bottom-1 -right-1 w-5 h-5 bg-white dark:bg-gray-900 border border-slate-200 dark:border-gray-700 rounded-full flex items-center justify-center">
                <div
                  :class="container.state === 'running' ? 'bg-emerald-500' : 'bg-slate-300 dark:bg-slate-600'"
                  class="w-3 h-3 rounded-full"
                ></div>
              </div>
            </div>

            <div class="min-w-0">
              <h3 class="font-bold text-lg text-slate-900 dark:text-white line-clamp-1 mb-1 group-hover:text-blue-600 dark:group-hover:text-blue-300 transition-colors">
                {{ container.app ? container.app.name : container.name.replace(/^\//, "") }}
              </h3>
              <div class="flex items-center gap-2 flex-wrap">
                <span
                  class="text-xs font-semibold px-2.5 py-1 rounded-lg border"
                  :class="
                    container.state === 'running'
                      ? 'bg-slate-900 text-white border-slate-900 dark:bg-white dark:text-slate-900 dark:border-white'
                      : 'bg-slate-50 text-slate-600 border-slate-200 dark:bg-gray-800/40 dark:text-slate-300 dark:border-gray-700'
                  "
                >
                  {{ container.state }}
                </span>
                <span
                  v-if="isTemporary(container)"
                  class="text-xs font-semibold px-2.5 py-1 rounded-lg border border-dashed bg-slate-50 text-slate-600 border-slate-200 dark:bg-gray-800/40 dark:text-slate-300 dark:border-gray-700"
                >
                  Temp
                </span>
              </div>
            </div>
          </div>
        </div>

        <div class="space-y-3 mb-5 flex-1">
          <div
            v-if="container.state === 'running' && formatUptime(container)"
            class="flex items-center justify-between text-sm py-2 border-b border-slate-200/70 dark:border-slate-700/60 last:border-0 border-dashed"
          >
            <span class="text-slate-500 dark:text-gray-400 font-medium">Uptime</span>
            <span class="text-slate-700 dark:text-gray-200 font-semibold font-mono tabular-nums">{{ formatUptime(container) }}</span>
          </div>

          <div
            v-if="isTemporary(container)"
            class="flex items-center justify-between text-sm py-2 border-b border-slate-200/70 dark:border-slate-700/60 last:border-0 border-dashed"
          >
            <span class="text-slate-500 dark:text-gray-400 font-medium">Expires</span>
            <span
              :class="[
                getExpirationInfo(container).timeRemaining === 'Expired'
                  ? 'text-rose-600 dark:text-rose-400'
                  : getExpirationInfo(container).isExpiringSoon
                    ? 'text-amber-600 dark:text-amber-400 animate-pulse underline underline-offset-2'
                    : 'text-slate-900 dark:text-white',
              ]"
              class="font-bold font-mono tabular-nums"
            >
              {{ getExpirationInfo(container).timeRemaining }}
            </span>
          </div>
        </div>

        <div class="mt-auto pt-4 flex items-center justify-between text-sm border-t border-slate-200 dark:border-gray-700">
          <span class="text-slate-500 dark:text-gray-400 font-medium group-hover:text-slate-900 dark:group-hover:text-white transition-colors">Manage App</span>
          <div
            class="w-10 h-10 rounded-full bg-slate-50 dark:bg-gray-800/40 border border-slate-200 dark:border-gray-700 flex items-center justify-center text-slate-700 dark:text-slate-200 group-hover:bg-slate-100 dark:group-hover:bg-gray-800/70 transition-colors"
          >
            <ArrowRight :size="16" class="transform group-hover:translate-x-0.5 transition-transform" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
