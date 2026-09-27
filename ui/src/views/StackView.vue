<script setup>
import { ref, computed, onMounted } from "vue";
import { usePolling } from "../composables/usePolling";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useApiUrl } from "../composables/useApiUrl";
import { useNotification } from "../composables/useNotification";
import { useJobs } from "../composables/useJobs";
import { appUrl, isNavigableProtocol } from "../utils/url";
import { useYantrAuth } from "../composables/useYantrAuth";
import AppLogo from "../components/AppLogo.vue";
import StackServiceList from "../components/StackServiceList.vue";
import {
  Globe,
  ExternalLink,
  Network,
  Trash2,
  HardDrive,
  RotateCcw,
  Download,
  Loader2,
  Bug,
  AlertTriangle,
  Play,
  Square,
} from "@lucide/vue";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const { apiUrl } = useApiUrl();
const toast = useNotification();
const { openVolumeBrowser } = useYantrAuth();
const { fetchActiveJob, pollJobUntilDone } = useJobs();

const projectId = computed(() => route.params.projectId);

const stack = ref(null);
const loading = ref(true);
// Set when a load failed and no stack data has ever arrived. The page body is
// gated on `stack`, so without this a failed first request rendered a blank
// screen with only a transient toast and no way to retry.
const loadFailed = ref(false);
const removing = ref(false);
const updating = ref(false);
const restarting = ref(false);

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
    try {
      const active = await fetchActiveJob("system");
      if (active && active.status === "running") {
        toast.info(t("stackView.updateStillRunning") || "Update is still running in background...");
        const pollResult = await pollJobUntilDone(active.id);
        if (pollResult.success) {
          const count = pollResult.job?.result?.updatedCount ?? 0;
          if (count > 0) {
            toast.success(t("stackView.updateComplete", { count }));
          } else {
            toast.info(t("stackView.updateAlreadyLatest"));
          }
          await fetchStack();
          return;
        } else if (pollResult.error) {
          toast.error(pollResult.error || t("stackView.updateFailed"));
          return;
        }
      }
    } catch (checkErr) {
      console.warn("Failed checking active autoupdate job:", checkErr);
    }

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

// Build a service-aware index from the x-yantr.ports array.
// Keyed by container port -> list of {protocol, label, service}.
function buildPortLabels(ports) {
  const byPort = {};
  if (!Array.isArray(ports)) return byPort;
  for (const p of ports) {
    if (p.port == null) continue;
    const key = String(p.port);
    if (!byPort[key]) byPort[key] = [];
    byPort[key].push({
      protocol: (p.protocol || "").toLowerCase(),
      label: p.label || null,
      service: p.service || null,
    });
  }
  return byPort;
}

function lookupPortLabel(cands, service) {
  if (!cands || cands.length === 0) return null;
  if (service) {
    const exact = cands.find((c) => c.service === service);
    if (exact) return exact;
  }
  return cands[0];
}

// Merge published ports with described labels from x-yantr.ports.
// Backend already enriches publishedPorts with label/displayProtocol/service;
// the catalog merge is a fallback for older API responses.
const enrichedPorts = computed(() => {
  if (!stack.value) return [];
  const portLabels = buildPortLabels(stack.value.app?.ports);
  return stack.value.publishedPorts.map((p) => {
    if (p.label || p.displayProtocol) {
      return {
        ...p,
        label: p.label || null,
        labeledProtocol: (p.displayProtocol || "").toLowerCase() || null,
      };
    }
    const match = lookupPortLabel(portLabels[String(p.containerPort)], p.service);
    return {
      ...p,
      label: match?.label || null,
      labeledProtocol: match?.protocol || null,
    };
  });
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

// Named Podman volumes only
const namedVolumes = computed(() => allMounts.value.filter((m) => m.type === "volume" && m.name));

// Bind mounts and tmpfs — shown in a simple compact list
const otherMounts = computed(() => allMounts.value.filter((m) => m.type !== "volume" || !m.name));

// appUrl is shared with ContainerDetail so both render the same link. The local
// copy used to emit a bare `2001:db8::1:8080` on IPv6 hosts, which no browser
// can parse.

// Prefilled GitHub issue URL — title carries the app name, body is left for the user
const reportIssueUrl = computed(() => {
  const base = "https://github.com/besoeasy/yantr/issues/new";
  if (!stack.value) return base;
  const appName = stack.value.app?.name || stack.value.appId || projectId.value;
  const title = `[app: ${appName}] `;
  return `${base}?title=${encodeURIComponent(title)}`;
});


// ── helpers ───────────────────────────────────────────────────────────────────


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
      loadFailed.value = false;
    } else if (res.status === 404) {
      // A stack that is genuinely gone should send the user back to the grid.
      toast.error(t("stackView.stackNotFound"));
      router.push("/");
      return;
    } else {
      // Report a malformed-but-present response without ejecting the user.
      loadFailed.value = true;
    }
  } catch {
    // Network/server failure. A toast alone is not enough: the whole page body
    // is gated on `stack`, so nothing rendered and there was no way to retry.
    toast.error(t("stackView.failedToLoadStack"));
    loadFailed.value = true;
  } finally {
    loading.value = false;
  }
}

