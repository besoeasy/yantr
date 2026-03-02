<script setup>
import { ref, computed, onMounted, onUnmounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useApiUrl } from "../composables/useApiUrl";
import { useCurrentTime } from "../composables/useCurrentTime";
import { formatDuration } from "../utils/metrics";
import {
  ArrowLeft, Layers, Server, Activity, Package, ExternalLink, Play, Clock,
} from "lucide-vue-next";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const { apiUrl } = useApiUrl();
const { currentTime } = useCurrentTime();

const appId = computed(() => route.params.appId);

const app = ref(null);
const containers = ref([]);
const loading = ref(true);

// Group containers for this app by compose project
const runningStacks = computed(() => {
  const appContainers = containers.value.filter((c) => c.app?.id === appId.value);
  const map = new Map();
  for (const c of appContainers) {
    const key = c.app?.projectId || c.id;
    if (!map.has(key)) map.set(key, { projectId: key, services: [] });
    map.get(key).services.push(c);
  }
  return [...map.values()];
});

function stackState(stack) {
  const states = stack.services.map((s) => s.state);
  if (states.every((s) => s === "running")) return "running";
  if (states.some((s) => s === "running")) return "partial";
  return "stopped";
}

function stackUptime(stack) {
  const running = stack.services.filter((s) => s.state === "running" && s.created);
  if (!running.length) return null;
  const oldest = Math.min(...running.map((s) => s.created * 1000));
  const delta = currentTime.value - oldest;
  if (delta <= 0) return t('appOverview.justStarted');
  return formatDuration(delta);
}

function stackExpiry(stack) {
  for (const svc of stack.services) {
    const expireAt = svc.labels?.["yantr.expireAt"];
    if (expireAt) {
      const remaining = parseInt(expireAt, 10) * 1000 - currentTime.value;
      if (remaining <= 0) return { label: t('appOverview.expired'), expired: true, soon: false };
      const totalSeconds = Math.floor(remaining / 1000);
      const totalMinutes = Math.floor(totalSeconds / 60);
      const hours = Math.floor(totalMinutes / 60);
      const minutes = totalMinutes % 60;
      const seconds = totalSeconds % 60;
      const label = hours > 0 ? `${hours}h ${minutes}m` : minutes > 0 ? `${minutes}m ${seconds}s` : `${seconds}s`;
      return { label, expired: false, soon: remaining < 60 * 60 * 1000 };
    }
  }
  return null;
}

async function fetchData() {
  try {
    const [appsRes, containersRes] = await Promise.all([
      fetch(`${apiUrl.value}/api/apps`),
      fetch(`${apiUrl.value}/api/containers`),
    ]);
    const appsData = await appsRes.json();
    const containersData = await containersRes.json();
    app.value = (appsData.apps || appsData).find((a) => a.id === appId.value) || null;
    containers.value = containersData.containers || containersData || [];

    // Auto-redirect when there is exactly one running instance
    if (loading.value && runningStacks.value.length === 1) {
      router.replace(`/stacks/${runningStacks.value[0].projectId}`);
      return;
    }
  } catch (e) {
    console.error("AppOverview fetch error", e);
  } finally {
    loading.value = false;
  }
}

let interval = null;
onMounted(() => {
  fetchData();
  interval = setInterval(fetchData, 8000);
});
onUnmounted(() => clearInterval(interval));
</script>

