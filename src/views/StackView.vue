<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useApiUrl } from "../composables/useApiUrl";
import { useCurrentTime } from "../composables/useCurrentTime";
import { useNotification } from "../composables/useNotification";
import { formatDuration, formatBytes } from "../utils/metrics";
import { useYantrAuth } from "../composables/useYantrAuth";
import AppLogo from "../components/AppLogo.vue";
import StackServiceList from "../components/StackServiceList.vue";
import {
  Globe,
  ExternalLink,
  Activity,
  Terminal,
  Server,
  Network,
  Trash2,
  HardDrive,
  FolderOpen,
  ChevronRight,
  RotateCcw,
  Plus,
  ShieldCheck,
} from "@lucide/vue";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const { apiUrl } = useApiUrl();
const { currentTime } = useCurrentTime();
const toast = useNotification();
const { openVolumeBrowser } = useYantrAuth();

const projectId = computed(() => route.params.projectId);

const stack = ref(null);
const loading = ref(true);
const removing = ref(false);
const updating = ref(false);

async function updateStack() {
  if (updating.value || !stack.value) return;
  updating.value = true;
  toast.info(t("stackView.updatingStack", { name: stack.value.app?.name || projectId.value }));
  try {
    const containerIds = stack.value.services.map((s) => s.id).filter(Boolean);
    const res = await fetch(`${apiUrl.value}/api/autoupdate/run`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ containerIds }),
    });
    const data = await res.json();
    if (data.success) {
      if (data.updatedCount > 0) {
        toast.success(t("stackView.updateComplete", { count: data.updatedCount }));
      } else {
        toast.info(t("stackView.updateAlreadyLatest"));
      }
      await fetchStack();
    } else {
      toast.error(data.error || t("stackView.updateFailed"));
    }
  } catch (e) {
    if (e?.message?.includes("timed out") || String(e).toLowerCase().includes("timeout")) {
      toast.error(t("stackView.updateTimedOut"));
    } else {
      toast.error(t("stackView.updateFailed"));
    }
  } finally {
    updating.value = false;
  }
}
const showOnlyDescribedPorts = ref(true);



// Volume browsing state
const browsingVolume = ref({});
const showVolumeMenu = ref({});

// Top-level section navigation
const activeSection = ref("containers"); // 'containers' | 'storage'

const sectionTabs = computed(() => [
  ...(namedVolumes.value.length > 0 || otherMounts.value.length > 0
    ? [{ id: "storage", label: t("system.volumes"), icon: HardDrive, tone: "text-emerald-500" }]
    : []),
  { id: "containers", label: t("stackView.containers"), icon: Server, tone: "text-amber-500" },
]);

const sectionTabsGridClass = computed(() => {
  switch (sectionTabs.value.length) {
    case 1:
      return "grid-cols-1";
    case 2:
      return "grid-cols-2";
    case 3:
      return "grid-cols-2 lg:grid-cols-3";
    case 4:
      return "grid-cols-2 lg:grid-cols-4";
    default:
      return "grid-cols-2 lg:grid-cols-5";
  }
});

// Build a port-number → {label, protocol} lookup from the info.json ports array
function buildPortLabels(ports) {
  const labels = {};
  if (!Array.isArray(ports)) return labels;
  for (const p of ports) {
    if (p.port != null) {
      labels[String(p.port)] = {
        protocol: (p.protocol || "").toLowerCase(),
        label: p.label || null,
      };
    }
  }
  return labels;
}

// Merge published ports with described labels from info.json
const enrichedPorts = computed(() => {
  if (!stack.value) return [];
  const portLabels = buildPortLabels(stack.value.app?.ports);
  return stack.value.publishedPorts.map((p) => ({
    ...p,
    label: portLabels[String(p.containerPort)]?.label || null,
    labeledProtocol: portLabels[String(p.containerPort)]?.protocol || null,
  }));
});

