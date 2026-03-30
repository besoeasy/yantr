<script setup>
import { useI18n } from "vue-i18n";
import { ArrowRight, Box } from "lucide-vue-next";

const { t } = useI18n();

const { containers, showHeader } = defineProps({
  containers: { type: Array, default: () => [] },
  showHeader: { type: Boolean, default: false },
});

const emit = defineEmits(["select"]);
</script>

<template>
  <div style="display: contents">
    <!-- Section Header -->
    <div v-if="showHeader" class="col-span-full flex items-center gap-2 pt-2 pb-1">
      <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-[var(--text-secondary)]">{{ t("home.dockerApps") }}</span>
      <span class="text-[10px] font-bold text-[var(--text-secondary)] opacity-50">{{ containers.length }}</span>
    </div>

    <div
      v-for="(container, index) in containers"
      :key="`other-${container.id}`"
      :style="{ animationDelay: `${index * 50}ms` }"
      @click="emit('select', container)"
      @keydown.enter.prevent="emit('select', container)"
      @keydown.space.prevent="emit('select', container)"
      role="button"
      tabindex="0"
      class="group relative h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl overflow-hidden transition-all duration-300 smooth-shadow hover:smooth-shadow-lg hover:-translate-y-0.5 cursor-pointer animate-fadeIn focus-visible:outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-500"
    >

      <div class="relative z-10 flex flex-col h-full p-5">
        <div class="flex items-start gap-4 mb-4">
          <!-- Icon -->
          <div class="relative shrink-0">
             <div class="w-10 h-10 rounded-lg bg-gray-50 dark:bg-zinc-900 flex items-center justify-center group-hover:scale-105 transition-all duration-300">
                <Box class="w-5 h-5 text-[var(--text-secondary)] group-hover:text-blue-500 transition-colors" />
             </div>
             
             <!-- Status Dot -->
             <div class="absolute -top-1 -right-1 w-2 h-2 rounded-full"
                  :class="container.state === 'running' ? 'bg-green-500' : 'bg-gray-400 dark:bg-zinc-600'">
             </div>
          </div>

          <div class="overflow-hidden">
            <h3 class="font-semibold text-sm text-gray-900 dark:text-white truncate mb-1 pr-2 tracking-tight group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors" :title="container.name">
              {{ container.name.replace(/^\//, "") }}
            </h3>
            <div class="flex items-center gap-1.5">
               <div class="text-[10px] font-bold uppercase tracking-wider"
                    :class="container.state === 'running'
                        ? 'text-green-600 dark:text-green-500'
                        : 'text-[var(--text-secondary)]'
                    ">
                  {{ container.state }}
               </div>
            </div>
          </div>
        </div>

        <div class="mt-auto pt-4 flex items-center justify-between overflow-hidden">
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
