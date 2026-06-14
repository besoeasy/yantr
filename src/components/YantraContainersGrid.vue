<script setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { Layers, ArrowRight } from "lucide-vue-next";
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
      class="group relative h-full flex flex-col bg-white dark:bg-[#0A0A0A] rounded-xl overflow-hidden transition-all duration-300 smooth-shadow hover:smooth-shadow-lg hover:-translate-y-0.5 cursor-pointer animate-fadeIn focus-visible:outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-500"
    >
      <div class="relative z-10 flex flex-col h-full p-5">

        <!-- Header: logo + name + status -->
        <div class="flex items-start gap-3.5 mb-5">
          <!-- Logo -->
          <div class="w-11 h-11 rounded-xl bg-(--surface-muted) flex items-center justify-center shrink-0 transition-transform duration-300 group-hover:scale-105">
            <AppLogo
              :logo="group.logo"
              :name="group.name"
              :seed="group.appId || group.projectId"
              img-class="w-6 h-6 object-contain"
              icon-class="w-5 h-5 text-(--text-secondary) group-hover:text-blue-500 transition-colors"
            />
          </div>

          <!-- Name + status -->
          <div class="min-w-0 flex-1 pt-0.5">
            <h3 class="font-semibold text-sm text-gray-900 dark:text-white line-clamp-1 tracking-tight group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors duration-300">
              {{ group.name }}
            </h3>
            <div class="mt-1 flex items-center gap-1.5">
              <span class="w-1.5 h-1.5 rounded-full shrink-0"
                    :class="groupState(group) === 'running' ? 'bg-green-500' : groupState(group) === 'partial' ? 'bg-amber-500' : 'bg-gray-400 dark:bg-zinc-600'">
              </span>
              <span class="text-[10px] font-semibold uppercase tracking-wider"
                    :class="groupState(group) === 'running' ? 'text-green-600 dark:text-green-500' : groupState(group) === 'partial' ? 'text-amber-600 dark:text-amber-500' : 'text-gray-400 dark:text-zinc-500'">
                {{ groupState(group) === 'partial' ? t("stackView.partial") : t("stackView." + groupState(group)) }}
              </span>
              <span v-if="hasTemporary(group)" class="text-[10px] font-semibold uppercase tracking-wider text-amber-500">
                · {{ t("appOverview.expiresIn") }}
              </span>
            </div>
          </div>
        </div>

        <!-- Services list -->
        <div class="flex flex-col gap-1.5 mb-5">
          <div
            v-for="c in group.containers"
            :key="c.id"
            class="flex items-center gap-2 px-3 py-2 rounded-lg bg-(--surface-muted)"
          >
            <span class="w-1.5 h-1.5 rounded-full shrink-0"
                  :class="c.state === 'running' ? 'bg-green-500' : 'bg-gray-300 dark:bg-zinc-600'">
            </span>
            <span class="text-xs font-medium truncate text-gray-700 dark:text-zinc-300">
              {{ c.app?.service || c.name }}
            </span>
            <span v-if="group.containers.length > 1" class="ml-auto text-[10px] font-mono text-gray-400 dark:text-zinc-500 shrink-0">
              {{ c.state }}
            </span>
          </div>
        </div>

        <!-- Footer -->
        <div class="mt-auto flex items-center justify-between pt-3 border-t border-gray-100 dark:border-zinc-800/60">
          <div class="flex items-center gap-1.5 text-gray-400 dark:text-zinc-500">
            <Layers :size="12" />
            <span class="text-[10px] font-semibold uppercase tracking-[0.15em]">
              {{ group.containers.length === 1 ? '1 service' : `${group.containers.length} services` }}
            </span>
          </div>
          <div class="flex items-center gap-1 text-blue-600 dark:text-blue-400 font-semibold text-xs opacity-0 -translate-x-2 group-hover:opacity-100 group-hover:translate-x-0 transition-all duration-300">
            <span>{{ t("home.yantraContainersGrid.open") }}</span>
            <ArrowRight :size="13" class="group-hover:translate-x-0.5 transition-transform duration-300" />
          </div>
        </div>

      </div>
    </div>
  </div>
</template>
