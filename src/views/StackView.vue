<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useApiUrl } from "../composables/useApiUrl";
import { useCurrentTime } from "../composables/useCurrentTime";
import { useNotification } from "../composables/useNotification";
import { formatDuration, formatBytes } from "../utils/metrics";
import AppLogo from "../components/AppLogo.vue";
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
} from "lucide-vue-next";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const { apiUrl } = useApiUrl();
const { currentTime } = useCurrentTime();
const toast = useNotification();

const projectId = computed(() => route.params.projectId);

const stack = ref(null);
const loading = ref(true);
const removing = ref(false);
const updating = ref(false);
const openingPort = ref(false);
const newPort = ref({ serviceName: "", mapping: "" });

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

// ── Caddy Auth deployment ────────────────────────────────────────────────────
const caddyAuth = ref({ targetPort: "", servePort: "", user: "admin", pass: "" });
const deployingCaddy = ref(false);
const disablingCaddy = ref(false);

const caddyProxies = computed(() => stack.value?.caddyProxies || []);

watch(
  () => stack.value?.publishedPorts,
  (ports) => {
    if (ports?.length && !caddyAuth.value.targetPort) {
      const first = ports.find((p) => p.protocol === "tcp") || ports[0];
      if (first) caddyAuth.value.targetPort = String(first.containerPort);
    }
  },
  { immediate: true },
);

async function deployCaddyAuth() {
  if (deployingCaddy.value) return;
  if (!caddyAuth.value.servePort) {
    toast.error(t("stackView.caddyAuthServePortRequired"));
    return;
  }
  if (!caddyAuth.value.targetPort) {
    toast.error(t("stackView.caddyAuthTargetPortRequired"));
    return;
  }
  deployingCaddy.value = true;
  toast.info(t("stackView.caddyAuthDeploying"));
  try {
    const body = {
      projectId: projectId.value,
      servePort: Number(caddyAuth.value.servePort),
      targetPort: Number(caddyAuth.value.targetPort),
    };
    if (caddyAuth.value.user.trim() && caddyAuth.value.pass.trim()) {
      body.authUser = caddyAuth.value.user.trim();
      body.authPass = caddyAuth.value.pass;
    }
    const res = await fetch(`${apiUrl.value}/api/proxy/enable`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    const data = await res.json();
    if (data.success) {
      toast.success(t("stackView.caddyAuthDeployed"));
      await fetchStack();
    } else {
      toast.error(data.error || t("stackView.caddyAuthFailed"));
    }
  } catch (e) {
    toast.error(t("stackView.caddyAuthFailed"));
  } finally {
    deployingCaddy.value = false;
  }
}

async function disableCaddyAuth() {
  if (disablingCaddy.value) return;
  disablingCaddy.value = true;
  try {
    const res = await fetch(`${apiUrl.value}/api/proxy/disable`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ projectId: projectId.value }),
    });
    const data = await res.json();
    if (data.success) {
      toast.success(t("stackView.caddyAuthDisabled"));
      await fetchStack();
    } else {
      toast.error(data.error || t("stackView.caddyAuthFailed"));
    }
  } catch (e) {
    toast.error(t("stackView.caddyAuthFailed"));
  } finally {
    disablingCaddy.value = false;
  }
}

// Volume browsing state
const browsingVolume = ref({});
const showVolumeMenu = ref({});

// Top-level section navigation
const activeSection = ref("ports"); // 'ports' | 'auth' | 'containers' | 'storage'

