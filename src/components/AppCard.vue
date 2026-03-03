<script setup>
import { toRefs, computed, ref } from "vue";
import { useI18n } from 'vue-i18n'
import { 
  Bot, 
  ArrowRight, 
  Tag, 
  Activity, 
  Box,
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

// Determine the state for UI color-coding
const appState = computed(() => {
  if (instanceCount.value > 0) return 'running';
  if (app.value?.isInstalled) return 'installed';
  return 'available';
});

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
      'group relative flex flex-col h-full bg-white dark:bg-[#0A0A0A] border rounded-xl p-5 overflow-hidden transition-all duration-300',
      appState === 'installed'
        ? 'cursor-default border-gray-200 dark:border-zinc-800'
        : 'cursor-pointer border-gray-200 dark:border-zinc-800 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:-translate-y-1 hover:border-gray-300 dark:hover:border-zinc-600',
      rippling ? 'border-yellow-400 dark:border-yellow-500' : ''
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
    <!-- Subtle Top Accent Line that reveals on hover (not for installed state) -->
    <div v-if="appState !== 'installed'" class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-blue-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>

    <!-- Background Pattern (Subtle grid that animates on hover) -->
    <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none [mask-image:linear-gradient(to_bottom,white,transparent)]"></div>

    <!-- Top row: Logo & Status -->
    <div class="relative z-10 flex items-start justify-between mb-5">
      <!-- Minimalist Logo Container -->
      <div class="w-12 h-12 rounded-lg bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center p-2.5 shrink-0 transition-all duration-500 group-hover:scale-105 group-hover:shadow-md">
        <img
          v-if="app?.logo"
          :src="app.logo"
          :alt="app.name"
          class="w-full h-full object-contain filter group-hover:brightness-110 transition-all"
          loading="lazy"
        />
        <Bot v-else :size="24" class="text-gray-400 dark:text-zinc-500 group-hover:text-blue-500 transition-colors" />
      </div>

      <!-- Status Indicator (always expanded on mobile, expands on hover on desktop) -->
      <div class="flex items-center justify-end">
        <!-- Running -->
        <div v-if="appState === 'running'" class="flex items-center gap-2 bg-green-50 dark:bg-green-500/10 text-green-600 dark:text-green-400 border border-green-200 dark:border-green-500/20 px-2 py-1.5 rounded-full">
          <div class="w-2 h-2 rounded-full bg-green-500 animate-pulse shrink-0"></div>
          <span class="text-[11px] font-bold tracking-wide uppercase whitespace-nowrap">
            {{ t('appCard.active') }} ({{ instanceCount }})
          </span>
        </div>
        
        <!-- Installed / Ready -->
        <div v-else-if="appState === 'installed'" class="flex items-center gap-2 bg-yellow-50 dark:bg-yellow-500/10 text-yellow-600 dark:text-yellow-500 border border-yellow-200 dark:border-yellow-500/20 px-2 py-1.5 rounded-full">
          <div class="w-2 h-2 rounded-full bg-yellow-500 shrink-0"></div>
          <span class="text-[11px] font-bold tracking-wide uppercase whitespace-nowrap">
            {{ t('appCard.ready') }}
          </span>
        </div>
      </div>
    </div>

    <!-- Content -->
    <div class="relative z-10 flex-1 flex flex-col">
      <!-- Clean Typography -->
      <h3 :class="['text-lg font-semibold text-gray-900 dark:text-white mb-1.5 tracking-tight transition-colors duration-300', appState !== 'installed' ? 'group-hover:text-blue-600 dark:group-hover:text-blue-400' : '']">
        {{ app?.name || t('appCard.unknownApp') }}
      </h3>

      <p class="text-sm text-gray-500 dark:text-zinc-400 leading-relaxed line-clamp-2 mb-6 font-medium">
        {{ app?.description || t('appCard.noDescription') }}
      </p>

      <div class="mt-auto pt-4 border-t border-gray-100 dark:border-zinc-800/80 flex items-center justify-between overflow-hidden">
        <!-- Category tag -->
        <div class="flex items-center gap-1.5 text-gray-400 dark:text-zinc-500 group-hover:text-gray-600 dark:group-hover:text-zinc-300 transition-colors duration-300">
          <Layers :size="14" />
          <span class="text-[11px] font-semibold uppercase tracking-wider">
            {{ app?.tags?.[0] || t('appCard.application') }}
          </span>
        </div>
        
        <!-- Action hint -->
        <div v-if="appState !== 'installed'" class="flex items-center gap-1 text-blue-600 dark:text-blue-400 font-semibold text-xs">
          <span>{{ t('appCard.manage') }}</span>
          <ArrowRight :size="14" class="group-hover:translate-x-1 transition-transform duration-300" />
        </div>
        <div v-else class="flex items-center gap-1 text-yellow-500 dark:text-yellow-400 font-semibold text-xs opacity-60">
          <span>{{ t('appCard.notRunning') }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
