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
      'group relative flex h-full flex-col overflow-hidden rounded-[24px] p-6 transition-all duration-500 border border-blue-100/50 dark:border-white/5 bg-gradient-to-br from-blue-50/50 to-indigo-50/50 dark:from-[#111827] dark:to-[#09090b]',
      isInteractive
        ? 'cursor-pointer hover:-translate-y-2 hover:shadow-[0_20px_40px_-15px_rgba(59,130,246,0.4)] dark:hover:shadow-[0_20px_40px_-15px_rgba(59,130,246,0.3)]'
        : 'cursor-default opacity-90'
    ]"
    role="button"
    tabindex="0"
    :aria-label="`Open ${app?.name ?? 'app'} details`"
  >
    <!-- Animated Gradient Glow -->
    <div class="absolute -top-[50%] -left-[50%] w-[200%] h-[200%] bg-[radial-gradient(circle_at_50%_50%,_rgba(59,130,246,0.15)_0%,_transparent_50%)] dark:bg-[radial-gradient(circle_at_50%_50%,_rgba(59,130,246,0.12)_0%,_transparent_40%)] opacity-0 group-hover:opacity-100 transition-opacity duration-1000 pointer-events-none group-hover:animate-[spin_20s_linear_infinite]"></div>

    <!-- Ripple ring for installed-but-not-running click -->
    <div
      v-if="rippling"
      class="absolute inset-0 rounded-[24px] pointer-events-none z-20 border-2 border-yellow-400 dark:border-yellow-500 animate-ripple-ping"
    ></div>

    <div class="relative z-10 flex items-start justify-between gap-4">
      <div class="relative w-16 h-16 rounded-2xl flex items-center justify-center shrink-0 bg-white/60 dark:bg-black/40 backdrop-blur-xl border border-white dark:border-white/5 shadow-lg shadow-blue-500/10 group-hover:shadow-blue-500/30 transition-all duration-500 group-hover:scale-105 overflow-hidden">
        <AppLogo
          :logo="app?.logo"
          :name="app?.name"
          :seed="app?.id || app?.name"
          img-class="h-full w-full object-contain p-2.5 transition-all duration-500 group-hover:brightness-110 group-hover:scale-110"
          icon-class="h-full w-full text-blue-400 dark:text-blue-500 transition-colors duration-500 group-hover:text-blue-600 dark:group-hover:text-blue-400"
        />
      </div>

      <div class="flex min-h-12 flex-col items-end justify-start gap-1.5 text-right">
        <span
          v-if="appState === 'running'"
          class="inline-flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-[0.2em] text-emerald-600 dark:text-emerald-400 drop-shadow-sm"
        >
          <span class="status-dot h-1.5 w-1.5 rounded-full bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.8)]"></span>
          {{ t('appCard.active') }}<span v-if="instanceCount > 1" class="opacity-80"> · {{ instanceCount }}</span>
        </span>
        <span
          v-else-if="appState === 'installed'"
          class="inline-flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-[0.2em] text-amber-600 dark:text-amber-400 drop-shadow-sm"
        >
          <span class="h-1.5 w-1.5 rounded-full bg-amber-500 shadow-[0_0_8px_rgba(245,158,11,0.8)]"></span>
          {{ t('appCard.ready') }}
        </span>
        <span
          v-if="app?.customapp"
          class="inline-block mt-1 px-2 py-0.5 rounded-md bg-blue-100/50 dark:bg-blue-900/30 border border-blue-200/50 dark:border-blue-800/50 text-[9px] font-bold uppercase tracking-[0.2em] text-blue-700 dark:text-blue-400 backdrop-blur-sm"
        >
          Yantr built
        </span>
      </div>
    </div>

    <div class="relative z-10 flex flex-1 flex-col pt-6">
      <h3 :class="['text-xl font-black tracking-tight text-blue-950 dark:text-white transition-colors duration-300', isInteractive ? 'group-hover:text-blue-600 dark:group-hover:text-blue-400' : '']">
        {{ app?.name || t('appCard.unknownApp') }}
      </h3>

      <p class="mt-2.5 line-clamp-2 text-sm font-medium leading-relaxed text-blue-900/70 dark:text-zinc-400 transition-colors duration-300 group-hover:text-blue-900/90 dark:group-hover:text-zinc-300">
        {{ app?.description || t('appCard.noDescription') }}
      </p>

      <div class="mt-auto flex items-center gap-3 pt-6 text-blue-800/60 dark:text-zinc-500 transition-colors duration-300">
        <div class="flex min-w-0 items-center gap-1.5 group-hover:text-blue-600 dark:group-hover:text-blue-400">
          <Layers :size="14" />
          <span class="line-clamp-1 text-[10px] font-bold uppercase tracking-[0.2em]">
            {{ randomTag || t('appCard.application') }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>
