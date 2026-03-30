<script setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { Box, Layers, ArrowRight } from "lucide-vue-next";

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
      <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-[var(--text-secondary)]">{{ t("home.yantrApps") }}</span>
      <span class="text-[10px] font-bold text-[var(--text-secondary)] opacity-50">{{ appGroups.length }}</span>
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
      class="group relative h-full flex flex-col bg-[var(--surface)] border border-gray-200 dark:border-zinc-800 rounded-xl overflow-hidden transition-all duration-300 smooth-shadow hover:smooth-shadow-lg hover:-translate-y-0.5 cursor-pointer animate-fadeIn focus-visible:outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-500"
    >

      <div class="relative z-10 flex flex-col h-full p-5">
        <!-- Header -->
        <div class="flex items-start justify-between mb-4">
          <div class="min-w-0 flex-1 pr-4">
            <h3 class="font-semibold text-base text-gray-900 dark:text-white line-clamp-1 mb-1 tracking-tight group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
              {{ group.name }}
            </h3>
            <p class="text-[10px] font-mono text-gray-400 dark:text-zinc-500 truncate mb-1">{{ group.projectId }}</p>

            <div class="flex items-center gap-2 flex-wrap">
              <span class="text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-md"
                    :class="groupState(group) === 'running' ? 'bg-green-50/50 dark:bg-green-500/10 text-green-600 dark:text-green-500' : 
                            groupState(group) === 'partial' ? 'bg-amber-50/50 dark:bg-amber-500/10 text-amber-600 dark:text-amber-500' : 
                            'bg-gray-100 dark:bg-zinc-800 text-gray-500 dark:text-zinc-400'">
                {{ groupState(group) === 'partial' ? t("stackView.partial") : t("stackView." + groupState(group)) }}
              </span>

              <span v-if="group.containers.length > 1" class="text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-md bg-gray-50 dark:bg-zinc-900 text-gray-500 dark:text-zinc-400 flex items-center gap-1">
                <Layers :size="10" />
                {{ group.containers.length }}
              </span>

              <span v-if="hasTemporary(group)" class="text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-md bg-amber-50/50 dark:bg-amber-500/10 text-amber-600 dark:text-amber-500">
                {{ t("appOverview.expiresIn") }}
              </span>
            </div>
          </div>

          <!-- Logo Container -->
          <div class="w-12 h-12 rounded-lg bg-[var(--surface-muted)] flex items-center justify-center shrink-0 group-hover:scale-105 transition-all duration-300 relative">
            <img v-if="group.logo" :src="group.logo" :alt="group.name" class="w-7 h-7 object-contain group-hover:brightness-110 transition-all" loading="lazy" />
            <Box v-else class="w-6 h-6 text-[var(--text-secondary)] group-hover:text-blue-500 transition-colors" />
            
            <div class="absolute -top-1 -right-1 w-2 h-2 rounded-full"
                 :class="groupState(group) === 'running' ? 'bg-green-500' : groupState(group) === 'partial' ? 'bg-amber-500' : 'bg-gray-400 dark:bg-zinc-600'">
            </div>
          </div>
        </div>

        <!-- Services -->
        <div class="flex flex-wrap gap-1.5 mb-6">
          <span v-for="c in group.containers" :key="c.id" 
                class="text-[10px] font-mono px-2 py-1 rounded-md flex items-center gap-1.5 bg-[var(--surface-muted)]"
                :class="c.state === 'running' ? 'text-gray-600 dark:text-zinc-300' : 'text-gray-400 dark:text-zinc-500'">
            <span class="w-1.5 h-1.5 rounded-full shrink-0" :class="c.state === 'running' ? 'bg-green-500' : 'bg-gray-300 dark:bg-zinc-600'"></span>
            <span class="truncate max-w-[120px]">{{ c.app?.service || c.name }}</span>
          </span>
        </div>

        <!-- Bottom Action -->
        <div class="mt-auto pt-4 flex items-center justify-between overflow-hidden">
          <div class="flex items-center gap-1.5 text-gray-400 dark:text-zinc-500 group-hover:text-gray-600 dark:group-hover:text-zinc-300 transition-colors duration-300">
            <span class="text-[10px] font-semibold uppercase tracking-[0.15em]">{{ t("home.yantraContainersGrid.stackView") }}</span>
          </div>
          
          <div class="flex items-center gap-1 text-blue-600 dark:text-blue-400 font-semibold text-xs transform translate-y-8 opacity-0 group-hover:translate-y-0 group-hover:opacity-100 transition-all duration-300 cubic-bezier(0.4, 0, 0.2, 1)">
            <span>{{ t("home.yantraContainersGrid.open") }}</span>
            <ArrowRight :size="14" class="group-hover:translate-x-1 transition-transform duration-300" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