function retryLoad() {
  loadFailed.value = false;
  loading.value = true;
  fetchStack();
}



async function removeStack() {
  if (removing.value) return;
  const name = stack.value?.app?.name || projectId.value;
  if (!confirm(t("stackView.removeStackConfirm", { name }))) return;

  removing.value = true;
  toast.info(t("stackView.removingStack", { name }));

  try {
    // Must be the stack endpoint, not DELETE /api/containers/{id}.
    //
    // The container endpoint only tears the whole project down when the compose
    // file is present *and* `podman compose` is available *and* `compose down`
    // exits 0. On any of those three failures it falls through to removing that
    // single container and still answers success — so the UI reported the stack
    // removed while every sibling kept running, contradicting the confirmation
    // the user just accepted.
    //
    // It also keyed its job on the container ID, so the fetchActiveJob
    // recovery below could never match; the stack endpoint keys on projectId.
    const res = await fetch(`${apiUrl.value}/api/stacks/${projectId.value}`, { method: "DELETE" });
    const data = await res.json();

    if (data.success && data.removed) {
      toast.success(t("stackView.stackRemoved", { name }));
      router.push("/");
    } else if (data.success) {
      // Defensive: the endpoint sets `removed`. Without it, treat as incomplete.
      throw new Error(t("stackView.removalFailed"));
    } else {
      throw new Error(data.message || t("stackView.removalFailed"));
    }
  } catch (e) {
    try {
      const active = await fetchActiveJob(projectId.value);
      if (active && active.status === "running") {
        toast.info(t("stackView.removalStillRunning") || "Stack removal is still in progress...");
        const pollResult = await pollJobUntilDone(active.id);
        if (pollResult.success) {
          toast.success(t("stackView.stackRemoved", { name }));
          router.push("/");
          return;
        } else if (pollResult.error) {
          toast.error(t("stackView.failedToRemove", { error: pollResult.error }));
          return;
        }
      }
    } catch (checkErr) {
      console.warn("Failed checking active removal job:", checkErr);
    }

    toast.error(t("stackView.failedToRemove", { error: e.message }));
  } finally {
    removing.value = false;
  }
}

// ── lifecycle ─────────────────────────────────────────────────────────────────

// POST /api/stacks/{projectId}/restart existed on the server with no UI caller at
// all. The per-container start/stop endpoints are wired up in ContainerDetail.
async function restartStack() {
  if (restarting.value) return;
  restarting.value = true;
  toast.info(t("stackView.restartingStack", { name: stack.value?.app?.name || projectId.value }));

  try {
    const res = await fetch(`${apiUrl.value}/api/stacks/${projectId.value}/restart`, { method: "POST" });
    const data = await res.json();
    if (!res.ok || !data.success) {
      throw new Error(data.message || t("stackView.restartFailed"));
    }
    // Let the poll pick up the new state; do not navigate away.
    await fetchStack();
    toast.success(t("stackView.stackRestarted"));
  } catch (e) {
    toast.error(t("stackView.restartFailed", { error: e.message }));
  } finally {
    restarting.value = false;
  }
}

