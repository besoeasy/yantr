<script>
// One concern: the host's silicon — how many cores, what CPU, what board.
// Static identity facts, read once. The count-up easing is kept from the old
// Host System sheet: a jump reads as a glitch. Everything that qualifies the
// numbers — load, frequency, pressure — is the Performance card's business.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { Cpu } from "@lucide/vue";
import { useHostMetrics } from "../composables/useHostMetrics";
import WidgetSkeleton from "../components/WidgetSkeleton.vue";

const { t } = useI18n();
const { loading, error, coreCount, cpuModel, board } = useHostMetrics();

// The core count counts up on arrival, one small piece of life in the card —
// the same easing the previous band used, kept: a jump reads as a glitch.
function countUpTo(targetRef, targetVal, duration = 900) {
  const startVal = targetRef.value;
  const startTime = Date.now();

  const tick = () => {
    const elapsed = Date.now() - startTime;
    const progress = Math.min(elapsed / duration, 1);
    const eased = 1 - Math.pow(1 - progress, 3);
    targetRef.value = Math.round(startVal + eased * (targetVal - startVal));
    if (progress < 1) requestAnimationFrame(tick);
    else targetRef.value = targetVal;
  };

  requestAnimationFrame(tick);
}

const displayCores = ref(0);
watch(coreCount, (cores) => countUpTo(displayCores, cores));

// The board as one spec line — "ASUSTeK · ProArt X670E". Product and board
// name are near-duplicates on most boards, so the row prefers the product and
// falls back to the board string only when there is no product slot.
const boardText = computed(() => {
  const b = board.value || {};
  const parts = [b.vendor, b.product || b.board].filter(Boolean);
  return parts.length ? parts.join(" · ") : null;
});
</script>

<template>
  <WidgetSkeleton v-if="loading" />

  <section
    v-else-if="error"
    class="yantra-card yantra-card-min group relative flex h-full w-full flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow] duration-200 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <div class="flex items-center gap-4">
      <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-red-200 bg-red-50 dark:border-red-900/50 dark:bg-red-900/20">
        <Cpu class="h-5 w-5 text-red-600 dark:text-red-500" />
      </div>
      <div>
        <h3 class="text-lg font-bold leading-none tracking-tight text-red-950 dark:text-red-50">
          {{ t("home.hostCard.hardware") }}
        </h3>
        <p class="mt-2 text-[10px] font-semibold uppercase tracking-widest text-red-600 dark:text-red-400">
          {{ t("quickMetrics.hostMetrics.connectionFailed") }}
        </p>
      </div>
    </div>
    <div class="mt-auto flex flex-col pt-6">
      <div class="text-4xl font-black leading-[1.1] tracking-tighter yantra-text-primary">
        {{ t("common.error") }}
      </div>
      <div class="mt-2 line-clamp-2 text-sm yantra-text-secondary">{{ error }}</div>
    </div>
  </section>

  <section
    v-else
    class="yantra-card yantra-card-min group relative flex h-full w-full flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow] duration-200 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <div
      aria-hidden="true"
      class="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-teal-500/30 to-transparent dark:via-teal-500/40"
    ></div>

    <!-- Header -->
    <div class="flex items-center gap-4">
      <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-teal-200 bg-teal-50 transition-transform duration-200 group-hover:scale-105 dark:border-teal-900/50 dark:bg-teal-900/20">
        <Cpu class="h-5 w-5 text-teal-600 dark:text-teal-400" />
      </div>
      <h3 class="text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
        {{ t("home.hostCard.hardware") }}
      </h3>
    </div>

    <!-- The headline is the core count; the CPU and board qualify it below. -->
    <div class="mt-6">
      <div class="tabular-nums text-4xl font-black leading-[1.05] tracking-tighter yantra-text-primary transition-transform duration-200 group-hover:-translate-y-[3px]">
        {{ displayCores }}
        <span class="text-sm font-bold tracking-normal yantra-text-secondary">
          {{ t("quickMetrics.hostMetrics.cores") }}
        </span>
      </div>
    </div>

    <!-- Spec rows, a spec sheet: absent facts shorten the list rather than
         print "undefined". -->
    <div class="mt-auto flex flex-col pt-6">
      <div
        v-if="cpuModel"
        class="flex items-baseline justify-between gap-4 border-b border-zinc-100 py-2 dark:border-zinc-900"
      >
        <span class="shrink-0 text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
          {{ t("quickMetrics.hostMetrics.cpu") }}
        </span>
        <span class="min-w-0 truncate text-right text-[13px] font-semibold yantra-text-primary" :title="cpuModel">
          {{ cpuModel }}
        </span>
      </div>
      <div v-if="boardText" class="flex items-baseline justify-between gap-4 py-2">
        <span class="shrink-0 text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
          {{ t("quickMetrics.hostMetrics.boardName") }}
        </span>
        <span class="min-w-0 truncate text-right text-[13px] font-semibold yantra-text-primary" :title="boardText">
          {{ boardText }}
        </span>
      </div>
    </div>
  </section>
</template>
