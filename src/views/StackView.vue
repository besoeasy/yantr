<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useApiUrl } from "../composables/useApiUrl";
import { useCurrentTime } from "../composables/useCurrentTime";
import { useNotification } from "../composables/useNotification";
import { formatDuration, formatBytes } from "../utils/metrics";
import {
  ArrowLeft,
  Globe,
  ExternalLink,
  Bot,
  Activity,
  Terminal,
  Server,
  Network,
  Trash2,
  RefreshCw,
  HardDrive,
  FolderOpen,
  AlertCircle,
  Eye,
  EyeOff,
  Settings2,
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

// Env vars reveal state
const revealedVars = ref(new Set());

function isSensitive(key) {
  const k = key.toLowerCase();
  return (
    k.includes("password") ||
    k.includes("secret") ||
    k.includes("token") ||
    k.includes("_key") ||
    k.endsWith("key") ||
    k.includes("passwd") ||
    k.includes("_pass") ||
    k.endsWith("pass") ||
    k.includes("auth") ||
    k.includes("credential") ||
    k.includes("private")
  );
}

function toggleReveal(key) {
  const s = new Set(revealedVars.value);
  s.has(key) ? s.delete(key) : s.add(key);
  revealedVars.value = s;
}

// Aggregate unique env vars across all services (primary service wins on conflict)
const stackEnvVars = computed(() => {
  if (!stack.value) return [];
  const map = new Map();
  // Process primary service first so its values take precedence
  const sorted = [...stack.value.services].sort((a, b) => (b.hasYantrLabel ? 1 : 0) - (a.hasYantrLabel ? 1 : 0));
  for (const svc of sorted) {
    for (const v of svc.env || []) {
      if (!map.has(v.key)) map.set(v.key, { ...v, service: svc.service });
    }
  }
  return [...map.values()].sort((a, b) => a.key.localeCompare(b.key));
});

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

// Browse / Backup state
const browsingVolume = ref({});
const showVolumeMenu = ref({});
const s3Configured = ref(false);
const volumeBackups = ref({});
const backingUp = ref(false);
const showRestoreMenu = ref({});

// Tabs for port options
const activePortTab = ref("list"); // 'list' | 'add' | 'caddy'

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

// Collect all unique mounts across all services (includes svcId for backup ops)
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

// Named Docker volumes only — these support browse / backup / restore
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

// ── Browse / Backup / Restore ────────────────────────────────────────────────

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

async function checkS3Config() {
  try {
    const res = await fetch(`${apiUrl.value}/api/backup/config`);
    const data = await res.json();
    s3Configured.value = data.configured;
  } catch (e) {
    /* silent */
  }
}

async function fetchVolumeBackups() {
  if (!stack.value) return;
  // Collect unique svcIds that have named volumes
  const svcIds = [...new Set(namedVolumes.value.map((m) => m.svcId))];
  const merged = {};
  await Promise.all(
    svcIds.map(async (svcId) => {
      try {
        const res = await fetch(`${apiUrl.value}/api/containers/${svcId}/backups`);
        const data = await res.json();
        if (data.success && data.backups) {
          Object.assign(merged, data.backups);
          if (data.configured !== false) s3Configured.value = true;
        }
      } catch (e) {
        /* silent */
      }
    }),
  );
  volumeBackups.value = merged;
}

async function backupVolume(svcId) {
  if (backingUp.value) return;
  backingUp.value = true;
  try {
    const res = await fetch(`${apiUrl.value}/api/containers/${svcId}/backup`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
    });
    const data = await res.json();
    if (data.success) {
      toast.success(t("stackView.backupStarted"));
      pollBackupJob(data.jobId);
    } else {
      toast.error(data.error || t("stackView.failedToStartBackup"));
    }
  } catch (e) {
    toast.error(t("stackView.failedToStartBackup"));
  } finally {
    backingUp.value = false;
  }
}

async function backupAll() {
  if (backingUp.value) return;
  const svcIds = [...new Set(namedVolumes.value.map((m) => m.svcId))];
  for (const svcId of svcIds) await backupVolume(svcId);
}

function pollBackupJob(jobId) {
  const iv = setInterval(async () => {
    try {
      const res = await fetch(`${apiUrl.value}/api/backup/jobs/${jobId}`);
      const data = await res.json();
      if (data.success && data.job) {
        if (data.job.status === "completed") {
          clearInterval(iv);
          toast.success(t("stackView.backupCompleted"));
          await fetchVolumeBackups();
        } else if (data.job.status === "failed") {
          clearInterval(iv);
          toast.error(t("stackView.backupFailed", { error: data.job.error }));
        }
      }
    } catch (e) {
      clearInterval(iv);
    }
  }, 2000);
}

async function restoreBackup(volumeName, snapshotId) {
  if (!confirm(t("stackView.restoreConfirm", { volume }))) return;
  try {
    const res = await fetch(`${apiUrl.value}/api/volumes/${volumeName}/restore`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ snapshotId, overwrite: true }),
    });
    const data = await res.json();
    if (data.success) {
      toast.success(t("stackView.restoreStarted"));
      pollRestoreJob(data.jobId);
    } else {
      toast.error(data.error || t("stackView.failedToStartRestore"));
    }
  } catch (e) {
    toast.error(t("stackView.failedToStartRestore"));
  }
  showRestoreMenu.value[volumeName] = false;
}

function pollRestoreJob(jobId) {
  const iv = setInterval(async () => {
    try {
      const res = await fetch(`${apiUrl.value}/api/restore/jobs/${jobId}`);
      const data = await res.json();
      if (data.success && data.job) {
        if (data.job.status === "completed") {
          clearInterval(iv);
          toast.success(t("stackView.restoreCompleted"));
        } else if (data.job.status === "failed") {
          clearInterval(iv);
          toast.error(t("stackView.restoreFailed", { error: data.job.error }));
        }
      }
    } catch (e) {
      clearInterval(iv);
    }
  }, 2000);
}

async function deleteBackupFile(volumeName, snapshotId) {
  if (!confirm(t("stackView.deleteBackupConfirm"))) return;
  try {
    const res = await fetch(`${apiUrl.value}/api/volumes/${volumeName}/backup/${snapshotId}`, { method: "DELETE" });
    const data = await res.json();
    if (data.success) {
      toast.success(t("stackView.backupDeleted"));
      await fetchVolumeBackups();
    } else toast.error(data.error || t("stackView.failedToDeleteBackup"));
  } catch (e) {
    toast.error(t("stackView.failedToDeleteBackup"));
  }
}

function toggleRestoreMenu(volumeName) {
  showRestoreMenu.value[volumeName] = !showRestoreMenu.value[volumeName];
}

function hasBackups(volumeName) {
  return volumeBackups.value[volumeName]?.length > 0;
}

function getLatestBackupAge(volumeName) {
  const backups = volumeBackups.value[volumeName];
  if (!backups?.length) return "Never";
  const diffMs = Date.now() - new Date(backups[0].timestamp);
  const m = Math.floor(diffMs / 60000);
  const h = Math.floor(m / 60);
  const d = Math.floor(h / 24);
  if (d > 0) return `${d}d ago`;
  if (h > 0) return `${h}h ago`;
  if (m > 0) return `${m}m ago`;
  return "Just now";
}

