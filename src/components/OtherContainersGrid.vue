<script setup>
import { useI18n } from "vue-i18n";
import { ArrowRight, Box } from "@lucide/vue";

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
      class="group relative h-full flex flex-col bg-white dark:bg-[#0A0A0A] rounded-xl overflow-hidden transition-all duration-300 hover:-translate-y-0.5 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 border border-gray-100 dark:border-zinc-800 text-left w-full cursor-pointer animate-fadeIn focus-visible:outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gray-500"
    >
      <!-- top accent line -->
      <div class="absolute top-0 left-0 w-full h-0.5 bg-gray-500 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>

      <div class="relative z-10 p-5 flex flex-col h-full gap-4">
        <!-- header -->
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-3 min-w-0">
            <div class="w-9 h-9 rounded-lg bg-gray-50 dark:bg-zinc-800/50 border border-gray-100 dark:border-zinc-700/50 flex items-center justify-center shrink-0 transition-transform group-hover:scale-110">
              <Box class="w-4.5 h-4.5 text-gray-500 dark:text-zinc-400 group-hover:text-gray-700 dark:group-hover:text-zinc-200 transition-colors" />
            </div>
            <div class="min-w-0">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight leading-none truncate">Docker App</h3>
              <div class="flex items-center gap-1.5 mt-1.5"
                   :class="container.state === 'running' ? 'text-emerald-600 dark:text-emerald-400' : 'text-gray-400 dark:text-zinc-500'">
                <div class="w-1.5 h-1.5 rounded-full bg-current" :class="{ 'animate-pulse': container.state === 'running' }"></div>
                <span class="text-[10px] font-bold uppercase tracking-wider">{{ container.state }}</span>
              </div>
            </div>
          </div>
          <div class="flex items-center justify-center w-8 h-8 rounded-full bg-gray-50 dark:bg-zinc-900/50 opacity-0 group-hover:opacity-100 group-hover:translate-x-1 transition-all duration-300 shrink-0">
             <ArrowRight class="w-4 h-4 text-gray-400 dark:text-zinc-500 group-hover:text-gray-700 dark:group-hover:text-zinc-200" />
          </div>
        </div>

        <!-- Name at bottom + image -->
        <div class="mt-auto pt-6 pb-1 flex flex-col relative z-10">
          <div class="text-2xl sm:text-3xl font-black tracking-tighter text-gray-900 dark:text-white group-hover:text-gray-700 dark:group-hover:text-gray-300 transition-colors line-clamp-2 leading-[1.1] mb-3 break-all" :title="container.name">
             {{ container.name.replace(/^\//, "") }}
          </div>
          
          <div class="flex items-center gap-2">
             <span class="text-[11px] font-medium text-gray-500 dark:text-zinc-400 truncate min-w-0" :title="container.image">
                {{ container.image.split(":")[0] }}
             </span>
             <span v-if="container.image.includes(':')" class="ml-auto text-[9px] font-mono font-bold uppercase tracking-widest px-1.5 py-0.5 rounded bg-gray-50 dark:bg-zinc-900/50 text-gray-400 dark:text-zinc-500 shrink-0">
                {{ container.image.split(":")[1] }}
             </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