const visiblePorts = computed(() => {
  if (!showOnlyDescribedPorts.value) return enrichedPorts.value;
  const described = enrichedPorts.value.filter((p) => p.label);
  // Fall back to all if none have descriptions
  return described.length > 0 ? described : enrichedPorts.value;
});

const hasDescribedPorts = computed(() => enrichedPorts.value.some((p) => p.label));

const servicesWithNetworks = computed(() => {
  if (!stack.value?.services) return [];
  return stack.value.services.filter((service) => Array.isArray(service.networks) && service.networks.length > 0);
});

// Collect all unique mounts across all services
const allMounts = computed(() => {
  if (!stack.value) return [];
  const seen = new Set();
  const result = [];
  for (const svc of stack.value.services) {
    for (const m of svc.mounts || []) {
      const key = `${m.type}:${m.source}:${m.destination}`;
      if (!seen.has(key)) {
        seen.add(key);
        result.push({ ...m, svcName: svc.service, svcId: svc.id });
      }
    }
  }
  const order = { volume: 0, bind: 1, tmpfs: 2 };
  result.sort((a, b) => (order[a.type] ?? 9) - (order[b.type] ?? 9));
  return result;
});

// Named Docker volumes only
const namedVolumes = computed(() => allMounts.value.filter((m) => m.type === "volume" && m.name));

// Bind mounts and tmpfs — shown in a simple compact list
const otherMounts = computed(() => allMounts.value.filter((m) => m.type !== "volume" || !m.name));

function appUrl(hostPort, proto) {
  const scheme = proto === "https" ? "https" : "http";
  return `${scheme}://${window.location.hostname}:${hostPort}`;
}

let refreshInterval = null;

// ── helpers ───────────────────────────────────────────────────────────────────

function formatUptime(service) {
  if (service.state !== "running" || !service.created) return null;
  const uptime = currentTime.value - service.created * 1000;
  if (uptime <= 0) return t("stackView.justStarted");
  return formatDuration(uptime);
}

const overallState = computed(() => {
  if (!stack.value) return "unknown";
  const states = stack.value.services.map((s) => s.state);
  if (states.every((s) => s === "running")) return t("stackView.running");
  if (states.some((s) => s === "running")) return t("stackView.partial");
  return t("stackView.stopped");
});

const stateClass = computed(() => {
  if (overallState.value === "running") return "bg-green-50 dark:bg-green-500/10 text-green-600 dark:text-green-400 border-green-200 dark:border-green-500/20";
  if (overallState.value === "partial") return "bg-amber-50 dark:bg-amber-500/10 text-amber-700 dark:text-amber-400 border-amber-200 dark:border-amber-500/20";
  return "bg-gray-100 dark:bg-zinc-900 text-gray-600 dark:text-zinc-400 border-gray-200 dark:border-zinc-800";
});

// ── API ────────────────────────────────────────────────────────────────────────

async function fetchStack() {
  try {
    const res = await fetch(`${apiUrl.value}/api/stacks/${projectId.value}`);
    const data = await res.json();
    if (data.success) {
      stack.value = data.stack;
    } else {
      toast.error(t("stackView.stackNotFound"));
      router.push("/");
    }
  } catch (e) {
    console.error("Failed to load stack:", e);
    toast.error(t("stackView.failedToLoadStack"));
  } finally {
    loading.value = false;
  }
}



async function removeStack() {
  if (removing.value) return;
  const name = stack.value?.app?.name || projectId.value;
  if (!confirm(t("stackView.removeStackConfirm", { name }))) return;

  removing.value = true;
  toast.info(t("stackView.removingStack", { name }));

  try {
    const firstId = stack.value?.services?.[0]?.id;
    if (!firstId) throw new Error(t("stackView.noContainerFound"));

    const res = await fetch(`${apiUrl.value}/api/containers/${firstId}`, { method: "DELETE" });
    const data = await res.json();
    if (data.success) {
      toast.success(t("stackView.stackRemoved", { name }));
      router.push("/");
    } else {
      throw new Error(data.message || t("stackView.removalFailed"));
    }
  } catch (e) {
    console.error("Remove error:", e);
    toast.error(t("stackView.failedToRemove", { error: e.message }));
  } finally {
    removing.value = false;
  }
}

