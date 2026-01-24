<script setup>
import { computed } from "vue";
import { Sparkles, Activity } from "lucide-vue-next";

const props = defineProps({
  runningApps: { type: Number, default: 0 },
  totalVolumes: { type: Number, default: 0 },
  temporaryCount: { type: Number, default: 0 },
  imagesCount: { type: Number, default: 0 },
});

const greeting = computed(() => {
  const hour = new Date().getHours();
  if (hour < 12) return "Good morning";
  if (hour < 18) return "Good afternoon";
  return "Good evening";
});
</script>

<template>
  <div class="relative h-full overflow-hidden group rounded-3xl transition-all duration-500 hover:-translate-y-1">
    <div class="absolute inset-0 bg-white dark:bg-black"></div>

    <div class="relative z-20 h-full p-6 flex flex-col gap-6 border border-black/10 dark:border-white/10 rounded-3xl backdrop-blur-sm group-hover:border-black/20 dark:group-hover:border-white/20 transition-none">
      <div class="flex items-start justify-between gap-4">
        <div class="flex items-center gap-4">
          <div class="relative">
            <div
              class="relative w-14 h-14 rounded-2xl bg-black text-white dark:bg-white dark:text-black flex items-center justify-center shadow-lg group-hover:scale-110 transition-transform duration-500"
            >
              <Sparkles class="w-7 h-7" />
            </div>
          </div>

          <div>
            <h1 class="text-xl font-bold text-slate-900 dark:text-white mb-1 group-hover:text-black dark:group-hover:text-white transition-colors">
              {{ greeting }}
            </h1>
            <p class="text-sm font-medium text-slate-600 dark:text-gray-400 group-hover:text-slate-700 dark:group-hover:text-gray-300 transition-colors">
              Unified health pulse for your stack.
            </p>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-2 lg:grid-cols-4 gap-3">
        <div class="rounded-2xl px-4 py-3 bg-white/70 dark:bg-white/5 border border-slate-200/70 dark:border-slate-700/60">
          <div class="text-3xl font-extrabold text-slate-900 dark:text-white leading-none tabular-nums">{{ props.runningApps }}</div>
          <div class="text-[11px] font-semibold text-slate-500 dark:text-gray-400 uppercase tracking-wider mt-1">Running Apps</div>
        </div>
        <div class="rounded-2xl px-4 py-3 bg-white/70 dark:bg-white/5 border border-slate-200/70 dark:border-slate-700/60">
          <div class="text-3xl font-extrabold text-slate-900 dark:text-white leading-none tabular-nums">{{ props.totalVolumes }}</div>
          <div class="text-[11px] font-semibold text-slate-500 dark:text-gray-400 uppercase tracking-wider mt-1">Volumes</div>
        </div>
        <div class="rounded-2xl px-4 py-3 bg-white/70 dark:bg-white/5 border border-slate-200/70 dark:border-slate-700/60">
          <div class="text-3xl font-extrabold text-slate-900 dark:text-white leading-none tabular-nums">{{ props.temporaryCount }}</div>
          <div class="text-[11px] font-semibold text-slate-500 dark:text-gray-400 uppercase tracking-wider mt-1">Temporary Installs</div>
        </div>
        <div class="rounded-2xl px-4 py-3 bg-white/70 dark:bg-white/5 border border-slate-200/70 dark:border-slate-700/60">
          <div class="text-3xl font-extrabold text-slate-900 dark:text-white leading-none tabular-nums">{{ props.imagesCount }}</div>
          <div class="text-[11px] font-semibold text-slate-500 dark:text-gray-400 uppercase tracking-wider mt-1">Images Cached</div>
        </div>
      </div>

      <div class="flex items-center justify-between border-t border-slate-200/70 dark:border-slate-700/60 pt-4 group-hover:border-black/15 dark:group-hover:border-white/15 transition-colors">
        <div class="flex items-center gap-2 text-xs font-bold text-black/70 dark:text-white/70 uppercase tracking-wider">
          <Activity class="w-3.5 h-3.5" />
          <span>Stack Signal</span>
        </div>
      </div>
    </div>
  </div>
</template>
