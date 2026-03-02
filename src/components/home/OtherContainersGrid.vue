<script setup>
import { useI18n } from "vue-i18n";
import { ArrowRight, Box } from "lucide-vue-next";

const { t } = useI18n();

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
      class="group relative h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl overflow-hidden transition-all duration-400 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:-translate-y-1 hover:border-gray-300 dark:hover:border-zinc-600 cursor-pointer animate-fadeIn focus:outline-none focus:ring-2 focus:ring-blue-500/50"
    >
      <!-- Hover Accents -->
      <div class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-blue-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
      <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none [mask-image:linear-gradient(to_bottom,white,transparent)]"></div>

      <div class="relative z-10 flex flex-col h-full p-5">
        <div class="flex items-start gap-4 mb-4">
          <!-- Icon -->
          <div class="relative shrink-0">
             <div class="w-10 h-10 rounded-lg bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center group-hover:scale-105 group-hover:border-zinc-700 transition-all duration-500">
                <Box class="w-5 h-5 text-gray-400 dark:text-zinc-500 group-hover:text-blue-500 transition-colors" />
             </div>
             
             <!-- Status Dot -->
             <div class="absolute -top-1 -right-1 w-2.5 h-2.5 rounded-full border-2 border-white dark:border-[#0A0A0A]"
                  :class="container.state === 'running' ? 'bg-green-500 animate-pulse' : 'bg-gray-400 dark:bg-zinc-600'">
             </div>
          </div>

          <div class="overflow-hidden">
            <h3 class="font-semibold text-sm text-gray-900 dark:text-white truncate mb-1 pr-2 tracking-tight group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors" :title="container.name">
              {{ container.name.replace(/^\//, "") }}
            </h3>
            <div class="flex items-center gap-1.5">
               <div class="text-[10px] font-bold uppercase tracking-wider px-1.5 py-0.5 rounded-md"
                    :class="container.state === 'running'
                        ? 'bg-green-50/50 dark:bg-green-500/10 text-green-600 dark:text-green-500'
                        : 'bg-gray-100 dark:bg-zinc-800 text-gray-500 dark:text-zinc-400'
                    ">
                  {{ container.state }}
               </div>
            </div>
          </div>
        </div>

        <div class="mt-auto pt-4 border-t border-gray-100 dark:border-zinc-800/80 flex items-center justify-between overflow-hidden">
          <div class="flex flex-col min-w-0 pr-4">
             <span class="text-[9px] uppercase font-bold text-gray-400 dark:text-zinc-500 tracking-[0.15em] mb-0.5">{{ t("home.otherContainersGrid.image") }}</span>
             <span class="font-mono text-xs text-gray-600 dark:text-gray-400 truncate" :title="container.image">{{ container.image.split(":")[0] }}</span>
          </div>
          
          <div class="flex items-center gap-1 text-blue-600 dark:text-blue-400 font-semibold text-xs transform translate-y-8 opacity-0 group-hover:translate-y-0 group-hover:opacity-100 transition-all duration-300 cubic-bezier(0.4, 0, 0.2, 1)">
            <span>{{ t("home.otherContainersGrid.inspect") }}</span>
            <ArrowRight :size="14" class="group-hover:translate-x-1 transition-transform duration-300" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
