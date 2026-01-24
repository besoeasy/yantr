<script setup>
import { computed } from "vue";
import { Sparkles, Activity, Layers, Database, Box, HardDrive } from "lucide-vue-next";

const props = defineProps({
  runningApps: { type: Number, default: 0 },
  totalVolumes: { type: Number, default: 0 },
  temporaryCount: { type: Number, default: 0 },
  imagesCount: { type: Number, default: 0 },
});

const greeting = computed(() => {
  const hour = new Date().getHours();
  if (hour < 12) return "Good morning";
  if (hour < 18) return "Good afternoon";
  return "Good evening";
});

const theme = {
  icon: Sparkles,
  text: 'text-sky-600 dark:text-sky-400',
  bg: 'bg-sky-500/10 dark:bg-sky-500/20',
  border: 'group-hover:border-sky-500/30 dark:group-hover:border-sky-400/30',
};
</script>

<template>
  <div class="relative h-full overflow-hidden group rounded-3xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 transition-all duration-300 hover:shadow-lg dark:hover:shadow-slate-900/50" :class="theme.border">
    <!-- Background Texture -->
    <div class="absolute inset-0 opacity-[0.03] dark:opacity-[0.05] pointer-events-none" 
         style="background-image: radial-gradient(circle at 1rem 1rem, currentColor 1px, transparent 0); background-size: 1rem 1rem;">
    </div>

    <div class="relative z-10 h-full p-6 flex flex-col justify-between">
      <!-- Header -->
      <div class="flex items-center justify-between mb-6">
        <div class="flex items-center gap-3">
          <div class="p-2.5 rounded-xl transition-colors duration-300" :class="theme.bg">
            <component :is="theme.icon" class="w-5 h-5 transition-transform duration-700 ease-in-out group-hover:rotate-12" :class="theme.text" />
          </div>
          <div>
            <h1 class="text-lg font-bold text-slate-900 dark:text-white leading-tight">
              {{ greeting }}
            </h1>
            <p class="text-xs font-medium text-slate-500 dark:text-slate-400">
              System Overview
            </p>
          </div>
        </div>
        
        <div class="flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-emerald-500/10 border border-emerald-500/20">
           <span class="relative flex h-2 w-2">
             <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
             <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
           </span>
           <span class="text-[10px] font-bold text-emerald-600 dark:text-emerald-400 uppercase tracking-wide">Online</span>
        </div>
      </div>

      <!-- Metrics Grid -->
      <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <!-- Running Apps -->
        <div class="flex flex-col gap-1 p-3 rounded-2xl bg-slate-50 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-800 group/item hover:border-sky-200 dark:hover:border-sky-500/30 transition-colors">
           <div class="flex items-center gap-2 text-slate-500 dark:text-slate-400">
             <Layers class="w-3.5 h-3.5" />
             <span class="text-[10px] uppercase font-bold tracking-wider">Apps</span>
           </div>
           <div class="text-2xl font-black text-slate-900 dark:text-white tabular-nums">{{ props.runningApps }}</div>
        </div>

        <!-- Volumes -->
        <div class="flex flex-col gap-1 p-3 rounded-2xl bg-slate-50 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-800 group/item hover:border-violet-200 dark:hover:border-violet-500/30 transition-colors">
           <div class="flex items-center gap-2 text-slate-500 dark:text-slate-400">
             <HardDrive class="w-3.5 h-3.5" />
             <span class="text-[10px] uppercase font-bold tracking-wider">Volumes</span>
           </div>
           <div class="text-2xl font-black text-slate-900 dark:text-white tabular-nums">{{ props.totalVolumes }}</div>
        </div>

        <!-- Temp -->
        <div class="flex flex-col gap-1 p-3 rounded-2xl bg-slate-50 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-800 group/item hover:border-amber-200 dark:hover:border-amber-500/30 transition-colors">
           <div class="flex items-center gap-2 text-slate-500 dark:text-slate-400">
             <Box class="w-3.5 h-3.5" />
             <span class="text-[10px] uppercase font-bold tracking-wider">Temp</span>
           </div>
           <div class="text-2xl font-black text-slate-900 dark:text-white tabular-nums">{{ props.temporaryCount }}</div>
        </div>

        <!-- Images -->
        <div class="flex flex-col gap-1 p-3 rounded-2xl bg-slate-50 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-800 group/item hover:border-blue-200 dark:hover:border-blue-500/30 transition-colors">
           <div class="flex items-center gap-2 text-slate-500 dark:text-slate-400">
             <Database class="w-3.5 h-3.5" />
             <span class="text-[10px] uppercase font-bold tracking-wider">Images</span>
           </div>
           <div class="text-2xl font-black text-slate-900 dark:text-white tabular-nums">{{ props.imagesCount }}</div>
        </div>
      </div>
    </div>
  </div>
</template>
