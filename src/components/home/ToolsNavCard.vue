<script setup>
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { Layers, HardDrive, ClipboardList, ChevronRight, Wrench } from "lucide-vue-next";

const router = useRouter();
const { t } = useI18n();

const tools = [
  {
    title: t("home.toolsNavCard.images"),
    path: "/images",
    description: t("home.toolsNavCard.imagesDesc"),
    icon: Layers,
    color: "text-blue-500",
    bg: "bg-blue-500/10",
  },
  {
    title: t("home.toolsNavCard.volumes"),
    path: "/volumes",
    description: t("home.toolsNavCard.volumesDesc"),
    icon: HardDrive,
    color: "text-violet-500",
    bg: "bg-violet-500/10",
  },
  {
    title: t("home.toolsNavCard.logs"),
    path: "/logs",
    description: t("home.toolsNavCard.logsDesc"),
    icon: ClipboardList,
    color: "text-green-500",
    bg: "bg-green-500/10",
  },
];
</script>

<template>
  <div class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl overflow-hidden transition-all duration-400 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:border-gray-300 dark:hover:border-zinc-600">
    <!-- Hover Accents -->
    <div class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-blue-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
    <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none [mask-image:linear-gradient(to_bottom,white,transparent)]"></div>

    <div class="relative z-10 flex flex-col h-full p-6">
      <div class="flex items-center gap-3 mb-6">
        <div class="w-10 h-10 rounded-lg bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center shrink-0 group-hover:scale-105 transition-all duration-500">
           <Wrench class="w-5 h-5 text-gray-400 dark:text-zinc-500 group-hover:text-blue-500 transition-colors" />
        </div>
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">{{ t("home.toolsNavCard.systemTools") }}</h3>
          <div class="text-[11px] font-medium text-gray-500 dark:text-zinc-400 uppercase tracking-wider mt-1">{{ t("home.toolsNavCard.managementLogs") }}</div>
        </div>
      </div>

      <div class="flex flex-col gap-2 mt-auto">
        <button
          v-for="tool in tools"
          :key="tool.title"
          @click="router.push(tool.path)"
          class="flex items-center justify-between p-3 rounded-lg bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50 hover:border-gray-300 dark:hover:border-zinc-700 transition-all duration-300 focus:outline-none group/btn"
        >
          <div class="flex items-center gap-3">
            <component :is="tool.icon" :class="['w-4 h-4', tool.color]" />
            <div class="flex flex-col text-left">
              <span class="text-xs font-semibold text-gray-900 dark:text-white">{{ tool.title }}</span>
              <span class="text-[10px] text-gray-500 dark:text-zinc-400">{{ tool.description }}</span>
            </div>
          </div>
          <ChevronRight class="w-4 h-4 text-gray-400 dark:text-zinc-600 group-hover/btn:text-gray-900 dark:group-hover/btn:text-white group-hover/btn:translate-x-1 transition-all" />
        </button>
      </div>
    </div>
  </div>
</template>