const sectionTabs = computed(() => [
  ...(namedVolumes.value.length > 0 || otherMounts.value.length > 0
    ? [{ id: "storage", label: t("system.volumes"), icon: HardDrive, tone: "text-emerald-500" }]
    : []),
  { id: "ports", label: t("stackView.openPortAction"), icon: Plus, tone: "text-blue-500" },
  ...(stack.value?.appId !== "caddy-yantr"
    ? [{ id: "auth", label: t("stackView.auth"), icon: ShieldCheck, tone: "text-violet-500" }]
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

const portServices = computed(() => {
  if (!stack.value?.services) return [];
  const seen = new Set();
  return stack.value.services
    .filter((service) => service.composeService)
    .filter((service) => {
      if (seen.has(service.composeService)) return false;
      seen.add(service.composeService);
      return true;
    })
    .map((service) => ({
      value: service.composeService,
      label: service.service || service.composeService,
    }));
});

const needsPortServiceSelection = computed(() => portServices.value.length > 1);

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
      if (!portServices.value.some((service) => service.value === newPort.value.serviceName)) {
        newPort.value.serviceName = portServices.value[0]?.value || "";
      }
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

async function openPort() {
  if (openingPort.value || !stack.value) return;

  const serviceName = String(newPort.value.serviceName || "").trim();
  const portMapping = String(newPort.value.mapping || "").trim();

  if ((!serviceName && needsPortServiceSelection.value) || !portMapping) {
    toast.error(t("stackView.portFormInvalid"));
    return;
  }

  openingPort.value = true;
  toast.info(t("stackView.openingPort", { portMapping }));

  try {
    const res = await fetch(`${apiUrl.value}/api/stacks/${projectId.value}/ports`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ serviceName, portMapping }),
    });
    const data = await res.json();
    if (!res.ok || !data.success) {
      throw new Error(data.error || t("stackView.failedToOpenPort"));
    }

    toast.success(t("stackView.portOpened", { portMapping: data.port?.portMapping || portMapping }));
    newPort.value.mapping = "";
    await fetchStack();
  } catch (error) {
    toast.error(error.message || t("stackView.failedToOpenPort"));
  } finally {
    openingPort.value = false;
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
      window.open(`/browse/${volumeName}/`, "_blank");
    }
  } catch (e) {
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
  <div class="min-h-screen" style="background: var(--bg-body); color: var(--text-primary)">

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center min-h-[60vh]">
      <div class="w-7 h-7 border-2 border-gray-200 dark:border-zinc-700 border-t-blue-500 rounded-full animate-spin"></div>
    </div>

    <!-- Content -->
    <div v-else-if="stack" class="max-w-5xl mx-auto px-4 sm:px-8 py-8 space-y-4 animate-fadeIn">
      <!-- ── App Header ───────────────────────────────────────────────────────────── -->
      <div class="mb-3 px-1 pb-3 sm:mb-4 sm:pb-4">
        <div class="flex items-start gap-4 sm:gap-5">
          <!-- Logo -->
          <div
            class="w-18 h-18 sm:w-20 sm:h-20 rounded-2xl shrink-0 flex items-center justify-center overflow-hidden border border-gray-100 dark:border-zinc-800 hover:scale-105 transition-transform"
            style="background: var(--surface-muted)"
          >
            <AppLogo
              :logo="stack.app?.logo"
              :name="stack.app?.name || stack.appId"
              :seed="stack.app?.id || stack.projectId || stack.appId"
              img-class="w-full h-full object-contain"
              icon-class="w-full h-full text-gray-400 dark:text-zinc-500"
            />
          </div>

          <!-- Info -->
          <div class="flex-1 min-w-0">
            <!-- Name + badges -->
            <div class="flex items-center flex-wrap gap-2 mb-1">
              <h1 class="text-xl sm:text-2xl font-black tracking-tight" style="color: var(--text-primary)">
                {{ stack.app?.name || stack.appId }}
              </h1>
              <span
                class="text-[10px] font-bold uppercase tracking-widest px-2 py-0.5 rounded-full border"
                :class="stateClass"
              >{{ overallState }}</span>
              <span
                class="text-[10px] font-bold uppercase tracking-widest px-2 py-0.5 rounded-full border border-gray-200 dark:border-zinc-800 bg-gray-100 dark:bg-zinc-900"
                style="color: var(--text-secondary)"
              >{{ stack.projectId }}</span>
            </div>

            <!-- Description -->
            <p v-if="stack.app?.short_description" class="text-sm mb-3 leading-relaxed" style="color: var(--text-secondary)">
              {{ stack.app.short_description }}
            </p>

            <!-- Tags -->
            <div v-if="stack.app?.tags?.length" class="flex flex-wrap gap-1.5 mb-3">
              <span
                v-for="tag in (stack.app.tags).slice(0, 6)"
                :key="tag"
                class="text-[10px] font-bold uppercase tracking-widest px-2 py-0.5 rounded-md border border-gray-200 dark:border-zinc-800 hover:border-gray-400 dark:hover:border-zinc-600 hover:scale-105 transition-all cursor-default"
                style="color: var(--text-secondary)"
              >{{ tag }}</span>
            </div>

            <!-- Actions -->
            <div class="flex flex-wrap items-center gap-2">
              <span class="flex-1"></span>

              <!-- Custom app badge -->
              <div
                v-if="stack.app?.customapp"
                class="inline-flex items-center gap-1.5 text-xs font-bold px-3 py-2 rounded-lg bg-purple-50 dark:bg-purple-500/10 text-purple-600 dark:text-purple-400 border border-purple-200 dark:border-purple-500/20"
              >
                <Bot :size="13" />{{ t("stackView.builtByYantr") }}
              </div>

              <button
                v-if="stack.app"
                @click="router.push(`/apps/${stack.appname || stack.appId}`)"
                class="inline-flex items-center gap-1.5 text-xs font-bold px-3 py-2 rounded-lg border border-gray-200 dark:border-zinc-800 hover:border-gray-300 dark:hover:border-zinc-700 hover:scale-[1.03] active:scale-95 transition-all"
                style="color: var(--text-secondary)"
              >
                <ExternalLink :size="13" />{{ t("stackView.appPage") }}
              </button>

              <!-- Update -->
              <button
                v-if="!stack.app?.customapp"
                @click="updateStack"
                :disabled="updating"
                class="inline-flex items-center gap-1.5 text-xs font-bold px-3 py-2 rounded-lg bg-gray-900 dark:bg-zinc-100 text-white dark:text-gray-900 hover:opacity-90 hover:scale-[1.03] active:scale-95 transition-all disabled:opacity-50 disabled:cursor-not-allowed disabled:scale-100"
              >
                <RotateCcw :size="13" :class="updating ? 'animate-spin' : ''" />
                {{ updating ? t("stackView.updating") : t("stackView.updateStack") }}
              </button>

              <!-- Remove -->
              <button
                @click="removeStack"
                :disabled="removing"
                class="inline-flex items-center gap-1.5 text-xs font-bold px-3 py-2 rounded-lg border bg-red-50 dark:bg-red-500/10 text-red-600 dark:text-red-500 border-red-200 dark:border-red-500/20 hover:bg-red-100 dark:hover:bg-red-500/20 hover:scale-[1.03] active:scale-95 transition-all disabled:opacity-50 disabled:cursor-not-allowed disabled:scale-100"
              >
                <Trash2 :size="13" />{{ removing ? t("stackView.removing") : t("stackView.removeStack") }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- ── Ports Overview ─────────────────────────────────────────────────────────── -->
      <div class="space-y-4 animate-fadeIn">
        <div class="flex items-center justify-between gap-3 mb-1">
          <div class="flex items-center gap-2 min-w-0">
            <div class="text-xs font-bold uppercase tracking-widest" style="color: var(--text-secondary)">
              {{ t("stackView.networkAccess") }}
            </div>
            <span
              v-if="enrichedPorts.length > 0"
              class="inline-flex items-center rounded-full px-2.5 py-1 text-[10px] font-bold uppercase tracking-[0.2em] border border-gray-200 dark:border-zinc-800"
              style="background: var(--surface-muted); color: var(--text-secondary)"
            >
              {{ visiblePorts.length }}
            </span>
          </div>
          <div v-if="hasDescribedPorts" class="flex gap-0.5 p-0.5 rounded-lg" style="background: var(--surface-muted)">
            <button @click="showOnlyDescribedPorts = false" :class="!showOnlyDescribedPorts ? 'bg-white dark:bg-zinc-800 smooth-shadow' : 'hover:bg-white/60 dark:hover:bg-zinc-800/60'" :style="!showOnlyDescribedPorts ? 'color: var(--text-primary)' : 'color: var(--text-secondary)'" class="px-3 py-1.5 text-xs font-bold uppercase tracking-wider rounded-md transition-all hover:scale-[1.02]">{{ t("stackView.allPorts") }}</button>
            <button @click="showOnlyDescribedPorts = true" :class="showOnlyDescribedPorts ? 'bg-white dark:bg-zinc-800 smooth-shadow' : 'hover:bg-white/60 dark:hover:bg-zinc-800/60'" :style="showOnlyDescribedPorts ? 'color: var(--text-primary)' : 'color: var(--text-secondary)'" class="px-3 py-1.5 text-xs font-bold uppercase tracking-wider rounded-md transition-all hover:scale-[1.02]">{{ t("stackView.described") }}</button>
          </div>
        </div>

        <div v-if="enrichedPorts.length > 0" class="mt-2 grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          <div
            v-for="(p, i) in visiblePorts"
            :key="i"
            class="group flex h-full flex-col rounded-2xl p-4 transition-all duration-300 hover:-translate-y-0.5 hover:smooth-shadow-lg"
            style="background: var(--surface)"
          >
            <div class="flex h-full flex-col gap-4">
              <div class="flex items-start gap-3 min-w-0">
                <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl transition-all duration-300 group-hover:scale-105" style="background: var(--surface-muted)">
                  <Globe v-if="p.labeledProtocol === 'http' || p.labeledProtocol === 'https'" :size="16" class="text-blue-600 dark:text-blue-400" />
                  <Network v-else :size="16" class="text-gray-500 dark:text-zinc-400" />
                </div>

                <div class="min-w-0 flex-1">
                  <div class="flex flex-wrap items-center gap-2">
                    <div class="text-sm font-semibold truncate" style="color: var(--text-primary)">
                      {{ p.label || p.service }}
                    </div>
                    <span class="inline-flex items-center rounded-full px-2 py-1 text-[10px] font-bold uppercase tracking-[0.18em] border border-gray-200 dark:border-zinc-800" style="background: var(--surface-muted); color: var(--text-secondary)">
                      {{ p.protocol }}
                    </span>
                    <span v-if="p.labeledProtocol" class="inline-flex items-center rounded-full px-2 py-1 text-[10px] font-bold uppercase tracking-[0.18em] border border-blue-200 dark:border-blue-500/20 text-blue-700 dark:text-blue-400 bg-blue-50 dark:bg-blue-500/10">
                      {{ p.labeledProtocol }}
                    </span>
                  </div>

                  <div class="mt-1 text-[11px] uppercase tracking-[0.2em] truncate" style="color: var(--text-secondary)">
                    {{ p.service }}
                  </div>
                </div>
              </div>

              <div class="rounded-xl px-3.5 py-3" style="background: var(--surface-muted)">
                <div class="text-[10px] font-bold uppercase tracking-[0.18em]" style="color: var(--text-secondary)">
                  Port Route
                </div>
                <div class="mt-2 flex flex-wrap items-center gap-2">
                  <span class="inline-flex items-center rounded-full px-3 py-1.5 font-mono font-bold text-xs bg-green-50 dark:bg-green-500/10 text-green-700 dark:text-green-400 border border-green-200 dark:border-green-500/20">
                    :{{ p.hostPort }}
                  </span>
                  <ChevronRight :size="14" class="text-gray-300 dark:text-zinc-600" />
                  <span class="inline-flex items-center rounded-full px-3 py-1.5 font-mono font-bold text-xs border border-gray-200 dark:border-zinc-800" style="color: var(--text-primary)">
                    {{ p.containerPort }}
                  </span>
                </div>
              </div>

              <div class="mt-auto flex">
                <a
                  v-if="p.protocol === 'tcp'"
                  :href="appUrl(p.hostPort, p.labeledProtocol || 'http')"
                  target="_blank"
                  class="inline-flex min-h-11 w-full items-center justify-center gap-2 rounded-xl px-3.5 py-2.5 text-xs font-bold uppercase tracking-wider bg-gray-900 text-white transition-all duration-300 hover:opacity-90 hover:scale-[1.01] active:scale-95 dark:bg-zinc-100 dark:text-gray-900"
                >
                  <ExternalLink :size="13" class="transition-transform duration-300 group-hover:translate-x-0.5" />{{ t("stackView.open") }}
                </a>
                <span
                  v-else
                  class="inline-flex min-h-11 w-full items-center justify-center rounded-xl px-3.5 py-2.5 text-[10px] font-bold uppercase tracking-wider border border-gray-200 dark:border-zinc-800"
                  style="color: var(--text-secondary)"
                >
                  {{ p.protocol.toUpperCase() }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <div v-else class="rounded-2xl p-10 flex flex-col items-center justify-center gap-3 border-2 border-dashed border-gray-200 dark:border-zinc-800">
          <Network :size="28" class="text-gray-300 dark:text-zinc-700" />
          <span class="text-xs font-bold uppercase tracking-widest" style="color: var(--text-secondary)">{{ t("stackView.noPortsPublished") }}</span>
        </div>
      </div>

      <!-- ── Section Navigation ───────────────────────────────────────────────────────── -->
      <div class="grid gap-2 rounded-2xl p-2" :class="sectionTabsGridClass" style="background: var(--surface-muted)">
        <button
          v-for="sec in sectionTabs"
          :key="sec.id"
          @click="activeSection = sec.id"
          class="group flex w-full min-h-13 items-center gap-3 rounded-xl px-3.5 py-3 text-left transition-all duration-300"
          :class="activeSection === sec.id
            ? 'bg-white dark:bg-zinc-800 smooth-shadow'
            : 'hover:bg-white/60 dark:hover:bg-zinc-800/60'"
          :style="activeSection === sec.id ? 'color: var(--text-primary)' : 'color: var(--text-secondary)'"
        >
          <div
            class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl transition-colors duration-300"
            :style="activeSection === sec.id ? 'background: var(--surface-muted)' : 'background: transparent'"
          >
            <component :is="sec.icon" :size="16" :class="sec.tone" />
          </div>

          <div class="min-w-0 flex-1">
            <div class="text-sm font-semibold leading-tight">
              {{ sec.label }}
            </div>
          </div>
        </button>
      </div>

      <!-- CONTAINERS SECTION -->
      <div v-show="activeSection === 'containers'" class="space-y-3 animate-fadeIn">
        <div class="flex items-center justify-between mb-1">
          <div class="text-xs font-bold uppercase tracking-widest" style="color: var(--text-secondary)">
            {{ t("stackView.containers") }}
          </div>
          <span class="text-xs font-mono font-bold px-2.5 py-1 rounded-lg" style="background: var(--surface-muted); color: var(--text-secondary)">
            {{ stack.services.length }}
          </span>
        </div>

        <div class="grid gap-3">
          <div
            v-for="(svc, i) in stack.services"
            :key="svc.id"
            @click="router.push(`/containers/${svc.id}`)"
            class="group cursor-pointer rounded-2xl p-5 border border-gray-100 dark:border-zinc-800 smooth-shadow hover:smooth-shadow-lg hover:-translate-y-1 hover:border-gray-200 dark:hover:border-zinc-700 transition-all duration-300"
            style="background: var(--surface)"
          >
            <div class="flex items-start gap-4">
              <div
                class="w-11 h-11 rounded-xl flex items-center justify-center shrink-0 relative border transition-all group-hover:scale-110"
                :class="svc.state === 'running'
                  ? 'bg-green-50 dark:bg-green-500/10 border-green-200 dark:border-green-500/20'
                  : 'bg-gray-50 dark:bg-zinc-900 border-gray-200 dark:border-zinc-800'"
              >
                <Server :size="18" :class="svc.state === 'running' ? 'text-green-600 dark:text-green-400' : 'text-gray-400 dark:text-zinc-600'" />
                <span v-if="svc.state === 'running'" class="absolute -top-1 -right-1 flex h-3 w-3">
                  <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                  <span class="relative inline-flex rounded-full h-3 w-3 bg-green-500"></span>
                </span>
              </div>
              <div class="flex-1 min-w-0">
                <div class="flex items-center flex-wrap gap-2 mb-1">
                  <span class="font-bold text-base group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors" style="color: var(--text-primary)">{{ svc.service }}</span>
                  <span
                    class="text-[10px] font-bold uppercase tracking-widest px-2 py-0.5 rounded-md border"
                    :class="svc.state === 'running'
                      ? 'bg-green-50 dark:bg-green-500/10 text-green-700 dark:text-green-400 border-green-200 dark:border-green-500/20'
                      : 'bg-gray-100 dark:bg-zinc-800 text-gray-500 dark:text-zinc-400 border-gray-200 dark:border-zinc-700'"
                  >{{ svc.state }}</span>
                  <span v-if="svc.hasYantrLabel" class="text-[10px] font-bold uppercase tracking-widest px-2 py-0.5 rounded-md border bg-blue-50 dark:bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-200 dark:border-blue-500/20">{{ t("stackView.primary") }}</span>
                </div>
                <div class="font-mono text-xs truncate" style="color: var(--text-secondary)" :title="svc.image">{{ svc.image }}</div>
              </div>
              <div class="flex items-center gap-3 shrink-0">
                <div v-if="formatUptime(svc)" class="text-right hidden sm:block">
                  <div class="text-[10px] font-bold uppercase tracking-widest mb-0.5" style="color: var(--text-secondary)">{{ t("stackView.uptime") }}</div>
                  <div class="font-mono text-xs font-bold tabular-nums" style="color: var(--text-primary)">{{ formatUptime(svc) }}</div>
                </div>
                <div class="w-8 h-8 rounded-full flex items-center justify-center border border-gray-100 dark:border-zinc-800 group-hover:bg-blue-50 dark:group-hover:bg-blue-900/30 group-hover:border-blue-200 transition-all" style="background: var(--surface-muted)">
                  <ChevronRight :size="15" class="text-gray-400 dark:text-zinc-500 group-hover:text-blue-500 group-hover:translate-x-0.5 transition-all" />
                </div>
              </div>
            </div>
            <div v-if="svc.rawPorts.length > 0" class="flex items-center gap-2 flex-wrap mt-3 pt-3 border-t border-gray-100 dark:border-zinc-800">
              <template v-if="svc.rawPorts.filter((p) => p.PublicPort).length > 0">
                <span
                  v-for="p in [...new Map(svc.rawPorts.filter((rp) => rp.PublicPort).map((rp) => [`${rp.PublicPort}:${rp.PrivatePort}:${rp.Type}`, rp])).values()]"
                  :key="`${p.PublicPort}-${p.Type}`"
                  class="inline-flex items-center gap-1.5 text-[10px] font-mono font-bold px-2.5 py-1 rounded-lg bg-green-50 dark:bg-green-500/10 text-green-700 dark:text-green-400 border border-green-200 dark:border-green-500/20 hover:scale-105 transition-transform"
                >
                  <Network :size="11" />:{{ p.PublicPort }} → {{ p.PrivatePort }}
                </span>
              </template>
              <template v-else>
                <span
                  v-for="p in [...new Map(svc.rawPorts.map((rp) => [`${rp.PrivatePort}:${rp.Type}`, rp])).values()]"
                  :key="`internal-${p.PrivatePort}-${p.Type}`"
                  class="inline-flex items-center gap-1.5 text-[10px] font-mono font-bold px-2.5 py-1 rounded-lg border border-gray-200 dark:border-zinc-800 hover:scale-105 transition-transform"
                  style="color: var(--text-secondary)"
                >
                  <Network :size="11" />{{ p.PrivatePort }}/{{ p.Type }}
                </span>
              </template>
            </div>
          </div>
        </div>
      </div>

      <!-- OPEN PORT SECTION -->
      <div v-show="activeSection === 'ports'" class="space-y-4 animate-fadeIn">
        <div class="rounded-2xl p-6 smooth-shadow" style="background: var(--surface)">
          <div class="flex items-center gap-3 mb-5">
            <div class="w-10 h-10 rounded-xl flex items-center justify-center border border-gray-100 dark:border-zinc-800 shrink-0" style="background: var(--surface-muted)">
              <Plus :size="18" class="text-gray-500 dark:text-zinc-400" />
            </div>
            <div>
              <div class="text-sm font-bold" style="color: var(--text-primary)">{{ t("stackView.openAnotherPort") }}</div>
              <p class="text-xs mt-0.5" style="color: var(--text-secondary)">{{ t("stackView.openAnotherPortHint") }}</p>
            </div>
          </div>
          <div :class="needsPortServiceSelection ? 'grid sm:grid-cols-2 gap-4 mb-5' : 'grid gap-4 mb-5'">
            <label v-if="needsPortServiceSelection" class="space-y-2">
              <span class="text-xs font-bold uppercase tracking-wider" style="color: var(--text-secondary)">{{ t("stackView.service") }}</span>
              <select v-model="newPort.serviceName" class="w-full px-4 py-3 rounded-xl border border-gray-200 dark:border-zinc-800 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/40 transition-shadow hover:border-blue-300" style="background: var(--surface-muted)">
                <option v-for="service in portServices" :key="service.value" :value="service.value">{{ service.label }}</option>
              </select>
            </label>
            <label class="space-y-2">
              <span class="text-xs font-bold uppercase tracking-wider" style="color: var(--text-secondary)">{{ t("stackView.portMappingLabel") }}</span>
              <input v-model="newPort.mapping" placeholder="9000:9000 or 9000 or 53:53/udp" class="w-full px-4 py-3 rounded-xl border border-gray-200 dark:border-zinc-800 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500/40 font-mono transition-shadow hover:border-blue-300" style="background: var(--surface-muted)" />
              <p class="text-xs" style="color: var(--text-secondary)">{{ t("stackView.portMappingHint") }}</p>
            </label>
          </div>
          <div class="flex justify-end">
            <button @click="openPort" :disabled="openingPort || portServices.length === 0" class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl text-xs font-bold uppercase tracking-wider bg-blue-50 dark:bg-blue-500/10 text-blue-700 dark:text-blue-400 border border-blue-200 dark:border-blue-500/20 hover:bg-blue-100 dark:hover:bg-blue-500/20 hover:scale-[1.03] active:scale-95 transition-all disabled:opacity-50 disabled:cursor-not-allowed">
              <Plus :size="14" />{{ openingPort ? t("stackView.openingPortAction") : t("stackView.openPortAction") }}
            </button>
          </div>
        </div>
      </div>

      <!-- AUTH SECTION -->
      <div v-show="activeSection === 'auth' && stack.appId !== 'caddy-yantr'" class="space-y-4 animate-fadeIn">
        <div v-if="caddyProxies.length > 0" class="space-y-2">
          <div class="text-xs font-bold uppercase tracking-wider mb-1" style="color: var(--text-secondary)">{{ t("stackView.caddyProxiesRunning") }}</div>
          <div v-for="proxy in caddyProxies" :key="proxy.servePort" class="flex items-center justify-between gap-4 rounded-2xl px-5 py-4 bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-200 dark:border-emerald-500/20 hover:-translate-y-0.5 hover:shadow-md transition-all">
            <div class="flex items-center gap-3 min-w-0">
              <div class="w-2.5 h-2.5 rounded-full bg-emerald-500 shrink-0 animate-pulse"></div>
              <span class="text-sm font-bold text-emerald-700 dark:text-emerald-400 font-mono">:{{ proxy.servePort }} → localhost:{{ proxy.targetPort }}</span>
              <span v-if="proxy.authEnabled" class="text-xs font-bold uppercase tracking-wider px-2.5 py-1 rounded-lg bg-purple-100 dark:bg-purple-500/20 text-purple-600 dark:text-purple-400 border border-purple-200 dark:border-purple-500/20">{{ proxy.authUser }}</span>
            </div>
            <button @click="disableCaddyAuth" :disabled="disablingCaddy" class="text-xs font-bold uppercase tracking-wider px-3 py-1.5 rounded-lg border border-red-200 dark:border-red-500/20 bg-red-50 dark:bg-red-500/10 text-red-600 dark:text-red-400 hover:bg-red-100 dark:hover:bg-red-500/20 hover:scale-[1.03] active:scale-95 transition-all disabled:opacity-50">{{ disablingCaddy ? "..." : t("stackView.caddyAuthDisable") }}</button>
          </div>
        </div>
        <div class="rounded-2xl p-6 smooth-shadow" style="background: var(--surface)">
          <div class="flex items-center gap-3 mb-5">
            <div class="w-10 h-10 rounded-xl flex items-center justify-center border border-purple-200 dark:border-purple-500/20 bg-purple-50 dark:bg-purple-500/10 shrink-0">
              <ShieldCheck :size="18" class="text-purple-500" />
            </div>
            <div>
              <div class="text-sm font-bold" style="color: var(--text-primary)">{{ t("stackView.caddyAuthHeading") }}</div>
              <p class="text-xs mt-0.5" style="color: var(--text-secondary)">{{ t("stackView.caddyAuthHint") }}</p>
            </div>
          </div>
          <div class="grid sm:grid-cols-2 gap-4 mb-5">
            <label class="space-y-2">
              <span class="text-xs font-bold uppercase tracking-wider" style="color: var(--text-secondary)">{{ t("stackView.caddyAuthTargetPort") }}</span>
              <input v-model="caddyAuth.targetPort" type="number" min="1" max="65535" placeholder="e.g. 8096" class="w-full px-4 py-3 rounded-xl border border-gray-200 dark:border-zinc-800 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500/40 font-mono transition-shadow hover:border-purple-300" style="background: var(--surface-muted)" />
              <p class="text-xs" style="color: var(--text-secondary)">{{ t("stackView.caddyAuthTargetHint") }}</p>
            </label>
            <label class="space-y-2">
              <span class="text-xs font-bold uppercase tracking-wider" style="color: var(--text-secondary)">{{ t("stackView.caddyAuthServePort") }}</span>
              <input v-model="caddyAuth.servePort" type="number" min="1024" max="65535" placeholder="e.g. 9096" class="w-full px-4 py-3 rounded-xl border border-gray-200 dark:border-zinc-800 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500/40 font-mono transition-shadow hover:border-purple-300" style="background: var(--surface-muted)" />
              <p class="text-xs" style="color: var(--text-secondary)">{{ t("stackView.caddyAuthServeHint") }}</p>
            </label>
            <label class="space-y-2">
              <span class="text-xs font-bold uppercase tracking-wider" style="color: var(--text-secondary)">{{ t("stackView.caddyAuthUser") }}</span>
              <input v-model="caddyAuth.user" type="text" placeholder="admin" class="w-full px-4 py-3 rounded-xl border border-gray-200 dark:border-zinc-800 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500/40 font-mono transition-shadow hover:border-purple-300" style="background: var(--surface-muted)" />
            </label>
            <label class="space-y-2">
              <span class="text-xs font-bold uppercase tracking-wider" style="color: var(--text-secondary)">{{ t("stackView.caddyAuthPass") }}</span>
              <input v-model="caddyAuth.pass" type="password" placeholder="••••••••" class="w-full px-4 py-3 rounded-xl border border-gray-200 dark:border-zinc-800 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-purple-500/40 font-mono transition-shadow hover:border-purple-300" style="background: var(--surface-muted)" />
              <p class="text-xs" style="color: var(--text-secondary)">{{ t("stackView.caddyAuthPassHint") }}</p>
            </label>
          </div>
          <div class="flex justify-end">
            <button @click="deployCaddyAuth" :disabled="deployingCaddy || !caddyAuth.servePort || !caddyAuth.targetPort" class="inline-flex items-center gap-2 px-5 py-2.5 rounded-xl text-xs font-bold uppercase tracking-wider bg-purple-50 dark:bg-purple-500/10 text-purple-700 dark:text-purple-400 border border-purple-200 dark:border-purple-500/20 hover:bg-purple-100 dark:hover:bg-purple-500/20 hover:scale-[1.03] active:scale-95 transition-all disabled:opacity-50 disabled:cursor-not-allowed">
              <ShieldCheck :size="14" :class="deployingCaddy ? 'animate-ping' : ''" />{{ deployingCaddy ? t("stackView.caddyAuthDeploying") : t("stackView.caddyAuthDeploy") }}
            </button>
          </div>
        </div>
      </div>

      <!-- STORAGE SECTION -->
      <div v-show="activeSection === 'storage'" class="space-y-4 animate-fadeIn">
        <div v-if="namedVolumes.length > 0" class="space-y-3">
          <div class="text-xs font-bold uppercase tracking-widest" style="color: var(--text-secondary)">{{ t("stackView.storageVolumes") }}</div>
          <div v-for="(vol, i) in namedVolumes" :key="vol.name" class="rounded-2xl p-5 border border-gray-100 dark:border-zinc-800 smooth-shadow hover:smooth-shadow-lg hover:-translate-y-0.5 transition-all group" style="background: var(--surface)">
            <div class="flex items-start gap-4 mb-4">
              <div class="w-11 h-11 rounded-xl flex items-center justify-center border border-gray-100 dark:border-zinc-800 shrink-0 group-hover:scale-110 group-hover:rotate-6 transition-all" style="background: var(--surface-muted)">
                <HardDrive :size="18" class="text-gray-500 dark:text-zinc-400 group-hover:text-blue-500 transition-colors" />
              </div>
              <div class="flex-1 min-w-0">
                <div class="font-bold text-sm truncate group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors" style="color: var(--text-primary)" :title="vol.name">{{ vol.name }}</div>
                <div class="font-mono text-xs truncate mt-0.5" style="color: var(--text-secondary)">{{ vol.destination }}</div>
                <div class="flex items-center gap-3 mt-2 text-xs flex-wrap" style="color: var(--text-secondary)">
                  <span>{{ t("stackView.serviceLabel") }} <span class="font-bold px-1.5 py-0.5 rounded-md ml-1 bg-gray-100 dark:bg-zinc-800" style="color: var(--text-primary)">{{ vol.svcName }}</span></span>
                </div>
              </div>
            </div>
            <div class="flex items-center gap-2 flex-wrap pt-3 border-t border-gray-100 dark:border-zinc-800">
              <div v-if="browsingVolume[vol.name]" class="text-xs font-bold uppercase tracking-wider text-blue-600 dark:text-blue-400 animate-pulse px-3 py-2 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800/50 rounded-lg">{{ t("stackView.startingWebDAV") }}</div>
              <template v-else-if="!showVolumeMenu[vol.name]">
                <button @click="showVolumeMenu[vol.name] = true" class="flex items-center gap-2 px-3 py-2 text-xs font-bold uppercase tracking-wider border border-gray-200 dark:border-zinc-800 rounded-xl hover:border-gray-300 dark:hover:border-zinc-700 hover:scale-[1.03] active:scale-95 transition-all" style="background: var(--surface-muted); color: var(--text-secondary)">
                  <FolderOpen :size="13" />{{ t("stackView.browseFiles") }}
                </button>
              </template>
              <template v-else>
                <button @click="browseVolume(vol.name, 60)" class="px-3 py-2 text-xs font-bold uppercase tracking-wider bg-gray-900 dark:bg-zinc-100 text-white dark:text-black rounded-xl hover:opacity-90 hover:scale-[1.03] active:scale-95 transition-all" :title="t('stackView.oneHourAccess')">1H</button>
                <button @click="browseVolume(vol.name, 0)" class="px-3 py-2 text-xs font-bold uppercase tracking-wider rounded-xl border border-gray-200 dark:border-zinc-800 hover:border-gray-300 dark:hover:border-zinc-700 hover:scale-[1.03] active:scale-95 transition-all" style="background: var(--surface-muted); color: var(--text-secondary)" :title="t('stackView.permanentAccess')">Perm</button>
              </template>
            </div>
          </div>
        </div>
        <div v-if="otherMounts.length > 0" class="space-y-3 mt-2">
          <div class="text-xs font-bold uppercase tracking-widest" style="color: var(--text-secondary)">{{ t("stackView.bindMounts") }}</div>
          <div class="rounded-2xl overflow-hidden border border-gray-100 dark:border-zinc-800 smooth-shadow" style="background: var(--surface)">
            <table class="w-full text-left min-w-80">
              <thead>
                <tr class="border-b border-gray-100 dark:border-zinc-800" style="background: var(--surface-muted)">
                  <th class="px-5 py-3 text-[10px] font-bold uppercase tracking-widest" style="color: var(--text-secondary)">{{ t("stackView.type") }}</th>
                  <th class="px-5 py-3 text-[10px] font-bold uppercase tracking-widest" style="color: var(--text-secondary)">{{ t("stackView.hostPath") }}</th>
                  <th class="px-5 py-3 text-[10px] font-bold uppercase tracking-widest" style="color: var(--text-secondary)">{{ t("stackView.containerPath") }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-zinc-800">
                <tr v-for="(m, i) in otherMounts" :key="i" class="hover:bg-gray-50 dark:hover:bg-zinc-900/50 transition-colors">
                  <td class="px-5 py-3.5">
                    <span class="text-[10px] font-bold uppercase tracking-widest px-2.5 py-1 rounded-md border" :class="m.type === 'bind' ? 'bg-amber-50 dark:bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-200 dark:border-amber-500/20' : 'bg-gray-100 dark:bg-zinc-800 text-gray-500 dark:text-zinc-400 border-gray-200 dark:border-zinc-700'">{{ m.type }}</span>
                  </td>
                  <td class="px-5 py-3.5 font-mono text-xs break-all max-w-xs" style="color: var(--text-primary)">{{ m.source || "—" }}</td>
                  <td class="px-5 py-3.5 font-mono text-xs break-all max-w-xs" style="color: var(--text-secondary)">{{ m.destination }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

    </div>
  </div>
</template>