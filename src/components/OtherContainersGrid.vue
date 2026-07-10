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
      class="group relative flex h-full w-full cursor-pointer flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-all duration-300 hover:-translate-y-1 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] animate-fadeIn focus-visible:outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-zinc-500 dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
    >
      <div class="relative z-10 flex h-full flex-col gap-4">
        <!-- header -->
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-4">
            <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 transition-transform duration-300 group-hover:scale-105 dark:border-zinc-800 dark:bg-zinc-900">
              <Box class="h-5 w-5 text-zinc-900 dark:text-zinc-100" />
            </div>
            <div>
              <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors group-hover:text-black dark:text-white dark:group-hover:text-white">Docker App</h3>
              <div class="mt-2 flex items-center gap-1.5"
                   :class="container.state === 'running' ? 'text-emerald-600 dark:text-emerald-400' : 'text-zinc-500'">
                <div class="h-1.5 w-1.5 rounded-full bg-current" :class="{ 'animate-pulse': container.state === 'running' }"></div>
                <span class="text-[10px] font-semibold uppercase tracking-widest">{{ container.state }}</span>
              </div>
            </div>
          </div>
          <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-zinc-50 opacity-0 transition-all duration-300 group-hover:translate-x-1 group-hover:opacity-100 dark:bg-zinc-900/50">
             <ArrowRight class="h-4 w-4 text-zinc-400 group-hover:text-zinc-900 dark:text-zinc-500 dark:group-hover:text-white" />
          </div>
        </div>

        <!-- Name at bottom + image -->
        <div class="relative z-10 mt-auto flex flex-col pt-6 pb-1">
          <div class="mb-3 line-clamp-2 break-all text-2xl font-black leading-[1.1] tracking-tighter text-zinc-900 transition-colors sm:text-3xl dark:text-white" :title="container.name">
             {{ container.name.replace(/^\//, "") }}
          </div>
          
          <div class="flex items-center gap-2">
             <span class="min-w-0 truncate text-[11px] font-medium text-zinc-500 dark:text-zinc-400" :title="container.image">
                {{ container.image.split(":")[0] }}
             </span>
             <span v-if="container.image.includes(':')" class="ml-auto shrink-0 rounded bg-zinc-50 px-1.5 py-0.5 font-mono text-[9px] font-bold uppercase tracking-widest text-zinc-500 dark:bg-zinc-900/50">
                {{ container.image.split(":")[1] }}
             </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
