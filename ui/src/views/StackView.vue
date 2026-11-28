<script setup>
import { ref, computed, onMounted } from "vue";
import { usePolling } from "../composables/usePolling";
import { useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { useApiUrl } from "../composables/useApiUrl";
import { useNotification } from "../composables/useNotification";
import { useJobs } from "../composables/useJobs";
import { buildIssueReportUrl } from "../utils/developer";
import { appDetail, dashboard } from "../utils/routes.js";
import { formatDuration } from "../utils/metrics";
import { isNavigableProtocol } from "../utils/url";
import { useCurrentTime } from "../composables/useCurrentTime";
import AppLogo from "../components/AppLogo.vue";
import DeveloperCredit from "../components/DeveloperCredit.vue";
import StackServiceTable from "../components/StackServiceTable.vue";
import AppLayout from "../components/AppLayout.vue";
import { ExternalLink, Trash2, RotateCcw, Bug, AlertTriangle, Play, Square } from "@lucide/vue";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const { apiUrl } = useApiUrl();
const toast = useNotification();
const { fetchActiveJob, pollJobUntilDone } = useJobs();
const { currentTime } = useCurrentTime();

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


const overallStateKey = computed(() => {
  if (!stack.value) return "unknown";
  const services = stack.value.services ?? [];
  if (services.length === 0) return "stopped";
  if (services.every((s) => s.state === "running")) return "running";
  if (services.some((s) => s.state === "running")) return "partial";
  return "stopped";
});

const overallState = computed(() => {
  if (overallStateKey.value === "unknown") return "unknown";
  if (overallStateKey.value === "running") return t("stackView.running");
  if (overallStateKey.value === "partial") return t("stackView.partial");
  return t("stackView.stopped");
});

// stateClass is a chip modifier rather than a palette: the colours live with the
// rest of the theme, so a state chip is one class and light and dark follow it.
const stateClass = computed(() => {
  if (overallStateKey.value === "running") return "yantra-chip--ok";
  if (overallStateKey.value === "partial") return "yantra-chip--warn";
  return "";
});

const serviceCount = computed(() => stack.value?.services?.length ?? 0);

const servicesHealthy = computed(() =>
  (stack.value?.services ?? []).filter((s) => s.state === "running").length
);

const accessPortCount = computed(() =>
  (stack.value?.publishedPorts ?? []).filter((port) =>
    port.show === true &&
    port.protocol?.toLowerCase() === "tcp" &&
    port.hostPort &&
    isNavigableProtocol(port.displayProtocol)
  ).length
);

const stackUptime = computed(() => {
  const running = (stack.value?.services ?? []).filter(
    (s) => s.state === "running" && Number.isFinite(s.created)
  );
  if (running.length === 0) return null;

  const oldestCreatedSeconds = Math.min(...running.map((s) => s.created));
  const uptimeMs = currentTime.value - oldestCreatedSeconds * 1000;
  if (uptimeMs <= 0) return t("stackView.justStarted");
  return formatDuration(uptimeMs);
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
      router.push(dashboard());
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
      router.push(dashboard());
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
          router.push(dashboard());
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
  <AppLayout width="reading" gap="none">

    <!-- Loading -->
    <div v-if="loading" class="flex justify-center py-32">
      <div class="h-8 w-8 animate-spin rounded-full border-[3px] border-zinc-200 border-t-zinc-900 dark:border-zinc-800 dark:border-t-white"></div>
    </div>

    <!-- Load failure: the page body is gated on `stack`, so without this the whole
         view rendered as a blank screen with only a transient toast. -->
    <div v-else-if="loadFailed" class="flex flex-col items-center gap-4 py-32 text-center">
      <AlertTriangle :size="28" class="yantra-text-danger" />
      <p class="text-sm font-semibold yantra-text-primary">{{ t("stackView.loadFailedTitle") }}</p>
      <p class="max-w-md text-xs yantra-text-secondary">{{ t("stackView.loadFailedHint", { id: projectId }) }}</p>
      <button @click="retryLoad" class="yantra-btn yantra-btn--primary mt-1">
        <RotateCcw :size="13" />{{ t("stackView.retry") }}
      </button>
    </div>

    <!-- Content -->
    <main v-else-if="stack" class="anim-enter space-y-6">
      <!-- Breadcrumb: dashboard → stack. Container detail links back to here. -->
      <nav class="flex flex-wrap items-center gap-2 text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
        <router-link :to="dashboard()" class="transition-colors hover:text-zinc-900 dark:hover:text-white">
          {{ t("nav.dashboard") }}
        </router-link>
        <span class="text-zinc-300 dark:text-zinc-700">/</span>
        <span class="yantra-text-primary">{{ stack.app?.name || stack.appId }}</span>
      </nav>

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
          @click="router.push(appDetail(stack.appId))"
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

              <button v-if="overallStateKey !== 'running'" @click="setStackServices('running')" class="yantra-btn">
                <Play :size="13" />{{ t("stackView.startStack") }}
              </button>
              <button v-else @click="setStackServices('stopped')" class="yantra-btn">
                <Square :size="13" />{{ t("stackView.stopStack") }}
              </button>

              <button v-if="stack.app" @click="router.push(appDetail(stack.appId))" class="yantra-btn">
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

      <!-- The three numbers a stack page is asked for first. -->
      <div class="yantra-facts">
        <div class="p-5">
          <div class="yantra-label">{{ t("stackView.servicesHealthy") }}</div>
          <div class="mt-2 font-mono text-2xl font-black tracking-tighter yantra-text-primary">
            {{ servicesHealthy }} / {{ serviceCount }}
          </div>
          <div class="mt-1 text-[10px] yantra-text-secondary">{{ t("stackView.services") }}</div>
        </div>
        <div class="p-5">
          <div class="yantra-label">{{ t("stackView.accessPorts") }}</div>
          <div class="mt-2 font-mono text-2xl font-black tracking-tighter yantra-text-primary">{{ accessPortCount }}</div>
          <div class="mt-1 text-[10px] yantra-text-secondary">{{ t("containerDetail.open") }}</div>
        </div>
        <div class="p-5">
          <div class="yantra-label">{{ t("stackView.uptime") }}</div>
          <div class="mt-2 font-mono text-2xl font-black tracking-tighter yantra-text-primary">{{ stackUptime ?? "—" }}</div>
          <div class="mt-1 text-[10px] yantra-text-secondary">{{ t("stackView.oldestService") }}</div>
        </div>
      </div>

      <!-- Services are the stack's children; container-level diagnostics live
           on ContainerDetail, not duplicated inline on the stack page. -->
      <StackServiceTable :services="stack.services" :published-ports="stack.publishedPorts" />
    </main>
  </AppLayout>
</template>
