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
      class="group relative h-full flex flex-col bg-white dark:bg-[#0A0A0A] rounded-xl overflow-hidden transition-all duration-300 smooth-shadow hover:smooth-shadow-lg hover:-translate-y-0.5 cursor-pointer animate-fadeIn focus-visible:outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-500"
    >
      <div class="relative z-10 flex flex-col h-full p-5">

        <!-- Header: icon + name + status -->
        <div class="flex items-start gap-3.5 mb-5">
          <!-- Icon -->
          <div class="w-11 h-11 rounded-xl bg-(--surface-muted) flex items-center justify-center shrink-0 transition-transform duration-300 group-hover:scale-105">
            <Box class="w-5 h-5 text-(--text-secondary) group-hover:text-blue-500 transition-colors duration-300" />
          </div>

          <!-- Name + status -->
          <div class="min-w-0 flex-1 pt-0.5">
            <h3 class="font-semibold text-sm text-gray-900 dark:text-white line-clamp-1 tracking-tight group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors duration-300"
                :title="container.name">
              {{ container.name.replace(/^\//, "") }}
            </h3>
            <div class="mt-1 flex items-center gap-1.5">
              <span class="w-1.5 h-1.5 rounded-full shrink-0"
                    :class="container.state === 'running' ? 'bg-green-500' : 'bg-gray-400 dark:bg-zinc-600'">
              </span>
              <span class="text-[10px] font-semibold uppercase tracking-wider"
                    :class="container.state === 'running' ? 'text-green-600 dark:text-green-500' : 'text-gray-400 dark:text-zinc-500'">
                {{ container.state }}
              </span>
            </div>
          </div>
        </div>

        <!-- Image row -->
        <div class="flex items-center gap-2 px-3 py-2 rounded-lg bg-(--surface-muted) mb-5">
          <span class="text-[10px] font-bold uppercase tracking-[0.15em] text-gray-400 dark:text-zinc-500 shrink-0">Image</span>
          <span class="font-mono text-xs text-gray-700 dark:text-zinc-300 truncate min-w-0" :title="container.image">
            {{ container.image.split(":")[0] }}
          </span>
          <span v-if="container.image.includes(':')" class="text-[10px] font-mono text-gray-400 dark:text-zinc-500 shrink-0 ml-auto">
            {{ container.image.split(":")[1] }}
          </span>
        </div>

        <!-- Footer -->
        <div class="mt-auto flex items-center justify-between pt-3 border-t border-gray-100 dark:border-zinc-800/60">
          <span class="text-[10px] font-semibold uppercase tracking-[0.15em] text-gray-400 dark:text-zinc-500">
            {{ t("home.otherContainersGrid.image") }}
          </span>
          <div class="flex items-center gap-1 text-blue-600 dark:text-blue-400 font-semibold text-xs opacity-0 -translate-x-2 group-hover:opacity-100 group-hover:translate-x-0 transition-all duration-300">
            <span>{{ t("home.otherContainersGrid.inspect") }}</span>
            <ArrowRight :size="13" class="group-hover:translate-x-0.5 transition-transform duration-300" />
          </div>
        </div>

      </div>
    </div>
  </div>
</template>
