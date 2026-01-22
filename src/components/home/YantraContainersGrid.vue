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
      class="relative h-full overflow-hidden group cursor-pointer transition-colors duration-300 animate-fadeIn"
    >
      <div class="absolute inset-0 bg-white dark:bg-gray-900">
        <div
          class="absolute inset-0 bg-linear-to-br from-slate-200/70 via-blue-200/25 to-white/80 dark:from-slate-600/25 dark:via-blue-600/10 dark:to-gray-900 z-10"
        ></div>
        <div
          class="absolute top-0 right-0 w-64 h-64 bg-blue-300/35 dark:bg-blue-500/15 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2 group-hover:bg-blue-400/45 dark:group-hover:bg-blue-500/25 transition-colors duration-700"
        ></div>
        <div
          class="absolute bottom-0 left-0 w-48 h-48 bg-slate-300/30 dark:bg-slate-500/20 rounded-full blur-3xl translate-y-1/2 -translate-x-1/2 group-hover:bg-slate-400/40 dark:group-hover:bg-slate-500/30 transition-colors duration-700"
        ></div>
      </div>

      <div
        class="relative z-20 h-full p-5 flex flex-col justify-between border border-slate-200/80 dark:border-slate-700/60 rounded-2xl backdrop-blur-sm group-hover:border-blue-300/70 dark:group-hover:border-blue-500/30 transition-none"
      >
        <div class="flex items-start justify-between mb-4">
          <div class="flex items-center gap-4 min-w-0">
            <div class="relative shrink-0">
              <div class="absolute inset-0 bg-blue-400/25 dark:bg-blue-500/15 rounded-2xl blur-lg group-hover:blur-xl transition-all duration-500"></div>

              <div
                class="relative w-14 h-14 rounded-2xl bg-white/80 dark:bg-white/5 border border-slate-200/70 dark:border-slate-700/60 flex items-center justify-center overflow-hidden"
              >
                <img
                  v-if="container.app && container.app.logo"
                  :src="container.app.logo"
                  :alt="container.name"
                  class="w-12 h-12 object-contain filter group-hover:brightness-110 transition-all"
                />
                <div v-else class="text-2xl">🐳</div>
              </div>

              <div class="absolute -bottom-1 -right-1 w-5 h-5 bg-white/80 dark:bg-gray-900/90 border border-slate-200/70 dark:border-slate-700/60 rounded-full flex items-center justify-center">
                <div :class="container.state === 'running' ? 'bg-emerald-500' : 'bg-slate-400'" class="w-3 h-3 rounded-full animate-pulse"></div>
              </div>
            </div>

            <div class="min-w-0">
              <h3 class="font-bold text-lg text-slate-900 dark:text-white line-clamp-1 mb-1 group-hover:text-blue-700 dark:group-hover:text-blue-200 transition-colors">
                {{ container.app ? container.app.name : container.name.replace(/^\//, "") }}
              </h3>
              <div class="flex items-center gap-2 flex-wrap">
                <span
                  class="text-xs font-semibold px-2.5 py-1 rounded-lg border"
                  :class="
                    container.state === 'running'
                      ? 'bg-emerald-100 text-emerald-700 border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-200 dark:border-emerald-500/20'
                      : 'bg-white/70 text-slate-600 border-slate-200 dark:bg-white/5 dark:text-gray-300 dark:border-slate-700/60'
                  "
                >
                  {{ container.state }}
                </span>
                <span
                  v-if="isTemporary(container)"
                  class="text-xs font-semibold px-2.5 py-1 rounded-lg border bg-orange-100 text-orange-700 border-orange-200 dark:bg-orange-500/10 dark:text-orange-200 dark:border-orange-500/20"
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
              :class="getExpirationInfo(container).isExpiringSoon ? 'text-red-500 dark:text-red-300 animate-pulse' : 'text-orange-600 dark:text-orange-200'"
              class="font-bold font-mono tabular-nums"
            >
              {{ getExpirationInfo(container).timeRemaining }}
            </span>
          </div>
        </div>

        <div class="mt-auto pt-4 flex items-center justify-between text-sm border-t border-slate-200/70 dark:border-slate-700/60 group-hover:border-blue-200/70 dark:group-hover:border-slate-600/70 transition-colors">
          <span class="text-slate-500 dark:text-gray-400 font-medium group-hover:text-blue-600 dark:group-hover:text-blue-200 transition-colors">Manage App</span>
          <div
            class="w-9 h-9 rounded-full bg-white/70 dark:bg-white/5 border border-slate-200/70 dark:border-slate-700/60 flex items-center justify-center text-slate-700 dark:text-white group-hover:border-blue-300/70 dark:group-hover:border-blue-500/30 group-hover:bg-blue-100 dark:group-hover:bg-blue-500/10 transition-colors"
          >
            <ArrowRight :size="16" class="transform group-hover:translate-x-0.5 transition-transform" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
