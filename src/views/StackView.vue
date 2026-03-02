<script setup>
import { ref, computed, onMounted, onUnmounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useApiUrl } from "../composables/useApiUrl";
import { useCurrentTime } from "../composables/useCurrentTime";
import { useNotification } from "../composables/useNotification";
import { formatDuration, formatBytes } from "../utils/metrics";
import {
  ArrowLeft, Globe, ExternalLink, Bot, Activity,
  Terminal, Server, Network, Trash2, RefreshCw, HardDrive, FolderOpen, AlertCircle,
  Eye, EyeOff, Settings2,
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
const showOnlyDescribedPorts = ref(true);

// Env vars reveal state
const revealedVars = ref(new Set());

function isSensitive(key) {
  const k = key.toLowerCase();
  return k.includes("password") || k.includes("secret") || k.includes("token")
    || k.includes("_key") || k.endsWith("key") || k.includes("passwd")
    || k.includes("_pass") || k.endsWith("pass") || k.includes("auth")
    || k.includes("credential") || k.includes("private");
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
    for (const v of (svc.env || [])) {
      if (!map.has(v.key)) map.set(v.key, { ...v, service: svc.service });
    }
  }
  return [...map.values()].sort((a, b) => a.key.localeCompare(b.key));
});

// Browse / Backup state
const browsingVolume = ref({});
const showVolumeMenu = ref({});
const s3Configured = ref(false);
const volumeBackups = ref({});
const backingUp = ref(false);
const showRestoreMenu = ref({});

