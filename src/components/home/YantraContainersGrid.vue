<script setup>
import { ArrowRight, Clock, Activity, Box } from "lucide-vue-next";

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
      class="group relative h-full overflow-hidden bg-white dark:bg-slate-900 rounded-3xl border border-slate-200 dark:border-slate-800 hover:shadow-xl dark:hover:shadow-slate-900/50 hover:-translate-y-1 transition-all duration-300 animate-fadeIn focus:outline-none focus:ring-4 focus:ring-blue-500/20"
    >
      <!-- Background Texture -->
      <div 
        class="absolute inset-0 opacity-[0.03] dark:opacity-[0.05] pointer-events-none transition-opacity duration-300 group-hover:opacity-[0.06] dark:group-hover:opacity-[0.08]" 
        style="background-image: radial-gradient(circle at 1rem 1rem, currentColor 1px, transparent 0); background-size: 1rem 1rem;"
      ></div>

      <!-- Hover Glow -->
      <div class="absolute top-0 right-0 w-32 h-32 bg-blue-100 dark:bg-blue-900/40 rounded-full blur-3xl opacity-0 group-hover:opacity-100 transition-opacity duration-500 translate-x-1/3 -translate-y-1/3 pointer-events-none"></div>

      <div class="relative z-10 h-full flex flex-col justify-between p-6">
        <div>
           <div class="flex items-start gap-4 mb-4">
            <!-- Icon/Logo -->
            <div class="relative shrink-0">
               <div class="w-14 h-14 rounded-2xl bg-white dark:bg-slate-800 flex items-center justify-center border border-slate-100 dark:border-slate-700/60 shadow-sm group-hover:scale-110 transition-transform duration-300 overflow-hidden">
                  <img
                    v-if="container.app && container.app.logo"
                    :src="container.app.logo"
                    :alt="container.name"
                    class="w-8 h-8 object-contain"
                  />
                  <Box v-else class="w-7 h-7 text-slate-400 dark:text-slate-500" />
               </div>
               
               <!-- Status Dot -->
               <div class="absolute -bottom-1 -right-1 w-5 h-5 bg-white dark:bg-slate-900 rounded-full flex items-center justify-center border border-slate-100 dark:border-slate-800">
                  <span class="relative flex h-2.5 w-2.5">
                    <span v-if="container.state === 'running'" class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                    <span class="relative inline-flex rounded-full h-2.5 w-2.5" 
                        :class="container.state === 'running' ? 'bg-emerald-500' : 'bg-slate-400 dark:bg-slate-600'"></span>
                  </span>
               </div>
            </div>

            <div class="min-w-0 pt-1">
               <h3 class="font-bold text-lg text-slate-900 dark:text-white line-clamp-1 mb-1 leading-tight group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
                 {{ container.app ? container.app.name : container.name.replace(/^\//, "") }}
               </h3>
               
               <div class="flex items-center gap-1.5 flex-wrap">
                  <span class="text-[10px] uppercase font-bold px-1.5 py-0.5 rounded border"
                      :class="container.state === 'running' 
                        ? 'bg-emerald-50 text-emerald-600 border-emerald-100 dark:bg-emerald-500/10 dark:text-emerald-400 dark:border-emerald-500/20' 
                        : 'bg-slate-100 text-slate-500 border-slate-200 dark:bg-slate-800 dark:text-slate-400 dark:border-slate-700'">
                    {{ container.state }}
                  </span>
                  
                  <span v-if="isTemporary(container)" 
                        class="text-[10px] uppercase font-bold px-1.5 py-0.5 rounded border bg-amber-50 text-amber-600 border-amber-100 dark:bg-amber-500/10 dark:text-amber-400 dark:border-amber-500/20">
                    Temp
                  </span>
               </div>
            </div>
          </div>
          
          <!-- Uptime / Expiry Grid -->
          <div class="grid grid-cols-1 gap-2 mt-4">
             <div v-if="container.state === 'running'" class="flex items-center justify-between text-xs px-3 py-2 rounded-xl bg-slate-50 dark:bg-slate-800/50 border border-slate-100 dark:border-slate-800/50 transition-colors group-hover:bg-blue-50/50 dark:group-hover:bg-blue-900/10 group-hover:border-blue-100 dark:group-hover:border-blue-800/30">
                <span class="text-slate-500 dark:text-slate-400 font-medium flex items-center gap-1.5">
                   <Activity class="w-3.5 h-3.5" />
                   Uptime
                </span>
                <span class="font-mono font-semibold tabular-nums text-slate-700 dark:text-slate-300">{{ formatUptime(container) }}</span>
             </div>

             <div v-if="isTemporary(container)" class="flex items-center justify-between text-xs px-3 py-2 rounded-xl bg-amber-50 dark:bg-amber-900/20 border border-amber-100 dark:border-amber-900/30">
                <span class="text-amber-700 dark:text-amber-400 font-medium flex items-center gap-1.5">
                   <Clock class="w-3.5 h-3.5" />
                   Expires
                </span>
                <span class="font-mono font-bold tabular-nums"
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
        </div>

        <div class="mt-4 pt-4 flex items-center justify-between text-xs border-t border-slate-100 dark:border-slate-800">
          <span class="text-slate-500 dark:text-slate-400 font-medium group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">Manage Application</span>
          <div class="w-8 h-8 rounded-full flex items-center justify-center bg-slate-50 dark:bg-slate-800 text-slate-400 group-hover:bg-blue-100 dark:group-hover:bg-blue-900/30 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
            <ArrowRight class="w-4 h-4 group-hover:-rotate-45 transition-transform duration-300" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
