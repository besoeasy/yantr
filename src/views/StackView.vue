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
  <div class="min-h-screen bg-white pb-20 font-sans text-zinc-900 selection:bg-blue-500/30 dark:bg-[#0A0A0A] dark:text-zinc-100">

    <!-- Loading -->
    <div v-if="loading" class="mx-auto flex max-w-7xl justify-center p-8 py-32">
       <div class="h-8 w-8 animate-spin rounded-full border-[3px] border-zinc-200 border-t-zinc-900 dark:border-zinc-800 dark:border-t-white"></div>
    </div>

    <!-- Content -->
    <main v-else-if="stack" class="mx-auto max-w-7xl animate-fadeIn space-y-6 px-6 py-8">
      <!-- ── App Header ───────────────────────────────────────────────────────────── -->
      <div class="group relative flex flex-col gap-6 rounded-2xl border border-zinc-200 bg-white p-6 transition-all duration-300 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] sm:flex-row dark:border-zinc-800 dark:bg-[#0A0A0A] dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
         <!-- Logo -->
         <div class="flex h-20 w-20 shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 p-4 transition-transform duration-500 group-hover:scale-105 dark:border-zinc-800 dark:bg-zinc-900">
           <AppLogo
             :logo="stack.app?.logo"
             :name="stack.app?.name || stack.appId"
             :seed="stack.app?.id || stack.projectId || stack.appId"
             img-class="h-full w-full object-contain filter transition-all dark:brightness-90 group-hover:brightness-100"
             icon-class="h-full w-full text-zinc-900 dark:text-zinc-100"
           />
         </div>

         <!-- Info -->
         <div class="flex-1 space-y-3">
            <div class="mb-1 flex flex-wrap items-center gap-2">
              <h1 class="text-2xl font-bold tracking-tight text-zinc-900 dark:text-white">
                {{ stack.app?.name || stack.appId }}
              </h1>
              <span
                class="rounded-full border px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest"
                :class="stateClass"
              >{{ overallState }}</span>
              <span
                class="rounded-md border border-zinc-200 bg-zinc-50 px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50"
              >{{ stack.projectId }}</span>
            </div>

            <!-- Description -->
            <p v-if="stack.app?.short_description" class="max-w-2xl text-sm leading-relaxed text-zinc-500">
              {{ stack.app.short_description }}
            </p>

            <!-- Tags -->
            <div v-if="stack.app?.tags?.length" class="flex flex-wrap gap-2 pt-2">
              <span
                v-for="tag in (stack.app.tags).slice(0, 6)"
                :key="tag"
                class="inline-flex items-center gap-1.5 rounded-md border border-zinc-200 bg-zinc-50 px-2.5 py-1 text-[10px] font-bold uppercase tracking-widest text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50"
              >{{ tag }}</span>
            </div>

            <!-- Actions -->
            <div class="flex flex-wrap items-center gap-2 pt-2">
              <!-- Custom app badge -->
              <div
                v-if="stack.app?.customapp"
                class="inline-flex items-center gap-1.5 rounded-lg border border-purple-200 bg-purple-50 px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-purple-600 dark:border-purple-500/20 dark:bg-purple-500/10 dark:text-purple-400"
              >
                <ShieldCheck :size="13" />{{ t("stackView.builtByYantr") }}
              </div>

              <button
                v-if="stack.app"
                @click="router.push(`/apps/${stack.appname || stack.appId}`)"
                class="inline-flex items-center gap-1.5 rounded-lg border border-zinc-200 px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-zinc-700 transition-all hover:bg-zinc-50 dark:border-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-900/50"
              >
                <ExternalLink :size="13" />{{ t("stackView.appPage") }}
              </button>

              <!-- Update -->
              <button
                v-if="!stack.app?.customapp"
                @click="updateStack"
                :disabled="updating"
                class="inline-flex items-center gap-1.5 rounded-lg border border-zinc-900 bg-zinc-900 px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-white transition-all hover:bg-black disabled:cursor-not-allowed disabled:opacity-50 dark:border-white dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-100"
              >
                <RotateCcw :size="13" :class="updating ? 'animate-spin' : ''" />
                {{ updating ? t("stackView.updating") : t("stackView.updateStack") }}
              </button>

              <!-- Remove -->
              <button
                @click="removeStack"
                :disabled="removing"
                class="inline-flex items-center gap-1.5 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-red-600 transition-all hover:border-red-300 hover:bg-red-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-red-900/30 dark:bg-red-900/10 dark:text-red-500 dark:hover:border-red-900/50 dark:hover:bg-red-900/20"
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
            <h3 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">
              {{ t("stackView.networkAccess") }}
            </h3>
            <span
              v-if="enrichedPorts.length > 0"
              class="inline-flex items-center rounded-md border border-zinc-200 bg-zinc-50 px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-widest text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50"
            >
              {{ visiblePorts.length }}
            </span>
          </div>
          <div v-if="hasDescribedPorts" class="flex items-center gap-1 rounded-lg bg-zinc-50 p-1 dark:bg-zinc-900">
            <button @click="showOnlyDescribedPorts = false" :class="!showOnlyDescribedPorts ? 'bg-white text-zinc-900 shadow-sm dark:bg-zinc-800 dark:text-white' : 'text-zinc-500 hover:text-zinc-900 dark:hover:text-zinc-300'" class="rounded-md px-3 py-1 text-[10px] font-bold uppercase tracking-wider transition-all">{{ t("stackView.allPorts") }}</button>
            <button @click="showOnlyDescribedPorts = true" :class="showOnlyDescribedPorts ? 'bg-white text-zinc-900 shadow-sm dark:bg-zinc-800 dark:text-white' : 'text-zinc-500 hover:text-zinc-900 dark:hover:text-zinc-300'" class="rounded-md px-3 py-1 text-[10px] font-bold uppercase tracking-wider transition-all">{{ t("stackView.described") }}</button>
          </div>
        </div>

        <div v-if="enrichedPorts.length > 0" class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          <div
            v-for="(p, i) in visiblePorts"
            :key="i"
            class="group flex h-full flex-col rounded-2xl border border-zinc-200 bg-white p-5 transition-all duration-300 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-[#0A0A0A] dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
          >
            <div class="mb-4 flex items-start justify-between">
              <div class="flex min-w-0 flex-1 items-start gap-4">
                <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 text-zinc-900 transition-transform duration-300 group-hover:scale-105 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100">
                  <Globe v-if="p.labeledProtocol === 'http' || p.labeledProtocol === 'https'" :size="18" />
                  <Network v-else :size="18" />
                </div>
                <div class="min-w-0 flex-1">
                  <div class="mb-1.5 flex items-center gap-2">
                    <span class="font-mono text-[10px] font-bold uppercase text-zinc-900 dark:text-white">{{ p.protocol }}</span>
                    <span v-if="p.labeledProtocol" class="rounded-md border border-zinc-200 bg-zinc-50 px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-widest text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50">
                      {{ p.labeledProtocol }}
                    </span>
                  </div>
                  <div class="truncate text-[11px] text-zinc-500" :title="p.label || p.service">
                    {{ p.label || p.service }}
                  </div>
                </div>
              </div>
            </div>

            <div class="mb-5 space-y-2">
              <div class="flex items-center justify-between text-[11px]">
                <span class="font-bold uppercase tracking-wider text-zinc-500">Host Port</span>
                <span v-if="p.hostPort" class="font-mono font-bold text-zinc-900 dark:text-white">:{{ p.hostPort }}</span>
                <span v-else class="italic text-zinc-400">Internal</span>
              </div>
              <div class="flex items-center justify-between text-[11px]">
                <span class="font-bold uppercase tracking-wider text-zinc-500">Container Port</span>
                <span class="font-mono font-medium text-zinc-700 dark:text-zinc-300">{{ p.containerPort }}</span>
              </div>
            </div>

            <div class="mt-auto">
              <a
                v-if="p.protocol === 'tcp'"
                :href="appUrl(p.hostPort, p.labeledProtocol || 'http')"
                target="_blank"
                class="flex w-full items-center justify-center gap-2 rounded-xl border border-zinc-900 bg-zinc-900 px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-white transition-colors hover:bg-black dark:border-white dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-100"
              >
                <ExternalLink :size="12" />{{ t("stackView.open") }}
              </a>
              <div
                v-else
                class="flex w-full items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50"
              >
                {{ p.protocol.toUpperCase() }}
              </div>
            </div>
          </div>
        </div>

        <div v-else class="group flex flex-col items-center justify-center gap-3 rounded-2xl border-2 border-dashed border-zinc-200 bg-white p-10 dark:border-zinc-800 dark:bg-[#0A0A0A]">
          <Network :size="28" class="text-zinc-300 dark:text-zinc-700" />
          <span class="text-xs font-bold uppercase tracking-widest text-zinc-400 dark:text-zinc-500">{{ t("stackView.noPortsPublished") }}</span>
        </div>
      </div>

      <!-- ── Internal Addresses Overview ─────────────────────────────────────────────── -->
      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <h3 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">
              {{ t("stackView.internalAddresses") }}
            </h3>
            <span
              v-if="servicesWithNetworks.length > 0"
              class="inline-flex items-center rounded-md border border-zinc-200 bg-zinc-50 px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-widest text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50"
            >
              {{ servicesWithNetworks.length }}
            </span>
          </div>
          <div class="text-[11px] font-medium text-zinc-500">
            {{ t("stackView.internalAddressesHint") }}
          </div>
        </div>

        <div v-if="servicesWithNetworks.length > 0" class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          <div
            v-for="svc in servicesWithNetworks"
            :key="`${svc.id}-networks`"
            class="group rounded-2xl border border-zinc-200 bg-white p-5 transition-all duration-300 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-[#0A0A0A] dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
          >
            <div class="mb-4 flex items-start justify-between gap-3">
              <div>
                <div class="text-sm font-bold text-zinc-900 dark:text-white">
                  {{ svc.service }}
                </div>
                <div class="mt-1 text-[11px] font-bold uppercase tracking-widest text-zinc-500">
                  {{ svc.composeService || svc.name }}
                </div>
              </div>
              <span class="inline-flex items-center rounded-md border border-zinc-200 bg-zinc-50 px-2 py-0.5 text-[9px] font-bold uppercase tracking-widest text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/50">
                {{ svc.networks.length }}
              </span>
            </div>

            <div class="space-y-2.5">
              <div
                v-for="network in svc.networks"
                :key="`${svc.id}-${network.name}`"
                class="rounded-xl border border-zinc-200 bg-zinc-50 px-3.5 py-3 dark:border-zinc-800 dark:bg-zinc-900/50"
              >
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <span class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">
                    {{ network.name }}
                  </span>
                  <span class="font-mono text-[11px] font-bold text-zinc-900 dark:text-white">
                    {{ network.ipAddress }}
                  </span>
                </div>
                <div v-if="network.aliases?.length" class="mt-2 text-[10px] text-zinc-500">
                  {{ t("stackView.networkAliases") }} {{ network.aliases.join(", ") }}
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-else class="group flex flex-col items-center justify-center gap-3 rounded-2xl border-2 border-dashed border-zinc-200 bg-white p-10 dark:border-zinc-800 dark:bg-[#0A0A0A]">
          <Network :size="28" class="text-zinc-300 dark:text-zinc-700" />
          <span class="text-xs font-bold uppercase tracking-widest text-zinc-400 dark:text-zinc-500">{{ t("stackView.noInternalAddresses") }}</span>
        </div>
      </div>

      <!-- ── Section Navigation ───────────────────────────────────────────────────────── -->
      <div class="overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-[#0A0A0A]">
        <div class="mx-6 mt-6 flex items-center gap-1 rounded-xl bg-zinc-50 p-1 dark:bg-zinc-900">
          <button
            v-for="sec in sectionTabs"
            :key="sec.id"
            @click="activeSection = sec.id"
            :class="activeSection === sec.id ? 'bg-white text-zinc-900 shadow-sm dark:bg-zinc-800 dark:text-white' : 'text-zinc-500 hover:text-zinc-900 dark:hover:text-zinc-300'"
            class="flex items-center gap-2 rounded-lg px-4 py-2 text-[11px] font-bold uppercase tracking-wider transition-all"
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
              <h3 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">
                {{ t("stackView.storageVolumes") }}
              </h3>
              <div class="grid gap-4">
                <div v-for="(vol, i) in namedVolumes" :key="vol.name" class="group rounded-2xl border border-zinc-200 bg-white p-5 transition-all duration-300 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-[#0A0A0A] dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
                  <div class="mb-5 flex items-start justify-between gap-4">
                    <div class="flex min-w-0 flex-1 items-start gap-4">
                      <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 text-zinc-900 transition-colors group-hover:bg-zinc-100 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-100 dark:group-hover:bg-zinc-800">
                        <HardDrive :size="18" />
                      </div>
                      <div class="min-w-0 flex-1">
                        <div class="truncate text-sm font-bold tracking-tight text-zinc-900 dark:text-white" :title="vol.name">{{ vol.name }}</div>
                        <div class="mt-1 truncate font-mono text-[11px] text-zinc-500">{{ vol.destination }}</div>
                        <div class="mt-2 flex flex-wrap items-center gap-3 text-[11px] text-zinc-500">
                          <span>{{ t("stackView.serviceLabel") }} <span class="ml-1 rounded-md border border-zinc-200 bg-zinc-50 px-1.5 py-0.5 font-bold text-zinc-700 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300">{{ vol.svcName }}</span></span>
                        </div>
                      </div>
                    </div>
                  </div>
                  <div class="flex flex-wrap items-center gap-2 border-t border-zinc-100 pt-4 dark:border-zinc-800">
                    <div v-if="browsingVolume[vol.name]" class="animate-pulse rounded-lg border border-zinc-900 bg-zinc-900 px-3 py-2 text-[10px] font-bold uppercase tracking-wider text-white dark:border-white dark:bg-white dark:text-zinc-900">{{ t("stackView.startingWebDAV") }}</div>
                    <template v-else-if="!showVolumeMenu[vol.name]">
                      <button @click="showVolumeMenu[vol.name] = true" class="rounded-lg border border-zinc-200 bg-zinc-50 px-3.5 py-2 text-[10px] font-bold uppercase tracking-wider text-zinc-700 transition-all hover:bg-zinc-100 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:bg-zinc-800">
                        {{ t("stackView.browseFiles") }}
                      </button>
                    </template>
                    <template v-else>
                      <button @click="browseVolume(vol.name, 60)" class="rounded-lg border border-zinc-900 bg-zinc-900 px-3 py-2 text-[10px] font-bold uppercase tracking-wider text-white transition-all hover:bg-black dark:border-white dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-100" :title="t('stackView.oneHourAccess')">1H</button>
                      <button @click="browseVolume(vol.name, 0)" class="rounded-lg border border-zinc-200 bg-zinc-50 px-3 py-2 text-[10px] font-bold uppercase tracking-wider text-zinc-800 transition-all hover:bg-zinc-100 dark:border-zinc-800 dark:bg-zinc-800 dark:text-zinc-200 dark:hover:bg-zinc-700" :title="t('stackView.permanentAccess')">Perm</button>
                    </template>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="otherMounts.length > 0" class="mt-6 space-y-4">
              <h3 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">
                {{ t("stackView.bindMounts") }}
              </h3>
              <div class="overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-[#0A0A0A]">
                <table class="min-w-80 w-full text-left">
                  <thead>
                    <tr class="border-b border-zinc-200 bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-900/50">
                      <th class="px-5 py-3 text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t("stackView.type") }}</th>
                      <th class="px-5 py-3 text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t("stackView.hostPath") }}</th>
                      <th class="px-5 py-3 text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t("stackView.containerPath") }}</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-zinc-100 dark:divide-zinc-800">
                    <tr v-for="(m, i) in otherMounts" :key="i" class="transition-colors hover:bg-zinc-50 dark:hover:bg-zinc-900/50">
                      <td class="px-5 py-3.5">
                        <span class="rounded-md border border-zinc-200 bg-zinc-50 px-2.5 py-1 text-[10px] font-bold uppercase tracking-widest text-zinc-500 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-400">{{ m.type }}</span>
                      </td>
                      <td class="max-w-xs break-all px-5 py-3.5 font-mono text-[11px] text-zinc-900 dark:text-white">{{ m.source || "—" }}</td>
                      <td class="max-w-xs break-all px-5 py-3.5 font-mono text-[11px] text-zinc-500 dark:text-zinc-400">{{ m.destination }}</td>
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