<template>
  <div class="min-h-screen bg-white dark:bg-[#0A0A0A] px-4 py-8 sm:px-6 lg:px-8 selection:bg-blue-500/30">

    <!-- Header -->
    <div class="max-w-5xl mx-auto mb-10">
      <button
        @click="router.back()"
        class="inline-flex items-center gap-2 text-[11px] font-bold uppercase tracking-wider text-gray-500 dark:text-zinc-500 hover:text-gray-900 dark:hover:text-white transition-colors mb-6 group"
      >
        <ArrowLeft :size="14" class="group-hover:-translate-x-0.5 transition-transform" />
        {{ t('appOverview.back') }}
      </button>

      <!-- Loading skeleton -->
      <div v-if="loading" class="flex items-center gap-5 animate-pulse">
        <div class="w-16 h-16 rounded-xl bg-gray-100 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 shrink-0"></div>
        <div class="space-y-3">
          <div class="h-5 w-40 bg-gray-200 dark:bg-zinc-800 rounded"></div>
          <div class="h-3 w-64 bg-gray-100 dark:bg-zinc-900 rounded"></div>
        </div>
      </div>

      <!-- App identity -->
      <div v-else-if="app" class="flex items-center gap-5">
        <div class="w-16 h-16 rounded-xl bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 flex items-center justify-center shrink-0 shadow-sm overflow-hidden">
          <img v-if="app.logo" :src="app.logo" :alt="app.name" loading="lazy" class="w-10 h-10 object-contain filter dark:brightness-90" />
          <Package v-else :size="24" class="text-gray-400 dark:text-zinc-600" />
        </div>
        <div>
          <h1 class="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">{{ app.name }}</h1>
          <p v-if="app.short_description" class="text-sm text-gray-500 dark:text-zinc-400 mt-1">{{ app.short_description }}</p>
          <div v-if="app.tags?.length" class="flex flex-wrap gap-2 mt-3">
            <span
              v-for="tag in app.tags.slice(0, 6)"
              :key="tag"
              class="text-[10px] font-bold uppercase tracking-widest px-2 py-0.5 rounded-md bg-gray-50 dark:bg-zinc-900/50 text-gray-500 dark:text-zinc-400 border border-gray-200 dark:border-zinc-800"
            >{{ tag }}</span>
          </div>
        </div>
      </div>
    </div>

    <div class="max-w-5xl mx-auto space-y-10">

      <!-- ── Row 1: Running Instances ─────────────────────────────────────── -->
      <section>
        <div class="flex items-center gap-2 mb-5">
          <Activity :size="14" class="text-gray-400 dark:text-zinc-500" />
          <h2 class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t('appOverview.runningInstances') }}</h2>
          <span
            v-if="runningStacks.length"
            class="text-[10px] font-mono px-2 py-0.5 rounded bg-green-50 dark:bg-green-500/10 text-green-600 dark:text-green-400 border border-green-200 dark:border-green-500/20"
          >{{ runningStacks.length }}</span>
        </div>

        <!-- No stacks -->
        <div
          v-if="!loading && runningStacks.length === 0"
          class="flex items-center justify-center gap-3 p-8 rounded-xl border border-dashed border-gray-300 dark:border-zinc-800 text-gray-400 dark:text-zinc-500 text-sm"
        >
          <Server :size="16" />
          <span>{{ t('appOverview.noRunningInstances') }}</span>
        </div>

        <!-- Stack cards -->
        <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
          <div
            v-for="(stack, idx) in runningStacks"
            :key="stack.projectId"
            @click="router.push(`/stacks/${stack.projectId}`)"
            role="button"
            tabindex="0"
            @keydown.enter.prevent="router.push(`/stacks/${stack.projectId}`)"
            class="group relative bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 p-5 cursor-pointer hover:border-gray-300 dark:hover:border-zinc-600 transition-all duration-300 focus:outline-none overflow-hidden"
          >
            <!-- Glowing Accent Line -->
            <div class="absolute top-0 left-0 w-full h-[2px] opacity-0 group-hover:opacity-100 transition-opacity duration-500" :class="{
                'bg-gradient-to-r from-transparent via-green-500 to-transparent': stackState(stack) === 'running',
                'bg-gradient-to-r from-transparent via-amber-500 to-transparent': stackState(stack) === 'partial',
                'bg-gradient-to-r from-transparent via-gray-500 to-transparent': stackState(stack) === 'stopped',
            }"></div>

            <div class="flex items-start justify-between gap-2 mb-5 relative z-10">
              <div class="flex items-center gap-2.5 min-w-0">
                <Layers :size="14" class="text-gray-400 dark:text-zinc-500 shrink-0" />
                <span class="text-xs font-mono font-semibold text-gray-900 dark:text-white truncate">{{ stack.projectId }}</span>
              </div>
              <!-- Multiple instances badge -->
              <span
                v-if="runningStacks.length > 1"
                class="text-[9px] font-bold uppercase tracking-wider px-1.5 py-0.5 rounded bg-blue-50 dark:bg-blue-500/10 text-blue-600 dark:text-blue-400 border border-blue-200 dark:border-blue-500/20 shrink-0"
              >#{{ idx + 1 }}</span>
            </div>

            <!-- Services list -->
            <div class="space-y-2 mb-5 relative z-10">
              <div
                v-for="svc in stack.services"
                :key="svc.id"
                class="flex items-center gap-2.5"
              >
                <div class="relative flex h-1.5 w-1.5 shrink-0">
                  <span
                    v-if="svc.state === 'running'"
                    class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"
                  ></span>
                  <span
                    class="relative inline-flex rounded-full h-1.5 w-1.5"
                    :class="svc.state === 'running' ? 'bg-green-500' : 'bg-gray-300 dark:bg-zinc-700'"
                  ></span>
                </div>
                <span class="text-xs text-gray-700 dark:text-zinc-300 truncate">{{ svc.app?.service || svc.name }}</span>
                <span
                  class="ml-auto text-[9px] font-bold uppercase tracking-wider px-1.5 py-0.5 rounded border shrink-0"
                  :class="svc.state === 'running'
                    ? 'bg-transparent text-green-600 dark:text-green-500 border-green-200 dark:border-green-500/30'
                    : 'bg-transparent text-gray-500 border-gray-200 dark:border-zinc-800'"
                >{{ svc.state }}</span>
              </div>
            </div>

            <!-- Footer: uptime + expiry + service count -->
            <div class="pt-4 border-t border-gray-100 dark:border-zinc-800/80 space-y-2.5 relative z-10">
              <div class="flex items-center justify-between text-[10px] font-mono text-gray-400 dark:text-zinc-500">
                <span>{{ stack.services.length }} {{ stack.services.length !== 1 ? t('appOverview.services') : t('appOverview.service') }}</span>
                <span v-if="stackUptime(stack)" class="flex items-center gap-1.5">
                  <Clock :size="10" />
                  {{ stackUptime(stack) }}
                </span>
              </div>
              <div v-if="stackExpiry(stack)" class="flex items-center justify-between">
                <span class="text-[9px] uppercase font-bold tracking-wider text-amber-600 dark:text-amber-500">{{ t('appOverview.expiresIn') }}</span>
                <span
                  class="text-[10px] font-mono font-bold tabular-nums"
                  :class="stackExpiry(stack).expired
                    ? 'text-red-600 dark:text-red-500'
                    : stackExpiry(stack).soon
                      ? 'text-amber-500 animate-pulse'
                      : 'text-amber-600 dark:text-amber-500'"
                >{{ stackExpiry(stack).label }}</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- ── Row 2: App Detail link ────────────────────────────────────────── -->
      <section>
        <div class="flex items-center gap-2 mb-5">
          <Package :size="14" class="text-gray-400 dark:text-zinc-500" />
          <h2 class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t('appOverview.appDetails') }}</h2>
        </div>

        <div
          @click="router.push(`/apps/${appId}`)"
          role="button"
          tabindex="0"
          @keydown.enter.prevent="router.push(`/apps/${appId}`)"
          class="group flex items-center justify-between bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 px-6 py-5 cursor-pointer hover:border-gray-300 dark:hover:border-zinc-600 transition-all duration-300"
        >
          <div class="flex items-center gap-5">
            <div class="w-12 h-12 rounded-xl bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center shrink-0 overflow-hidden shadow-sm group-hover:scale-105 transition-transform duration-500">
              <img v-if="app?.logo" :src="app.logo" :alt="app?.name" loading="lazy" class="w-7 h-7 object-contain filter dark:brightness-90 group-hover:brightness-100 transition-all" />
              <Package v-else :size="20" class="text-gray-400 dark:text-zinc-500" />
            </div>
            <div>
            <div class="text-sm font-semibold text-gray-900 dark:text-white mb-0.5 tracking-tight group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">{{ t('appOverview.appDetails') }}</div>
            <div class="text-xs text-gray-500 dark:text-zinc-400">{{ t('appOverview.appDetailsDesc') }}</div>
          </div>
          </div>
          <div class="flex items-center gap-3 shrink-0">
            <span class="hidden sm:flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider px-3 py-1.5 rounded-lg bg-black dark:bg-white text-white dark:text-black group-hover:bg-gray-800 dark:group-hover:bg-gray-200 transition-colors">
              <Play :size="10" />
              {{ t('appOverview.deploy') }}
            </span>
            <ExternalLink :size="14" class="text-gray-400 group-hover:text-gray-900 dark:group-hover:text-white transition-colors" />
          </div>
        </div>
      </section>

    </div>
  </div>
</template>
