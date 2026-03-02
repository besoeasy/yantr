<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { useI18n } from 'vue-i18n'
import { Boxes, HardDrive, Database, RefreshCw, Layers } from "lucide-vue-next";
import { formatBytes } from "../../utils/metrics.js";

const { t } = useI18n()
const props = defineProps({
  images: { type: Array, default: () => [] },
  volumes: { type: Array, default: () => [] },
  intervalMs: { type: Number, default: 10_000 },
  hoverCooldownMs: { type: Number, default: 3_000 },
});

const hovered = ref(false);
const active = ref("images"); // 'images' | 'volumes'
const lastHoverFlipAt = ref(0);
let intervalHandle = null;

const hasImages = computed(() => Array.isArray(props.images) && props.images.length > 0);
const hasVolumes = computed(() => Array.isArray(props.volumes) && props.volumes.length > 0);

const availableModes = computed(() => {
  const modes = [];
  if (hasImages.value) modes.push("images");
  if (hasVolumes.value) modes.push("volumes");
  return modes;
});

function pickInitialMode() {
  if (hasImages.value) return "images";
  if (hasVolumes.value) return "volumes";
  return "images";
}

function getOtherMode(mode) {
  if (mode === "images") return hasVolumes.value ? "volumes" : "images";
  return hasImages.value ? "images" : "volumes";
}

function flip() {
  if (availableModes.value.length < 2) return;
  active.value = getOtherMode(active.value);
}

function flipOnHoverIfAllowed() {
  if (availableModes.value.length < 2) return;
  const now = Date.now();
  const cooldown = Math.max(0, Number(props.hoverCooldownMs) || 0);
  if (cooldown > 0 && now - lastHoverFlipAt.value < cooldown) return;
  lastHoverFlipAt.value = now;
  flip();
}

function startInterval() {
  stopInterval();
  if (availableModes.value.length < 2) return;
  intervalHandle = setInterval(() => {
    if (!hovered.value) flip();
  }, Math.max(1000, Number(props.intervalMs) || 10_000));
}

function stopInterval() {
  if (intervalHandle) {
    clearInterval(intervalHandle);
    intervalHandle = null;
  }
}

function onEnter() {
  hovered.value = true;
  stopInterval();
  flipOnHoverIfAllowed();
}

function onLeave() {
  hovered.value = false;
  startInterval();
}

watch(
  () => [hasImages.value, hasVolumes.value],
  () => {
    const initial = pickInitialMode();
    if (!availableModes.value.includes(active.value)) active.value = initial;
    startInterval();
  },
  { immediate: true },
);

onMounted(() => {
  active.value = pickInitialMode();
  startInterval();
});

onUnmounted(() => {
  stopInterval();
});

const metrics = computed(() => {
  if (active.value === "images") {
    const total = props.images.reduce((sum, img) => sum + (img?.sizeBytes || 0), 0);
    const unused = props.images.filter((img) => !img?.isUsed).reduce((sum, img) => sum + (img?.sizeBytes || 0), 0);
    const used = total - unused;
    return { total, used, unused };
  }

  const total = props.volumes.reduce((sum, vol) => sum + (vol?.sizeBytes || 0), 0);
  const unused = props.volumes.filter((vol) => !vol?.isUsed).reduce((sum, vol) => sum + (vol?.sizeBytes || 0), 0);
  const used = total - unused;
  return { total, used, unused };
});

const usageStats = computed(() => {
  const { total, used } = metrics.value;
  if (!total) return { percent: 0, label: '0%' };
  const pct = Math.min(100, Math.round((used / total) * 100));
  return { percent: pct, label: `${pct}%` };
});

const theme = computed(() => {
  const isImages = active.value === "images";
  if (isImages) {
    return {
      text: 'text-blue-600 dark:text-blue-500',
      bg: 'bg-blue-50/50 dark:bg-blue-500/10',
      border: 'border-blue-100 dark:border-blue-500/20',
      progress: 'bg-blue-500/80',
      hoverGlow: 'bg-gradient-to-r from-transparent via-blue-500 to-transparent',
      icon: Layers,
      label: t('quickMetrics.rotatingDiskUsage.containerImages')
    };
  }
  return {
    text: 'text-violet-600 dark:text-violet-500',
    bg: 'bg-violet-50/50 dark:bg-violet-500/10',
    border: 'border-violet-100 dark:border-violet-500/20',
    progress: 'bg-violet-500/80',
    hoverGlow: 'bg-gradient-to-r from-transparent via-violet-500 to-transparent',
    icon: Database,
    label: t('quickMetrics.rotatingDiskUsage.dataVolumes')
  };
});
</script>

