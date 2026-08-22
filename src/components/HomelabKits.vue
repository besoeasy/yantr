<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  House,
  Clapperboard,
  Lamp,
  Shield,
  FolderSync,
  Activity,
} from "@lucide/vue";
import { HOMELAB_KITS } from "../data/homelabKits";
import AppLogo from "./AppLogo.vue";

const props = defineProps({
  selectedId: {
    type: String,
    default: null,
  },
  apps: {
    type: Array,
    default: () => [],
  },
});

const emit = defineEmits(["select"]);
const { t } = useI18n();

const ICONS = {
  essentials: House,
  media: Clapperboard,
  smarthome: Lamp,
  privacy: Shield,
  files: FolderSync,
  monitor: Activity,
};

const appsById = computed(() => {
  const map = new Map();
  for (const app of props.apps) {
    if (app?.id) map.set(app.id, app);
  }
  return map;
});

const kits = computed(() =>
  HOMELAB_KITS.map((kit) => ({
    ...kit,
    icon: ICONS[kit.id] || House,
    name: t(`kits.${kit.id}.name`),
    description: t(`kits.${kit.id}.description`),
    preview: kit.apps
      .map((id) => appsById.value.get(id))
      .filter(Boolean)
      .slice(0, 4),
  }))
);

function selectKit(id) {
  emit("select", id);
}
</script>

<template>
  <div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3">
    <button
      v-for="kit in kits"
      :key="kit.id"
      type="button"
      @click="selectKit(kit.id)"
      :class="[
        'group flex flex-col text-left rounded-2xl border p-5 transition-all duration-300',
        selectedId === kit.id
          ? 'border-black bg-gray-50 shadow-sm dark:border-white dark:bg-zinc-900'
          : 'border-zinc-200 bg-white hover:border-zinc-300 hover:-translate-y-0.5 hover:shadow-sm dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700',
      ]"
    >
      <div class="flex items-start justify-between gap-3">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 text-zinc-800 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100">
          <component :is="kit.icon" :size="18" />
        </div>
        <div v-if="kit.preview.length" class="flex -space-x-2">
          <div
            v-for="app in kit.preview"
            :key="app.id"
            class="h-8 w-8 rounded-lg border border-white bg-white p-0.5 dark:border-black dark:bg-zinc-950"
          >
            <AppLogo
              :logo="app.logo"
              :name="app.name"
              :seed="app.id"
              img-class="h-full w-full object-contain"
              icon-class="h-full w-full text-zinc-300 dark:text-zinc-700"
            />
          </div>
        </div>
      </div>

      <h3 class="mt-4 text-sm font-semibold tracking-tight text-zinc-900 dark:text-white">
        {{ kit.name }}
      </h3>
      <p class="mt-1.5 text-sm leading-relaxed text-zinc-500 dark:text-zinc-400">
        {{ kit.description }}
      </p>
      <p class="mt-4 text-[10px] font-bold uppercase tracking-widest text-zinc-400 dark:text-zinc-500">
        {{ t("kits.appsInKit", { count: kit.apps.length }) }}
      </p>
    </button>
  </div>
</template>
