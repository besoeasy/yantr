<script setup>
import { computed, toRefs } from "vue";
import { Bot, Check, ChevronRight } from "lucide-vue-next";

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

const { app, instanceCount } = toRefs(props);

function hashStringToUint32(value) {
  // FNV-1a 32-bit
  let hash = 0x811c9dc5;
  for (let i = 0; i < value.length; i++) {
    hash ^= value.charCodeAt(i);
    hash = Math.imul(hash, 0x01000193);
  }
  return hash >>> 0;
}

const categories = computed(() => {
  const raw = app.value?.category ?? "";
  const parts = raw
    .split(",")
    .map((c) => c.trim())
    .filter(Boolean);

  if (parts.length <= 2) {
    return { shown: parts };
  }

  const seedSource = String(app.value?.id ?? app.value?.name ?? raw);
  const h1 = hashStringToUint32(seedSource);
  const h2 = hashStringToUint32(`${seedSource}::2`);

  const firstIndex = h1 % parts.length;
  const remainingIndices = [];
  for (let i = 0; i < parts.length; i++) {
    if (i !== firstIndex) remainingIndices.push(i);
  }
  const secondIndex = remainingIndices[h2 % remainingIndices.length];

  const shown = [parts[firstIndex], parts[secondIndex]];
  return { shown };
});

const accent = computed(() => {
  const installed = Boolean(app.value?.isInstalled);
  if (installed) {
    return {
      ring: "focus:ring-emerald-500",
      border: "border-emerald-200 dark:border-emerald-500/30",
      glow: "bg-emerald-200 dark:bg-emerald-400/20",
      iconBg: "bg-emerald-900/10 dark:bg-emerald-900/30",
      iconHover: "group-hover:bg-emerald-900/15 dark:group-hover:bg-emerald-800/40",
      textHover: "group-hover:text-emerald-600 dark:group-hover:text-emerald-400",
      check: "text-emerald-500 dark:text-emerald-400",
      arrow: "text-emerald-600 dark:text-emerald-400",
      badge: "bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-200",
    };
  }

  return {
    ring: "focus:ring-sky-500",
    border: "border-slate-200 dark:border-gray-700",
    glow: "bg-sky-200 dark:bg-sky-400/20",
    iconBg: "bg-slate-900/5 dark:bg-slate-900/30",
    iconHover: "group-hover:bg-slate-900/10 dark:group-hover:bg-slate-800/40",
    textHover: "group-hover:text-sky-700 dark:group-hover:text-sky-300",
    check: "text-sky-600 dark:text-sky-300",
    arrow: "text-sky-700 dark:text-sky-300",
    badge: "bg-sky-100 text-sky-700 dark:bg-sky-500/20 dark:text-sky-200",
  };
});
</script>

<template>
  <div
    class="group relative w-full text-left overflow-hidden bg-white dark:bg-gray-900 rounded-3xl p-6 sm:p-7 border hover:shadow-2xl hover:-translate-y-2 transition-all duration-300 focus:outline-none focus:ring-4 cursor-pointer"
    :class="[accent.ring, accent.border]"
    role="button"
    tabindex="0"
    :aria-label="`Open ${app?.name ?? 'app'} details`"
    @keydown.enter.prevent="$event.currentTarget.click()"
    @keydown.space.prevent="$event.currentTarget.click()"
  >
    <!-- Installed marker -->
    <div v-if="app?.isInstalled" class="absolute inset-y-0 left-0 w-1 bg-emerald-500/70 dark:bg-emerald-400/60" aria-hidden="true"></div>

    <!-- Gradient Glow -->
    <div
      class="absolute top-0 right-0 -mt-8 -mr-8 w-40 h-40 rounded-full blur-3xl opacity-0 group-hover:opacity-70 transition-opacity duration-500"
      :class="accent.glow"
    ></div>

    <div class="relative z-10">
      <!-- Icon -->
      <div
        class="flex items-center justify-center w-20 h-20 rounded-2xl mb-5 transition-all duration-300 group-hover:scale-110"
        :class="[accent.iconBg, accent.iconHover]"
      >
        <img v-if="app?.logo" :src="app.logo" :alt="app.name" class="w-12 h-12 object-contain" loading="lazy" />
        <div v-else class="text-2xl font-bold text-slate-900 dark:text-white">
          {{ (app?.name ?? "?").slice(0, 1).toUpperCase() }}
        </div>
      </div>

      <!-- Title + badges -->
      <div class="flex items-start justify-between gap-3">
        <h3 class="text-xl font-bold text-slate-900 dark:text-white mb-1 transition-colors" :class="accent.textHover">
          {{ app.name }}
        </h3>

        <div class="flex items-center gap-2 pt-0.5">
          <span v-if="app?.isInstalled" class="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-semibold" :class="accent.badge"> Installed </span>
          <span
            v-if="instanceCount > 0"
            class="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-semibold bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-200"
            :title="`${instanceCount} running`"
          >
            {{ instanceCount }}
          </span>
        </div>
      </div>

      <p class="text-slate-600 dark:text-gray-300 leading-relaxed line-clamp-2 mb-5">
        {{ app.description || "No description available" }}
      </p>

      <!-- Categories (side-by-side) -->
      <div v-if="categories.shown.length" class="flex flex-wrap gap-2 text-sm">
        <div
          v-for="cat in categories.shown"
          :key="cat"
          class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-slate-100 text-slate-700 dark:bg-slate-800/70 dark:text-slate-200"
        >
          <Check :size="16" :class="accent.check" />
          <span class="capitalize">{{ cat }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