// Build a port-number → {label, protocol} lookup from the info.json ports array
function buildPortLabels(ports) {
  const labels = {};
  if (!Array.isArray(ports)) return labels;
  for (const p of ports) {
    if (p.port != null) {
      labels[String(p.port)] = {
        protocol: (p.protocol || '').toLowerCase(),
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

// Collect all unique mounts across all services (includes svcId for backup ops)
const allMounts = computed(() => {
  if (!stack.value) return [];
  const seen = new Set();
  const result = [];
  for (const svc of stack.value.services) {
    for (const m of (svc.mounts || [])) {
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
const namedVolumes = computed(() => allMounts.value.filter((m) => m.type === 'volume' && m.name));

// Bind mounts and tmpfs — shown in a simple compact list
const otherMounts = computed(() => allMounts.value.filter((m) => m.type !== 'volume' || !m.name));

function appUrl(hostPort, proto) {
  const scheme = proto === 'https' ? 'https' : 'http';
  return `${scheme}://${window.location.hostname}:${hostPort}`;
}

let refreshInterval = null;

// ── helpers ───────────────────────────────────────────────────────────────────

function formatUptime(service) {
  if (service.state !== "running" || !service.created) return null;
  const uptime = currentTime.value - service.created * 1000;
  if (uptime <= 0) return t('stackView.justStarted');
  return formatDuration(uptime);
}

const overallState = computed(() => {
  if (!stack.value) return "unknown";
  const states = stack.value.services.map((s) => s.state);
  if (states.every((s) => s === "running")) return t('stackView.running');
  if (states.some((s) => s === "running")) return t('stackView.partial');
  return t('stackView.stopped');
});

const stateClass = computed(() => {
  if (overallState.value === "running")
    return "bg-green-50 dark:bg-green-500/10 text-green-600 dark:text-green-400 border-green-200 dark:border-green-500/20";
  if (overallState.value === "partial")
    return "bg-amber-50 dark:bg-amber-500/10 text-amber-700 dark:text-amber-400 border-amber-200 dark:border-amber-500/20";
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
      toast.error(t('stackView.stackNotFound'));
      router.push("/");
    }
  } catch (e) {
    console.error("Failed to load stack:", e);
    toast.error(t('stackView.failedToLoadStack'));
  } finally {
    loading.value = false;
  }
}

async function removeStack() {
  if (removing.value) return;
  const name = stack.value?.app?.name || projectId.value;
  if (!confirm(t('stackView.removeStackConfirm', { name }))) return;

  removing.value = true;
  toast.info(t('stackView.removingStack', { name }));

  try {
    const firstId = stack.value?.services?.[0]?.id;
    if (!firstId) throw new Error(t('stackView.noContainerFound'));

    const res = await fetch(`${apiUrl.value}/api/containers/${firstId}`, { method: "DELETE" });
    const data = await res.json();
    if (data.success) {
      toast.success(t('stackView.stackRemoved', { name }));
      router.push("/");
    } else {
      throw new Error(data.message || t('stackView.removalFailed'));
    }
  } catch (e) {
    console.error("Remove error:", e);
    toast.error(t('stackView.failedToRemove', { error: e.message }));
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
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ expiryMinutes }),
    });
    const data = await response.json();
    if (data.success) {
      const expiryText = expiryMinutes > 0 ? ` (${t('stackView.expiresIn', { minutes: expiryMinutes })})` : ` (${t('stackView.noExpiry')})`;
      toast.success(t('stackView.volumeBrowserStarted', { expiry: expiryText }));
      window.open(`/browse/${volumeName}/`, '_blank');
    }
  } catch (e) {
    toast.error(t('stackView.failedToStartVolumeBrowser'));
  } finally {
    delete browsingVolume.value[volumeName];
  }
}

async function checkS3Config() {
  try {
    const res = await fetch(`${apiUrl.value}/api/backup/config`);
    const data = await res.json();
    s3Configured.value = data.configured;
  } catch (e) { /* silent */ }
}

async function fetchVolumeBackups() {
  if (!stack.value) return;
  // Collect unique svcIds that have named volumes
  const svcIds = [...new Set(namedVolumes.value.map((m) => m.svcId))];
  const merged = {};
  await Promise.all(svcIds.map(async (svcId) => {
    try {
      const res = await fetch(`${apiUrl.value}/api/containers/${svcId}/backups`);
      const data = await res.json();
      if (data.success && data.backups) {
        Object.assign(merged, data.backups);
        if (data.configured !== false) s3Configured.value = true;
      }
    } catch (e) { /* silent */ }
  }));
  volumeBackups.value = merged;
}

async function backupVolume(svcId) {
  if (backingUp.value) return;
  backingUp.value = true;
  try {
    const res = await fetch(`${apiUrl.value}/api/containers/${svcId}/backup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
    });
    const data = await res.json();
    if (data.success) {
      toast.success(t('stackView.backupStarted'));
      pollBackupJob(data.jobId);
    } else {
      toast.error(data.error || t('stackView.failedToStartBackup'));
    }
  } catch (e) {
    toast.error(t('stackView.failedToStartBackup'));
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
        if (data.job.status === 'completed') {
          clearInterval(iv);
          toast.success(t('stackView.backupCompleted'));
          await fetchVolumeBackups();
        } else if (data.job.status === 'failed') {
          clearInterval(iv);
          toast.error(t('stackView.backupFailed', { error: data.job.error }));
        }
      }
    } catch (e) { clearInterval(iv); }
  }, 2000);
}

async function restoreBackup(volumeName, snapshotId) {
  if (!confirm(t('stackView.restoreConfirm', { volume }))) return;
  try {
    const res = await fetch(`${apiUrl.value}/api/volumes/${volumeName}/restore`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ snapshotId, overwrite: true }),
    });
    const data = await res.json();
    if (data.success) {
      toast.success(t('stackView.restoreStarted'));
      pollRestoreJob(data.jobId);
    } else {
      toast.error(data.error || t('stackView.failedToStartRestore'));
    }
  } catch (e) {
    toast.error(t('stackView.failedToStartRestore'));
  }
  showRestoreMenu.value[volumeName] = false;
}

function pollRestoreJob(jobId) {
  const iv = setInterval(async () => {
    try {
      const res = await fetch(`${apiUrl.value}/api/restore/jobs/${jobId}`);
      const data = await res.json();
      if (data.success && data.job) {
        if (data.job.status === 'completed') { clearInterval(iv); toast.success(t('stackView.restoreCompleted')); }
        else if (data.job.status === 'failed') { clearInterval(iv); toast.error(t('stackView.restoreFailed', { error: data.job.error })); }
      }
    } catch (e) { clearInterval(iv); }
  }, 2000);
}

async function deleteBackupFile(volumeName, snapshotId) {
  if (!confirm(t('stackView.deleteBackupConfirm'))) return;
  try {
    const res = await fetch(`${apiUrl.value}/api/volumes/${volumeName}/backup/${snapshotId}`, { method: 'DELETE' });
    const data = await res.json();
    if (data.success) { toast.success(t('stackView.backupDeleted')); await fetchVolumeBackups(); }
    else toast.error(data.error || t('stackView.failedToDeleteBackup'));
  } catch (e) {
    toast.error(t('stackView.failedToDeleteBackup'));
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
  if (!backups?.length) return 'Never';
  const diffMs = Date.now() - new Date(backups[0].timestamp);
  const m = Math.floor(diffMs / 60000);
  const h = Math.floor(m / 60);
  const d = Math.floor(h / 24);
  if (d > 0) return `${d}d ago`;
  if (h > 0) return `${h}h ago`;
  if (m > 0) return `${m}m ago`;
  return 'Just now';
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
    <header class="bg-white/80 dark:bg-[#0A0A0A]/80 backdrop-blur-md border-b border-gray-200 dark:border-zinc-800 sticky top-0 z-30">
      <div class="max-w-5xl mx-auto px-4 sm:px-6 h-14 sm:h-16 flex items-center justify-between">

        <div class="flex items-center gap-2 sm:gap-4">
          <button
            @click="router.back()"
            class="inline-flex items-center justify-center w-10 h-10 rounded-lg hover:bg-gray-100 dark:hover:bg-zinc-900 transition-all text-gray-500 dark:text-zinc-400 group"
          >
            <ArrowLeft :size="16" class="group-hover:-translate-x-0.5 transition-transform" />
          </button>
          <div class="h-4 w-px bg-gray-300 dark:bg-zinc-800"></div>
          <div class="flex items-center gap-2.5 text-sm min-w-0">
            <button @click="router.push('/')" class="hidden sm:inline text-[10px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500 hover:text-gray-700 dark:hover:text-zinc-300 transition-colors">{{ t('stackView.dashboard') }}</button>
            <span class="hidden sm:inline text-gray-300 dark:text-zinc-700">/</span>
            <span class="font-semibold tracking-tight text-gray-900 dark:text-white truncate max-w-xs">
              {{ stack?.app?.name || projectId }}
            </span>
          </div>
        </div>

        <div class="flex items-center gap-3">
          <!-- Refresh -->
          <button
            @click="fetchStack"
            class="inline-flex items-center justify-center w-8 h-8 rounded-lg hover:bg-gray-100 dark:hover:bg-zinc-900 transition-all text-gray-500 dark:text-zinc-400"
            :title="t('stackView.refresh')"
          >
            <RefreshCw :size="14" />
          </button>

          <!-- State badge -->
          <div
            v-if="stack"
            class="flex items-center gap-1.5 px-2.5 py-1 rounded-md text-[10px] font-bold uppercase tracking-wider border"
            :class="stateClass"
          >
            <span
              v-if="overallState === 'running'"
              class="w-1.5 h-1.5 rounded-full bg-green-500 animate-pulse"
            ></span>
            <span
              v-else-if="overallState === 'partial'"
              class="w-1.5 h-1.5 rounded-full bg-amber-500 animate-pulse"
            ></span>
            <span v-else class="w-1.5 h-1.5 rounded-full bg-gray-400"></span>
            {{ overallState }}
          </div>
        </div>
      </div>
    </header>

    <!-- Loading -->
    <div v-if="loading" class="flex flex-col items-center justify-center min-h-[60vh]">
      <div class="w-8 h-8 border-[3px] border-gray-200 dark:border-zinc-800 border-t-blue-500 dark:border-t-blue-500 rounded-full animate-spin"></div>
    </div>

    <!-- Content -->
    <div v-else-if="stack" class="max-w-5xl mx-auto px-4 sm:px-6 py-6 sm:py-8 space-y-8">

      <!-- ── Identity card ─────────────────────────────────────────────────── -->
      <div class="group relative bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 p-6 flex flex-col sm:flex-row sm:items-center gap-6 hover:border-gray-300 dark:hover:border-zinc-700 transition-all duration-300">
        <!-- Glow Accent -->
        <div class="absolute top-0 left-0 w-full h-[2px] bg-gradient-to-r from-transparent via-blue-500 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500"></div>

        <!-- Logo -->
        <div class="w-20 h-20 shrink-0 rounded-xl bg-gray-50 dark:bg-zinc-900 flex items-center justify-center overflow-hidden border border-gray-100 dark:border-zinc-800 shadow-sm transition-transform duration-500 group-hover:scale-105">
          <img
            v-if="stack.app?.logo"
            :src="stack.app.logo"
            :alt="stack.app.name"
            loading="lazy"
            class="w-12 h-12 object-contain filter dark:brightness-90 group-hover:brightness-100 transition-all"
          />
          <Bot v-else :size="28" class="text-gray-400 dark:text-zinc-600" />
        </div>

        <!-- Info -->
        <div class="flex-1 min-w-0 space-y-2">
          <div class="flex items-center gap-3 flex-wrap">
            <h1 class="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">
              {{ stack.app?.name || stack.appId }}
            </h1>
            <!-- Project ID badge -->
            <span class="text-[10px] font-bold uppercase tracking-widest px-2 py-0.5 rounded-md bg-gray-100 dark:bg-zinc-900 text-gray-500 dark:text-zinc-400 border border-gray-200 dark:border-zinc-800">
              {{ stack.projectId }}
            </span>
          </div>

          <p v-if="stack.app?.short_description" class="text-sm text-gray-500 dark:text-zinc-400">
            {{ stack.app.short_description }}
          </p>

          <div class="flex flex-wrap gap-2 pt-1">
            <span
              v-for="tag in (stack.app?.tags || []).slice(0, 6)"
              :key="tag"
              class="text-[9px] font-bold uppercase tracking-widest px-1.5 py-0.5 rounded-md bg-gray-50 dark:bg-zinc-900/50 text-gray-600 dark:text-zinc-400 border border-gray-200 dark:border-zinc-800"
            >{{ tag }}</span>
          </div>
        </div>

        <!-- Actions -->
        <div class="flex flex-row sm:flex-col gap-2 sm:gap-2.5 shrink-0 flex-wrap">
          <a
            v-if="stack.app?.website"
            :href="stack.app.website"
            target="_blank"
            class="inline-flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-[10px] font-bold uppercase tracking-wider border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900 text-gray-700 dark:text-zinc-300 hover:bg-gray-100 dark:hover:bg-zinc-800 transition-all"
          >
            <Globe :size="12" />
            {{ t('stackView.website') }}
          </a>
          <button
            v-if="stack.app"
            @click="router.push(`/apps/${stack.appId}`)"
            class="inline-flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-[10px] font-bold uppercase tracking-wider border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900 text-gray-700 dark:text-zinc-300 hover:bg-gray-100 dark:hover:bg-zinc-800 transition-all"
          >
            <ExternalLink :size="12" />
            {{ t('stackView.appPage') }}
          </button>
          <button
            @click="removeStack"
            :disabled="removing"
            class="inline-flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-[10px] font-bold uppercase tracking-wider bg-white dark:bg-[#0A0A0A] border border-red-200 dark:border-red-500/20 text-red-600 dark:text-red-500 hover:bg-red-50 dark:hover:bg-red-500/10 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Trash2 :size="12" />
            {{ removing ? t('stackView.removing') : t('stackView.removeStack') }}
          </button>
        </div>
      </div>

      <!-- ── Published Ports ────────────────────────────────────────────────── -->
      <div v-if="enrichedPorts.length > 0" class="space-y-4">
        <div class="flex items-center justify-between">
          <h2 class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 flex items-center gap-2">
            <Network :size="12" />
            {{ t('stackView.networkAccess') }}
          </h2>
          <!-- Toggle only shown when there are described ports -->
          <div v-if="hasDescribedPorts" class="flex items-center gap-1 rounded-lg bg-gray-100 dark:bg-zinc-900 p-1">
            <button
              @click="showOnlyDescribedPorts = false"
              :class="!showOnlyDescribedPorts ? 'bg-white dark:bg-zinc-800 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 hover:text-gray-700 dark:hover:text-zinc-300'"
              class="px-3 py-1 text-[10px] font-bold uppercase tracking-wider rounded-md transition-all"
            >{{ t('stackView.allPorts') }}</button>
            <button
              @click="showOnlyDescribedPorts = true"
              :class="showOnlyDescribedPorts ? 'bg-white dark:bg-zinc-800 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 hover:text-gray-700 dark:hover:text-zinc-300'"
              class="px-3 py-1 text-[10px] font-bold uppercase tracking-wider rounded-md transition-all"
            >{{ t('stackView.described') }}</button>
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          <div
            v-for="(p, i) in visiblePorts"
            :key="i"
            class="group bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl p-5 hover:border-gray-300 dark:hover:border-zinc-600 transition-all duration-300"
          >
            <div class="flex items-start gap-3.5 mb-5">
              <div class="w-10 h-10 rounded-lg bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center text-gray-600 dark:text-zinc-400 shrink-0 shadow-sm group-hover:text-purple-500 transition-colors">
                <Network :size="18" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2 mb-1.5">
                  <span class="font-mono text-[10px] font-bold uppercase text-gray-900 dark:text-white">{{ p.protocol }}</span>
                  <span v-if="p.labeledProtocol" class="text-[9px] px-1.5 py-0.5 bg-gray-100 dark:bg-zinc-800 text-gray-500 dark:text-zinc-400 rounded-md uppercase font-bold tracking-widest border border-gray-200 dark:border-zinc-700">{{ p.labeledProtocol }}</span>
                </div>
                <div class="text-[11px] text-gray-500 dark:text-zinc-400 truncate">
                  {{ p.label || p.service }}
                </div>
              </div>
            </div>

            <div class="space-y-2 mb-5">
              <div class="flex items-center justify-between text-[11px]">
                <span class="text-gray-500 dark:text-zinc-500 uppercase font-bold tracking-wider">{{ t('stackView.hostPort') }}</span>
                <span class="font-mono font-bold text-gray-900 dark:text-white">{{ p.hostPort }}</span>
              </div>
              <div class="flex items-center justify-between text-[11px]">
                <span class="text-gray-500 dark:text-zinc-500 uppercase font-bold tracking-wider">{{ t('stackView.containerPort') }}</span>
                <span class="font-mono font-medium text-gray-700 dark:text-zinc-300">{{ p.containerPort }}</span>
              </div>
              <div class="flex items-center justify-between text-[11px]">
                <span class="text-gray-500 dark:text-zinc-500 uppercase font-bold tracking-wider">{{ t('stackView.service') }}</span>
                <span class="font-mono text-gray-500 dark:text-zinc-400">{{ p.service }}</span>
              </div>
            </div>

            <a
              v-if="p.protocol === 'tcp'"
              :href="appUrl(p.hostPort, p.labeledProtocol || 'http')"
              target="_blank"
              class="w-full flex items-center justify-center gap-2 px-3 py-2 bg-black dark:bg-white text-white dark:text-black rounded-lg hover:bg-gray-800 dark:hover:bg-gray-200 transition-all text-[11px] font-bold uppercase tracking-wider"
            >
              <ExternalLink :size="12" />
              {{ t('stackView.open') }}
            </a>
            <div v-else class="w-full flex items-center justify-center px-3 py-2 bg-gray-50 dark:bg-zinc-900/50 border border-gray-200 dark:border-zinc-800 text-gray-400 dark:text-zinc-500 rounded-lg text-[11px] font-bold uppercase tracking-wider">
              {{ p.protocol.toUpperCase() }} {{ t('stackView.port') }}
            </div>
          </div>
        </div>
      </div>

      <div v-else class="bg-gray-50 dark:bg-zinc-900/40 border border-dashed border-gray-200 dark:border-zinc-800 rounded-xl px-6 py-5 flex items-center gap-3 text-gray-400 dark:text-zinc-500">
        <Network :size="16" class="shrink-0" />
        <span class="text-xs font-medium">{{ t('stackView.noPortsPublished') }}</span>
      </div>

      <!-- ── Storage (Named Volumes) ────────────────────────────────────── -->
      <div v-if="namedVolumes.length > 0" class="space-y-4">
        <div class="flex items-center justify-between">
          <h2 class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 flex items-center gap-2">
            <HardDrive :size="12" />
            {{ t('stackView.storageVolumes') }}
          </h2>
          <button
            v-if="s3Configured && namedVolumes.length > 0"
            @click="backupAll"
            :disabled="backingUp"
            class="text-[10px] uppercase tracking-wider px-3 py-1.5 bg-black dark:bg-white text-white dark:text-black rounded-lg hover:bg-gray-800 dark:hover:bg-gray-200 disabled:opacity-50 disabled:cursor-not-allowed transition-all font-bold"
          >{{ backingUp ? t('stackView.backingUp') : t('stackView.backupAll') }}</button>
        </div>

        <!-- S3 warning -->
        <div v-if="!s3Configured" class="bg-amber-50 dark:bg-amber-500/10 border border-amber-200 dark:border-amber-500/20 rounded-xl p-4 flex items-start gap-3">
          <AlertCircle :size="14" class="text-amber-600 dark:text-amber-500 shrink-0 mt-0.5" />
          <p class="text-xs text-amber-900 dark:text-amber-200">
            <span class="font-bold">{{ t('stackView.s3NotConfigured') }}</span>
            <router-link to="/backup-config" class="underline hover:text-amber-700 font-semibold ml-1">{{ t('stackView.configureNow') }}</router-link> {{ t('stackView.toEnableBackups') }}
          </p>
        </div>

        <div class="grid gap-4">
          <div
            v-for="vol in namedVolumes"
            :key="vol.name"
            class="group bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl p-5 hover:border-gray-300 dark:hover:border-zinc-600 transition-all duration-300"
          >
            <div class="flex items-start gap-4 mb-5">
              <div class="w-10 h-10 rounded-lg bg-gray-50 dark:bg-zinc-900 border border-gray-100 dark:border-zinc-800 flex items-center justify-center text-gray-500 dark:text-zinc-400 shrink-0 shadow-sm group-hover:text-blue-500 transition-colors">
                <HardDrive :size="18" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="font-bold text-sm text-gray-900 dark:text-white truncate tracking-tight" :title="vol.name">{{ vol.name }}</div>
                <div class="text-[11px] text-gray-500 dark:text-zinc-400 font-mono truncate mt-1">{{ vol.destination }}</div>
                <div class="text-[10px] text-gray-400 dark:text-zinc-500 mt-2 font-bold uppercase tracking-wider">
                  {{ t('stackView.serviceLabel') }} <span class="font-medium text-gray-600 dark:text-zinc-300">{{ vol.svcName }}</span>
                  <span v-if="s3Configured" class="ml-3">{{ t('stackView.backupLabel') }} <span class="font-medium text-gray-600 dark:text-zinc-300">{{ getLatestBackupAge(vol.name) }}</span></span>
                </div>
              </div>
            </div>

            <div class="flex items-center gap-2 flex-wrap pt-4 border-t border-gray-100 dark:border-zinc-800">
              <!-- Browse -->
              <div v-if="browsingVolume[vol.name]" class="text-[10px] font-bold uppercase tracking-wider text-blue-600 dark:text-blue-400 animate-pulse px-3 py-2 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800/50 rounded-lg">
                {{ t('stackView.startingWebDAV') }}
              </div>
              <button
                v-else-if="!showVolumeMenu[vol.name]"
                @click="showVolumeMenu[vol.name] = true"
                class="flex items-center gap-1.5 px-3.5 py-2 text-[10px] font-bold uppercase tracking-wider border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900 text-gray-700 dark:text-zinc-300 rounded-lg hover:bg-gray-100 dark:hover:bg-zinc-800 transition-all"
              >
                <FolderOpen :size="12" />
                {{ t('stackView.browseFiles') }}
              </button>
              <div v-else class="flex items-center gap-1.5">
                <button @click="browseVolume(vol.name, 60)" class="px-3 py-2 text-[10px] font-bold uppercase tracking-wider bg-black dark:bg-white text-white dark:text-black rounded-lg hover:bg-gray-800 dark:hover:bg-gray-200 transition-all" :title="t('stackView.oneHourAccess')">1H</button>
                <button @click="browseVolume(vol.name, 0)" class="px-3 py-2 text-[10px] font-bold uppercase tracking-wider bg-gray-200 dark:bg-zinc-800 text-gray-800 dark:text-zinc-200 rounded-lg hover:bg-gray-300 dark:hover:bg-zinc-700 transition-all" :title="t('stackView.permanentAccess')">Perm</button>
              </div>
              <!-- Backup -->
              <button
                @click="backupVolume(vol.svcId)"
                :disabled="backingUp || !s3Configured"
                class="flex items-center gap-1.5 px-3.5 py-2 text-[10px] font-bold uppercase tracking-wider bg-black dark:bg-white text-white dark:text-black rounded-lg hover:bg-gray-800 dark:hover:bg-gray-200 disabled:opacity-40 disabled:cursor-not-allowed transition-all"
              >{{ t('stackView.backup') }}</button>
              <!-- Restore -->
              <button
                @click="toggleRestoreMenu(vol.name)"
                :disabled="!hasBackups(vol.name) || !s3Configured"
                class="flex items-center gap-1.5 px-3.5 py-2 text-[10px] font-bold uppercase tracking-wider border border-gray-200 dark:border-zinc-800 bg-gray-50 dark:bg-zinc-900 text-gray-700 dark:text-zinc-300 rounded-lg hover:bg-gray-100 dark:hover:bg-zinc-800 disabled:opacity-30 disabled:cursor-not-allowed transition-all"
              >{{ t('stackView.restore') }}</button>
            </div>

            <!-- Restore dropdown -->
            <div
              v-if="showRestoreMenu[vol.name] && hasBackups(vol.name)"
              class="mt-4 pt-4 border-t border-gray-200 dark:border-zinc-800"
            >
              <div class="text-[10px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500 mb-3">{{ t('stackView.availableBackups') }}</div>
              <div class="space-y-2 max-h-40 overflow-y-auto scrollbar-thin">
                <div
                   v-for="backup in volumeBackups[vol.name]"
                   :key="backup.snapshotId"
                   class="flex items-center justify-between py-2.5 px-3 bg-gray-50 dark:bg-zinc-900/50 hover:bg-gray-100 dark:hover:bg-zinc-900 rounded-lg border border-gray-200 dark:border-zinc-800 transition-all"
                 >
                   <div class="flex-1 min-w-0">
                     <div class="font-mono text-[11px] font-medium text-gray-900 dark:text-white">{{ formatBackupDate(backup.timestamp) }}</div>
                     <div class="text-gray-500 dark:text-zinc-400 text-[10px] mt-0.5 font-bold uppercase tracking-wider">{{ backup.sizeMB != null ? backup.sizeMB + ' MB' : '' }}</div>
                   </div>
                   <div class="flex gap-2 ml-3">
                     <button @click="restoreBackup(vol.name, backup.snapshotId)" class="px-2.5 py-1.5 border border-gray-300 dark:border-zinc-700 text-gray-700 dark:text-zinc-300 bg-white dark:bg-[#0A0A0A] rounded-md hover:bg-gray-50 dark:hover:bg-zinc-800 transition-all text-[10px] font-bold uppercase tracking-wider">{{ t('stackView.restore') }}</button>
                     <button @click="deleteBackupFile(vol.name, backup.snapshotId)" class="px-2.5 py-1.5 border border-red-200 dark:border-red-500/20 text-red-600 dark:text-red-500 bg-red-50 dark:bg-red-500/10 rounded-md hover:bg-red-100 dark:hover:bg-red-500/20 transition-all text-[10px] font-bold uppercase tracking-wider">{{ t('common.delete') }}</button>
                   </div>
                 </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Bind / tmpfs mounts compact list -->
      <div v-if="otherMounts.length > 0" class="space-y-4">
        <h2 class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 flex items-center gap-2">
          <HardDrive :size="12" />
          {{ t('stackView.bindMounts') }}
        </h2>
        <div class="overflow-x-auto bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800">
          <table class="w-full text-left min-w-[480px]">
            <thead>
              <tr class="bg-gray-50 dark:bg-zinc-900 border-b border-gray-200 dark:border-zinc-800">
                <th class="px-5 py-3 text-[9px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-400">{{ t('stackView.type') }}</th>
                <th class="px-5 py-3 text-[9px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-400">{{ t('stackView.hostPath') }}</th>
                <th class="px-5 py-3 text-[9px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-400">{{ t('stackView.containerPath') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-zinc-800">
              <tr v-for="(m, i) in otherMounts" :key="i" class="hover:bg-gray-50 dark:hover:bg-zinc-900/50 transition-colors">
                <td class="px-5 py-3">
                  <span class="text-[9px] font-bold tracking-widest uppercase px-2 py-0.5 rounded-md border"
                    :class="m.type === 'bind' ? 'bg-amber-50 dark:bg-amber-500/10 text-amber-600 dark:text-amber-400 border-amber-200 dark:border-amber-500/20' : 'bg-gray-100 dark:bg-zinc-800 text-gray-500 dark:text-zinc-400 border-gray-200 dark:border-zinc-700'"
                  >{{ m.type }}</span>
                </td>
                <td class="px-5 py-3 font-mono text-[11px] text-gray-700 dark:text-zinc-300 break-all max-w-xs">{{ m.source || '—' }}</td>
                <td class="px-5 py-3 font-mono text-[11px] text-gray-500 dark:text-zinc-400 break-all max-w-xs">{{ m.destination }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- ── Containers ─────────────────────────────────────────────────────── -->
      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <h2 class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 flex items-center gap-2">
            <Server :size="12" />
            {{ t('stackView.containers') }}
          </h2>
          <span class="text-[10px] font-mono text-gray-500 bg-gray-100 dark:bg-zinc-800 px-2 py-0.5 rounded">
            {{ stack.services.length }}
          </span>
        </div>

        <div class="grid gap-4">
          <div
            v-for="svc in stack.services"
            :key="svc.id"
            class="group bg-white dark:bg-[#0A0A0A] border border-gray-200 dark:border-zinc-800 rounded-xl p-5 hover:border-gray-300 dark:hover:border-zinc-600 transition-all duration-300"
          >
            <!-- Top row: icon + name/image + uptime -->
            <div class="flex items-start gap-4 mb-5">
              <!-- State icon -->
              <div
                class="w-10 h-10 rounded-lg flex items-center justify-center shrink-0 shadow-sm relative transition-colors"
                :class="svc.state === 'running'
                  ? 'bg-green-50 dark:bg-green-500/10 border border-green-200 dark:border-green-500/20 text-green-600 dark:text-green-500'
                  : 'bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-800 text-gray-400 dark:text-zinc-500'"
              >
                <Server :size="18" />
                <!-- Ping indicator for running -->
                <span
                  v-if="svc.state === 'running'"
                  class="absolute -top-1 -right-1 flex h-2.5 w-2.5"
                >
                  <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                  <span class="relative inline-flex rounded-full h-2.5 w-2.5 bg-green-500 border border-white dark:border-[#0A0A0A]"></span>
                </span>
              </div>

              <!-- Name + image + badges -->
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2.5 flex-wrap mb-1">
                  <span class="font-bold text-gray-900 dark:text-white text-sm tracking-tight">{{ svc.service }}</span>
                  <span
                    class="text-[9px] font-bold uppercase tracking-widest px-1.5 py-0.5 rounded-md border"
                    :class="svc.state === 'running'
                      ? 'bg-green-50 dark:bg-green-500/10 text-green-600 dark:text-green-500 border-green-200 dark:border-green-500/20'
                      : 'bg-gray-100 dark:bg-zinc-800 text-gray-500 dark:text-zinc-400 border-gray-200 dark:border-zinc-700'"
                  >{{ svc.state }}</span>
                  <span
                    v-if="svc.hasYantrLabel"
                    class="text-[9px] font-bold uppercase tracking-widest px-1.5 py-0.5 rounded-md border bg-blue-50 dark:bg-blue-500/10 text-blue-600 dark:text-blue-400 border-blue-200 dark:border-blue-500/20"
                  >{{ t('stackView.primary') }}</span>
                </div>
                <div class="font-mono text-[11px] text-gray-500 dark:text-zinc-400 truncate" :title="svc.image">{{ svc.image }}</div>
              </div>

              <!-- Uptime -->
              <div v-if="formatUptime(svc)" class="text-right shrink-0 hidden sm:block">
                <div class="text-[9px] uppercase font-bold text-gray-400 dark:text-zinc-500 tracking-wider mb-0.5">{{ t('stackView.uptime') }}</div>
                <div class="font-mono font-medium text-xs tabular-nums text-gray-700 dark:text-zinc-300">{{ formatUptime(svc) }}</div>
              </div>
              <div v-else-if="svc.state !== 'running'" class="text-[10px] font-bold uppercase tracking-wider text-gray-400 dark:text-zinc-600 hidden sm:block self-center">{{ t('stackView.stopped') }}</div>
            </div>

            <!-- Bottom row: ports + logs button -->
            <div class="flex items-center justify-between gap-3 flex-wrap pt-4 border-t border-gray-100 dark:border-zinc-800">
              <!-- Ports -->
              <div class="flex items-center gap-2 flex-wrap">
                <!-- Published ports -->
                <template v-if="svc.rawPorts.filter(p => p.PublicPort).length > 0">
                  <span
                    v-for="p in [...new Map(svc.rawPorts.filter(rp => rp.PublicPort).map(rp => [`${rp.PublicPort}:${rp.PrivatePort}:${rp.Type}`, rp])).values()]"
                    :key="`${p.PublicPort}-${p.Type}`"
                    class="inline-flex items-center gap-1.5 text-[10px] font-mono px-2 py-1 rounded-md bg-green-50 dark:bg-green-500/10 text-green-700 dark:text-green-400 border border-green-200 dark:border-green-500/20"
                  >
                    <Network :size="10" />
                    :{{ p.PublicPort }} → {{ p.PrivatePort }}
                  </span>
                </template>
                <!-- Internal-only ports (not exposed to host) -->
                <template v-else-if="svc.rawPorts.length > 0">
                  <span
                    v-for="p in [...new Map(svc.rawPorts.map(rp => [`${rp.PrivatePort}:${rp.Type}`, rp])).values()]"
                    :key="`internal-${p.PrivatePort}-${p.Type}`"
                    class="inline-flex items-center gap-1.5 text-[10px] font-mono px-2 py-1 rounded-md bg-gray-50 dark:bg-zinc-900 text-gray-500 dark:text-zinc-400 border border-gray-200 dark:border-zinc-800"
                    :title="t('stackView.internalPort')"
                  >
                    <Network :size="10" />
                    {{ p.PrivatePort }}/{{ p.Type }}
                  </span>
                </template>
              </div>

              <!-- Logs button -->
              <button
                @click="router.push(`/containers/${svc.id}`)"
                class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-[10px] font-bold uppercase tracking-wider bg-gray-100 dark:bg-zinc-800 text-gray-600 dark:text-zinc-300 hover:bg-gray-200 dark:hover:bg-zinc-700 hover:text-gray-900 dark:hover:text-white transition-all shrink-0"
              >
                <Terminal :size="12" />
                {{ t('stackView.logs') }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- ── Configuration (Env Vars) ───────────────────────────────────────── -->
      <div v-if="stackEnvVars.length > 0" class="space-y-4">
        <div class="flex items-center justify-between">
          <h2 class="text-[10px] font-bold uppercase tracking-widest text-gray-500 dark:text-zinc-500 flex items-center gap-2">
            <Settings2 :size="12" />
            {{ t('stackView.configurationVariables') }}
          </h2>
          <span class="text-[10px] font-mono text-gray-500 bg-gray-100 dark:bg-zinc-800 px-2 py-0.5 rounded">{{ stackEnvVars.length }}</span>
        </div>

        <div class="bg-white dark:bg-[#0A0A0A] rounded-xl border border-gray-200 dark:border-zinc-800 overflow-hidden divide-y divide-gray-100 dark:divide-zinc-800">
          <div
            v-for="v in stackEnvVars"
            :key="v.key"
            class="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4 px-5 py-3.5 hover:bg-gray-50 dark:hover:bg-zinc-900/50 transition-colors"
          >
            <!-- Key -->
            <div class="sm:w-64 shrink-0 min-w-0">
              <span class="font-mono text-[11px] font-bold text-gray-700 dark:text-zinc-300 truncate block" :title="v.key">{{ v.key }}</span>
              <span v-if="stackEnvVars.some(x => x.key === v.key && x.service !== v.service) || stack.services.length > 1"
                class="text-[9px] font-bold uppercase tracking-widest text-gray-400 dark:text-zinc-500 mt-0.5 block">{{ v.service }}</span>
            </div>

            <!-- Value -->
            <div class="flex-1 min-w-0 flex items-center justify-between gap-3">
              <span
                v-if="!isSensitive(v.key) || revealedVars.has(v.key)"
                class="font-mono text-[11px] text-gray-900 dark:text-zinc-100 break-all select-all"
              >{{ v.value || '—' }}</span>
              <span v-else class="font-mono text-[11px] text-gray-400 dark:text-zinc-600 tracking-widest select-none mt-1">••••••••</span>
              
              <!-- Reveal toggle for sensitive vars -->
              <button
                v-if="isSensitive(v.key)"
                @click="toggleReveal(v.key)"
                class="shrink-0 p-1.5 rounded-md text-gray-400 hover:text-gray-700 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-zinc-800 transition-all"
                :title="revealedVars.has(v.key) ? t('stackView.hide') : t('stackView.show')"
              >
                <EyeOff v-if="revealedVars.has(v.key)" :size="14" />
                <Eye v-else :size="14" />
              </button>
            </div>

          </div>
        </div>
      </div>

    </div>
  </div>
</template>
