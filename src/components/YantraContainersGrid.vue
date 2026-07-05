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
      <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-(--text-secondary)">{{ t("home.yantrApps") }}</span>
      <span class="text-[10px] font-bold text-(--text-secondary) opacity-50">{{ appGroups.length }}</span>
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
      class="group relative h-full flex flex-col bg-white dark:bg-[#0A0A0A] rounded-xl overflow-hidden transition-all duration-300 hover:-translate-y-0.5 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 border border-gray-100 dark:border-zinc-800 text-left w-full cursor-pointer animate-fadeIn focus-visible:outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-500"
    >
      <!-- top accent line -->
      <div class="absolute top-0 left-0 w-full h-0.5 bg-blue-500 opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>

      <div class="relative z-10 p-5 flex flex-col h-full gap-4">
        <!-- header -->
        <div class="flex items-center justify-between gap-3">
          <div class="flex items-center gap-3 min-w-0">
            <div class="w-9 h-9 rounded-lg bg-blue-50 dark:bg-blue-500/10 border border-blue-100 dark:border-blue-500/20 flex items-center justify-center shrink-0 transition-transform group-hover:scale-110">
              <AppLogo
                :logo="group.logo"
                :name="group.name"
                :seed="group.appId || group.projectId"
                img-class="w-5 h-5 object-contain"
                icon-class="w-4.5 h-4.5 text-blue-600 dark:text-blue-400 group-hover:text-blue-500 transition-colors"
              />
            </div>
            <div class="min-w-0">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight leading-none truncate">{{ t("home.yantrApps").replace(/s$/, '') }}</h3>
              <div class="flex items-center gap-1.5 mt-1.5"
                   :class="groupState(group) === 'running' ? 'text-emerald-600 dark:text-emerald-400' : groupState(group) === 'partial' ? 'text-amber-600 dark:text-amber-400' : 'text-gray-400 dark:text-zinc-500'">
                <div class="w-1.5 h-1.5 rounded-full bg-current" :class="{ 'animate-pulse': groupState(group) === 'running' }"></div>
                <span class="text-[10px] font-bold uppercase tracking-wider">{{ groupState(group) === 'partial' ? t("stackView.partial") : t("stackView." + groupState(group)) }}</span>
                <span v-if="hasTemporary(group)" class="text-[10px] font-bold uppercase tracking-wider text-amber-500 ml-1">· {{ t("appOverview.expiresIn") }}</span>
              </div>
            </div>
          </div>
          <div class="flex items-center justify-center w-8 h-8 rounded-full bg-gray-50 dark:bg-zinc-900/50 opacity-0 group-hover:opacity-100 group-hover:translate-x-1 transition-all duration-300 shrink-0">
             <ArrowRight class="w-4 h-4 text-gray-400 dark:text-zinc-500 group-hover:text-blue-500" />
          </div>
        </div>

        <!-- Name at bottom + services -->
        <div class="mt-auto pt-6 pb-1 flex flex-col relative z-10">
          <div class="text-2xl sm:text-3xl font-black tracking-tighter text-gray-900 dark:text-white group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors line-clamp-2 leading-[1.1] mb-3">
             {{ group.name }}
          </div>
          <div class="flex flex-col gap-1.5">
             <div v-for="c in group.containers" :key="c.id" class="flex items-center gap-2">
                <span class="text-[11px] font-medium text-gray-500 dark:text-zinc-400 truncate">{{ c.app?.service || c.name }}</span>
                <span class="ml-auto text-[9px] font-mono font-bold uppercase tracking-widest px-1.5 py-0.5 rounded bg-gray-50 dark:bg-zinc-900/50" :class="c.state === 'running' ? 'text-emerald-600 dark:text-emerald-400' : 'text-gray-400 dark:text-zinc-500'">
                   {{ c.state }}
                </span>
             </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
