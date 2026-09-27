<script setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { Layers, ArrowRight } from "@lucide/vue";
import AppLogo from "./AppLogo.vue";

const { t } = useI18n();
const props = defineProps({
  containers: { type: Array, default: () => [] },
  showHeader: { type: Boolean, default: false },
});

const router = useRouter();

// Group containers into stacks by projectId
const appGroups = computed(() => {
  const map = new Map();
  for (const c of props.containers) {
    const projectId = c.app?.projectId || c.id;
    if (!map.has(projectId)) {
      map.set(projectId, {
        projectId,
        appId: c.app?.id,
        name: c.app?.name || projectId,
        logo: c.app?.logo || null,
        containers: [],
      });
    }
    map.get(projectId).containers.push(c);
  }
  return [...map.values()];
});

function groupState(group) {
  const states = group.containers.map((c) => c.state);
  if (states.every((s) => s === "running")) return "running";
  if (states.some((s) => s === "running")) return "partial";
  return "stopped";
}

function hasTemporary(group) {
  return group.containers.some((c) => c?.labels?.["yantr.expireAt"]);
}

function navigate(group) {
  router.push(`/stacks/${group.projectId}`);
}
</script>

<template>
  <div style="display: contents">
    <!-- Section Header -->
    <div v-if="props.showHeader" class="col-span-full flex items-center gap-2 pt-2 pb-1">
      <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-zinc-500 dark:text-zinc-400">{{ t("home.yantrApps") }}</span>
      <span class="text-[10px] font-bold text-zinc-400 dark:text-zinc-500">{{ appGroups.length }}</span>
    </div>

    <div
      v-for="(group, index) in appGroups"
      :key="group.projectId"
      :style="{ animationDelay: `${index * 50}ms` }"
      @click="navigate(group)"
      @keydown.enter.prevent="navigate(group)"
      @keydown.space.prevent="navigate(group)"
      role="button"
      tabindex="0"
      class="group relative flex h-full w-full cursor-pointer flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-6 text-left transition-all duration-300 hover:-translate-y-1 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)] animate-fadeIn focus-visible:outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-zinc-500"
    >
      <div class="flex w-full items-start justify-between gap-3">
        <div class="flex items-center gap-4 min-w-0">
          <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 transition-transform duration-300 group-hover:scale-105 dark:border-zinc-800 dark:bg-zinc-900">
            <AppLogo
              :logo="group.logo"
              :name="group.name"
              :seed="group.appId || group.projectId"
              img-class="w-6 h-6 object-contain"
              icon-class="w-5 h-5 text-zinc-900 dark:text-zinc-100 transition-colors"
            />
          </div>
          <div class="min-w-0">
            <h3 class="text-lg font-bold leading-none tracking-tight text-zinc-900 transition-colors group-hover:text-black dark:text-white dark:group-hover:text-white truncate">
              {{ t("home.yantrApps").replace(/s$/, '') }}
            </h3>
            <div class="mt-2 flex items-center gap-2"
                 :class="groupState(group) === 'running' ? 'text-zinc-900 dark:text-white' : groupState(group) === 'partial' ? 'text-zinc-700 dark:text-zinc-300' : 'text-zinc-500 dark:text-zinc-400'">
              <span v-if="groupState(group) === 'running'" class="relative flex h-2 w-2">
                <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
                <span class="relative inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
              </span>
              <span v-else-if="groupState(group) === 'partial'" class="h-2 w-2 rounded-full bg-amber-500"></span>
              <span v-else class="h-2 w-2 rounded-full bg-zinc-300 dark:bg-zinc-700"></span>
              <span class="text-[10px] font-semibold uppercase tracking-widest">{{ groupState(group) === 'partial' ? t("stackView.partial") : t("stackView." + groupState(group)) }}</span>
              <span v-if="hasTemporary(group)" class="ml-1 text-[10px] font-semibold uppercase tracking-widest text-amber-600 dark:text-amber-500">· {{ t("appOverview.expiresIn") }}</span>
            </div>
          </div>
        </div>
        
        <div class="flex h-8 w-8 shrink-0 -translate-y-1 items-center justify-center rounded-full opacity-0 transition-all duration-300 group-hover:translate-y-0 group-hover:opacity-100">
           <ArrowRight class="h-5 w-5 text-zinc-400 transition-colors hover:text-zinc-900 dark:text-zinc-600 dark:hover:text-white" />
        </div>
      </div>

      <!-- Name at bottom + services -->
      <div class="relative z-10 mt-auto flex flex-col pt-6">
        <div class="mb-4 text-3xl font-black leading-[1.1] tracking-tighter text-zinc-900 transition-colors duration-500 group-hover:-translate-y-1 sm:text-4xl dark:text-white line-clamp-2">
           {{ group.name }} 
        </div>
        <div class="flex flex-col gap-2">
           <div v-for="c in group.containers" :key="c.id" class="flex items-center gap-3">
              <span class="truncate text-xs font-medium text-zinc-600 dark:text-zinc-400">{{ c.app?.service || c.name }}</span>
              <span class="ml-auto rounded border border-zinc-200 bg-zinc-50 px-2 py-0.5 text-[9px] font-semibold uppercase tracking-widest dark:border-zinc-800 dark:bg-zinc-900" :class="c.state === 'running' ? 'text-emerald-600 dark:text-emerald-500' : 'text-zinc-500 dark:text-zinc-400'">
                 {{ c.state }}
              </span>
           </div>
        </div>
      </div>
    </div>
  </div>
</template>