// ── lifecycle ─────────────────────────────────────────────────────────────────

// ── Volume browsing ──────────────────────────────────────────────────────────

async function browseVolume(volumeName, expiryMinutes = 60) {
  console.log(`[StackView] Starting volume browser for volume: ${volumeName} (Expiry: ${expiryMinutes}m)`);
  browsingVolume.value[volumeName] = true;
  showVolumeMenu.value[volumeName] = false;
  try {
    const response = await fetch(`${apiUrl.value}/api/volumes/${volumeName}/browse`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ expiryMinutes }),
    });
    const data = await response.json();
    if (data.success) {
      const expiryText = expiryMinutes > 0 ? ` (${t("stackView.expiresIn", { minutes: expiryMinutes })})` : ` (${t("stackView.noExpiry")})`;
      toast.success(t("stackView.volumeBrowserStarted", { expiry: expiryText }));
      openVolumeBrowser(volumeName);
    } else {
      console.error(`[StackView] Failed to start browser for volume: ${volumeName}`, data);
    }
  } catch (e) {
    console.error(`[StackView] Error starting browser for volume: ${volumeName}`, e);
    toast.error(t("stackView.failedToStartVolumeBrowser"));
  } finally {
    delete browsingVolume.value[volumeName];
  }
}

onMounted(async () => {
  await fetchStack();
  refreshInterval = setInterval(fetchStack, 8000);
});

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval);
});
</script>

