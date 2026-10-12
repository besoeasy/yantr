<script setup>
import { computed, useId } from 'vue'

// Sparkline — a dependency-free SVG area/line plot for a short rolling series.
//
// Deliberately hand-rolled rather than a charting library: the project's only
// runtime deps are vue, vue-router, vue-i18n and lucide, and a sparkline is
// not worth a third-party bundle. The viewBox is stretched with
// preserveAspectRatio="none" so the path fills any width, and the stroke is
// pinned with a non-scaling stroke so it stays 1px at every size.
const props = defineProps({
  points: { type: Array, default: () => [] },
  max: { type: Number, default: 100 },
  label: { type: String, default: '' },
  // Accent tone. main.css is a hand-curated utility layer, so each tone needs
  // its own stroke/stop-color rules present there — adding a tone means adding
  // the four matching classes, not just this string.
  tone: { type: String, default: 'sky' },
})

const W = 100
const H = 28
const gradientId = `spark-${useId().replace(/:/g, '')}`

const coords = computed(() => {
  const pts = props.points
  // A rate needs two readings to exist, so a single point is not a line yet.
  // The backend sends ready:false for exactly that first sample, and holding a
  // flat baseline reads as "waiting" rather than as a dead host.
  if (!Array.isArray(pts) || pts.length < 2) return null
  const step = W / (pts.length - 1)
  return pts.map((v, i) => {
    const clamped = Math.min(Math.max(Number(v) || 0, 0), props.max)
    return [i * step, H - (clamped / props.max) * H]
  })
})

const linePath = computed(() => {
  const c = coords.value
  if (!c) return ''
  return c.map(([x, y], i) => `${i === 0 ? 'M' : 'L'}${x.toFixed(2)},${y.toFixed(2)}`).join(' ')
})

const areaPath = computed(() => {
  if (!coords.value) return ''
  return `${linePath.value} L${W},${H} L0,${H} Z`
})
</script>

<template>
  <svg
    class="h-full w-full"
    :viewBox="`0 0 ${W} ${H}`"
    preserveAspectRatio="none"
    role="img"
    :aria-label="label"
  >
    <defs>
      <linearGradient :id="gradientId" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" :class="`stop-color-${tone}-500/25`" />
        <stop offset="100%" :class="`stop-color-${tone}-500/0`" />
      </linearGradient>
    </defs>

    <path v-if="areaPath" :d="areaPath" :fill="`url(#${gradientId})`" />
    <path
      v-if="linePath"
      :d="linePath"
      fill="none"
      :class="`stroke-${tone}-500 dark:stroke-${tone}-400`"
      stroke-width="1.5"
      stroke-linejoin="round"
      stroke-linecap="round"
      vector-effect="non-scaling-stroke"
    />
    <line
      v-else
      x1="0"
      :y1="H"
      :x2="W"
      :y2="H"
      class="stroke-zinc-300 dark:stroke-zinc-700"
      stroke-width="1"
      stroke-dasharray="3 3"
      vector-effect="non-scaling-stroke"
    />
  </svg>
</template>