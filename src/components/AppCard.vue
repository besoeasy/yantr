<script setup>
import { toRefs, computed, ref } from "vue";
import { useI18n } from 'vue-i18n'
import AppLogo from "./AppLogo.vue";
import { 
  Layers
} from "lucide-vue-next";

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
      'group relative flex h-full flex-col overflow-hidden rounded-xl bg-(--surface) p-5 smooth-shadow transition-all duration-300',
      isInteractive
        ? 'cursor-pointer hover:-translate-y-0.5'
        : 'cursor-default'
    ]"
    role="button"
    tabindex="0"
    :aria-label="`Open ${app?.name ?? 'app'} details`"
  >
    <!-- Ripple ring for installed-but-not-running click -->
    <div
      v-if="rippling"
      class="absolute inset-0 rounded-xl pointer-events-none z-20 border-2 border-yellow-400 dark:border-yellow-500 animate-ripple-ping"
    ></div>

    <div class="relative z-10 flex items-start justify-between gap-4">
      <div class="h-12 w-12 shrink-0 text-gray-400 transition-transform duration-300 group-hover:scale-105 dark:text-zinc-500">
        <AppLogo
          :logo="app?.logo"
          :name="app?.name"
          :seed="app?.id || app?.name"
          img-class="h-full w-full object-contain transition-all duration-300 group-hover:brightness-110"
          icon-class="h-full w-full text-gray-400 transition-colors duration-300 group-hover:text-blue-500 dark:text-zinc-500"
        />
      </div>

      <div class="flex min-h-12 flex-col items-end justify-start gap-1 text-right">
        <span
          v-if="appState === 'running'"
          class="inline-flex items-center gap-2 text-[11px] font-semibold uppercase tracking-[0.18em] text-emerald-600 dark:text-emerald-400"
        >
          <span class="status-dot h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
          {{ t('appCard.active') }}<span v-if="instanceCount > 1"> · {{ instanceCount }}</span>
        </span>
        <span
          v-else-if="appState === 'installed'"
          class="inline-flex items-center gap-2 text-[11px] font-semibold uppercase tracking-[0.18em] text-amber-600 dark:text-amber-400"
        >
          <span class="h-1.5 w-1.5 rounded-full bg-amber-500"></span>
          {{ t('appCard.ready') }}
        </span>
        <span
          v-if="app?.customapp"
          class="text-[11px] font-semibold uppercase tracking-[0.18em] text-blue-600 dark:text-blue-400"
        >
          Yantr built
        </span>
      </div>
    </div>

    <div class="relative z-10 flex flex-1 flex-col pt-5">
      <h3 :class="['text-lg font-semibold tracking-tight text-(--text-primary) transition-colors duration-300', isInteractive ? 'group-hover:text-blue-600 dark:group-hover:text-blue-400' : '']">
        {{ app?.name || t('appCard.unknownApp') }}
      </h3>

      <p class="mt-2 line-clamp-2 text-sm leading-relaxed text-(--text-secondary)">
        {{ app?.description || t('appCard.noDescription') }}
      </p>

      <div class="mt-auto flex items-center gap-3 pt-6 text-(--text-secondary) transition-colors duration-300">
        <div class="flex min-w-0 items-center gap-1.5 group-hover:text-(--text-primary)">
          <Layers :size="14" />
          <span class="line-clamp-1 text-[11px] font-semibold uppercase tracking-[0.18em]">
            {{ randomTag || t('appCard.application') }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>