// Start/stop every service of the stack.
//
// There is no stack-level start/stop endpoint. Driving the containers one by one
// is safe *now* precisely because the server records the intent per compose
// service (see #87): a service stopped here stays stopped across a reboot and
// the watchdog leaves it alone, while its running siblings keep crash recovery.
// Under the old per-project model this would have been the wrong thing to add.
async function setStackServices(desired) {
  const action = desired === "running" ? "start" : "stop";
  const services = stack.value?.services ?? [];
  if (services.length === 0) {
    toast.error(t("stackView.noContainerFound"));
    return;
  }

  const name = stack.value?.app?.name || projectId.value;
  toast.info(desired === "running" ? t("stackView.startingStack", { name }) : t("stackView.stoppingStack", { name }));

  const results = await Promise.allSettled(
    services.map((s) =>
      fetch(`${apiUrl.value}/api/containers/${s.id}/${action}`, { method: "POST" }).then((r) => {
        if (!r.ok) throw new Error(r.statusText || String(r.status));
        return r.json();
      })
    )
  );

  const failed = results.filter((r) => r.status === "rejected").length;
  if (failed > 0) {
    toast.error(t("stackView.serviceActionFailed", { failed, total: services.length }));
  } else {
    toast.success(desired === "running" ? t("stackView.stackStarted") : t("stackView.stackStopped"));
  }
  await fetchStack();
}

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
    if (!data.success) {
      throw new Error(data.message || t("stackView.failedToStartVolumeBrowser"));
    }
    const expiryText = expiryMinutes > 0 ? ` (${t("stackView.expiresIn", { minutes: expiryMinutes })})` : ` (${t("stackView.noExpiry")})`;
    toast.success(t("stackView.volumeBrowserStarted", { expiry: expiryText }));
    openVolumeBrowser(volumeName);
  } catch {
    toast.error(t("stackView.failedToStartVolumeBrowser"));
  } finally {
    delete browsingVolume.value[volumeName];
  }
}

const exportingVolume = ref({});

async function exportVolume(volumeName) {
  exportingVolume.value[volumeName] = true;
  toast.info(t("stackView.backingUp") || "Backing up…");
  try {
    const res = await fetch(`${apiUrl.value}/api/volumes/${encodeURIComponent(volumeName)}/export`);
    if (!res.ok) {
      throw new Error(`HTTP ${res.status}`);
    }
    const blob = await res.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    const dateStr = new Date().toISOString().slice(0, 10);
    a.download = `${volumeName}-backup-${dateStr}.tar.gz`;
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);
    toast.success(t("stackView.backupSuccess", { name: volumeName }) || "Backup downloaded");
  } catch (error) {
    toast.error(t("stackView.backupFailed", { message: error.message }) || "Backup failed");
  } finally {
    delete exportingVolume.value[volumeName];
  }
}

// usePolling owns the timer, the in-flight guard, the visibility pause and the
// abort, so slow 8s ticks against an endpoint that re-inspects every container
// can no longer overlap.
const { start: startStackPolling } = usePolling(fetchStack, 8000);

onMounted(() => {
  startStackPolling();
});
</script>

