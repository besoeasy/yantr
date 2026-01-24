<script setup>
import { ArrowRight, FolderOpen, Clock } from "lucide-vue-next";

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
      class="group relative h-full overflow-hidden bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 hover:shadow-xl dark:hover:shadow-slate-900/50 hover:-translate-y-1 transition-all duration-300 animate-fadeIn focus:outline-none focus:ring-4 focus:ring-violet-500/20"
    >
      <!-- Background Texture -->
      <div 
        class="absolute inset-0 opacity-[0.03] dark:opacity-[0.05] pointer-events-none transition-opacity duration-300 group-hover:opacity-[0.06] dark:group-hover:opacity-[0.08]" 
        style="background-image: radial-gradient(circle at 1rem 1rem, currentColor 1px, transparent 0); background-size: 1rem 1rem;"
      ></div>

      <!-- Hover Glow -->
      <div class="absolute top-0 right-0 w-32 h-32 bg-violet-100 dark:bg-violet-900/40 rounded-full blur-3xl opacity-0 group-hover:opacity-100 transition-opacity duration-500 translate-x-1/3 -translate-y-1/3 pointer-events-none"></div>

      <div class="relative z-10 h-full flex flex-col justify-between p-6">
        <div>
           <div class="flex items-start gap-4 mb-4">
             <!-- Icon -->
            <div class="relative shrink-0">
               <div class="w-12 h-12 rounded-xl bg-violet-50 dark:bg-violet-900/20 flex items-center justify-center border border-violet-100 dark:border-violet-900/30 group-hover:scale-110 transition-transform duration-300">
                  <FolderOpen class="w-6 h-6 text-violet-600 dark:text-violet-400 group-hover:text-violet-700 dark:group-hover:text-violet-300 transition-colors" />
               </div>
            </div>

            <div class="min-w-0">
               <h3 class="font-bold text-base text-slate-900 dark:text-white line-clamp-1 mb-1" :title="container.labels?.['yantra.volume-browser'] || container.name">
                 {{ container.labels?.["yantra.volume-browser"] || container.name }}
               </h3>
               <div class="flex items-center gap-1.5">
                  <span class="text-[10px] uppercase font-bold px-1.5 py-0.5 rounded border bg-slate-100 text-slate-500 border-slate-200 dark:bg-slate-800 dark:text-slate-400 dark:border-slate-700">
                    File Browser
                  </span>
               </div>
            </div>
          </div>
          
          <!-- Temporary Expiration Badge -->
          <div v-if="isTemporary(container)" class="mt-2 text-xs">
            <div class="flex items-center gap-2 p-2 rounded-lg bg-orange-50 dark:bg-orange-900/20 border border-orange-100 dark:border-orange-900/30">
               <Clock class="w-3.5 h-3.5 text-orange-500" />
               <div class="flex flex-col">
                  <span class="text-[10px] font-semibold text-orange-600/80 dark:text-orange-400/80 uppercase tracking-wide">Expires</span>
                  <span 
                    class="font-mono font-bold leading-none tabular-nums"
                    :class="[
                      getExpirationInfo(container).timeRemaining === 'Expired'
                        ? 'text-rose-600 dark:text-rose-400'
                        : getExpirationInfo(container).isExpiringSoon
                          ? 'text-orange-600 dark:text-orange-400 animate-pulse'
                          : 'text-orange-700 dark:text-orange-300',
                    ]"
                  >
                    {{ getExpirationInfo(container).timeRemaining }}
                  </span>
               </div>
            </div>
          </div>
        </div>

        <div class="mt-4 pt-4 flex items-center justify-between text-xs border-t border-slate-100 dark:border-slate-800">
          <span class="text-slate-500 dark:text-slate-400 font-medium group-hover:text-violet-600 dark:group-hover:text-violet-400 transition-colors">Open Browser</span>
          <div class="w-8 h-8 rounded-full flex items-center justify-center bg-slate-50 dark:bg-slate-800 text-slate-400 group-hover:bg-violet-100 dark:group-hover:bg-violet-900/30 group-hover:text-violet-600 dark:group-hover:text-violet-400 transition-colors">
            <ArrowRight class="w-4 h-4 group-hover:-rotate-45 transition-transform duration-300" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
