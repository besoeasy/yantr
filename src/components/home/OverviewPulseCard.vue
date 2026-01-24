<script setup>
import { computed, ref, onMounted } from "vue";
import { Sparkles, Activity, Layers, Database, Box, HardDrive, Zap, Cpu } from "lucide-vue-next";

const props = defineProps({
  runningApps: { type: Number, default: 0 },
  totalVolumes: { type: Number, default: 0 },
  temporaryCount: { type: Number, default: 0 },
  imagesCount: { type: Number, default: 0 },
});

const greeting = computed(() => {
  const hour = new Date().getHours();
  if (hour < 5) return "Late night coding?";
  if (hour < 12) return "Good morning";
  if (hour < 18) return "Good afternoon";
  return "Good evening";
});

const theme = {
  border: 'group-hover:border-sky-500/30 dark:group-hover:border-sky-400/30',
};

// Simple count animation ref (optional enhancement, sticking to CSS for simplicity)
const showContent = ref(false);
onMounted(() => {
  requestAnimationFrame(() => showContent.value = true);
});
</script>

<template>
  <div class="relative h-full overflow-hidden group rounded-3xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 transition-all duration-300 hover:shadow-xl dark:hover:shadow-slate-900/50" :class="theme.border">
    <!-- Animated Background Mesh -->
    <div class="absolute inset-0 overflow-hidden pointer-events-none">
       <div class="absolute top-[-50%] left-[-20%] w-[80%] h-[200%] bg-gradient-to-r from-transparent via-sky-100/20 dark:via-sky-900/10 to-transparent transform -rotate-12 translate-x-[-100%] animate-shine"></div>
       <div class="absolute inset-0 opacity-[0.03] dark:opacity-[0.05]" 
         style="background-image: radial-gradient(circle at 1rem 1rem, currentColor 1px, transparent 0); background-size: 1rem 1rem;">
       </div>
    </div>
    
    <!-- Big Background Icon for decorative feel -->
    <div class="absolute -right-6 -bottom-6 opacity-5 dark:opacity-10 transform rotate-12 group-hover:scale-110 group-hover:rotate-6 transition-transform duration-700 ease-in-out pointer-events-none">
       <Activity class="w-64 h-64 text-sky-900 dark:text-sky-100" />
    </div>

    <div class="relative z-10 h-full p-6 sm:p-8 flex flex-col md:flex-row items-stretch gap-8">
      
      <!-- Hero Section (Left) -->
      <div class="flex-1 flex flex-col justify-center min-w-[200px]">
        <div class="flex items-center gap-2 mb-4">
           <div class="relative flex h-3 w-3">
             <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
             <span class="relative inline-flex rounded-full h-3 w-3 bg-emerald-500"></span>
           </div>
           <span class="text-xs font-bold text-emerald-600 dark:text-emerald-400 uppercase tracking-widest">System Online</span>
        </div>
        
        <h1 class="text-3xl sm:text-4xl font-black text-slate-900 dark:text-white tracking-tight mb-2 leading-tight">
          {{ greeting }}
        </h1>
        <p class="text-sm text-slate-500 dark:text-slate-400 font-medium max-w-xs leading-relaxed">
          Your stack is running smoothly. Here is your live performance overview.
        </p>
        
        <div class="mt-6 flex items-center gap-3">
           <div class="px-3 py-1.5 rounded-lg bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 flex items-center gap-2">
              <Cpu class="w-4 h-4 text-slate-500" />
              <span class="text-xs font-bold text-slate-700 dark:text-slate-300">Healthy</span>
           </div>
           <div class="px-3 py-1.5 rounded-lg bg-sky-50 dark:bg-sky-900/20 border border-sky-100 dark:border-sky-800 flex items-center gap-2">
              <Zap class="w-4 h-4 text-sky-500" />
              <span class="text-xs font-bold text-sky-700 dark:text-sky-300">Active</span>
           </div>
        </div>
      </div>

      <!-- Stats Grid (Right) -->
      <div class="flex-none w-full md:w-auto md:min-w-[400px] grid grid-cols-2 gap-3 sm:gap-4">
        
        <!-- Running Apps -->
        <div class="relative overflow-hidden p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/40 border border-slate-100 dark:border-slate-700/60 group/stat hover:bg-white dark:hover:bg-slate-800 transition-colors duration-200 hover:shadow-sm">
           <div class="absolute right-0 top-0 p-3 opacity-20 group-hover/stat:opacity-40 transition-opacity">
              <Layers class="w-12 h-12 text-blue-500 transform rotate-12" />
           </div>
           
           <div class="relative z-10 flex flex-col justify-between h-full gap-2">
             <div class="flex items-center gap-2 text-slate-500 dark:text-slate-400">
               <span class="p-1.5 rounded-md bg-white dark:bg-slate-900 shadow-sm border border-slate-100 dark:border-slate-700 text-blue-500">
                 <Layers class="w-4 h-4" />
               </span>
               <span class="text-xs font-bold uppercase tracking-wider">Apps</span>
             </div>
             <div>
                <div class="text-3xl font-black text-slate-900 dark:text-white tabular-nums tracking-tight">{{ props.runningApps }}</div>
                <div class="h-1 w-12 bg-blue-500 rounded-full mt-1.5 opacity-80"></div>
             </div>
           </div>
        </div>

        <!-- Volumes -->
        <div class="relative overflow-hidden p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/40 border border-slate-100 dark:border-slate-700/60 group/stat hover:bg-white dark:hover:bg-slate-800 transition-colors duration-200 hover:shadow-sm">
           <div class="absolute right-0 top-0 p-3 opacity-20 group-hover/stat:opacity-40 transition-opacity">
              <HardDrive class="w-12 h-12 text-violet-500 transform -rotate-6" />
           </div>
           
           <div class="relative z-10 flex flex-col justify-between h-full gap-2">
             <div class="flex items-center gap-2 text-slate-500 dark:text-slate-400">
               <span class="p-1.5 rounded-md bg-white dark:bg-slate-900 shadow-sm border border-slate-100 dark:border-slate-700 text-violet-500">
                 <HardDrive class="w-4 h-4" />
               </span>
               <span class="text-xs font-bold uppercase tracking-wider">Volumes</span>
             </div>
             <div>
                <div class="text-3xl font-black text-slate-900 dark:text-white tabular-nums tracking-tight">{{ props.totalVolumes }}</div>
                <div class="h-1 w-12 bg-violet-500 rounded-full mt-1.5 opacity-80"></div>
             </div>
           </div>
        </div>

        <!-- Images -->
        <div class="relative overflow-hidden p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/40 border border-slate-100 dark:border-slate-700/60 group/stat hover:bg-white dark:hover:bg-slate-800 transition-colors duration-200 hover:shadow-sm">
           <div class="absolute right-0 top-0 p-3 opacity-20 group-hover/stat:opacity-40 transition-opacity">
              <Database class="w-12 h-12 text-emerald-500 transform rotate-6" />
           </div>
           
           <div class="relative z-10 flex flex-col justify-between h-full gap-2">
             <div class="flex items-center gap-2 text-slate-500 dark:text-slate-400">
               <span class="p-1.5 rounded-md bg-white dark:bg-slate-900 shadow-sm border border-slate-100 dark:border-slate-700 text-emerald-500">
                 <Database class="w-4 h-4" />
               </span>
               <span class="text-xs font-bold uppercase tracking-wider">Images</span>
             </div>
             <div>
                <div class="text-3xl font-black text-slate-900 dark:text-white tabular-nums tracking-tight">{{ props.imagesCount }}</div>
                <div class="h-1 w-12 bg-emerald-500 rounded-full mt-1.5 opacity-80"></div>
             </div>
           </div>
        </div>
        
        <!-- Temp -->
        <div class="relative overflow-hidden p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/40 border border-slate-100 dark:border-slate-700/60 group/stat hover:bg-white dark:hover:bg-slate-800 transition-colors duration-200 hover:shadow-sm">
           <div class="absolute right-0 top-0 p-3 opacity-20 group-hover/stat:opacity-40 transition-opacity">
              <Box class="w-12 h-12 text-amber-500 transform -rotate-12" />
           </div>
           
           <div class="relative z-10 flex flex-col justify-between h-full gap-2">
             <div class="flex items-center gap-2 text-slate-500 dark:text-slate-400">
               <span class="p-1.5 rounded-md bg-white dark:bg-slate-900 shadow-sm border border-slate-100 dark:border-slate-700 text-amber-500">
                 <Box class="w-4 h-4" />
               </span>
               <span class="text-xs font-bold uppercase tracking-wider">Temp</span>
             </div>
             <div>
                <div class="text-3xl font-black text-slate-900 dark:text-white tabular-nums tracking-tight">{{ props.temporaryCount }}</div>
                <div class="h-1 w-12 bg-amber-500 rounded-full mt-1.5 opacity-80"></div>
             </div>
           </div>
        </div>

      </div>
    </div>
  </div>
</template>

<style scoped>
@keyframes shine {
  0% { transform: translateX(-100%) rotate(-12deg); }
  20%, 100% { transform: translateX(200%) rotate(-12deg); }
}

.animate-shine {
  animation: shine 8s infinite ease-in-out;
}
</style>
