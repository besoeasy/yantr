<script>
// Dashboard engine: one column — the memory bar and its where-the-bytes-went
// legend stack into a single readable rail; wider would only stretch the bar.
// The bar is stacked (used · buffers · slab · cached) so "near-full but mostly
// cache" needs no prose. Swap occupancy is its own card now: a separate
// device, a separate question. See Home.vue for how meta.cols is consumed.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { MemoryStick } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";

const { t } = useI18n();

const {
  metrics,
  metricsStale,
  isDegraded,
  memPct,
  memText,
  cachedText,
  buffersText,
  slabText,
} = useHostMetrics();

const mem = computed(() => metrics.value?.mem ?? null);

const isHot = computed(() => (memPct.value ?? 0) >= 90);

// The stacked segments: everything the kernel holds beyond the working set is
// a slice of the same bar, not a row of prose. Cached/buffers/slab are
// accounted subsets of used on most kernels, so the working set is whatever
// used has left after them; cumulative clamping keeps a kernel that disagrees
// from pushing later segments past 100%.
const segments = computed(() => {
  const m = mem.value;
  if (!m?.ready || !m.totalBytes) return null;
  const total = m.totalBytes;
  const share = (bytes) => Math.max(0, Math.min(100, (Number(bytes) / total) * 100));
  const buffersPct = share(m.buffersBytes);
  const slabPct = share(m.slabBytes);
  const cachedPct = share(m.cachedBytes);
  const workingPct = Math.max(
    0,
    (memPct.value ?? 0) - buffersPct - slabPct - cachedPct
  );
  const segments = [
    {
      key: "working",
      pct: workingPct,
      color: isHot.value ? "var(--color-rose-500)" : "var(--color-sky-500)",
      label: t("quickMetrics.hostMetrics.memoryUsed"),
      value: null,
    },
    {
      key: "buffers",
      pct: buffersPct,
      color: "var(--color-emerald-500)",
      label: t("quickMetrics.hostMetrics.memoryBuffers"),
      value: buffersText.value ?? "—",
    },
    {
      key: "slab",
      pct: slabPct,
      color: "var(--color-zinc-500)",
      label: t("quickMetrics.hostMetrics.memorySlab"),
      value: slabText.value ?? "—",
    },
    {
      key: "cached",
      pct: cachedPct,
      color: "color-mix(in srgb, var(--color-emerald-500) 35%, transparent)",
      label: t("quickMetrics.hostMetrics.memoryCached"),
      value: cachedText.value ?? "—",
    },
  ];
  // Clamp cumulatively so a kernel whose accounting overlaps still leaves the
  // bar a bar: earlier segments win the space they claim.
  let usedWidth = 0;
  const out = segments
    .filter((s) => s.pct > 0.4)
    .map((s) => {
      const width = Math.max(0, Math.min(100 - usedWidth, s.pct));
      usedWidth += width;
      return { ...s, pct: width };
    })
    .filter((s) => s.pct > 0.4);
  return out.length ? out : null;
});

// The non-working-set pages the kernel holds: reclaimable in a pinch, and the
// reason a near-full bar is usually fine. Each value degrades to an em-dash —
// "unknown" — because a breakdown row reading 0 could just mean an old kernel
// line was absent, and a fake zero here would look like reclaimed memory.
const legendItems = computed(() => {
  if (isDegraded("memory") || !segments.value) return [];
  return segments.value.filter((s) => s.key !== "working");
});
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-rose-200 bg-rose-50 transition-transform duration-200 group-hover:scale-105 dark:border-rose-900/50 dark:bg-rose-900/20">
          <MemoryStick class="h-5 w-5 text-rose-600 dark:text-rose-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
            {{ t("quickMetrics.hostMetrics.memory") }}
          </h3>
          <div class="mt-2 flex items-center gap-2">
            <span class="relative flex h-2 w-2">
              <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
              <span class="inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
            </span>
            <span class="text-[10px] font-semibold uppercase tracking-widest text-emerald-600 dark:text-emerald-500">{{ t("quickMetrics.hostMetrics.live") }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="relative z-10 mt-auto flex flex-col pt-6 transition-opacity duration-200" :class="metricsStale ? 'opacity-50' : ''">
      <template v-if="!isDegraded('memory') && memPct !== null && segments">
        <!-- Headline: the number on the left, the bytes on the right. -->
        <div class="flex items-baseline justify-between gap-3">
          <span
            class="text-4xl font-black tabular-nums leading-[1.1] tracking-tighter yantra-text-primary transition-colors duration-200 group-hover:-translate-y-[3px] yantra-text-primary sm:text-5xl"
            :class="isHot ? 'text-rose-600 dark:text-rose-400' : ''"
          >
            {{ Math.round(memPct) }}%
          </span>
          <span class="truncate text-[9px] font-semibold tabular-nums yantra-text-secondary">
            {{ memText ?? "" }}
          </span>
        </div>

        <!-- Stacked bar: working set · buffers · slab · cached share one rail. -->
        <div class="mb-1.5 mt-2 flex h-2 w-full overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
          <div
            v-for="seg in segments"
            :key="seg.key"
            class="h-full transition-[width] duration-200"
            :style="{ width: seg.pct + '%', backgroundColor: seg.color }"
          ></div>
        </div>

        <!-- Legend: the dots carry the colours the bar just used. -->
        <div class="mb-4 flex flex-wrap gap-x-4 gap-y-1">
          <span
            v-for="item in legendItems"
            :key="item.key"
            class="flex items-center gap-1.5 text-[9px] font-semibold yantra-text-secondary"
          >
            <span
              class="h-1.5 w-1.5 shrink-0 rounded-full"
              :style="{ backgroundColor: item.color }"
            ></span>
            {{ item.label }} {{ item.value }}
          </span>
        </div>
      </template>

      <!-- Degraded: the old text rows, because "unknown" must stay legible. -->
      <template v-else>
        <div class="mb-1 flex items-baseline justify-between gap-3">
          <span class="text-4xl font-black tabular-nums leading-[1.1] tracking-tighter yantra-text-primary sm:text-5xl">—</span>
        </div>
        <div
          v-for="(item, i) in [
            { key: 'cached', label: t('quickMetrics.hostMetrics.memoryCached'), value: isDegraded('memory') ? '—' : (cachedText.value ?? '—') },
            { key: 'buffers', label: t('quickMetrics.hostMetrics.memoryBuffers'), value: isDegraded('memory') ? '—' : (buffersText.value ?? '—') },
            { key: 'slab', label: t('quickMetrics.hostMetrics.memorySlab'), value: isDegraded('memory') ? '—' : (slabText.value ?? '—') },
          ]"
          :key="item.key"
          class="flex items-baseline justify-between gap-2 rounded-xl bg-zinc-50 p-3"
          :class="i < 2 ? 'mb-2' : ''"
        >
          <span class="text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
            {{ item.label }}
          </span>
          <span class="truncate text-sm font-black tabular-nums tracking-tight yantra-text-primary">
            {{ item.value }}
          </span>
        </div>
      </template>
    </div>
  </div>
</template>