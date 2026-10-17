<script setup>
import { ref, computed, onMounted } from "vue";
import { usePolling } from "../composables/usePolling";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useApiUrl } from "../composables/useApiUrl";
import { useNotification } from "../composables/useNotification";
import { useJobs } from "../composables/useJobs";
import { appUrl, isNavigableProtocol } from "../utils/url";
import { buildIssueReportUrl } from "../utils/developer";
import { useYantrAuth } from "../composables/useYantrAuth";
import AppComments from "../components/AppComments.vue";
import AppLogo from "../components/AppLogo.vue";
import DeveloperCredit from "../components/DeveloperCredit.vue";
import StackServiceList from "../components/StackServiceList.vue";
import {
  Globe,
  ExternalLink,
  Network,
  Wrench,
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
// True when the catalog definition moved after this stack was deployed.
// Fetched once per page load: the catalog only changes on Yantr updates,
// which reload the page anyway, so the 8s stack poll does not need it.
const drifted = ref(false);

async function fetchDrift() {
  try {
    const res = await fetch(`${apiUrl.value}/api/stacks/${projectId.value}/drift`);
    const data = await res.json();
    drifted.value = res.ok && data.success === true && data.drifted === true;
  } catch {
    drifted.value = false;
  }
}

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
// "Access" shows only ports the app marked show: true; "All" also lists work
// ports — ports that exist for protocol plumbing and have no UI surface.
const showAccessPorts = ref(true);

// Volume browsing state
const browsingVolume = ref({});
const showVolumeMenu = ref({});

// Build a service-aware index from the x-yantr.ports array.
// Keyed by container port -> list of {protocol, label, service, show}.
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
      show: p.show === true,
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
// Backend already enriches publishedPorts with label/displayProtocol/service/show;
// the catalog merge is a fallback for older API responses.
const enrichedPorts = computed(() => {
  if (!stack.value) return [];
  const portLabels = buildPortLabels(stack.value.app?.ports);
  return stack.value.publishedPorts.map((p) => {
    const match = lookupPortLabel(portLabels[String(p.containerPort)], p.service);
    return {
      ...p,
      label: p.label || match?.label || null,
      labeledProtocol: (p.displayProtocol || match?.protocol || "").toLowerCase() || null,
      // The backend is authoritative: it resolves show against the running
      // service. The catalog entry is the fallback for responses that predate
      // the field, and `=== true` keeps undefined (old payload) hidden.
      show: p.show === true || (p.show === undefined && match?.show === true),
    };
  });
});

// Ports the user is meant to reach. A port that is published but not catalogued
// has no show flag at all, so it is not access — it only appears under "All".
const accessPorts = computed(() => enrichedPorts.value.filter((p) => p.show));

const visiblePorts = computed(() => {
  if (!showAccessPorts.value) return enrichedPorts.value;
  // Fall back to the catalogued ports if nothing is flagged show: true — an
  // app that has not opted any port in yet would otherwise render an empty
  // grid and look like it published nothing.
  return accessPorts.value.length > 0 ? accessPorts.value : enrichedPorts.value.filter((p) => p.label);
});

const hasAccessPorts = computed(() => accessPorts.value.length > 0);

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

// Prefilled GitHub issue URL — the title carries the app name and the body
// mentions the app's GitHub authors, so a report filed from a running stack
// reaches the people who can fix it and not just the Yantr maintainer.
const reportIssueUrl = computed(() =>
  buildIssueReportUrl(stack.value?.app, {
    prefix: "app",
    // A runtime report is about the deployed instance, which is the useful
    // thing to pin down before the author reads it.
    extraLines: projectId.value ? [`Stack: ${projectId.value}`] : [],
  })
);


// ── helpers ───────────────────────────────────────────────────────────────────


const overallState = computed(() => {
  if (!stack.value) return "unknown";
  const states = stack.value.services.map((s) => s.state);
  if (states.every((s) => s === "running")) return t("stackView.running");
  if (states.some((s) => s === "running")) return t("stackView.partial");
  return t("stackView.stopped");
});

// stateClass is a chip modifier rather than a palette: the colours live with the
// rest of the theme, so a state chip is one class and light and dark follow it.
const stateClass = computed(() => {
  if (overallState.value === "running") return "yantra-chip--ok";
  if (overallState.value === "partial") return "yantra-chip--warn";
  return "";
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
  fetchDrift();
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
  fetchDrift();
});
</script>

<template>
  <div class="min-h-screen bg-white pb-20 font-sans text-zinc-900 selection:bg-blue-500/30 dark:bg-[#0A0A0A] dark:text-zinc-100">

    <!-- Loading -->
    <div v-if="loading" class="mx-auto flex max-w-7xl justify-center p-8 py-32">
      <div class="h-8 w-8 animate-spin rounded-full border-[3px] border-zinc-200 border-t-zinc-900 dark:border-zinc-800 dark:border-t-white"></div>
    </div>

    <!-- Load failure: the page body is gated on `stack`, so without this the whole
         view rendered as a blank screen with only a transient toast. -->
    <div v-else-if="loadFailed" class="mx-auto flex max-w-7xl flex-col items-center gap-4 px-6 py-32 text-center">
      <AlertTriangle :size="28" class="yantra-text-danger" />
      <p class="text-sm font-semibold yantra-text-primary">{{ t("stackView.loadFailedTitle") }}</p>
      <p class="max-w-md text-xs yantra-text-secondary">{{ t("stackView.loadFailedHint", { id: projectId }) }}</p>
      <button @click="retryLoad" class="yantra-btn yantra-btn--primary mt-1">
        <RotateCcw :size="13" />{{ t("stackView.retry") }}
      </button>
    </div>

    <!-- Content -->
    <main v-else-if="stack" class="mx-auto max-w-7xl animate-fadeIn space-y-8 px-6 py-8">
      <!-- Drift: the catalog definition changed after this stack was deployed.
           Update only refreshes images, so point at the app page for a redeploy. -->
      <div
        v-if="drifted"
        class="flex flex-col gap-3 rounded-xl border border-amber-200 bg-amber-50 p-4 sm:flex-row sm:items-center dark:border-amber-500/20 dark:bg-amber-500/10"
      >
        <div class="flex flex-1 items-start gap-2.5">
          <AlertTriangle :size="16" class="mt-0.5 shrink-0 text-amber-600 dark:text-amber-400" />
          <div>
            <p class="text-[11px] font-bold uppercase tracking-wider text-amber-800 dark:text-amber-300">{{ t("stackView.definitionChanged") }}</p>
            <p class="mt-0.5 text-xs text-amber-700 dark:text-amber-400/90">{{ t("stackView.definitionChangedHint") }}</p>
          </div>
        </div>
        <button
          @click="router.push(`/apps/${stack.appname || stack.appId}`)"
          class="yantra-btn yantra-btn--warn shrink-0"
        >
          <ExternalLink :size="13" />{{ t("stackView.appPage") }}
        </button>
      </div>

      <!-- ── App Header ─────────────────────────────────────────────────────────────
           One panel for the whole identity: who it is, what state it is in, what
           it runs, who writes it, and the four things you can do about it. The
           actions are one strong button and three ghosts, with remove last and
           coloured as the thing it is. -->
      <div class="yantra-surface p-6">
        <div class="flex flex-col gap-6 sm:flex-row">
          <div class="yantra-tile h-20 w-20 shrink-0 p-4">
            <AppLogo
              :logo="stack.app?.logo"
              :name="stack.app?.name || stack.appId"
              :seed="stack.app?.id || stack.projectId || stack.appId"
              img-class="h-full w-full object-contain filter dark:brightness-90"
              icon-class="h-full w-full yantra-text-primary"
            />
          </div>

          <div class="min-w-0 flex-1 space-y-4">
            <div class="flex flex-wrap items-center gap-2">
              <h1 class="text-2xl font-bold tracking-tight yantra-text-primary">
                {{ stack.app?.name || stack.appId }}
              </h1>
              <span class="yantra-chip" :class="stateClass">{{ overallState }}</span>
              <span class="yantra-chip font-mono normal-case tracking-normal">{{ stack.projectId }}</span>
            </div>

            <p v-if="stack.app?.short_description" class="max-w-2xl text-sm leading-relaxed yantra-text-secondary">
              {{ stack.app.short_description }}
            </p>

            <div v-if="stack.app?.tags?.length" class="flex flex-wrap gap-1.5">
              <span
                v-for="tag in (stack.app.tags).slice(0, 6)"
                :key="tag"
                class="yantra-chip"
              >{{ tag }}</span>
            </div>

            <!-- Who writes this app. A running stack is somebody's app, and the
                 catalog knows whose — the same credit the app page shows. -->
            <DeveloperCredit v-if="stack.app" :app="stack.app" />

            <div class="flex flex-wrap items-center gap-2 pt-1">
              <button
                @click="updateStack"
                :disabled="updating"
                class="yantra-btn yantra-btn--primary"
              >
                <RotateCcw :size="13" :class="updating ? 'animate-spin' : ''" />
                {{ updating ? t("stackView.updating") : t("stackView.updateStack") }}
              </button>

              <button @click="restartStack" :disabled="restarting" class="yantra-btn">
                <RotateCcw :size="13" :class="restarting ? 'animate-spin' : ''" />
                {{ restarting ? t("stackView.restarting") : t("stackView.restartStack") }}
              </button>

              <button v-if="overallState !== 'running'" @click="setStackServices('running')" class="yantra-btn">
                <Play :size="13" />{{ t("stackView.startStack") }}
              </button>
              <button v-else @click="setStackServices('stopped')" class="yantra-btn">
                <Square :size="13" />{{ t("stackView.stopStack") }}
              </button>

              <button v-if="stack.app" @click="router.push(`/apps/${stack.appname || stack.appId}`)" class="yantra-btn">
                <ExternalLink :size="13" />{{ t("stackView.appPage") }}
              </button>

              <!-- Remove sits apart from the rest: same weight as an update is a
                   lie about what it does. -->
              <button
                @click="removeStack"
                :disabled="removing"
                class="yantra-btn yantra-btn--danger sm:ml-auto"
              >
                <Trash2 :size="13" />{{ removing ? t("stackView.removing") : t("stackView.removeStack") }}
              </button>
            </div>

            <!-- A report is a link, not an action on the stack, so it reads like
                 one — the way the app page lists its own links. -->
            <a
              :href="reportIssueUrl"
              target="_blank"
              rel="noopener"
              class="inline-flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider yantra-text-secondary transition-colors hover:text-zinc-900 dark:hover:text-white"
            >
              <Bug :size="12" />{{ t("home.externalLinks.reportIssue") }}
            </a>
          </div>
        </div>
      </div>

      <!-- ── Network access ────────────────────────────────────────────────────────
           The same shape the container page uses for its ports: a label with a
           count, a filter, and one card per port. -->
      <div class="space-y-4">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex items-center gap-2">
            <h3 class="yantra-label">{{ t("stackView.networkAccess") }}</h3>
            <span
              v-if="enrichedPorts.length > 0"
              class="yantra-chip"
              :title="accessPorts.length < enrichedPorts.length ? t('stackView.portsWithWorkCount', { access: accessPorts.length, total: enrichedPorts.length }) : ''"
            >
              {{ showAccessPorts && accessPorts.length < enrichedPorts.length ? `${accessPorts.length}/${enrichedPorts.length}` : visiblePorts.length }}
            </span>
          </div>
          <div v-if="hasAccessPorts" class="yantra-segment">
            <button
              type="button"
              class="yantra-chip"
              :class="{ 'yantra-chip--active': !showAccessPorts }"
              @click="showAccessPorts = false"
            >{{ t("stackView.allPorts") }}</button>
            <button
              type="button"
              class="yantra-chip"
              :class="{ 'yantra-chip--active': showAccessPorts }"
              @click="showAccessPorts = true"
            >{{ t("stackView.access") }}</button>
          </div>
        </div>

        <div v-if="enrichedPorts.length > 0" class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          <div v-for="(p, i) in visiblePorts" :key="i" class="yantra-surface flex h-full flex-col p-5">
            <div class="mb-4 flex items-start gap-4">
              <div class="yantra-tile h-10 w-10">
                <Globe v-if="p.labeledProtocol === 'http' || p.labeledProtocol === 'https'" :size="17" />
                <Network v-else :size="17" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="mb-1 flex items-center gap-2">
                  <span class="font-mono text-[10px] font-bold uppercase yantra-text-primary">{{ p.protocol }}</span>
                  <span v-if="p.labeledProtocol" class="yantra-chip yantra-chip--accent">{{ p.labeledProtocol }}</span>
                </div>
                <div class="truncate text-[11px] yantra-text-secondary" :title="p.label || p.service">
                  {{ p.label || p.service }}
                </div>
              </div>
            </div>

            <!-- Two facts, on hairlines rather than in a second box. -->
            <div class="yantra-panel mb-4">
              <div class="yantra-fact">
                <span class="yantra-fact__label">{{ t("containerDetail.hostPort") }}</span>
                <span v-if="p.hostPort" class="yantra-fact__value">:{{ p.hostPort }}</span>
                <span v-else class="yantra-text-secondary italic">—</span>
              </div>
              <div class="yantra-fact">
                <span class="yantra-fact__label">{{ t("containerDetail.containerPort") }}</span>
                <span class="yantra-fact__value">{{ p.containerPort }}</span>
              </div>
            </div>

            <div class="mt-auto">
              <a
                v-if="p.show && p.protocol === 'tcp' && p.hostPort && isNavigableProtocol(p.labeledProtocol)"
                :href="appUrl(p.hostPort, p.labeledProtocol)"
                target="_blank"
                rel="noopener noreferrer"
                class="yantra-btn yantra-btn--primary w-full"
              >
                <ExternalLink :size="12" />{{ t("stackView.open") }}
              </a>
              <!-- A work port is protocol plumbing with nothing to open, so it
                   says so rather than offering a link that would fail. -->
              <div
                v-else-if="!p.show"
                class="yantra-btn w-full cursor-default justify-between opacity-100"
                :title="t('stackView.workPortHint')"
              >
                <span class="inline-flex items-center gap-2"><Wrench :size="12" />{{ t("stackView.workPort") }}</span>
              </div>
              <div v-else class="yantra-btn w-full cursor-default justify-between opacity-100">
                <span>{{ p.protocol.toUpperCase() }}</span>
              </div>
            </div>
          </div>
        </div>

        <div v-else class="yantra-empty">
          <Network :size="24" />
          <span>{{ t("stackView.noPortsPublished") }}</span>
        </div>
      </div>

      <!-- ── Internal addresses ─────────────────────────────────────────────────────
           One panel per service, one hairline row per network. This used to be a
           card inside a card inside a card. -->
      <div class="space-y-4">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex items-center gap-2">
            <h3 class="yantra-label">{{ t("stackView.internalAddresses") }}</h3>
            <span v-if="servicesWithNetworks.length > 0" class="yantra-chip">{{ servicesWithNetworks.length }}</span>
          </div>
          <p class="text-[11px] yantra-text-secondary">{{ t("stackView.internalAddressesHint") }}</p>
        </div>

        <div v-if="servicesWithNetworks.length > 0" class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          <div v-for="svc in servicesWithNetworks" :key="`${svc.id}-networks`" class="yantra-surface p-5">
            <div class="mb-3 flex items-center justify-between gap-3">
              <div class="min-w-0">
                <div class="truncate text-sm font-bold yantra-text-primary">{{ svc.service }}</div>
                <div class="mt-1 truncate text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
                  {{ svc.composeService || svc.name }}
                </div>
              </div>
              <span class="yantra-chip">{{ svc.networks.length }}</span>
            </div>

            <div class="yantra-panel">
              <div v-for="network in svc.networks" :key="`${svc.id}-${network.name}`" class="yantra-fact">
                <span class="yantra-fact__label">{{ network.name }}</span>
                <span class="yantra-fact__value" :title="network.aliases?.join(', ')">{{ network.ipAddress }}</span>
              </div>
              <div v-if="svc.networks.some((n) => n.aliases?.length)" class="yantra-fact">
                <span class="yantra-fact__label">{{ t("stackView.networkAliases") }}</span>
                <span class="truncate yantra-text-secondary">{{ svc.networks.flatMap((n) => n.aliases || []).join(", ") }}</span>
              </div>
            </div>
          </div>
        </div>

        <div v-else class="yantra-empty">
          <Network :size="24" />
          <span>{{ t("stackView.noInternalAddresses") }}</span>
        </div>
      </div>

      <!-- ── Storage ────────────────────────────────────────────────────────────────
           Named volumes as cards, because they carry actions; the bind and tmpfs
           mounts as one panel of rows, because they carry nothing but text. -->
      <div v-if="namedVolumes.length > 0 || otherMounts.length > 0" class="space-y-4">
        <div class="flex items-center gap-2">
          <h3 class="yantra-label">{{ t("stackView.storageVolumes") }}</h3>
          <span v-if="namedVolumes.length > 0" class="yantra-chip yantra-chip--warn">{{ namedVolumes.length }}</span>
        </div>

        <div v-if="namedVolumes.length > 0" class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          <div v-for="vol in namedVolumes" :key="vol.name" class="yantra-surface flex h-full flex-col p-5">
            <div class="mb-4 flex items-start gap-4">
              <div class="yantra-tile yantra-tile--warn h-10 w-10">
                <HardDrive :size="17" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="truncate text-sm font-bold tracking-tight yantra-text-primary" :title="vol.name">{{ vol.name }}</div>
                <div class="mt-1 truncate font-mono text-[11px] yantra-text-secondary" :title="vol.destination">{{ vol.destination }}</div>
                <div class="mt-2"><span class="yantra-chip">{{ vol.svcName }}</span></div>
              </div>
            </div>

            <div class="mt-auto flex flex-wrap items-center gap-2 border-t border-zinc-100 pt-4 dark:border-zinc-800">
              <div
                v-if="browsingVolume[vol.name]"
                class="yantra-btn yantra-btn--warn cursor-default"
              >
                {{ t("stackView.startingWebDAV") }}
              </div>
              <template v-else-if="!showVolumeMenu[vol.name]">
                <button @click="showVolumeMenu[vol.name] = true" class="yantra-btn">
                  {{ t("stackView.browseFiles") }}
                </button>
              </template>
              <template v-else>
                <button
                  @click="browseVolume(vol.name, 60)"
                  class="yantra-btn"
                  :title="t('stackView.oneHourAccess')"
                >1H</button>
                <button
                  @click="browseVolume(vol.name, 0)"
                  class="yantra-btn"
                  :title="t('stackView.permanentAccess')"
                >{{ t("stackView.permanentAccess") }}</button>
              </template>
              <button
                @click="exportVolume(vol.name)"
                :disabled="exportingVolume[vol.name]"
                class="yantra-btn ml-auto"
                :title="t('stackView.backupVolume')"
              >
                <Loader2 v-if="exportingVolume[vol.name]" class="h-3 w-3 animate-spin" />
                <Download v-else class="h-3 w-3" />
                {{ t("stackView.backupVolume") }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="otherMounts.length > 0" class="space-y-3">
          <h4 class="yantra-label">{{ t("stackView.bindMounts") }}</h4>
          <div class="yantra-panel">
            <div v-for="(m, i) in otherMounts" :key="i" class="yantra-fact">
              <span class="yantra-chip shrink-0">{{ m.type }}</span>
              <span class="min-w-0 truncate yantra-fact__value" :title="`${m.source} → ${m.destination}`">
                {{ m.source || "—" }} <span class="yantra-text-secondary">→</span> {{ m.destination }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- ── Containers ──────────────────────────────────────────────────────────────── -->
      <div class="space-y-4">
        <StackServiceList :services="stack.services" />
      </div>

      <!-- ── Reviews & comments ─────────────────────────────────────────────────────── -->
      <!-- The same section the app page ends with, on the same catalog tag, so
           notes written before or after a deploy are the same notes on both
           pages. Needs the catalog entry: without one there is no app id to
           tag a comment with, and nothing to show. -->
      <AppComments v-if="stack.app" :app="stack.app" />
    </main>
  </div>
</template>
