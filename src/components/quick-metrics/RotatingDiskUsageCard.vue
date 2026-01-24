<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { Boxes, HardDrive } from "lucide-vue-next";
import DonutChart from "../charts/DonutChart.vue";
import { formatBytes } from "../../utils/metrics.js";

const props = defineProps({
  images: { type: Array, default: () => [] },
  volumes: { type: Array, default: () => [] },
  intervalMs: { type: Number, default: 10_000 },
  hoverCooldownMs: { type: Number, default: 3_000 },
});

const isDark = ref(false);
let themeObserver = null;

function syncTheme() {
  isDark.value = document.documentElement.classList.contains('dark');
}

onMounted(() => {
  syncTheme();
  themeObserver = new MutationObserver(syncTheme);
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] });
  const media = window.matchMedia('(prefers-color-scheme: dark)');
  media.addEventListener('change', syncTheme);
});

onUnmounted(() => {
  const media = window.matchMedia('(prefers-color-scheme: dark)');
  media.removeEventListener('change', syncTheme);
  if (themeObserver) {
    themeObserver.disconnect();
    themeObserver = null;
  }
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

const donut = computed(() => {
  const value = metrics.value;
  const isImages = active.value === "images";

  return {
    total: value.total,
    used: value.used,
    unused: value.unused,
    series: [value.used, value.unused],
    labels: ["Used", "Unused"],
    colors: isImages ? ["#3b82f6", "#fdba74"] : ["#6366f1", "#fdba74"],
    donutLabel: isImages ? "Images" : "Volumes",
  };
});

const usedPercent = computed(() => {
  const total = Number(donut.value.total) || 0;
  const used = Number(donut.value.used) || 0;
  if (!total) return 0;
  return Math.max(0, Math.min(100, Math.round((used / total) * 100)));
});

const unusedPercent = computed(() => {
  const total = Number(donut.value.total) || 0;
  const unused = Number(donut.value.unused) || 0;
  if (!total) return 0;
  return Math.max(0, Math.min(100, Math.round((unused / total) * 100)));
});

const ui = computed(() => {
  const isImages = active.value === "images";

  return {
    title: isImages ? "Image Usage" : "Volume Usage",
    subtitle: isImages ? "Used vs unused storage" : "Used vs unused storage",
    accentText: isImages ? "group-hover:text-blue-700 dark:group-hover:text-blue-200" : "group-hover:text-indigo-700 dark:group-hover:text-indigo-200",
    borderHover: isImages ? "group-hover:border-blue-300/60 dark:group-hover:border-blue-500/30" : "group-hover:border-indigo-300/60 dark:group-hover:border-indigo-500/30",
    orbA: isImages ? "bg-blue-300/35 group-hover:bg-blue-400/45 dark:bg-blue-500/20 dark:group-hover:bg-blue-500/30" : "bg-indigo-300/35 group-hover:bg-indigo-400/45 dark:bg-indigo-500/20 dark:group-hover:bg-indigo-500/30",
    orbB: isImages ? "bg-slate-300/30 group-hover:bg-slate-400/40 dark:bg-slate-500/20 dark:group-hover:bg-slate-500/30" : "bg-purple-300/30 group-hover:bg-purple-400/40 dark:bg-purple-600/20 dark:group-hover:bg-purple-600/30",
    iconGlow: isImages ? "bg-blue-400/25 dark:bg-blue-500/20" : "bg-indigo-400/25 dark:bg-indigo-500/20",
    iconBg: isImages ? "from-blue-500 to-slate-600" : "from-indigo-500 to-purple-600",
    icon: isImages ? Boxes : HardDrive,
    gradient: isImages ? "from-blue-200/60 via-slate-200/30 to-white/80 dark:from-blue-600/25 dark:via-slate-600/10 dark:to-gray-900" : "from-indigo-200/60 via-purple-200/30 to-white/80 dark:from-indigo-600/25 dark:via-purple-600/10 dark:to-gray-900",
  };
});
</script>

<template>
  <div
    class="relative h-full overflow-hidden group rounded-2xl transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)] hover:-translate-y-1"
    @mouseenter="onEnter"
    @mouseleave="onLeave"
  >
    <div class="absolute inset-0 bg-white dark:bg-gray-900">
      <div class="absolute inset-0 bg-linear-to-br z-10" :class="ui.gradient"></div>
      <div
        class="absolute top-0 right-0 w-64 h-64 rounded-full blur-3xl -translate-y-1/2 translate-x-1/2 transition-colors duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"
        :class="ui.orbA"
      ></div>
      <div
        class="absolute bottom-0 left-0 w-48 h-48 rounded-full blur-3xl translate-y-1/2 -translate-x-1/2 transition-colors duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"
        :class="ui.orbB"
      ></div>
    </div>

    <div
      class="relative z-20 h-full p-6 flex flex-col border border-slate-200/80 dark:border-slate-700/60 rounded-2xl backdrop-blur-sm transition-none"
      :class="ui.borderHover"
    >
      <div class="flex items-start justify-between gap-4">
        <div class="flex items-center gap-4">
          <div class="relative">
            <div class="absolute inset-0 rounded-xl blur-lg group-hover:blur-xl transition-all duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]" :class="ui.iconGlow"></div>
            <div
              class="relative w-12 h-12 bg-linear-to-br rounded-xl flex items-center justify-center shadow-lg group-hover:scale-110 transition-transform duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"
              :class="ui.iconBg"
            >
              <component :is="ui.icon" class="w-6 h-6 text-white" />
            </div>
          </div>

          <div>
            <h3 class="text-lg font-bold text-slate-900 dark:text-white mb-1 transition-colors" :class="ui.accentText">Disk Usage</h3>
            <p class="text-sm font-medium text-slate-600 dark:text-gray-400 group-hover:text-slate-700 dark:group-hover:text-gray-300 transition-colors">
              {{ ui.title }}
              <span v-if="availableModes.length > 1" class="text-slate-500 dark:text-gray-500">• flips every 10s</span>
            </p>
          </div>
        </div>

        <div class="inline-flex items-center gap-2 rounded-xl bg-white/70 dark:bg-white/5 border border-slate-200/70 dark:border-slate-700/60 px-3 py-1.5">
          <span class="text-xs font-semibold text-slate-600 dark:text-gray-300">Total</span>
          <span class="text-xs font-extrabold text-slate-900 dark:text-white tabular-nums">{{ formatBytes(donut.total) }}</span>
        </div>
      </div>

      <div class="mt-5 flex-1">
        <div v-if="donut.total === 0" class="rounded-2xl border border-dashed border-slate-200 dark:border-slate-700/60 bg-white/70 dark:bg-white/5 px-4 py-6 text-center">
          <div class="text-sm font-semibold text-slate-700 dark:text-gray-200">No disk usage data</div>
          <div class="text-xs text-slate-500 dark:text-gray-400 mt-1">Nothing to visualize yet.</div>
        </div>

        <div v-else class="disk-flip-surface rounded-2xl">
          <transition name="disk-flip" mode="out-in">
            <div :key="active" class="rounded-2xl p-4 bg-white/60 dark:bg-transparent">
              <div class="flex items-center justify-between gap-3 mb-3">
                <div>
                  <div class="text-sm font-bold text-slate-900 dark:text-white">Breakdown</div>
                  <div class="text-xs text-slate-500 dark:text-gray-400">{{ ui.subtitle }}</div>
                </div>
                <div class="text-xs font-semibold text-slate-500 dark:text-gray-400">
                  <span v-if="hovered" class="text-slate-700 dark:text-gray-300">Hover: instant flip</span>
                  <span v-else>Hover to flip</span>
                </div>
              </div>

              <div class="grid grid-cols-1 sm:grid-cols-5 gap-4 items-start">
                <div class="sm:col-span-2">
                  <DonutChart
                    :series="donut.series"
                    :labels="donut.labels"
                    :colors="donut.colors"
                    :height="180"
                    :donut-label="donut.donutLabel"
                    :theme="isDark ? 'dark' : 'light'"
                    :value-formatter="formatBytes"
                    :total-formatter="() => formatBytes(donut.total)"
                  />
                </div>

                <div class="sm:col-span-3 space-y-3">
                  <div class="rounded-xl px-3 py-2">
                    <div class="flex items-center justify-between gap-3 text-xs">
                      <div class="inline-flex items-center gap-2 min-w-0">
                        <span class="w-2 h-2 rounded-full" :style="{ backgroundColor: donut.colors[0] }"></span>
                        <span class="text-slate-600 dark:text-gray-300 font-semibold">Used</span>
                      </div>
                      <div class="shrink-0 text-slate-800 dark:text-gray-200 font-bold tabular-nums">
                        {{ formatBytes(donut.used) }}
                        <span class="text-slate-500 dark:text-gray-400 font-semibold">({{ usedPercent }}%)</span>
                      </div>
                    </div>
                    <div class="mt-2 h-2 rounded-full bg-slate-200/70 dark:bg-black/20 overflow-hidden">
                      <div class="h-full rounded-full" :style="{ width: `${usedPercent}%`, backgroundColor: donut.colors[0] }"></div>
                    </div>
                  </div>

                  <div class="rounded-xl px-3 py-2">
                    <div class="flex items-center justify-between gap-3 text-xs">
                      <div class="inline-flex items-center gap-2 min-w-0">
                        <span class="w-2 h-2 rounded-full" :style="{ backgroundColor: donut.colors[1] }"></span>
                        <span class="text-slate-600 dark:text-gray-300 font-semibold">Unused</span>
                      </div>
                      <div class="shrink-0 text-slate-800 dark:text-gray-200 font-bold tabular-nums">
                        {{ formatBytes(donut.unused) }}
                        <span class="text-slate-500 dark:text-gray-400 font-semibold">({{ unusedPercent }}%)</span>
                      </div>
                    </div>
                    <div class="mt-2 h-2 rounded-full bg-slate-200/70 dark:bg-black/20 overflow-hidden">
                      <div class="h-full rounded-full" :style="{ width: `${unusedPercent}%`, backgroundColor: donut.colors[1] }"></div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </transition>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.disk-flip-surface {
  /* keep a stable surface for ApexCharts */
}

.disk-flip-enter-active,
.disk-flip-leave-active {
  transition:
    transform 260ms cubic-bezier(0.2, 0.9, 0.2, 1),
    opacity 260ms cubic-bezier(0.2, 0.9, 0.2, 1);
}

.disk-flip-enter-from {
  opacity: 0;
  transform: translateY(8px);
}

.disk-flip-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