<template>
  <div
    class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl p-6 overflow-hidden transition-all duration-400 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:border-gray-300 dark:hover:border-zinc-600"
    @mouseenter="onEnter"
    @mouseleave="onLeave"
  >
    <!-- Hover Accents -->
    <div class="absolute top-0 left-0 w-full h-[2px] opacity-0 group-hover:opacity-100 transition-opacity duration-500" :class="theme.hoverGlow"></div>
    <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none [mask-image:linear-gradient(to_bottom,white,transparent)]"></div>

    <!-- Rotation Progress Bar (Top) -->
    <div v-if="availableModes.length > 1 && !hovered" class="absolute top-0 left-0 right-0 h-0.5 bg-transparent overflow-hidden">
       <div class="h-full bg-gray-300 dark:bg-zinc-700 animate-progress origin-left" 
            :style="{ animationDuration: `${intervalMs}ms` }"></div>
    </div>

    <!-- Header -->
    <div class="relative z-10 flex items-start justify-between mb-6">
      <div class="flex items-center gap-4">
        <!-- Icon Container -->
        <div class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0 group-hover:scale-105 transition-all duration-500" :class="[theme.bg, theme.border]">
          <component :is="theme.icon" class="w-5 h-5" :class="theme.text" />
        </div>
        
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight group-hover:text-gray-700 dark:group-hover:text-gray-300 transition-colors">
            {{ t('quickMetrics.rotatingDiskUsage.title') }}
          </h3>
          <div class="mt-1 flex items-center gap-1.5">
             <span class="text-[11px] font-medium text-gray-500 dark:text-zinc-400 uppercase tracking-wider transition-colors duration-300">
               {{ theme.label }}
             </span>
          </div>
        </div>
      </div>
      
      <div class="w-8 h-8 rounded-md flex items-center justify-center text-gray-400 bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 transition-all duration-300"
           :class="hovered ? 'opacity-100 text-gray-900 dark:text-white border-gray-300 dark:border-zinc-700' : 'opacity-0 scale-95'">
        <RefreshCw class="w-3.5 h-3.5" :class="{'animate-spin-slow': hovered}" />
      </div>
    </div>

    <!-- Main Content -->
    <div class="relative z-10 flex-1 flex flex-col justify-end">
      <transition 
          mode="out-in"
          enter-active-class="transition duration-200 ease-out"
          enter-from-class="transform translate-y-2 opacity-0"
          enter-to-class="transform translate-y-0 opacity-100"
          leave-active-class="transition duration-150 ease-in"
          leave-from-class="transform translate-y-0 opacity-100"
          leave-to-class="transform -translate-y-2 opacity-0"
      >
        <div :key="active" class="flex flex-col">
          <!-- Big Metric -->
          <div class="mb-5">
             <div class="text-[10px] font-bold uppercase tracking-[0.2em] text-gray-400 dark:text-zinc-500 mb-2 flex justify-between">
               <span>{{ t('quickMetrics.rotatingDiskUsage.usedSpace') }}</span>
             </div>
             
             <div class="flex items-baseline gap-2">
                <div class="text-4xl font-bold tracking-tighter leading-none text-gray-900 dark:text-white tabular-nums">
                  {{ usageStats.percent }}<span class="text-2xl text-gray-400 dark:text-zinc-500 font-semibold ml-0.5">%</span>
                </div>
                <div class="font-mono text-sm font-semibold text-gray-500 dark:text-zinc-500 ml-auto">
                  {{ formatBytes(metrics.used) }}
                </div>
             </div>
          </div>

          <!-- Progress Bar -->
          <div class="space-y-3">
            <div class="h-1.5 w-full bg-gray-100 dark:bg-zinc-800/80 rounded-full overflow-hidden flex">
               <div class="h-full transition-all duration-1000 ease-out" 
                    :class="theme.progress"
                    :style="{ width: `${usageStats.percent}%` }">
               </div>
            </div>
            
            <div class="flex justify-between items-center text-[10px] font-bold uppercase tracking-wider text-gray-400 dark:text-zinc-500">
              <span>0%</span>
              <span class="flex items-center gap-1.5">
                 {{ t('quickMetrics.rotatingDiskUsage.total') }}: <span class="text-gray-900 dark:text-white font-mono">{{ formatBytes(metrics.total) }}</span>
              </span>
            </div>
          </div>
        </div>
      </transition>
    </div>
  </div>
</template>

<style scoped>
@keyframes progress {
  from { transform: scaleX(0); }
  to { transform: scaleX(1); }
}
.animate-progress {
  animation-name: progress;
  animation-timing-function: linear;
  animation-iteration-count: infinite;
}
.animate-spin-slow {
  animation: spin 3s linear infinite;
}
</style>
