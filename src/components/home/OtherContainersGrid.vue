<script setup>
import { ArrowRight, Box, Activity, Circle, Package } from "lucide-vue-next";

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
      class="group relative overflow-hidden bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 hover:shadow-xl dark:hover:shadow-slate-900/50 hover:-translate-y-1 transition-all duration-300 animate-fadeIn cursor-pointer focus:outline-none focus:ring-4 focus:ring-slate-500/20"
    >
      <!-- Background Texture -->
      <div 
        class="absolute inset-0 opacity-[0.03] dark:opacity-[0.05] pointer-events-none transition-opacity duration-300 group-hover:opacity-[0.06] dark:group-hover:opacity-[0.08]" 
        style="background-image: radial-gradient(circle at 1rem 1rem, currentColor 1px, transparent 0); background-size: 1rem 1rem;"
      ></div>
      
      <!-- Hover Glow -->
      <div class="absolute top-0 right-0 w-32 h-32 bg-slate-100 dark:bg-slate-800 rounded-full blur-3xl opacity-0 group-hover:opacity-100 transition-opacity duration-500 translate-x-1/3 -translate-y-1/3 pointer-events-none"></div>

      <div class="relative z-10 flex flex-col h-full p-6">
        <div class="flex items-start gap-4 mb-4">
          <!-- Icon -->
          <div class="relative shrink-0">
             <div class="w-12 h-12 rounded-xl bg-slate-100 dark:bg-slate-800 flex items-center justify-center border border-slate-200 dark:border-slate-700 group-hover:scale-110 transition-transform duration-300">
                <Box class="w-6 h-6 text-slate-500 dark:text-slate-400 group-hover:text-slate-700 dark:group-hover:text-slate-200 transition-colors" />
             </div>
             
             <!-- Status Dot -->
             <div class="absolute -bottom-1 -right-1 w-4 h-4 bg-white dark:bg-slate-900 rounded-full flex items-center justify-center border border-slate-100 dark:border-slate-800">
               <span class="relative flex h-2.5 w-2.5">
                  <span v-if="container.state === 'running'" class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                  <span class="relative inline-flex rounded-full h-2.5 w-2.5" 
                      :class="container.state === 'running' ? 'bg-emerald-500' : 'bg-slate-400 dark:bg-slate-600'"></span>
               </span>
             </div>
          </div>

          <div class="overflow-hidden">
            <h3 class="font-bold text-base text-slate-700 dark:text-slate-200 truncate mb-1 pr-2" :title="container.name">
              {{ container.name.replace(/^\//, "") }}
            </h3>
            <div class="flex items-center gap-1.5">
               <div class="text-[10px] font-bold uppercase tracking-wider px-1.5 py-0.5 rounded border"
                    :class="
                      container.state === 'running'
                        ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border-emerald-500/20'
                        : 'bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-400 border-slate-200 dark:border-slate-700'
                    ">
                  {{ container.state }}
               </div>
            </div>
          </div>
        </div>

        <div class="mt-auto pt-4 border-t border-slate-100 dark:border-slate-800 flex items-center justify-between">
          <div class="flex flex-col min-w-0 pr-4">
             <span class="text-[10px] uppercase font-bold text-slate-400 dark:text-slate-500 tracking-wider mb-0.5">Image</span>
             <span class="font-mono text-xs text-slate-600 dark:text-slate-400 truncate" :title="container.image">{{ container.image.split(":")[0] }}</span>
          </div>
          
          <div class="w-8 h-8 rounded-full flex items-center justify-center bg-slate-50 dark:bg-slate-800 text-slate-400 group-hover:bg-slate-100 dark:group-hover:bg-slate-700 group-hover:text-slate-600 dark:group-hover:text-slate-300 transition-colors">
            <ArrowRight class="w-4 h-4 group-hover:-rotate-45 transition-transform duration-300" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
