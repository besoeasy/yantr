<script setup>
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import { Bot, ChevronRight } from "lucide-vue-next";
import { useApiUrl } from "../../composables/useApiUrl";

const router = useRouter();
const { apiUrl } = useApiUrl();
const opencodePort = ref(null);
const opencodeInstalled = ref(false);

onMounted(async () => {
  try {
    const res = await fetch(`${apiUrl.value}/api/containers`);
    const data = await res.json();
    if (data.success) {
      const matchingContainers = data.containers.filter(
        (c) => c.app?.id === "opencode-yantr"
      );
      opencodeInstalled.value = matchingContainers.length > 0;

      const runningContainer = matchingContainers.find((c) => c.state === "running");
      if (runningContainer) {
        const port = runningContainer.ports?.find((p) => p.PrivatePort === 4096 && p.PublicPort);
        if (port) opencodePort.value = port.PublicPort;
      }
    }
  } catch {
    // not installed — card still renders
  }
});

function openOpenCode() {
  if (opencodePort.value) {
    window.open(`http://${window.location.hostname}:${opencodePort.value}`, "_blank");
  } else {
    router.push("/apps/opencode-yantr");
  }
}
</script>

<template>
  <div @click="openOpenCode" class="relative group h-full flex flex-col bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl overflow-hidden transition-all duration-400 hover:shadow-2xl hover:shadow-black/5 dark:hover:shadow-black/40 hover:border-gray-300 dark:hover:border-zinc-600 cursor-pointer">
    <div class="absolute top-0 left-0 w-full h-0.5 bg-linear-to-r from-transparent via-purple-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>
    <div class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjAiIGhlaWdodD0iMjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGNpcmNsZSBjeD0iMSIgY3k9IjEiIHI9IjEiIGZpbGw9InJnYmEoMTUwLCAxNTAsIDE1MCwgMC4xKSIvPjwvc3ZnPg==')] opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none mask-[linear-gradient(to_bottom,white,transparent)]"></div>

    <div class="relative z-10 flex flex-col h-full p-6">
      <div class="flex items-center gap-3 mb-6">
        <div class="w-10 h-10 rounded-lg bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center shrink-0 group-hover:scale-105 transition-all duration-500">
          <Bot class="w-5 h-5 text-gray-400 dark:text-zinc-500 group-hover:text-purple-500 transition-colors" />
        </div>
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white tracking-tight group-hover:text-purple-600 dark:group-hover:text-purple-400 transition-colors">AI App Manager</h3>
          <div class="text-[11px] font-medium text-gray-500 dark:text-zinc-400 uppercase tracking-wider mt-1">Powered by OpenCode</div>
        </div>
      </div>

      <p class="text-xs text-gray-500 dark:text-zinc-400 leading-relaxed mb-4">
        Deploy and manage apps using AI. Just describe what you want in chat — OpenCode handles the rest.
      </p>

      <button
        type="button"
        @click.stop="openOpenCode"
        class="mt-auto flex items-center justify-between p-3 rounded-lg bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800/50 hover:border-gray-300 dark:hover:border-zinc-700 transition-all duration-300 focus:outline-none group/btn"
      >
        <div class="flex items-center gap-3 min-w-0">
          <div v-if="opencodePort" class="w-2 h-2 rounded-full bg-green-500 animate-pulse shrink-0"></div>
          <div class="flex flex-col text-left min-w-0">
            <span class="text-xs font-semibold text-gray-900 dark:text-white">{{ opencodePort ? "Open OpenCode" : opencodeInstalled ? "Open OpenCode" : "Install OpenCode Yantr" }}</span>
            <span class="text-[10px] text-gray-500 dark:text-zinc-400">{{ opencodePort ? `Running on port ${opencodePort}` : opencodeInstalled ? "Installed, but host port is not available" : "Not installed yet" }}</span>
          </div>
        </div>
        <ChevronRight class="w-4 h-4 text-gray-400 dark:text-zinc-600 group-hover/btn:text-gray-900 dark:group-hover/btn:text-white group-hover/btn:translate-x-1 transition-all" />
      </button>
    </div>
  </div>
</template>
