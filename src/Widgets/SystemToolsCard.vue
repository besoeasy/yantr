<script setup>
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { ChevronRight, HardDrive, Layers, Wrench } from "lucide-vue-next";

const router = useRouter();
const { t } = useI18n();

const tools = [
  {
    title: t("home.toolsNavCard.images"),
    path: "/images",
    description: t("home.toolsNavCard.imagesDesc"),
    icon: Layers,
    iconColor: "text-blue-500",
    hoverClass: "group-hover:bg-blue-500/5",
  },
  {
    title: t("home.toolsNavCard.volumes"),
    path: "/volumes",
    description: t("home.toolsNavCard.volumesDesc"),
    icon: HardDrive,
    iconColor: "text-emerald-500",
    hoverClass: "group-hover:bg-emerald-500/5",
  },
];
</script>

<template>
  <div
    class="flex h-full flex-col rounded-2xl p-6 smooth-shadow transition-all duration-300 hover:-translate-y-0.5 hover:smooth-shadow-lg bg-white dark:bg-[#0A0A0A] text-(--text-primary)"
  >
    <div class="flex items-start gap-3.5">
      <div
        class="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl"
        style="background: var(--surface-muted)"
      >
        <Wrench class="h-5 w-5 text-blue-500" />
      </div>

      <div class="min-w-0">
        <h3 class="text-sm font-semibold tracking-tight">
          {{ t("home.toolsNavCard.systemTools") }}
        </h3>
        <p class="mt-1 text-xs leading-relaxed" style="color: var(--text-secondary)">
          {{ t("home.toolsNavCard.images") }} & {{ t("home.toolsNavCard.volumes") }}
        </p>
      </div>
    </div>

    <div class="mt-6 flex flex-col gap-2.5">
      <button
        v-for="tool in tools"
        :key="tool.title"
        type="button"
        @click="router.push(tool.path)"
        class="group flex items-center justify-between rounded-xl px-4 py-3 text-left transition-all duration-300"
        :class="tool.hoverClass"
        style="background: var(--surface-muted)"
      >
        <div class="flex min-w-0 items-center gap-3">
          <div
            class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl"
            style="background: var(--surface)"
          >
            <component :is="tool.icon" :class="['h-4 w-4', tool.iconColor]" />
          </div>

          <div class="min-w-0">
            <div class="text-xs font-semibold">
              {{ tool.title }}
            </div>
            <div class="mt-0.5 text-[11px] truncate" style="color: var(--text-secondary)">
              {{ tool.description }}
            </div>
          </div>
        </div>

        <ChevronRight class="h-4 w-4 shrink-0" style="color: var(--text-secondary)" />
      </button>
    </div>
  </div>
</template>