function formatBackupDate(ts) {
  return new Date(ts).toLocaleString();
}

onMounted(async () => {
  await fetchStack();
  await Promise.all([checkS3Config(), fetchVolumeBackups()]);
  refreshInterval = setInterval(fetchStack, 8000);
});

onUnmounted(() => {
  if (refreshInterval) clearInterval(refreshInterval);
});
</script>

<template>
  <div class="min-h-screen bg-white dark:bg-[#0A0A0A] text-gray-900 dark:text-zinc-100 font-sans selection:bg-blue-500/30">
    <!-- Header bar -->
    <header class="bg-white/90 dark:bg-[#0A0A0A]/90 backdrop-blur-xl border-b border-gray-200 dark:border-zinc-800 sticky top-0 z-40 shadow-sm dark:shadow-none">
      <div class="max-w-6xl mx-auto px-6 sm:px-12 h-16 sm:h-20 flex items-center justify-between">
        <div class="flex items-center gap-3 sm:gap-5">
          <button
            @click="router.back()"
            class="inline-flex items-center justify-center w-10 h-10 rounded-xl bg-gray-50 hover:bg-gray-100 dark:bg-zinc-900 shadow-sm border border-gray-100 dark:border-zinc-800 dark:hover:bg-zinc-800 transition-all duration-300 text-gray-500 dark:text-zinc-400 group focus:outline-none focus:ring-2 focus:ring-blue-500/50"
          >
            <ArrowLeft :size="18" class="group-hover:-translate-x-1 transition-transform duration-300" />
          </button>
          <div class="h-6 w-px bg-gray-200 dark:bg-zinc-800"></div>
          <div class="flex flex-col min-w-0">
            <div class="flex items-center gap-2 text-[10px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500">
              <button @click="router.push('/')" class="hover:text-blue-500 transition-colors">
                {{ t("stackView.dashboard") }}
              </button>
              <span class="text-gray-300 dark:text-zinc-700">/</span>
              <span>Stack</span>
            </div>
            <div class="font-extrabold tracking-tight text-gray-900 dark:text-white text-base sm:text-lg truncate max-w-[200px] sm:max-w-md">
              {{ stack?.app?.name || projectId }}
            </div>
          </div>
        </div>

        <div class="flex items-center gap-3 sm:gap-4 shrink-0">
          <!-- State badge -->
          <div v-if="stack" class="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-lg text-[10px] font-bold uppercase tracking-widest border shadow-sm transition-colors duration-300" :class="stateClass">
            <span v-if="overallState === 'running'" class="w-2 h-2 rounded-full bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.5)] animate-pulse"></span>
            <span v-else-if="overallState === 'partial'" class="w-2 h-2 rounded-full bg-amber-500 shadow-[0_0_8px_rgba(245,158,11,0.5)] animate-pulse"></span>
            <span v-else class="w-2 h-2 rounded-full bg-gray-400 shadow-[0_0_8px_rgba(156,163,175,0.5)]"></span>
            {{ overallState }}
          </div>

          <!-- Refresh -->
          <button
            @click="fetchStack"
            class="inline-flex items-center justify-center px-3 py-2 h-9 rounded-lg bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 hover:bg-gray-50 dark:hover:bg-zinc-900 hover:border-gray-300 dark:hover:border-zinc-700 transition-all duration-300 text-gray-600 dark:text-zinc-300 text-xs font-bold uppercase tracking-wider gap-2 group shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500/50"
          >
            <RefreshCw :size="14" class="group-hover:rotate-180 transition-transform duration-500" />
            <span class="hidden sm:inline">{{ t('stackView.refresh') }}</span>
          </button>
        </div>
      </div>
    </header>

    <!-- Loading -->
    <div v-if="loading" class="flex flex-col items-center justify-center min-h-[60vh]">
      <div class="w-8 h-8 border-[3px] border-gray-200 dark:border-zinc-800 border-t-blue-500 dark:border-t-blue-500 rounded-full animate-spin"></div>
    </div>

    <!-- Content -->
    <div v-else-if="stack" class="max-w-6xl mx-auto px-6 sm:px-12 py-10 sm:py-16 space-y-12 animate-fadeIn">
      <!-- ── Page Header (Identity) ─────────────────────────────────────────────────── -->
      <div class="flex flex-col sm:flex-row sm:items-start gap-6 sm:gap-8 pb-8 border-b border-gray-100 dark:border-zinc-800/50">
        <!-- Logo -->
        <div
          class="w-24 h-24 sm:w-32 sm:h-32 shrink-0 rounded-[2rem] bg-gray-50/50 dark:bg-zinc-900/30 flex items-center justify-center p-4 transition-transform duration-700 hover:scale-105"
        >
          <img
            v-if="stack.app?.logo"
            :src="stack.app.logo"
            :alt="stack.app.name"
            loading="lazy"
            class="w-full h-full object-contain filter dark:brightness-90 hover:brightness-100 transition-all duration-500 drop-shadow-sm"
          />
          <Bot v-else :size="48" class="text-gray-400 dark:text-zinc-600 transition-colors duration-500 hover:text-blue-500" />
        </div>

        <!-- Info -->
        <div class="flex-1 min-w-0 flex flex-col justify-center space-y-4 pt-2">
          <div class="flex flex-col gap-2">
            <div class="flex items-center gap-3 flex-wrap">
              <h1 class="text-3xl sm:text-5xl font-black tracking-tight text-gray-900 dark:text-white">
                {{ stack.app?.name || stack.appId }}
              </h1>
              <!-- Project ID badge -->
              <span
                class="text-[10px] sm:text-xs font-bold uppercase tracking-widest px-3 py-1 sm:py-1.5 rounded-full bg-gray-100/80 dark:bg-zinc-900 text-gray-500 dark:text-zinc-400 mt-1 sm:mt-2"
              >
                {{ stack.projectId }}
              </span>
            </div>
            
            <p v-if="stack.app?.short_description" class="text-base sm:text-lg text-gray-500 dark:text-zinc-400 leading-relaxed max-w-3xl font-medium">
              {{ stack.app.short_description }}
            </p>
          </div>

          <div class="flex flex-wrap gap-2.5 pt-1">
            <span
              v-for="(tag, index) in (stack.app?.tags || []).slice(0, 6)"
              :key="tag"
              class="text-[10px] font-bold uppercase tracking-widest px-2.5 py-1 rounded-md bg-transparent text-gray-400 dark:text-zinc-500 border border-gray-200 dark:border-zinc-800 hover:text-gray-700 dark:hover:text-zinc-300 hover:border-gray-300 dark:hover:border-zinc-700 transition-all duration-300"
              >{{ tag }}</span
            >
          </div>
          
          <!-- Actions Row -->
          <div class="flex flex-wrap items-center gap-3 pt-4 sm:pt-6">
            <a
              v-if="stack.app?.website"
              :href="stack.app.website"
              target="_blank"
              class="inline-flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-gray-500 dark:text-zinc-400 hover:text-blue-600 dark:hover:text-blue-400 transition-colors py-2 pr-4"
            >
              <Globe :size="14" />
              {{ t("stackView.website") }}
            </a>
            
            <button
              v-if="stack.app"
              @click="router.push(`/apps/${stack.appname || stack.appId}`)"
              class="inline-flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-gray-500 dark:text-zinc-400 hover:text-blue-600 dark:hover:text-blue-400 transition-colors py-2 pr-4"
            >
              <ExternalLink :size="14" />
              {{ t("stackView.appPage") }}
            </button>
            
            <div class="w-px h-4 bg-gray-200 dark:bg-zinc-800 mx-2 hidden sm:block"></div>

            <div
              v-if="stack.app?.customapp"
              class="inline-flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-purple-600 dark:text-purple-400 py-2"
              :title="t('stackView.customAppNoUpdate')"
            >
              <Bot :size="14" />
              {{ t("stackView.builtByYantr") }}
            </div>
            <button
              v-else
              @click="updateStack"
              :disabled="updating"
              class="inline-flex items-center gap-2 px-4 py-2 rounded-lg text-xs font-bold uppercase tracking-wider bg-gray-900 text-white dark:bg-white dark:text-gray-900 hover:opacity-90 transition-opacity disabled:opacity-50 disabled:cursor-not-allowed group/update"
            >
              <RotateCcw :size="14" :class="updating ? 'animate-spin' : 'group-hover/update:-rotate-90 transition-transform duration-500'" />
              {{ updating ? t("stackView.updating") : t("stackView.updateStack") }}
            </button>
            
            <button
              @click="removeStack"
              :disabled="removing"
              class="inline-flex items-center gap-2 px-4 py-2 rounded-lg text-xs font-bold uppercase tracking-wider bg-red-50 text-red-600 dark:bg-red-500/10 dark:text-red-500 hover:bg-red-100 dark:hover:bg-red-500/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Trash2 :size="14" />
              {{ removing ? t("stackView.removing") : t("stackView.removeStack") }}
            </button>
          </div>
        </div>
      </div>

      <!-- ── Published Ports and Authentication ────────────────────────────────────────────────── -->
      <div class="space-y-6 animate-fadeIn">
        <div class="flex items-center justify-between pb-2">
          <h2
            class="text-sm font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 flex items-center gap-3 hover:text-gray-800 dark:hover:text-gray-200 transition-colors duration-300"
          >
            <Network :size="18" class="animate-pulse" />
            <span v-if="activePortTab === 'caddy'" class="flex items-center gap-2"
              ><ShieldCheck :size="16" class="text-purple-500" /> {{ t("stackView.caddyAuthTitle") }}</span
            >
            <span v-else>{{ t("stackView.networkAccess") }}</span>
          </h2>
          <!-- Tab navigation -->
          <div class="flex items-center gap-2 rounded-xl bg-gray-100 dark:bg-zinc-900 p-1.5 hover:shadow-inner transition-shadow duration-300">
            <button
              @click="activePortTab = 'list'"
              :class="
                activePortTab === 'list'
                  ? 'bg-white dark:bg-zinc-800 text-gray-900 dark:text-white shadow-lg scale-105'
                  : 'text-gray-500 hover:text-gray-700 dark:hover:text-zinc-300 hover:scale-105'
              "
              class="px-4 py-2 text-xs font-bold uppercase tracking-wider rounded-lg transition-all duration-300"
            >
              {{ t("stackView.ports") || "PORTS" }}
            </button>
            <button
              @click="activePortTab = 'add'"
              :class="
                activePortTab === 'add'
                  ? 'bg-white dark:bg-zinc-800 text-gray-900 dark:text-white shadow-lg scale-105'
                  : 'text-gray-500 hover:text-gray-700 dark:hover:text-zinc-300 hover:scale-105'
              "
              class="px-4 py-2 text-xs font-bold uppercase tracking-wider rounded-lg transition-all duration-300 flex items-center gap-1.5"
            >
              <Plus :size="12" />{{ t("stackView.openAnotherPort") || "OPEN PORT" }}
            </button>
            <button
              v-if="stack.appId !== 'caddy-yantr'"
              @click="activePortTab = 'caddy'"
              :class="
                activePortTab === 'caddy'
                  ? 'bg-white dark:bg-zinc-800 text-gray-900 dark:text-white shadow-lg scale-105'
                  : 'text-gray-500 hover:text-gray-700 dark:hover:text-zinc-300 hover:scale-105'
              "
              class="px-4 py-2 text-xs font-bold uppercase tracking-wider rounded-lg transition-all duration-300 flex items-center gap-1.5"
            >
              <ShieldCheck :size="12" />{{ t("stackView.auth") || "AUTH" }}
            </button>
          </div>
        </div>

        <div
          v-show="activePortTab === 'add'"
          class="bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-3xl p-8 sm:p-10 space-y-6 hover:shadow-lg hover:border-gray-300 dark:hover:border-zinc-700 transition-all duration-500 ease-out transform group"
        >
          <div class="flex items-start justify-between gap-6">
            <div>
              <div
                class="text-sm font-bold uppercase tracking-wider text-gray-700 dark:text-zinc-300 group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors"
              >
                {{ t("stackView.openAnotherPort") }}
              </div>
              <p class="mt-2 text-sm text-gray-500 dark:text-zinc-400">{{ t("stackView.openAnotherPortHint") }}</p>
            </div>
            <div
              class="hidden sm:flex items-center justify-center w-12 h-12 rounded-xl bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 text-gray-500 dark:text-zinc-400 group-hover:scale-110 group-hover:-rotate-90 transition-transform duration-500"
            >
              <Plus :size="20" />
            </div>
          </div>

          <div :class="needsPortServiceSelection ? 'grid grid-cols-1 sm:grid-cols-2 gap-6' : 'grid grid-cols-1 gap-6'">
            <label v-if="needsPortServiceSelection" class="space-y-3">
              <span class="text-xs font-bold uppercase tracking-wider text-gray-500 dark:text-zinc-500">{{ t("stackView.service") }}</span>
              <select
                v-model="newPort.serviceName"
                class="w-full px-5 py-4 rounded-xl border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900 text-base text-gray-900 dark:text-white focus:outline-none focus:ring-4 focus:ring-blue-500/30 transition-shadow duration-300 hover:border-blue-300"
              >
                <option v-for="service in portServices" :key="service.value" :value="service.value">{{ service.label }}</option>
              </select>
            </label>

            <label class="space-y-3">
              <span class="text-xs font-bold uppercase tracking-wider text-gray-500 dark:text-zinc-500">{{ t("stackView.portMappingLabel") }}</span>
              <input
                v-model="newPort.mapping"
                placeholder="9000:9000 or 9000 or 53:53/udp"
                class="w-full px-5 py-4 rounded-xl border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900 text-base text-gray-900 dark:text-white focus:outline-none focus:ring-4 focus:ring-blue-500/30 font-mono transition-shadow duration-300 hover:border-blue-300"
              />
              <p class="text-xs text-gray-500 dark:text-zinc-400">{{ t("stackView.portMappingHint") }}</p>
            </label>
          </div>

          <div class="flex justify-end pt-2">
            <button
              @click="openPort"
              :disabled="openingPort || portServices.length === 0"
              class="inline-flex items-center justify-center gap-3 px-6 py-4 rounded-xl text-xs font-bold uppercase tracking-wider border border-blue-200 dark:border-blue-500/20 bg-blue-50 dark:bg-blue-500/10 text-blue-700 dark:text-blue-400 hover:bg-blue-100 dark:hover:bg-blue-500/30 hover:-translate-y-1 hover:shadow-lg transition-all duration-300 active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed group/btn"
            >
              <Plus :size="16" class="group-hover/btn:rotate-180 transition-transform duration-500" />
              {{ openingPort ? t("stackView.openingPortAction") : t("stackView.openPortAction") }}
            </button>
          </div>
        </div>

        <div v-show="activePortTab === 'list' && enrichedPorts.length > 0">
          <div class="flex items-center justify-end mb-4">
            <!-- Toggle only shown when there are described ports -->
            <div
              v-if="hasDescribedPorts"
              class="flex items-center gap-2 rounded-xl bg-gray-100 dark:bg-zinc-900 p-1.5 hover:shadow-inner transition-shadow duration-300"
            >
              <button
                @click="showOnlyDescribedPorts = false"
                :class="
                  !showOnlyDescribedPorts
                    ? 'bg-white dark:bg-zinc-800 text-gray-900 dark:text-white shadow-lg scale-105'
                    : 'text-gray-500 hover:text-gray-700 dark:hover:text-zinc-300 hover:scale-105'
                "
                class="px-5 py-2 text-xs font-bold uppercase tracking-wider rounded-lg transition-all duration-300"
              >
                {{ t("stackView.allPorts") }}
              </button>
              <button
                @click="showOnlyDescribedPorts = true"
                :class="
                  showOnlyDescribedPorts
                    ? 'bg-white dark:bg-zinc-800 text-gray-900 dark:text-white shadow-lg scale-105'
                    : 'text-gray-500 hover:text-gray-700 dark:hover:text-zinc-300 hover:scale-105'
                "
                class="px-5 py-2 text-xs font-bold uppercase tracking-wider rounded-lg transition-all duration-300"
              >
                {{ t("stackView.described") }}
              </button>
            </div>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
            <div
              v-for="(p, i) in visiblePorts"
              :key="i"
              class="group bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-3xl p-8 hover:-translate-y-2 hover:shadow-xl hover:border-purple-300 dark:hover:border-purple-500/50 transition-all duration-500 ease-out"
              :style="{ animationDelay: `${i * 100}ms` }"
            >
              <div class="flex items-start gap-5 mb-8">
                <div
                  class="w-14 h-14 rounded-2xl bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center text-gray-600 dark:text-zinc-400 shrink-0 shadow-sm group-hover:text-purple-500 group-hover:scale-110 group-hover:-rotate-6 transition-all duration-500"
                >
                  <Network :size="24" />
                </div>
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-3 mb-2">
                    <span class="font-mono text-xs font-bold uppercase text-gray-900 dark:text-white">{{ p.protocol }}</span>
                    <span
                      v-if="p.labeledProtocol"
                      class="text-xs px-2.5 py-1 bg-gray-100 dark:bg-zinc-800 text-gray-500 dark:text-zinc-400 rounded-lg uppercase font-bold tracking-widest border border-gray-200 dark:border-zinc-700 hover:scale-105 transition-transform"
                      >{{ p.labeledProtocol }}</span
                    >
                  </div>
                  <div
                    class="text-sm text-gray-500 dark:text-zinc-400 truncate group-hover:text-gray-800 dark:group-hover:text-zinc-200 transition-colors duration-300"
                  >
                    {{ p.label || p.service }}
                  </div>
                </div>
              </div>

              <div class="space-y-4 mb-8">
                <div class="flex items-center justify-between text-xs">
                  <span class="text-gray-500 dark:text-zinc-500 uppercase font-bold tracking-wider">{{ t("stackView.hostPort") }}</span>
                  <span class="font-mono font-bold text-gray-900 dark:text-white group-hover:scale-110 transition-transform origin-right">{{
                    p.hostPort
                  }}</span>
                </div>
                <div class="flex items-center justify-between text-xs">
                  <span class="text-gray-500 dark:text-zinc-500 uppercase font-bold tracking-wider">{{ t("stackView.containerPort") }}</span>
                  <span class="font-mono font-medium text-gray-700 dark:text-zinc-300 group-hover:scale-110 transition-transform origin-right">{{
                    p.containerPort
                  }}</span>
                </div>
                <div class="flex items-center justify-between text-xs">
                  <span class="text-gray-500 dark:text-zinc-500 uppercase font-bold tracking-wider">{{ t("stackView.service") }}</span>
                  <span class="font-mono text-gray-500 dark:text-zinc-400 truncate max-w-[120px]">{{ p.service }}</span>
                </div>
              </div>

              <a
                v-if="p.protocol === 'tcp'"
                :href="appUrl(p.hostPort, p.labeledProtocol || 'http')"
                target="_blank"
                class="w-full flex items-center justify-center gap-3 px-6 py-4 bg-black dark:bg-white text-white dark:text-black rounded-xl hover:bg-gray-800 dark:hover:bg-gray-200 hover:-translate-y-1 hover:shadow-lg transition-all duration-300 text-xs font-bold uppercase tracking-wider active:scale-95 group/link"
              >
                <ExternalLink :size="16" class="group-hover/link:translate-x-1 group-hover/link:-translate-y-1 transition-transform" />
                {{ t("stackView.open") }}
              </a>
              <div
                v-else
                class="w-full flex items-center justify-center px-6 py-4 bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 text-gray-400 dark:text-zinc-500 rounded-xl text-xs font-bold uppercase tracking-wider hover:bg-gray-100 dark:hover:bg-zinc-800 transition-colors duration-300"
              >
                {{ p.protocol.toUpperCase() }} {{ t("stackView.port") }}
              </div>
            </div>
          </div>
        </div>

        <div
          v-show="activePortTab === 'list' && enrichedPorts.length === 0"
          class="bg-gray-50 dark:bg-zinc-900/40 border border-dashed border-gray-200 dark:border-zinc-800 rounded-3xl p-10 flex flex-col items-center justify-center gap-6 text-gray-400 dark:text-zinc-500 hover:bg-gray-100 dark:hover:bg-zinc-800/60 transition-colors duration-500"
        >
          <Network :size="32" class="shrink-0 opacity-50 mb-2" />
          <span class="text-sm font-bold uppercase tracking-widest">{{ t("stackView.noPortsPublished") }}</span>
        </div>

        <!-- ── Caddy Auth Proxy ──────────────────────────────────────────────── -->
        <div v-show="activePortTab === 'caddy' && stack.appId !== 'caddy-yantr'" class="space-y-6">
          <!-- Active proxy badges -->
          <div v-if="caddyProxies.length > 0" class="space-y-4">
            <div class="text-xs font-bold uppercase tracking-wider text-gray-400 dark:text-zinc-500 mb-2 ml-1">{{ t("stackView.caddyProxiesRunning") }}</div>
            <div
              v-for="proxy in caddyProxies"
              :key="proxy.servePort"
              class="group/proxy flex items-center justify-between gap-6 bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-200 dark:border-emerald-500/20 rounded-2xl px-6 py-5 hover:-translate-y-1 hover:shadow-md transition-all duration-300"
            >
              <div class="flex items-center gap-4 min-w-0">
                <div class="w-3 h-3 rounded-full bg-emerald-500 shrink-0 shadow-[0_0_10px_rgba(16,185,129,0.5)] animate-pulse"></div>
                <span
                  class="text-sm font-bold text-emerald-700 dark:text-emerald-400 font-mono tracking-tight group-hover/proxy:scale-105 transition-transform origin-left"
                >
                  :{{ proxy.servePort }} → localhost:{{ proxy.targetPort }}
                </span>
                <span
                  v-if="proxy.authEnabled"
                  class="text-xs font-bold uppercase tracking-wider px-3 py-1.5 rounded-lg bg-purple-100 dark:bg-purple-500/20 text-purple-600 dark:text-purple-400 border border-purple-200 dark:border-purple-500/30 hover:scale-110 transition-transform"
                >
                  {{ proxy.authUser }}
                </span>
              </div>
              <button
                @click="disableCaddyAuth"
                :disabled="disablingCaddy"
                class="text-xs font-bold uppercase tracking-wider px-4 py-2 rounded-xl border border-red-200 dark:border-red-500/20 bg-red-50 dark:bg-red-500/10 text-red-600 dark:text-red-400 hover:bg-red-100 dark:hover:bg-red-500/30 hover:-translate-y-0.5 hover:shadow-sm transition-all duration-300 active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {{ disablingCaddy ? "…" : t("stackView.caddyAuthDisable") }}
              </button>
            </div>
          </div>

          <div
            class="bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-3xl p-8 sm:p-10 space-y-8 hover:shadow-lg hover:border-purple-300 dark:hover:border-purple-500/40 transition-all duration-500 ease-out group"
          >
            <div class="flex items-start justify-between gap-6">
              <div>
                <div
                  class="text-sm font-bold uppercase tracking-wider text-gray-700 dark:text-zinc-300 group-hover:text-purple-600 dark:group-hover:text-purple-400 transition-colors"
                >
                  {{ t("stackView.caddyAuthHeading") }}
                </div>
                <p class="mt-2 text-sm text-gray-500 dark:text-zinc-400">{{ t("stackView.caddyAuthHint") }}</p>
              </div>
              <div
                class="hidden sm:flex items-center justify-center w-12 h-12 rounded-xl bg-purple-50 dark:bg-purple-500/10 border border-purple-100 dark:border-purple-500/20 text-purple-500 dark:text-purple-400 shrink-0 group-hover:scale-110 group-hover:rotate-12 transition-transform duration-500"
              >
                <ShieldCheck :size="20" />
              </div>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-2 gap-6">
              <label class="space-y-3">
                <span class="text-xs font-bold uppercase tracking-wider text-gray-500 dark:text-zinc-500">{{ t("stackView.caddyAuthTargetPort") }}</span>
                <input
                  v-model="caddyAuth.targetPort"
                  type="number"
                  min="1"
                  max="65535"
                  placeholder="e.g. 8096"
                  class="w-full px-5 py-4 rounded-xl border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900 text-base text-gray-900 dark:text-white focus:outline-none focus:ring-4 focus:ring-purple-500/30 font-mono transition-shadow duration-300 hover:border-purple-300"
                />
                <p class="text-xs text-gray-500 dark:text-zinc-400">{{ t("stackView.caddyAuthTargetHint") }}</p>
              </label>
              <label class="space-y-3">
                <span class="text-xs font-bold uppercase tracking-wider text-gray-500 dark:text-zinc-500">{{ t("stackView.caddyAuthServePort") }}</span>
                <input
                  v-model="caddyAuth.servePort"
                  type="number"
                  min="1024"
                  max="65535"
                  placeholder="e.g. 9096"
                  class="w-full px-5 py-4 rounded-xl border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900 text-base text-gray-900 dark:text-white focus:outline-none focus:ring-4 focus:ring-purple-500/30 font-mono transition-shadow duration-300 hover:border-purple-300"
                />
                <p class="text-xs text-gray-500 dark:text-zinc-400">{{ t("stackView.caddyAuthServeHint") }}</p>
              </label>
              <label class="space-y-3">
                <span class="text-xs font-bold uppercase tracking-wider text-gray-500 dark:text-zinc-500">{{ t("stackView.caddyAuthUser") }}</span>
                <input
                  v-model="caddyAuth.user"
                  type="text"
                  placeholder="admin"
                  class="w-full px-5 py-4 rounded-xl border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900 text-base text-gray-900 dark:text-white focus:outline-none focus:ring-4 focus:ring-purple-500/30 font-mono transition-shadow duration-300 hover:border-purple-300"
                />
              </label>
              <label class="space-y-3">
                <span class="text-xs font-bold uppercase tracking-wider text-gray-500 dark:text-zinc-500">{{ t("stackView.caddyAuthPass") }}</span>
                <input
                  v-model="caddyAuth.pass"
                  type="password"
                  placeholder="••••••••"
                  class="w-full px-5 py-4 rounded-xl border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900 text-base text-gray-900 dark:text-white focus:outline-none focus:ring-4 focus:ring-purple-500/30 font-mono transition-shadow duration-300 hover:border-purple-300"
                />
                <p class="text-xs text-gray-500 dark:text-zinc-400">{{ t("stackView.caddyAuthPassHint") }}</p>
              </label>
            </div>

            <div class="flex justify-end pt-2">
              <button
                @click="deployCaddyAuth"
                :disabled="deployingCaddy || !caddyAuth.servePort || !caddyAuth.targetPort"
                class="inline-flex items-center justify-center gap-3 px-6 py-4 rounded-xl text-xs font-bold uppercase tracking-wider border border-purple-200 dark:border-purple-500/20 bg-purple-50 dark:bg-purple-500/10 text-purple-700 dark:text-purple-400 hover:bg-purple-100 dark:hover:bg-purple-500/30 hover:-translate-y-1 hover:shadow-lg transition-all duration-300 active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed group/btn2"
              >
                <ShieldCheck
                  :size="16"
                  :class="deployingCaddy ? 'animate-ping' : 'group-hover/btn2:scale-110 group-hover/btn2:rotate-3 transition-transform duration-300'"
                />
                {{ deployingCaddy ? t("stackView.caddyAuthDeploying") : t("stackView.caddyAuthDeploy") }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- ── Storage (Named Volumes) ────────────────────────────────────── -->
      <div v-if="namedVolumes.length > 0" class="space-y-6 animate-fadeIn">
        <div class="flex items-center justify-between pb-2">
          <h2
            class="text-sm font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 flex items-center gap-3 hover:text-gray-800 dark:hover:text-gray-200 transition-colors duration-300"
          >
            <HardDrive :size="18" class="animate-pulse" style="animation-duration: 2.5s" />
            {{ t("stackView.storageVolumes") }}
          </h2>
          <button
            v-if="s3Configured && namedVolumes.length > 0"
            @click="backupAll"
            :disabled="backingUp"
            class="text-xs uppercase tracking-wider px-5 py-2.5 bg-black dark:bg-white text-white dark:text-black rounded-xl hover:bg-gray-800 dark:hover:bg-gray-200 hover:-translate-y-1 hover:shadow-lg active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-300 font-bold"
          >
            {{ backingUp ? t("stackView.backingUp") : t("stackView.backupAll") }}
          </button>
        </div>

        <!-- S3 warning -->
        <div
          v-if="!s3Configured"
          class="bg-amber-50 dark:bg-amber-500/10 border border-amber-200 dark:border-amber-500/20 rounded-3xl p-6 sm:p-8 flex items-start gap-4 hover:-translate-y-1 hover:shadow-md transition-all duration-300"
        >
          <AlertCircle :size="20" class="text-amber-600 dark:text-amber-500 shrink-0 mt-0.5 animate-bounce" style="animation-duration: 2s" />
          <p class="text-sm text-amber-900 dark:text-amber-200 leading-relaxed">
            <span class="font-bold">{{ t("stackView.s3NotConfigured") }}</span>
            <router-link to="/backup-config" class="underline hover:text-amber-700 font-extrabold ml-1 transition-colors">{{
              t("stackView.configureNow")
            }}</router-link>
            {{ t("stackView.toEnableBackups") }}
          </p>
        </div>

        <div class="grid gap-6">
          <div
            v-for="(vol, i) in namedVolumes"
            :key="vol.name"
            class="group bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-3xl p-8 sm:p-10 hover:-translate-y-2 hover:shadow-xl hover:border-blue-300 dark:hover:border-zinc-600 transition-all duration-500 ease-out"
            :style="{ animationDelay: `${i * 100}ms` }"
          >
            <div class="flex items-start gap-6 mb-8">
              <div
                class="w-16 h-16 rounded-2xl bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center text-gray-500 dark:text-zinc-400 shrink-0 shadow-sm group-hover:text-blue-500 group-hover:scale-110 group-hover:rotate-12 transition-all duration-500"
              >
                <HardDrive :size="28" />
              </div>
              <div class="min-w-0 flex-1">
                <div
                  class="font-extrabold text-lg text-gray-900 dark:text-white truncate tracking-tight group-hover:text-blue-600 transition-colors duration-300"
                  :title="vol.name"
                >
                  {{ vol.name }}
                </div>
                <div class="text-xs text-gray-500 dark:text-zinc-400 font-mono truncate mt-2 group-hover:translate-x-1 transition-transform duration-300">
                  {{ vol.destination }}
                </div>
                <div class="text-xs text-gray-400 dark:text-zinc-500 mt-4 font-bold uppercase tracking-wider">
                  {{ t("stackView.serviceLabel") }}
                  <span class="font-extrabold text-gray-600 dark:text-zinc-300 ml-1 bg-gray-100 dark:bg-zinc-800 px-2 py-1 rounded-md">{{ vol.svcName }}</span>
                  <span v-if="s3Configured" class="ml-5"
                    >{{ t("stackView.backupLabel") }}
                    <span class="font-extrabold text-gray-600 dark:text-zinc-300 ml-1 bg-gray-100 dark:bg-zinc-800 px-2 py-1 rounded-md">{{
                      getLatestBackupAge(vol.name)
                    }}</span></span
                  >
                </div>
              </div>
            </div>

            <div class="flex items-center gap-4 flex-wrap pt-6 border-t border-gray-100 dark:border-zinc-800">
              <!-- Browse -->
              <div
                v-if="browsingVolume[vol.name]"
                class="text-xs font-bold uppercase tracking-wider text-blue-600 dark:text-blue-400 animate-pulse px-5 py-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800/50 rounded-xl"
              >
                {{ t("stackView.startingWebDAV") }}
              </div>
              <button
                v-else-if="!showVolumeMenu[vol.name]"
                @click="showVolumeMenu[vol.name] = true"
                class="flex items-center gap-2.5 px-5 py-3 text-xs font-bold uppercase tracking-wider border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900 text-gray-700 dark:text-zinc-300 rounded-xl hover:bg-gray-100 dark:hover:bg-zinc-800 hover:-translate-y-1 hover:shadow-md active:scale-95 transition-all duration-300 group/browse"
              >
                <FolderOpen :size="16" class="group-hover/browse:scale-110 group-hover/browse:-rotate-6 transition-transform" />
                {{ t("stackView.browseFiles") }}
              </button>
              <div v-else class="flex items-center gap-2">
                <button
                  @click="browseVolume(vol.name, 60)"
                  class="px-5 py-3 text-xs font-bold uppercase tracking-wider bg-black dark:bg-white text-white dark:text-black rounded-xl hover:bg-gray-800 dark:hover:bg-gray-200 hover:-translate-y-1 hover:shadow-md active:scale-95 transition-all duration-300"
                  :title="t('stackView.oneHourAccess')"
                >
                  1H
                </button>
                <button
                  @click="browseVolume(vol.name, 0)"
                  class="px-5 py-3 text-xs font-bold uppercase tracking-wider bg-gray-200 dark:bg-zinc-800 text-gray-800 dark:text-zinc-200 rounded-xl hover:bg-gray-300 dark:hover:bg-zinc-700 hover:-translate-y-1 hover:shadow-md active:scale-95 transition-all duration-300"
                  :title="t('stackView.permanentAccess')"
                >
                  Perm
                </button>
              </div>
              <!-- Backup -->
              <button
                @click="backupVolume(vol.svcId)"
                :disabled="backingUp || !s3Configured"
                class="flex items-center gap-2.5 px-5 py-3 text-xs font-bold uppercase tracking-wider bg-black dark:bg-white text-white dark:text-black rounded-xl hover:bg-gray-800 dark:hover:bg-gray-200 disabled:opacity-40 disabled:cursor-not-allowed hover:-translate-y-1 hover:shadow-md active:scale-95 transition-all duration-300"
              >
                {{ t("stackView.backup") }}
              </button>
              <!-- Restore -->
              <button
                @click="toggleRestoreMenu(vol.name)"
                :disabled="!hasBackups(vol.name) || !s3Configured"
                class="flex items-center gap-2.5 px-5 py-3 text-xs font-bold uppercase tracking-wider border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900 text-gray-700 dark:text-zinc-300 rounded-xl hover:bg-gray-100 dark:hover:bg-zinc-800 disabled:opacity-30 disabled:cursor-not-allowed hover:-translate-y-1 hover:shadow-md active:scale-95 transition-all duration-300"
              >
                {{ t("stackView.restore") }}
              </button>
            </div>

            <!-- Restore dropdown -->
            <div v-if="showRestoreMenu[vol.name] && hasBackups(vol.name)" class="mt-6 pt-6 border-t border-gray-200 dark:border-zinc-800 animate-fadeIn">
              <div class="text-xs font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500 mb-4">{{ t("stackView.availableBackups") }}</div>
              <div class="space-y-3 max-h-60 overflow-y-auto scrollbar-thin rounded-xl p-2 bg-gray-50 dark:bg-black/20">
                <div
                  v-for="backup in volumeBackups[vol.name]"
                  :key="backup.snapshotId"
                  class="flex items-center justify-between py-4 px-5 bg-white dark:bg-zinc-900/80 hover:bg-gray-50 dark:hover:bg-zinc-800 rounded-xl border border-gray-200 dark:border-zinc-700 transition-all duration-300 hover:-translate-x-1 hover:shadow-sm"
                >
                  <div class="flex-1 min-w-0">
                    <div class="font-mono text-xs font-bold text-gray-900 dark:text-white">{{ formatBackupDate(backup.timestamp) }}</div>
                    <div class="text-gray-500 dark:text-zinc-400 text-xs mt-1.5 font-bold uppercase tracking-wider">
                      {{ backup.sizeMB != null ? backup.sizeMB + " MB" : "" }}
                    </div>
                  </div>
                  <div class="flex gap-3 ml-4">
                    <button
                      @click="restoreBackup(vol.name, backup.snapshotId)"
                      class="px-4 py-2 border border-gray-300 dark:border-zinc-600 text-gray-700 dark:text-zinc-300 bg-white dark:bg-[#0A0A0A] rounded-lg hover:bg-gray-50 dark:hover:bg-zinc-800 hover:scale-105 active:scale-95 transition-all duration-300 text-xs font-bold uppercase tracking-wider"
                    >
                      {{ t("stackView.restore") }}
                    </button>
                    <button
                      @click="deleteBackupFile(vol.name, backup.snapshotId)"
                      class="px-4 py-2 border border-red-200 dark:border-red-500/20 text-red-600 dark:text-red-500 bg-red-50 dark:bg-red-500/10 rounded-lg hover:bg-red-100 dark:hover:bg-red-500/30 hover:scale-105 active:scale-95 transition-all duration-300 text-xs font-bold uppercase tracking-wider"
                    >
                      {{ t("common.delete") }}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Bind / tmpfs mounts compact list -->
      <div v-if="otherMounts.length > 0" class="space-y-6 animate-fadeIn">
        <h2 class="text-sm font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 flex items-center gap-3 pb-2">
          <HardDrive :size="18" class="animate-pulse" style="animation-duration: 3s" />
          {{ t("stackView.bindMounts") }}
        </h2>
        <div
          class="overflow-x-auto bg-white dark:bg-[#0A0A0A] rounded-3xl border border-gray-200 dark:border-zinc-800 shadow-sm hover:shadow-lg transition-shadow duration-500"
        >
          <table class="w-full text-left min-w-120">
            <thead>
              <tr class="bg-gray-50 dark:bg-zinc-900 border-b border-gray-200 dark:border-zinc-800">
                <th class="px-8 py-5 text-xs font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-400">{{ t("stackView.type") }}</th>
                <th class="px-8 py-5 text-xs font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-400">{{ t("stackView.hostPath") }}</th>
                <th class="px-8 py-5 text-xs font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-400">{{ t("stackView.containerPath") }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-zinc-800">
              <tr v-for="(m, i) in otherMounts" :key="i" class="hover:bg-gray-50 dark:hover:bg-zinc-900/50 transition-colors duration-300 group/row">
                <td class="px-8 py-5">
                  <span
                    class="text-xs font-bold tracking-widest uppercase px-3 py-1 rounded-lg border inline-block group-hover/row:scale-105 transition-transform"
                    :class="
                      m.type === 'bind'
                        ? 'bg-amber-50 dark:bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-200 dark:border-amber-500/20'
                        : 'bg-gray-100 dark:bg-zinc-800 text-gray-500 dark:text-zinc-400 border-gray-200 dark:border-zinc-700'
                    "
                    >{{ m.type }}</span
                  >
                </td>
                <td
                  class="px-8 py-5 font-mono text-xs text-gray-700 dark:text-zinc-300 break-all max-w-sm group-hover/row:text-gray-900 dark:group-hover/row:text-white transition-colors"
                >
                  {{ m.source || "—" }}
                </td>
                <td
                  class="px-8 py-5 font-mono text-xs text-gray-500 dark:text-zinc-400 break-all max-w-sm group-hover/row:text-gray-700 dark:group-hover/row:text-zinc-300 transition-colors"
                >
                  {{ m.destination }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- ── Containers ─────────────────────────────────────────────────────── -->
      <div class="space-y-6 animate-fadeIn">
        <div class="flex items-center justify-between pb-2">
          <h2 class="text-sm font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 flex items-center gap-3">
            <Server :size="18" class="animate-bounce" style="animation-duration: 2.5s" />
            {{ t("stackView.containers") }}
          </h2>
          <span
            class="text-xs font-mono font-bold text-gray-500 bg-gray-100 dark:bg-zinc-800 px-3 py-1 rounded-lg hover:scale-110 transition-transform cursor-default"
          >
            {{ stack.services.length }}
          </span>
        </div>

        <div class="grid gap-6">
          <div
            v-for="(svc, i) in stack.services"
            :key="svc.id"
            @click="router.push(`/containers/${svc.id}`)"
            class="group cursor-pointer bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-3xl p-8 sm:p-10 hover:border-blue-400 dark:hover:border-blue-500/60 hover:-translate-y-2 hover:shadow-xl transition-all duration-500 ease-out"
            :style="{ animationDelay: `${i * 100}ms` }"
          >
            <!-- Top row: icon + name/image + uptime -->
            <div class="flex items-start gap-6 mb-8">
              <!-- State icon -->
              <div
                class="w-16 h-16 rounded-2xl flex items-center justify-center shrink-0 shadow-sm relative transition-all duration-500 group-hover:scale-110 group-hover:rotate-6"
                :class="
                  svc.state === 'running'
                    ? 'bg-green-50 dark:bg-green-500/10 border border-green-200 dark:border-green-500/20 text-green-600 dark:text-green-500'
                    : 'bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 text-gray-400 dark:text-zinc-500'
                "
              >
                <Server :size="28" />
                <!-- Ping indicator for running -->
                <span v-if="svc.state === 'running'" class="absolute -top-1.5 -right-1.5 flex h-4 w-4">
                  <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75 duration-1000"></span>
                  <span class="relative inline-flex rounded-full h-4 w-4 bg-green-500 border-2 border-white dark:border-[#0A0A0A]"></span>
                </span>
              </div>

              <!-- Name + image + badges -->
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-4 flex-wrap mb-2">
                  <span
                    class="font-extrabold text-gray-900 dark:text-white text-lg tracking-tight group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors duration-300"
                    >{{ svc.service }}</span
                  >
                  <span
                    class="text-xs font-bold uppercase tracking-widest px-3 py-1 rounded-lg border hover:scale-105 transition-transform"
                    :class="
                      svc.state === 'running'
                        ? 'bg-green-50 dark:bg-green-500/10 text-green-600 dark:text-green-500 border-green-200 dark:border-green-500/20 shadow-[0_0_10px_rgba(16,185,129,0.1)]'
                        : 'bg-gray-100 dark:bg-zinc-800 text-gray-500 dark:text-zinc-400 border-gray-200 dark:border-zinc-700'
                    "
                    >{{ svc.state }}</span
                  >
                  <span
                    v-if="svc.hasYantrLabel"
                    class="text-xs font-bold uppercase tracking-widest px-3 py-1 rounded-lg border bg-blue-50 dark:bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-200 dark:border-blue-500/20 hover:scale-105 transition-transform"
                    >{{ t("stackView.primary") }}</span
                  >
                </div>
                <div
                  class="font-mono text-sm text-gray-500 dark:text-zinc-400 truncate group-hover:translate-x-1 transition-transform duration-300"
                  :title="svc.image"
                >
                  {{ svc.image }}
                </div>
              </div>

              <!-- Uptime + nav hint -->
              <div class="flex items-center gap-4 shrink-0">
                <div v-if="formatUptime(svc)" class="text-right hidden sm:block group-hover:scale-105 transition-transform duration-300">
                  <div class="text-xs uppercase font-bold text-gray-400 dark:text-zinc-500 tracking-widest mb-1">{{ t("stackView.uptime") }}</div>
                  <div class="font-mono font-extrabold text-sm tabular-nums text-gray-700 dark:text-zinc-300">{{ formatUptime(svc) }}</div>
                </div>
                <div
                  v-else-if="svc.state !== 'running'"
                  class="text-xs font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-600 hidden sm:block self-center"
                >
                  {{ t("stackView.stopped") }}
                </div>
                <div
                  class="w-10 h-10 rounded-full bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center group-hover:bg-blue-50 dark:group-hover:bg-blue-900/30 group-hover:border-blue-200 dark:group-hover:border-blue-800/50 transition-colors duration-300"
                >
                  <ChevronRight
                    :size="20"
                    class="text-gray-400 dark:text-zinc-500 group-hover:text-blue-500 group-hover:translate-x-1 transition-all duration-300"
                  />
                </div>
              </div>
            </div>

            <!-- Bottom row: ports + hover hint -->
            <div class="flex items-center justify-between gap-4 flex-wrap pt-6 border-t border-gray-100 dark:border-zinc-800">
              <!-- Ports -->
              <div class="flex items-center gap-3 flex-wrap">
                <!-- Published ports -->
                <template v-if="svc.rawPorts.filter((p) => p.PublicPort).length > 0">
                  <span
                    v-for="p in [
                      ...new Map(svc.rawPorts.filter((rp) => rp.PublicPort).map((rp) => [`${rp.PublicPort}:${rp.PrivatePort}:${rp.Type}`, rp])).values(),
                    ]"
                    :key="`${p.PublicPort}-${p.Type}`"
                    class="inline-flex items-center gap-2 text-xs font-mono font-bold px-3 py-1.5 rounded-lg bg-green-50 dark:bg-green-500/10 text-green-700 dark:text-green-400 border border-green-200 dark:border-green-500/20 hover:scale-105 transition-transform"
                  >
                    <Network :size="14" />
                    :{{ p.PublicPort }} → {{ p.PrivatePort }}
                  </span>
                </template>
                <!-- Internal-only ports (not exposed to host) -->
                <template v-else-if="svc.rawPorts.length > 0">
                  <span
                    v-for="p in [...new Map(svc.rawPorts.map((rp) => [`${rp.PrivatePort}:${rp.Type}`, rp])).values()]"
                    :key="`internal-${p.PrivatePort}-${p.Type}`"
                    class="inline-flex items-center gap-2 text-xs font-mono font-bold px-3 py-1.5 rounded-lg bg-gray-50 dark:bg-zinc-900 text-gray-500 dark:text-zinc-400 border border-gray-200 dark:border-zinc-800 hover:scale-105 transition-transform"
                    :title="t('stackView.internalPort')"
                  >
                    <Network :size="14" />
                    {{ p.PrivatePort }}/{{ p.Type }}
                  </span>
                </template>
              </div>

              <!-- Hover hint (visible on hover, acts as affordance for mobile too) -->
              <span
                class="flex items-center gap-2 text-xs font-bold uppercase tracking-widest text-blue-500 dark:text-blue-400 opacity-0 group-hover:opacity-100 group-hover:-translate-x-2 transition-all duration-300 shrink-0 select-none"
              >
                <Terminal :size="14" class="animate-bounce" style="animation-duration: 2s" />
                {{ t("stackView.logs") }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- ── Configuration (Env Vars) ───────────────────────────────────────── -->
      <div v-if="stackEnvVars.length > 0" class="space-y-6 animate-fadeIn mb-16">
        <div class="flex items-center justify-between pb-2">
          <h2 class="text-sm font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 flex items-center gap-3">
            <Settings2 :size="18" class="animate-spin-slow" />
            {{ t("stackView.configurationVariables") }}
          </h2>
          <span
            class="text-xs font-mono font-bold text-gray-500 bg-gray-100 dark:bg-zinc-800 px-3 py-1 rounded-lg cursor-default hover:scale-110 transition-transform"
            >{{ stackEnvVars.length }}</span
          >
        </div>

        <div
          class="bg-white dark:bg-[#0A0A0A] rounded-3xl border border-gray-200 dark:border-zinc-800 overflow-hidden divide-y divide-gray-100 dark:divide-zinc-800 shadow-sm hover:shadow-lg transition-shadow duration-500"
        >
          <div
            v-for="(v, i) in stackEnvVars"
            :key="v.key"
            class="flex flex-col sm:flex-row sm:items-center gap-4 sm:gap-6 px-8 py-5 hover:bg-gray-50 dark:hover:bg-zinc-900/50 transition-colors duration-300 group/env"
            :style="{ animationDelay: `${i * 30}ms` }"
          >
            <!-- Key -->
            <div class="sm:w-80 shrink-0 min-w-0">
              <span
                class="font-mono text-sm font-extrabold text-gray-700 dark:text-zinc-300 truncate block group-hover/env:text-blue-600 dark:group-hover/env:text-blue-400 transition-colors duration-300"
                :title="v.key"
                >{{ v.key }}</span
              >
              <span
                v-if="stackEnvVars.some((x) => x.key === v.key && x.service !== v.service) || stack.services.length > 1"
                class="text-xs font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500 mt-1.5 block group-hover/env:translate-x-1 transition-transform"
                >{{ v.service }}</span
              >
            </div>

            <!-- Value -->
            <div class="flex-1 min-w-0 flex items-center justify-between gap-4">
              <span
                v-if="!isSensitive(v.key) || revealedVars.has(v.key)"
                class="font-mono text-sm text-gray-900 dark:text-zinc-100 break-all select-all group-hover/env:bg-blue-50 dark:group-hover/env:bg-blue-900/20 px-2 py-1 -ml-2 rounded-md transition-colors duration-300"
                >{{ v.value || "—" }}</span
              >
              <span v-else class="font-mono text-sm text-gray-400 dark:text-zinc-600 tracking-widest select-none mt-1">••••••••</span>

              <!-- Reveal toggle for sensitive vars -->
              <button
                v-if="isSensitive(v.key)"
                @click="toggleReveal(v.key)"
                class="shrink-0 p-2.5 rounded-xl text-gray-400 bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 hover:text-gray-700 dark:hover:text-white hover:bg-gray-200 dark:hover:bg-zinc-800 hover:scale-110 active:scale-95 transition-all duration-300 shadow-sm"
                :title="revealedVars.has(v.key) ? t('stackView.hide') : t('stackView.show')"
              >
                <EyeOff v-if="revealedVars.has(v.key)" :size="16" class="animate-pulse" />
                <Eye v-else :size="16" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