<template>
  <div class="min-h-screen bg-white dark:bg-[#0A0A0A] text-gray-900 dark:text-zinc-100 font-sans selection:bg-blue-500/30">

    <!-- Loading -->
    <div v-if="loading" class="max-w-7xl mx-auto p-8 flex justify-center py-32">
       <div class="w-8 h-8 border-[3px] border-gray-200 dark:border-zinc-800 border-t-blue-500 dark:border-t-blue-500 rounded-full animate-spin"></div>
    </div>

    <!-- Content -->
    <main v-else-if="stack" class="max-w-7xl mx-auto px-6 py-8 space-y-6 animate-fadeIn">
      <!-- ── App Header ───────────────────────────────────────────────────────────── -->
      <div class="group relative bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 p-6 flex flex-col sm:flex-row gap-6 hover:border-gray-300 dark:hover:border-zinc-700 transition-all duration-300">
         <div class="absolute top-0 left-0 w-full h-0.5 bg-linear-to-r from-transparent via-blue-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>

         <!-- Logo -->
         <div class="w-20 h-20 bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 rounded-xl flex items-center justify-center p-4 shrink-0 shadow-sm transition-transform group-hover:scale-105 duration-500">
           <AppLogo
             :logo="stack.app?.logo"
             :name="stack.app?.name || stack.appId"
             :seed="stack.app?.id || stack.projectId || stack.appId"
             img-class="w-full h-full object-contain filter dark:brightness-90 group-hover:brightness-100 transition-all"
             icon-class="w-full h-full text-gray-400 dark:text-zinc-600"
           />
         </div>

         <!-- Info -->
         <div class="flex-1 space-y-3">
            <div class="flex items-center flex-wrap gap-2 mb-1">
              <h1 class="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">
                {{ stack.app?.name || stack.appId }}
              </h1>
              <span
                class="text-[10px] font-bold uppercase tracking-widest px-2 py-0.5 rounded-full border"
                :class="stateClass"
              >{{ overallState }}</span>
              <span
                class="text-[10px] font-bold uppercase tracking-widest px-2 py-0.5 rounded-md border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900/50 text-gray-600 dark:text-zinc-400"
              >{{ stack.projectId }}</span>
            </div>

            <!-- Description -->
            <p v-if="stack.app?.short_description" class="text-gray-500 dark:text-zinc-400 text-sm leading-relaxed max-w-2xl">
              {{ stack.app.short_description }}
            </p>

            <!-- Tags -->
            <div v-if="stack.app?.tags?.length" class="pt-2 flex flex-wrap gap-2">
              <span
                v-for="tag in (stack.app.tags).slice(0, 6)"
                :key="tag"
                class="inline-flex items-center gap-1.5 px-2.5 py-1 border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900/50 text-[10px] font-bold tracking-widest text-gray-600 dark:text-zinc-400 rounded-md uppercase"
              >{{ tag }}</span>
            </div>

            <!-- Actions -->
            <div class="flex flex-wrap items-center gap-2 pt-2">
              <!-- Custom app badge -->
              <div
                v-if="stack.app?.customapp"
                class="inline-flex items-center gap-1.5 text-[11px] font-bold px-3 py-2 rounded-lg bg-purple-50 dark:bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-200 dark:border-purple-500/20 uppercase tracking-wider"
              >
                <ShieldCheck :size="13" />{{ t("stackView.builtByYantr") }}
              </div>

              <button
                v-if="stack.app"
                @click="router.push(`/apps/${stack.appname || stack.appId}`)"
                class="inline-flex items-center gap-1.5 text-[11px] font-bold px-3 py-2 rounded-lg border border-gray-200 dark:border-zinc-800 text-gray-700 dark:text-zinc-300 hover:bg-gray-50 dark:hover:bg-zinc-900/50 transition-all uppercase tracking-wider"
              >
                <ExternalLink :size="13" />{{ t("stackView.appPage") }}
              </button>

              <!-- Update -->
              <button
                v-if="!stack.app?.customapp"
                @click="updateStack"
                :disabled="updating"
                class="inline-flex items-center gap-1.5 text-[11px] font-bold px-3 py-2 rounded-lg bg-black dark:bg-white text-white dark:text-black hover:bg-gray-800 dark:hover:bg-gray-200 transition-all disabled:opacity-50 disabled:cursor-not-allowed uppercase tracking-wider"
              >
                <RotateCcw :size="13" :class="updating ? 'animate-spin' : ''" />
                {{ updating ? t("stackView.updating") : t("stackView.updateStack") }}
              </button>

              <!-- Remove -->
              <button
                @click="removeStack"
                :disabled="removing"
                class="inline-flex items-center gap-1.5 text-[11px] font-bold px-3 py-2 rounded-lg border bg-red-50 dark:bg-red-500/10 text-red-600 dark:text-red-500 border-red-200 dark:border-red-500/20 hover:bg-red-100 dark:hover:bg-red-500/20 transition-all disabled:opacity-50 disabled:cursor-not-allowed uppercase tracking-wider"
              >
                <Trash2 :size="13" />{{ removing ? t("stackView.removing") : t("stackView.removeStack") }}
              </button>
            </div>
         </div>
      </div>

      <!-- ── Ports Overview ─────────────────────────────────────────────────────────── -->
      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <h3 class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">
              {{ t("stackView.networkAccess") }}
            </h3>
            <span
              v-if="enrichedPorts.length > 0"
              class="inline-flex items-center rounded-md px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-widest border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900/50 text-gray-500 dark:text-zinc-400"
            >
              {{ visiblePorts.length }}
            </span>
          </div>
          <div v-if="hasDescribedPorts" class="flex items-center gap-1 rounded-lg bg-gray-100 dark:bg-zinc-900 p-1">
            <button @click="showOnlyDescribedPorts = false" :class="!showOnlyDescribedPorts ? 'bg-white dark:bg-zinc-800 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 hover:text-gray-700 dark:hover:text-zinc-300'" class="px-3 py-1 text-[10px] font-bold uppercase tracking-wider rounded-md transition-all">{{ t("stackView.allPorts") }}</button>
            <button @click="showOnlyDescribedPorts = true" :class="showOnlyDescribedPorts ? 'bg-white dark:bg-zinc-800 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 hover:text-gray-700 dark:hover:text-zinc-300'" class="px-3 py-1 text-[10px] font-bold uppercase tracking-wider rounded-md transition-all">{{ t("stackView.described") }}</button>
          </div>
        </div>

        <div v-if="enrichedPorts.length > 0" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <div
            v-for="(p, i) in visiblePorts"
            :key="i"
            class="group bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl p-5 hover:border-gray-300 dark:hover:border-zinc-600 transition-all duration-300 flex flex-col h-full"
          >
            <div class="flex items-start justify-between mb-4">
              <div class="flex items-start gap-3.5 flex-1 min-w-0">
                <div class="w-10 h-10 rounded-lg bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center shrink-0 shadow-sm transition-colors group-hover:text-blue-500 text-gray-600 dark:text-zinc-400">
                  <Globe v-if="p.labeledProtocol === 'http' || p.labeledProtocol === 'https'" :size="18" />
                  <Network v-else :size="18" />
                </div>
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-2 mb-1.5">
                    <span class="font-mono text-[10px] font-bold uppercase text-gray-900 dark:text-white">{{ p.protocol }}</span>
                    <span v-if="p.labeledProtocol" class="text-[9px] px-1.5 py-0.5 bg-gray-100 dark:bg-zinc-800 text-gray-500 dark:text-zinc-400 rounded-md uppercase font-bold tracking-widest border border-gray-200 dark:border-zinc-700">
                      {{ p.labeledProtocol }}
                    </span>
                  </div>
                  <div class="text-[11px] text-gray-500 dark:text-zinc-400 truncate" :title="p.label || p.service">
                    {{ p.label || p.service }}
                  </div>
                </div>
              </div>
            </div>

            <div class="space-y-2 mb-5">
              <div class="flex items-center justify-between text-[11px]">
                <span class="text-gray-500 dark:text-zinc-500 uppercase font-bold tracking-wider">Host Port</span>
                <span v-if="p.hostPort" class="font-mono font-bold text-gray-900 dark:text-white">:{{ p.hostPort }}</span>
                <span v-else class="text-gray-400 italic">Internal</span>
              </div>
              <div class="flex items-center justify-between text-[11px]">
                <span class="text-gray-500 dark:text-zinc-500 uppercase font-bold tracking-wider">Container Port</span>
                <span class="font-mono font-medium text-gray-700 dark:text-zinc-300">{{ p.containerPort }}</span>
              </div>
            </div>

            <div class="mt-auto">
              <a
                v-if="p.protocol === 'tcp'"
                :href="appUrl(p.hostPort, p.labeledProtocol || 'http')"
                target="_blank"
                class="w-full flex items-center justify-center gap-2 px-3 py-2 bg-black dark:bg-white text-white dark:text-black rounded-lg hover:bg-gray-800 dark:hover:bg-gray-200 transition-all text-[11px] font-bold uppercase tracking-wider"
              >
                <ExternalLink :size="12" />{{ t("stackView.open") }}
              </a>
              <div
                v-else
                class="w-full flex items-center justify-center px-3 py-2 bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 text-gray-400 dark:text-zinc-500 rounded-lg text-[11px] font-bold uppercase tracking-wider"
              >
                {{ p.protocol.toUpperCase() }}
              </div>
            </div>
          </div>
        </div>

        <div v-else class="group bg-white dark:bg-[#0A0A0A] border-2 border-dashed border-gray-200 dark:border-zinc-800 rounded-xl p-10 flex flex-col items-center justify-center gap-3">
          <Network :size="28" class="text-gray-300 dark:text-zinc-700" />
          <span class="text-xs font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500">{{ t("stackView.noPortsPublished") }}</span>
        </div>
      </div>

      <!-- ── Internal Addresses Overview ─────────────────────────────────────────────── -->
      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <h3 class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">
              {{ t("stackView.internalAddresses") }}
            </h3>
            <span
              v-if="servicesWithNetworks.length > 0"
              class="inline-flex items-center rounded-md px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-widest border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900/50 text-gray-500 dark:text-zinc-400"
            >
              {{ servicesWithNetworks.length }}
            </span>
          </div>
          <div class="text-[11px] text-gray-500 dark:text-zinc-500 font-medium">
            {{ t("stackView.internalAddressesHint") }}
          </div>
        </div>

        <div v-if="servicesWithNetworks.length > 0" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <div
            v-for="svc in servicesWithNetworks"
            :key="`${svc.id}-networks`"
            class="group bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl p-5 hover:border-gray-300 dark:hover:border-zinc-600 transition-all duration-300"
          >
            <div class="flex items-start justify-between gap-3 mb-4">
              <div>
                <div class="text-sm font-bold text-gray-900 dark:text-white">
                  {{ svc.service }}
                </div>
                <div class="mt-1 text-[11px] uppercase tracking-widest text-gray-500 dark:text-zinc-400 font-bold">
                  {{ svc.composeService || svc.name }}
                </div>
              </div>
              <span class="inline-flex items-center rounded-md px-2 py-0.5 text-[9px] font-bold uppercase tracking-widest border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900/50 text-gray-500 dark:text-zinc-400">
                {{ svc.networks.length }}
              </span>
            </div>

            <div class="space-y-2.5">
              <div
                v-for="network in svc.networks"
                :key="`${svc.id}-${network.name}`"
                class="rounded-lg px-3.5 py-3 bg-gray-50 dark:bg-zinc-900/50 border border-gray-100 dark:border-zinc-800"
              >
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <span class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-400">
                    {{ network.name }}
                  </span>
                  <span class="font-mono font-bold text-[11px] text-gray-900 dark:text-white">
                    {{ network.ipAddress }}
                  </span>
                </div>
                <div v-if="network.aliases?.length" class="mt-2 text-[10px] text-gray-500 dark:text-zinc-500">
                  {{ t("stackView.networkAliases") }} {{ network.aliases.join(", ") }}
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-else class="group bg-white dark:bg-[#0A0A0A] border-2 border-dashed border-gray-200 dark:border-zinc-800 rounded-xl p-10 flex flex-col items-center justify-center gap-3">
          <Network :size="28" class="text-gray-300 dark:text-zinc-700" />
          <span class="text-xs font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500">{{ t("stackView.noInternalAddresses") }}</span>
        </div>
      </div>

      <!-- ── Section Navigation ───────────────────────────────────────────────────────── -->
      <div class="bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 overflow-hidden shadow-sm">
        <div class="flex items-center gap-1 rounded-lg bg-gray-100 dark:bg-zinc-900 p-1 mx-6 mt-6">
          <button
            v-for="sec in sectionTabs"
            :key="sec.id"
            @click="activeSection = sec.id"
            :class="activeSection === sec.id ? 'bg-white dark:bg-zinc-800 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 hover:text-gray-700 dark:hover:text-zinc-300'"
            class="flex items-center gap-2 px-4 py-2 text-[11px] font-bold uppercase tracking-wider rounded-md transition-all"
          >
            <component :is="sec.icon" :size="14" />
            {{ sec.label }}
          </button>
        </div>

        <div class="p-6">
          <!-- CONTAINERS SECTION -->
          <div v-show="activeSection === 'containers'">
            <StackServiceList 
              :services="stack.services" 
            />
          </div>

          <!-- STORAGE SECTION -->
          <div v-show="activeSection === 'storage'" class="space-y-4">
            <div v-if="namedVolumes.length > 0" class="space-y-4">
              <h3 class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">
                {{ t("stackView.storageVolumes") }}
              </h3>
              <div class="grid gap-4">
                <div v-for="(vol, i) in namedVolumes" :key="vol.name" class="group bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl p-5 hover:border-gray-300 dark:hover:border-zinc-600 transition-all duration-300">
                  <div class="flex items-start justify-between gap-4 mb-5">
                    <div class="flex items-start gap-4 min-w-0 flex-1">
                      <div class="w-10 h-10 rounded-lg bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center text-gray-500 dark:text-zinc-400 shrink-0 shadow-sm group-hover:text-blue-500 transition-colors">
                        <HardDrive :size="18" />
                      </div>
                      <div class="min-w-0 flex-1">
                        <div class="font-bold text-sm text-gray-900 dark:text-white truncate tracking-tight" :title="vol.name">{{ vol.name }}</div>
                        <div class="text-[11px] text-gray-500 dark:text-zinc-400 font-mono truncate mt-1">{{ vol.destination }}</div>
                        <div class="flex items-center gap-3 mt-2 text-[11px] text-gray-500 dark:text-zinc-400 flex-wrap">
                          <span>{{ t("stackView.serviceLabel") }} <span class="font-bold px-1.5 py-0.5 rounded-md ml-1 bg-gray-100 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 text-gray-700 dark:text-zinc-300">{{ vol.svcName }}</span></span>
                        </div>
                      </div>
                    </div>
                  </div>
                  <div class="flex items-center gap-2 flex-wrap pt-4 border-t border-gray-100 dark:border-zinc-800">
                    <div v-if="browsingVolume[vol.name]" class="text-[10px] font-bold uppercase tracking-wider text-blue-600 dark:text-blue-400 animate-pulse px-3 py-2 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800/50 rounded-lg">{{ t("stackView.startingWebDAV") }}</div>
                    <template v-else-if="!showVolumeMenu[vol.name]">
                      <button @click="showVolumeMenu[vol.name] = true" class="px-3.5 py-2 text-[10px] font-bold uppercase tracking-wider border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900 text-gray-700 dark:text-zinc-300 rounded-lg hover:bg-gray-100 dark:hover:bg-zinc-800 transition-all">
                        {{ t("stackView.browseFiles") }}
                      </button>
                    </template>
                    <template v-else>
                      <button @click="browseVolume(vol.name, 60)" class="px-3 py-2 text-[10px] font-bold uppercase tracking-wider bg-black dark:bg-white text-white dark:text-black rounded-lg hover:bg-gray-800 dark:hover:bg-gray-200 transition-all" :title="t('stackView.oneHourAccess')">1H</button>
                      <button @click="browseVolume(vol.name, 0)" class="px-3 py-2 text-[10px] font-bold uppercase tracking-wider bg-gray-200 dark:bg-zinc-800 text-gray-800 dark:text-zinc-200 rounded-lg hover:bg-gray-300 dark:hover:bg-zinc-700 transition-all" :title="t('stackView.permanentAccess')">Perm</button>
                    </template>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="otherMounts.length > 0" class="space-y-4 mt-6">
              <h3 class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">
                {{ t("stackView.bindMounts") }}
              </h3>
              <div class="bg-white dark:bg-[#0A0A0A] rounded-xl overflow-hidden border border-gray-200 dark:border-zinc-800 shadow-sm">
                <table class="w-full text-left min-w-80">
                  <thead>
                    <tr class="border-b border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900/50">
                      <th class="px-5 py-3 text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t("stackView.type") }}</th>
                      <th class="px-5 py-3 text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t("stackView.hostPath") }}</th>
                      <th class="px-5 py-3 text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500">{{ t("stackView.containerPath") }}</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-gray-100 dark:divide-zinc-800">
                    <tr v-for="(m, i) in otherMounts" :key="i" class="hover:bg-gray-50 dark:hover:bg-zinc-900/50 transition-colors">
                      <td class="px-5 py-3.5">
                        <span class="text-[10px] font-bold uppercase tracking-widest px-2.5 py-1 rounded-md border" :class="m.type === 'bind' ? 'bg-amber-50 dark:bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-200 dark:border-amber-500/20' : 'bg-gray-100 dark:bg-zinc-800 text-gray-500 dark:text-zinc-400 border-gray-200 dark:border-zinc-700'">{{ m.type }}</span>
                      </td>
                      <td class="px-5 py-3.5 font-mono text-[11px] text-gray-900 dark:text-white break-all max-w-xs">{{ m.source || "—" }}</td>
                      <td class="px-5 py-3.5 font-mono text-[11px] text-gray-500 dark:text-zinc-400 break-all max-w-xs">{{ m.destination }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>