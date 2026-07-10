<script setup>
import { toRefs, computed, ref } from "vue";
import { useI18n } from 'vue-i18n'
import AppLogo from "./AppLogo.vue";
import { 
  Layers
} from "@lucide/vue";

const { t } = useI18n()
const props = defineProps({
  app: {
    type: Object,
    required: true,
  },
  instanceCount: {
    type: Number,
    default: 0,
  },
});

const emit = defineEmits(['select']);
const { app, instanceCount } = toRefs(props);

const randomTag = computed(() => {
  const tags = app.value?.tags;
  if (!Array.isArray(tags) || tags.length === 0) return null;
  const index = Math.floor(Math.random() * tags.length);
  return tags[index];
});

// Determine the state for UI color-coding
const appState = computed(() => {
  if (instanceCount.value > 0) return 'running';
  if (app.value?.isInstalled) return 'installed';
  return 'available';
});

const isInteractive = computed(() => appState.value !== 'installed');

const rippling = ref(false);

function handleClick() {
  if (appState.value === 'installed') {
    if (rippling.value) return;
    rippling.value = true;
    setTimeout(() => { rippling.value = false; }, 700);
    return;
  }
  emit('select');
}
</script>

<template>
  <div
    @click="handleClick"
    @keydown.enter.prevent="handleClick"
    @keydown.space.prevent="handleClick"
    :class="[
      'group relative flex h-full flex-col overflow-hidden rounded-2xl bg-white dark:bg-black p-6 border border-zinc-200 dark:border-zinc-800 transition-all duration-300',
      isInteractive
        ? 'cursor-pointer hover:border-zinc-300 dark:hover:border-zinc-700 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)] hover:-translate-y-1'
        : 'cursor-default opacity-80'
    ]"
    role="button"
    tabindex="0"
    :aria-label="`Open ${app?.name ?? 'app'} details`"
  >
    <div class="flex items-start justify-between gap-4">
      <div class="w-14 h-14 shrink-0 transition-transform duration-300 group-hover:scale-105">
        <AppLogo
          :logo="app?.logo"
          :name="app?.name"
          :seed="app?.id || app?.name"
          img-class="h-full w-full object-contain"
          icon-class="h-full w-full text-zinc-300 dark:text-zinc-700 transition-colors duration-300 group-hover:text-zinc-600 dark:group-hover:text-zinc-400"
        />
      </div>

      <div class="flex flex-col items-end gap-2 text-right">
        <span
          v-if="appState === 'running'"
          class="inline-flex items-center gap-2 text-[10px] font-semibold uppercase tracking-widest text-zinc-900 dark:text-white"
        >
          <span class="relative flex h-2 w-2">
            <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
            <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
          </span>
          {{ t('appCard.active') }}<span v-if="instanceCount > 1" class="text-zinc-400 dark:text-zinc-500 font-normal"> ({{ instanceCount }})</span>
        </span>
        <span
          v-else-if="appState === 'installed'"
          class="inline-flex items-center gap-2 text-[10px] font-semibold uppercase tracking-widest text-zinc-500 dark:text-zinc-400"
        >
          <span class="h-2 w-2 rounded-full bg-zinc-300 dark:bg-zinc-700"></span>
          {{ t('appCard.ready') }}
        </span>
        
        <span
          v-if="app?.customapp"
          class="inline-block px-2 py-0.5 rounded bg-zinc-100 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 text-[9px] font-medium uppercase tracking-[0.1em] text-zinc-600 dark:text-zinc-400"
        >
          Yantr built
        </span>
      </div>
    </div>

    <div class="mt-6 flex flex-1 flex-col">
      <h3 class="text-lg font-bold tracking-tight text-zinc-900 dark:text-white group-hover:text-black dark:group-hover:text-white transition-colors duration-300">
        {{ app?.name || t('appCard.unknownApp') }}
      </h3>

      <p class="mt-2 line-clamp-2 text-sm leading-relaxed text-zinc-500 dark:text-zinc-400">
        {{ app?.description || t('appCard.noDescription') }}
      </p>

      <div class="mt-auto pt-6 flex items-center gap-2">
        <div class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-zinc-50 dark:bg-zinc-900/50 border border-zinc-100 dark:border-zinc-800 text-zinc-500 dark:text-zinc-400 transition-colors duration-300 group-hover:border-zinc-200 dark:group-hover:border-zinc-700">
          <Layers :size="12" />
          <span class="text-[10px] font-medium tracking-wide">
            {{ randomTag || t('appCard.application') }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>
