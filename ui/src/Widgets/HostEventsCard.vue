<script>
// Dashboard engine: two columns — a feed reads like a ledger: each row wants
// name, verb and age spread across the width, and one column would clip the
// names the moment a stack gets a long one. Width honours the two-column
// ceiling every widget lives by; see Home.vue for how meta.cols is consumed.
export const meta = { cols: 2 };
</script>
<script setup>
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { Activity } from "@lucide/vue";
import { useApiUrl } from "../composables/useApiUrl";
import { expectApiSuccess } from "../composables/useApiResponse";

// The events endpoint is a finite one-hour replay, so a poll every 8s keeps
// the ledger alive without turning the socket into a treadmill: the daemon
// scans one bounded log slice per tick and closes.
const POLL_MS = 8000;

const { t } = useI18n();
const { apiUrl } = useApiUrl();

const events = ref([]);
const degraded = ref(false);
const loaded = ref(false);
let timer = null;
let pollInFlight = false;

async function poll() {
  if (pollInFlight || document.hidden) return;
  pollInFlight = true;
  try {
    const res = await fetch(`${apiUrl.value}/api/system/events`);
    const data = await expectApiSuccess(res, "Failed to fetch system events");
    events.value = Array.isArray(data.events) ? data.events : events.value;
    degraded.value = Array.isArray(data.degraded) && data.degraded.includes("events");
    loaded.value = true;
  } catch {
    // The message is deliberately not surfaced: the previous poll's list
    // stays on screen and the unavailable state reads from it diminishing.
    degraded.value = true;
  } finally {
    pollInFlight = false;
  }
}

onMounted(() => {
  poll();
  timer = setInterval(poll, POLL_MS);
});
onUnmounted(() => clearInterval(timer));

const nowMs = ref(Date.now());

// Verb + tone per daemon action. The kernel's verbs are the vocabulary the
// dashboard translates; anything unmapped keeps its raw spelling with a
// neutral dot rather than pretending it said nothing.
const rowFor = (event) => {
  const action = (event.action ?? "").toLowerCase();
  if (action.startsWith("exec") || action === "output" || action === "attach") {
    return { verb: t("quickMetrics.events.exec"), tone: "bg-sky-500" };
  }
  if (action === "restart" || action === "recreate") {
    return { verb: t("quickMetrics.events.restarted"), tone: "bg-amber-500" };
  }
  if (action === "die" || action === "kill" || action === "destroy" || action === "stop" || action === "exit") {
    return { verb: t("quickMetrics.events.died"), tone: "bg-rose-500" };
  }
  if (action.includes("health_status: unhealthy") || action.includes("unhealthy")) {
    return { verb: t("quickMetrics.events.unhealthy"), tone: "bg-rose-500" };
  }
  if (action === "start" || action === "restart_failed" || action.includes("healthy")) {
    return { verb: t("quickMetrics.events.started"), tone: "bg-emerald-500" };
  }
  return { verb: event.action, tone: "bg-zinc-400" };
};

const rows = computed(() =>
  events.value.map((event, i) => {
    const { verb, tone } = rowFor(event);
    const settled = Math.max(1, Math.round((nowMs.value - event.atMs) / 1000));
    const age = settled < 60 ? `${settled}s` : settled < 3600 ? `${Math.round(settled / 60)}m` : `${Math.round(settled / 3600)}h`;
    return { key: `${event.atMs}-${event.name}-${i}`, name: event.name, verb, tone, age };
  })
);

const emptyText = computed(() => t("quickMetrics.events.empty"));
const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]">
    <!-- Header -->
    <div class="flex w-full items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border-zinc-200 bg-zinc-50 transition-transform duration-200 group-hover:scale-105 dark:border-zinc-800 dark:bg-zinc-900/50">
          <Activity class="h-5 w-5 text-zinc-600 dark:text-zinc-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black yantra-text-primary dark:group-hover:text-white">
            {{ t("quickMetrics.events.label") }}
          </h3>
          <div class="mt-2 flex items-center gap-2">
            <span class="relative flex h-2 w-2">
              <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
              <span class="inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
            </span>
            <span class="text-[10px] font-semibold uppercase tracking-widest text-emerald-600 dark:text-emerald-500">{{ t("quickMetrics.events.subtitle") }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Ledger rows: grey on the wall by design — every measured card already
         has its colour; a feed is the only card where colour belongs to the
         verb, so the dot carries green/red/amber and the row stays quiet. -->
    <div class="relative z-10 mt-auto flex flex-col gap-2 pt-6 transition-opacity duration-200" :class="degraded ? 'opacity-50' : ''">
      <template v-if="rows.length">
        <div
          v-for="row in rows"
          :key="row.key"
          class="flex items-center justify-between gap-3 rounded-xl bg-zinc-50 p-3 dark:border-zinc-800 dark:bg-zinc-900/50"
        >
          <span class="flex min-w-0 items-center gap-2">
            <span class="h-2 w-2 shrink-0 rounded-full" :class="row.tone"></span>
            <span class="flex min-w-0 items-baseline gap-2">
              <span class="truncate text-sm font-bold yantra-text-primary">{{ row.name }}</span>
              <span class="shrink-0 text-[10px] font-semibold uppercase tracking-widest yantra-text-secondary">
                {{ row.verb }}
              </span>
            </span>
          </span>
          <span class="shrink-0 text-sm font-black tabular-nums tracking-tight yantra-text-secondary">
            {{ row.age }}
          </span>
        </div>
      </template>

      <!-- Empty vs unavailable are different facts again: nothing happening
           is a clean reading of a quiet box, a degraded socket is not. -->
      <div
        v-else
        class="flex items-center gap-2 rounded-xl bg-zinc-50 p-4 text-sm yantra-text-secondary dark:border-zinc-800 dark:bg-zinc-900/50"
      >
        <Activity class="h-4 w-4 shrink-0" :class="degraded ? '' : 'animate-pulse'" />
        {{ degraded ? unavailable : emptyText }}
      </div>
    </div>
  </div>
</template>
