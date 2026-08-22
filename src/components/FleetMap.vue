<script setup>
import { computed, ref } from "vue";

const props = defineProps({
  countries: {
    type: Array,
    default: () => [],
  },
});

const hover = ref(null);

const maxCount = computed(() =>
  props.countries.reduce((max, item) => Math.max(max, item.count || 0), 0)
);

const points = computed(() =>
  props.countries
    .filter((item) => item.code && item.code !== "??" && (item.lat || item.lng))
    .map((item) => {
      const x = ((Number(item.lng) + 180) / 360) * 1000;
      const y = ((90 - Number(item.lat)) / 180) * 500;
      const t = maxCount.value ? item.count / maxCount.value : 0;
      return {
        ...item,
        x,
        y,
        r: 7 + t * 18,
      };
    })
);

function projectLabel(point) {
  return {
    left: `${(point.x / 1000) * 100}%`,
    top: `${(point.y / 500) * 100}%`,
  };
}
</script>

<template>
  <div class="relative overflow-hidden rounded-2xl border border-zinc-200 bg-zinc-50 dark:border-zinc-800 dark:bg-[#0A0A0A]">
    <svg viewBox="0 0 1000 500" class="block h-auto w-full" role="img" aria-label="Nodes by country">
      <defs>
        <pattern id="fleet-grid" width="50" height="50" patternUnits="userSpaceOnUse">
          <path d="M 50 0 L 0 0 0 50" fill="none" class="stroke-zinc-200 dark:stroke-zinc-800" stroke-width="1" />
        </pattern>
      </defs>
      <rect width="1000" height="500" fill="url(#fleet-grid)" />
      <line
        v-for="y in [125, 250, 375]"
        :key="'h'+y"
        x1="0"
        :y1="y"
        x2="1000"
        :y2="y"
        class="stroke-zinc-200/80 dark:stroke-zinc-800"
        stroke-dasharray="4 8"
      />
      <line
        v-for="x in [250, 500, 750]"
        :key="'v'+x"
        :x1="x"
        y1="0"
        :x2="x"
        y2="500"
        class="stroke-zinc-200/80 dark:stroke-zinc-800"
        stroke-dasharray="4 8"
      />
      <g>
        <circle
          v-for="point in points"
          :key="point.code"
          :cx="point.x"
          :cy="point.y"
          :r="point.r"
          class="fill-sky-500/25 stroke-sky-600 dark:fill-sky-400/20 dark:stroke-sky-400"
          stroke-width="2"
          @mouseenter="hover = point"
          @mouseleave="hover = null"
        />
      </g>
    </svg>

    <div
      v-if="hover"
      class="pointer-events-none absolute -translate-x-1/2 -translate-y-full rounded-lg border border-zinc-200 bg-white px-3 py-2 text-xs shadow-lg dark:border-zinc-700 dark:bg-zinc-900"
      :style="projectLabel(hover)"
    >
      <p class="font-semibold text-zinc-900 dark:text-white">{{ hover.name }}</p>
      <p class="tabular-nums text-zinc-500 dark:text-zinc-400">{{ hover.count }} {{ hover.count === 1 ? 'node' : 'nodes' }}</p>
    </div>

    <p class="absolute bottom-3 left-4 text-[10px] font-bold uppercase tracking-widest text-zinc-400 dark:text-zinc-600">
      Equirectangular · country centroids
    </p>
  </div>
</template>