<template>
  <div class="min-h-screen bg-white pb-20 font-sans text-zinc-900 selection:bg-blue-500/30 dark:bg-[#0A0A0A] dark:text-zinc-100">

    <!-- Loading -->
    <div v-if="loading" class="mx-auto flex max-w-7xl justify-center p-8 py-32">
       <div class="h-8 w-8 animate-spin rounded-full border-[3px] border-zinc-200 border-t-zinc-900 dark:border-zinc-800 dark:border-t-white"></div>    </div>

    <!-- Load failure: the page body is gated on `stack`, so without this the
         whole view rendered as a blank screen with only a transient toast. -->
    <div v-else-if="loadFailed" class="mx-auto flex max-w-7xl flex-col items-center gap-4 px-6 py-32 text-center">
      <AlertTriangle :size="32" class="text-amber-500" />
      <p class="text-sm font-semibold text-zinc-700 dark:text-zinc-200">
        {{ t("stackView.loadFailedTitle") }}
      </p>
      <p class="max-w-md text-xs text-zinc-500">{{ t("stackView.loadFailedHint", { id: projectId }) }}</p>
      <button
        @click="retryLoad"
        class="inline-flex items-center gap-1.5 rounded-lg border border-zinc-900 bg-zinc-900 px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-white transition-all hover:bg-black dark:border-white dark:bg-white dark:text-zinc-900"
      >
        <RotateCcw :size="13" />{{ t("stackView.retry") }}
      </button>
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
              <button
                v-if="stack.app"
                @click="router.push(`/apps/${stack.appname || stack.appId}`)"
                class="inline-flex items-center gap-1.5 rounded-lg border border-zinc-200 px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-zinc-700 transition-all hover:bg-zinc-50 dark:border-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-900/50"
              >
                <ExternalLink :size="13" />{{ t("stackView.appPage") }}
              </button>

              <!-- Restart (server endpoint, previously unreachable from the UI) -->
              <button
                @click="restartStack"
                :disabled="restarting"
                class="inline-flex items-center gap-1.5 rounded-lg border border-zinc-200 px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-zinc-700 transition-all hover:bg-zinc-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-900/50"
              >
                <RotateCcw :size="13" :class="restarting ? 'animate-spin' : ''" />
                {{ restarting ? t("stackView.restarting") : t("stackView.restartStack") }}
              </button>

              <!-- Start / Stop every service of the stack -->
              <button
                v-if="overallState !== 'running'"
                @click="setStackServices('running')"
                class="inline-flex items-center gap-1.5 rounded-lg border border-zinc-200 px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-zinc-700 transition-all hover:bg-zinc-50 dark:border-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-900/50"
              >
                <Play :size="13" />{{ t("stackView.startStack") }}
              </button>
              <button
                v-else
                @click="setStackServices('stopped')"
                class="inline-flex items-center gap-1.5 rounded-lg border border-zinc-200 px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-zinc-700 transition-all hover:bg-zinc-50 dark:border-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-900/50"
              >
                <Square :size="13" />{{ t("stackView.stopStack") }}
              </button>

              <!-- Update -->
              <button
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

              <!-- Report Issue -->
              <a
                :href="reportIssueUrl"
                target="_blank"
                rel="noopener"
                class="inline-flex items-center gap-1.5 rounded-lg border border-zinc-200 px-3 py-2 text-[11px] font-bold uppercase tracking-wider text-zinc-700 transition-all hover:bg-zinc-50 dark:border-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-900/50"
              >
                <Bug :size="13" />{{ t("home.externalLinks.reportIssue") }}
              </a>
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
                v-if="p.protocol === 'tcp' && p.hostPort && isNavigableProtocol(p.labeledProtocol)"
                :href="appUrl(p.hostPort, p.labeledProtocol)"
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

      <!-- ── Volumes ─────────────────────────────────────────────────────────────────── -->
      <div v-if="namedVolumes.length > 0 || otherMounts.length > 0" class="space-y-4">
        <div class="flex items-center gap-2">
          <h3 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">
            {{ t("stackView.storageVolumes") }}
          </h3>
          <span
            v-if="namedVolumes.length > 0"
            class="inline-flex items-center rounded-md border border-amber-200 bg-amber-50 px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-widest text-amber-700 dark:border-amber-900/40 dark:bg-amber-900/20 dark:text-amber-400"
          >
            {{ namedVolumes.length }}
          </span>
        </div>

        <div v-if="namedVolumes.length > 0" class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          <div
            v-for="vol in namedVolumes"
            :key="vol.name"
            class="group flex h-full flex-col rounded-2xl border border-zinc-200 bg-white p-5 transition-all duration-300 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-[#0A0A0A] dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
          >
            <div class="mb-4 flex items-start gap-4">
              <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-amber-200 bg-amber-50 text-amber-600 transition-transform duration-300 group-hover:scale-105 dark:border-amber-900/50 dark:bg-amber-900/20 dark:text-amber-400">
                <HardDrive :size="18" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="truncate text-sm font-bold tracking-tight text-zinc-900 dark:text-white" :title="vol.name">
                  {{ vol.name }}
                </div>
                <div class="mt-1 truncate font-mono text-[11px] text-zinc-500" :title="vol.destination">
                  {{ vol.destination }}
                </div>
                <div class="mt-2">
                  <span class="rounded-md border border-zinc-200 bg-zinc-50 px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-widest text-zinc-600 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300">
                    {{ vol.svcName }}
                  </span>
                </div>
              </div>
            </div>

            <div class="mt-auto flex flex-wrap items-center gap-2 border-t border-zinc-100 pt-4 dark:border-zinc-800">
              <div
                v-if="browsingVolume[vol.name]"
                class="animate-pulse rounded-lg border border-amber-600 bg-amber-600 px-3 py-2 text-[10px] font-bold uppercase tracking-wider text-white"
              >
                {{ t("stackView.startingWebDAV") }}
              </div>
              <template v-else-if="!showVolumeMenu[vol.name]">
                <button
                  @click="showVolumeMenu[vol.name] = true"
                  class="rounded-lg border border-amber-200 bg-amber-50 px-3.5 py-2 text-[10px] font-bold uppercase tracking-wider text-amber-700 transition-all hover:bg-amber-100 dark:border-amber-900/40 dark:bg-amber-900/20 dark:text-amber-400 dark:hover:bg-amber-900/30"
                >
                  {{ t("stackView.browseFiles") }}
                </button>
              </template>
              <template v-else>
                <button
                  @click="browseVolume(vol.name, 60)"
                  class="rounded-lg border border-zinc-900 bg-zinc-900 px-3 py-2 text-[10px] font-bold uppercase tracking-wider text-white transition-all hover:bg-black dark:border-white dark:bg-white dark:text-zinc-900 dark:hover:bg-zinc-100"
                  :title="t('stackView.oneHourAccess')"
                >
                  1H
                </button>
                <button
                  @click="browseVolume(vol.name, 0)"
                  class="rounded-lg border border-zinc-200 bg-zinc-50 px-3 py-2 text-[10px] font-bold uppercase tracking-wider text-zinc-800 transition-all hover:bg-zinc-100 dark:border-zinc-800 dark:bg-zinc-800 dark:text-zinc-200 dark:hover:bg-zinc-700"
                  :title="t('stackView.permanentAccess')"
                >
                  Perm
                </button>
              </template>
              <button
                @click="exportVolume(vol.name)"
                :disabled="exportingVolume[vol.name]"
                class="inline-flex items-center gap-1.5 rounded-lg border border-zinc-200 bg-zinc-50 px-3.5 py-2 text-[10px] font-bold uppercase tracking-wider text-zinc-700 transition-all hover:border-zinc-300 hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-300 dark:hover:border-zinc-700 dark:hover:text-white"
                :title="t('stackView.backupVolume')"
              >
                <Loader2 v-if="exportingVolume[vol.name]" class="h-3 w-3 animate-spin text-amber-500" />
                <Download v-else class="h-3 w-3" />
                {{ t('stackView.backupVolume') }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="otherMounts.length > 0" class="space-y-3">
          <h4 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">
            {{ t("stackView.bindMounts") }}
          </h4>
          <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
            <div
              v-for="(m, i) in otherMounts"
              :key="i"
              class="rounded-2xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-[#0A0A0A]"
            >
              <div class="mb-2">
                <span class="rounded-md border border-zinc-200 bg-zinc-50 px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest text-zinc-500 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-400">
                  {{ m.type }}
                </span>
              </div>
              <div class="break-all font-mono text-[11px] font-bold text-zinc-900 dark:text-white">
                {{ m.source || "—" }}
              </div>
              <div class="mt-1 break-all font-mono text-[11px] text-zinc-500">
                {{ m.destination }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- ── Containers ──────────────────────────────────────────────────────────────── -->
      <div class="space-y-4">
        <StackServiceList :services="stack.services" />
      </div>
    </main>
  </div>
</template>