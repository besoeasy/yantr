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
      class="group relative h-full overflow-hidden bg-white dark:bg-slate-900 rounded-none border border-slate-200 dark:border-slate-800 hover:shadow-xl dark:hover:shadow-slate-900/50 hover:-translate-y-1 transition-all duration-300 animate-fadeIn cursor-pointer focus:outline-none focus:ring-4 focus:ring-violet-500/20"
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
           <div class="flex items-start justify-between mb-4">
               <div class="min-w-0">
                   <h3 class="font-bold text-lg text-slate-900 dark:text-white line-clamp-1 mb-2 leading-tight group-hover:text-violet-600 dark:group-hover:text-violet-400 transition-colors" :title="container.labels?.['yantr.volume-browser'] || container.name">
                     {{ container.labels?.["yantr.volume-browser"] || container.name }}
                   </h3>
               
                   <div class="flex items-center gap-1.5 flex-wrap">
                      <span class="text-[10px] uppercase font-bold px-1.5 py-0.5 rounded border bg-slate-100 text-slate-500 border-slate-200 dark:bg-slate-800 dark:text-slate-400 dark:border-slate-700">
                        File Browser
                      </span>
                      
                      <span v-if="isTemporary(container)" 
                            class="text-[10px] uppercase font-bold px-1.5 py-0.5 rounded border bg-amber-50 text-amber-600 border-amber-100 dark:bg-amber-500/10 dark:text-amber-400 dark:border-amber-500/20">
                        Temp
                      </span>
                   </div>
               </div>
               
               <!-- Status Dot -->
               <div class="relative flex h-3 w-3 mt-1.5">
                    <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                    <span class="relative inline-flex rounded-full h-3 w-3 bg-emerald-500"></span>
               </div>
          </div>
          
          <!-- Bottom Row: Metrics Left, Icon Right -->
          <div class="flex items-end justify-between mt-6 gap-4">
             <!-- Metrics Grid (Left) -->
             <div class="flex-1 space-y-2 min-w-0">
                 <div class="flex flex-col gap-0.5">
                    <span class="text-[10px] font-bold text-slate-400 dark:text-slate-500 uppercase tracking-wider">Status</span>
                    <span class="font-mono font-bold text-base tabular-nums text-slate-700 dark:text-slate-200">Active</span>
                 </div>

                 <div v-if="isTemporary(container)" class="pt-1">
                    <span class="text-[10px] uppercase font-bold text-amber-600 dark:text-amber-500/80 block mb-0.5">Expires In</span>
                    <span class="font-mono text-sm font-bold tabular-nums"
                      :class="[
                          getExpirationInfo(container).timeRemaining === 'Expired'
                            ? 'text-rose-600 dark:text-rose-400'
                            : getExpirationInfo(container).isExpiringSoon
                              ? 'text-amber-600 dark:text-amber-400 animate-pulse underline'
                              : 'text-amber-800 dark:text-amber-300',
                        ]"
                    >{{ getExpirationInfo(container).timeRemaining }}</span>
                 </div>
             </div>

             <!-- Big Icon (Right) -->
             <div class="relative shrink-0">
               <div class="w-16 h-16 rounded-2xl bg-violet-50 dark:bg-violet-900/20 flex items-center justify-center border border-violet-100 dark:border-violet-700/60 shadow-sm group-hover:scale-110 group-hover:rotate-3 group-hover:shadow-md transition-all duration-500 ease-out">
                  <FolderOpen class="w-8 h-8 text-violet-600 dark:text-violet-400" />
               </div>
            </div>
          </div>
        </div>
      </div>

    </div>
  </div>
</template>
