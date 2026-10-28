<script>
// Dashboard engine: two columns — the machine's spec sheet reads better
// stacked than cramped into one. See Home.vue for how meta.cols is consumed.
export const meta = { cols: 2 };
</script>
<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Cpu, Server, CircuitBoard } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";

const { t } = useI18n();

const {
  loading,
  error,
  osInfo,
  coreCount,
  cpuModel,
  board,
} = useHostMetrics();

const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));

const boardLines = computed(() => {
  const b = board.value || {};
  const lines = [];
  if (b.vendor) lines.push({ key: "vendor", label: t("quickMetrics.hostMetrics.boardVendor"), value: b.vendor });
  if (b.board) lines.push({ key: "board", label: t("quickMetrics.hostMetrics.boardName"), value: b.board });
  if (b.product) lines.push({ key: "product", label: t("quickMetrics.hostMetrics.boardProduct"), value: b.product });
  return lines;
});

const hasBoard = computed(() => boardLines.value.length > 0);
</script>

<template>
  <section class="yantra-card yantra-card-min group relative flex h-full w-full flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow] duration-200 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex w-full items-start justify-between gap-4">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-zinc-50 transition-transform duration-200 group-hover:scale-105 dark:border-zinc-800 dark:bg-zinc-900">
          <CircuitBoard class="h-5 w-5 text-zinc-600 transition-colors group-hover:text-zinc-900 yantra-text-secondary dark:group-hover:text-zinc-100" />
        </div>
        <h3 class="text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
          {{ t("quickMetrics.hostMetrics.hardware") }}
        </h3>
      </div>
      <div class="flex shrink-0 items-center gap-2 text-emerald-600 dark:text-emerald-400">
        <span class="relative flex h-2 w-2">
          <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
          <span class="relative inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
        </span>
        <span class="text-[10px] font-semibold uppercase tracking-widest">
          {{ t("quickMetrics.hostMetrics.online") }}
        </span>
      </div>
    </div>

    <div v-if="loading" class="mt-6 flex flex-1 flex-col gap-5">
      <span class="block h-9 w-48 animate-pulse rounded-lg bg-zinc-100 dark:bg-zinc-800"></span>
      <span class="block h-3 w-48 animate-pulse rounded-full bg-zinc-100 dark:bg-zinc-800"></span>
    </div>

    <div v-else-if="error" class="mt-6 flex flex-1 flex-col gap-4">
      <div class="text-4xl font-black leading-[1.1] tracking-tighter text-red-950 yantra-text-primary">
        {{ t("common.error") }}
      </div>
      <div class="mt-2 line-clamp-2 text-sm yantra-text-secondary">{{ error }}</div>
    </div>

    <template v-else>
      <!-- Identity: OS name + kernel -->
      <div class="mt-6 flex flex-col">
        <div class="truncate text-4xl font-black leading-[1.05] tracking-tighter yantra-text-primary">
          {{ osInfo.name }}
        </div>
        <div class="mt-2 truncate font-mono text-[11px] yantra-text-secondary" :title="`${osInfo.type} · ${osInfo.arch} · ${osInfo.kernel}`">
          {{ osInfo.type }} · {{ osInfo.arch }} · {{ osInfo.kernel }}
        </div>
      </div>

      <!-- CPU model + core count -->
      <div class="mt-6 flex items-center gap-3 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50">
        <Cpu class="h-4 w-4 shrink-0 yantra-text-secondary" />
        <div class="min-w-0 flex-1">
          <div class="truncate text-sm font-black tracking-tight yantra-text-primary" :title="cpuModel || ''">
            {{ cpuModel || unavailable }}
          </div>
          <div class="text-[9px] font-bold uppercase tracking-widest yantra-text-secondary">
            {{ coreCount }} {{ t("quickMetrics.hostMetrics.cores") }}
          </div>
        </div>
      </div>

      <!-- Board identity rows -->
      <div class="mt-auto flex flex-col pt-6">
        <div v-if="!hasBoard" class="text-[11px] yantra-text-secondary">
          {{ unavailable }}
        </div>
        <div
          v-for="(row, i) in boardLines"
          :key="row.key"
          class="flex items-baseline justify-between gap-4 py-2"
          :class="i < boardLines.length - 1 ? 'border-b border-zinc-100 dark:border-zinc-900' : ''"
        >
          <span class="shrink-0 text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
            {{ row.label }}
          </span>
          <span class="min-w-0 truncate text-right text-[13px] font-semibold" :title="row.value">
            {{ row.value }}
          </span>
        </div>
      </div>
    </template>
  </section>
</template>