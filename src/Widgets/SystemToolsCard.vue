<script setup>
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { ChevronRight, ClipboardList, HardDrive, Layers, Wrench } from "lucide-vue-next";

const router = useRouter();
const { t } = useI18n();

const tools = [
  {
    title: t("home.toolsNavCard.images"),
    path: "/images",
    description: t("home.toolsNavCard.imagesDesc"),
    icon: Layers,
    iconColor: "text-blue-500",
    hoverBg: "hover:bg-blue-500/5",
  },
  {
    title: t("home.toolsNavCard.volumes"),
    path: "/volumes",
    description: t("home.toolsNavCard.volumesDesc"),
    icon: HardDrive,
    iconColor: "text-emerald-500",
    hoverBg: "hover:bg-emerald-500/5",
  },
  {
    title: t("home.toolsNavCard.logs"),
    path: "/logs",
    description: t("home.toolsNavCard.logsDesc"),
    icon: ClipboardList,
    iconColor: "text-violet-500",
    hoverBg: "hover:bg-violet-500/5",
  },
];
</script>

<template>
  <div class="flex h-full flex-col rounded-2xl p-5 smooth-shadow transition-all duration-300 bg-white dark:bg-[#0A0A0A] text-(--text-primary)">

    <!-- Header -->
    <div class="flex items-center gap-3 mb-5">
      <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-(--surface-muted)">
        <Wrench class="h-4 w-4 text-blue-500" />
      </div>
      <div>
        <h3 class="text-sm font-semibold tracking-tight">{{ t("home.toolsNavCard.systemTools") }}</h3>
        <p class="mt-0.5 text-[11px] text-(--text-secondary)">{{ t("home.toolsNavCard.images") }}, {{ t("home.toolsNavCard.volumes") }} & {{ t("home.toolsNavCard.logs") }}</p>
      </div>
    </div>

    <!-- Nav rows -->
    <div class="flex flex-col gap-2">
      <button
        v-for="tool in tools"
        :key="tool.title"
        type="button"
        @click="router.push(tool.path)"
        :class="['group flex items-center justify-between rounded-xl px-3.5 py-3 text-left transition-all duration-200 bg-(--surface-muted)', tool.hoverBg]"
      >
        <div class="flex items-center gap-3 min-w-0">
          <div class="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-white dark:bg-[#0A0A0A]">
            <component :is="tool.icon" :class="['h-4 w-4', tool.iconColor]" />
          </div>
          <div class="min-w-0">
            <div class="text-xs font-semibold">{{ tool.title }}</div>
            <div class="mt-0.5 text-[11px] truncate text-(--text-secondary)">{{ tool.description }}</div>
          </div>
        </div>
        <ChevronRight class="h-4 w-4 shrink-0 text-(--text-secondary) transition-transform duration-200 group-hover:translate-x-0.5" />
      </button>
    </div>

  </div>
</template>
