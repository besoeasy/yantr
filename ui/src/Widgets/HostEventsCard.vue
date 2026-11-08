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
import WidgetSkeleton from "../components/WidgetSkeleton.vue";

// The events endpoint is a finite one-hour replay, so a poll every 8s keeps
// the ledger alive without turning the socket into a treadmill: the daemon
// scans one bounded log slice per tick and closes.
const POLL_MS = 8000;

// The endpoint can return up to twenty events. A ledger that renders all of
// them decides the height of its whole grid row, and twenty rows is a page,
// not a card. The card is two columns wide, so the feed reads as two columns
// of three: six events, newest left to right, and a count for the older ones
// it leaves out — same overflow shape the expiring-containers card uses.
const MAX_ROWS = 6;

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

// Verb + tone per daemon action. The kernel's verbs are the vocabulary the
// dashboard translates; anything unmapped keeps its raw spelling with a
// neutral dot rather than pretending it said nothing.
const rowFor = (event) => {
  const action = (event.action ?? "").toLowerCase();
  if (action.startsWith("exec") || action === "output" || action === "attach") {
    return { kind: "exec", verb: t("quickMetrics.events.exec"), tone: "bg-sky-500" };
  }
  if (action === "restart" || action === "recreate") {
    return { kind: "restart", verb: t("quickMetrics.events.restarted"), tone: "bg-amber-500" };
  }
  if (action === "die" || action === "kill" || action === "destroy" || action === "stop" || action === "exit") {
    return { kind: "stop", verb: t("quickMetrics.events.died"), tone: "bg-rose-500" };
  }
  if (action.includes("health_status: unhealthy") || action.includes("unhealthy")) {
    return { kind: "unhealthy", verb: t("quickMetrics.events.unhealthy"), tone: "bg-rose-500" };
  }
  if (action === "start" || action === "restart_failed" || action.includes("healthy")) {
    return { kind: "start", verb: t("quickMetrics.events.started"), tone: "bg-emerald-500" };
  }
  return { kind: action, verb: event.action, tone: "bg-zinc-400" };
};

function badgeClassFor(kind) {
  switch (kind) {
    case "start":
      return "border border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/50 dark:bg-emerald-900/20 dark:text-emerald-400";
    case "stop":
    case "unhealthy":
      return "border border-rose-200 bg-rose-50 text-rose-700 dark:border-rose-900/50 dark:bg-rose-900/20 dark:text-rose-400";
    case "restart":
      return "border border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900/50 dark:bg-amber-900/20 dark:text-amber-400";
    case "exec":
      return "border border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-900/50 dark:bg-sky-900/20 dark:text-sky-400";
    default:
      return "border border-zinc-200 bg-zinc-100 text-zinc-600 dark:border-zinc-800 dark:bg-zinc-800 dark:text-zinc-400";
  }
}

// A crash-loop run is the same container alternating start/stop with no other
// event between the beats — one restart per start. Two full cycles is the
// threshold: a lone start/stop pair is an ordinary run, not a loop, and reads
// better as the two rows it is.
const crashLoopAt = (classified, start) => {
  const name = classified[start].event.name;
  if (classified[start].kind !== "start" && classified[start].kind !== "stop") return null;

  let end = start;
  let starts = 0;
  let stops = 0;
  let previous = null;
  while (end < classified.length && classified[end].event.name === name) {
    const { kind } = classified[end];
    // Same verb twice is the ordinary burst, handled by the merge below.
    if ((kind !== "start" && kind !== "stop") || kind === previous) break;
    if (kind === "start") starts += 1;
    else stops += 1;
    previous = kind;
    end += 1;
  }

  if (starts < 2 || stops < 2) return null;
  return { length: end - start, starts };
};

// A daemon storm — a healthcheck flapping every second, a crash loop — fills
// the one-hour window with the same line over and over. Collapse consecutive
// identical rows (same container, same verb) into one carrying a ×N count, so
// the six slots stay a summary of what happened instead of a transcript of the
// loudest container. A crash loop never repeats the same verb, so it gets its
// own pass: start/stop beats become one "restarted ×N" row, N counting starts.
// The age stays the newest event's: the burst started earlier, but "when did I
// last see it" is the useful reading. It is read from the clock at poll time,
// not from a captured `Date.now()` — that snapshot never advanced, so every
// event after the first poll read "1s" no matter how old it was.
const rows = computed(() => {
  const nowMs = Date.now();
  const ageOf = (event) => {
    const settled = Math.max(1, Math.round((nowMs - event.atMs) / 1000));
    return settled < 60 ? `${settled}s` : settled < 3600 ? `${Math.round(settled / 60)}m` : `${Math.round(settled / 3600)}h`;
  };

  const classified = events.value.map((event, i) => ({
    event,
    i,
    ...rowFor(event),
    age: ageOf(event),
  }));

  const out = [];
  let i = 0;
  while (i < classified.length) {
    const head = classified[i];
    const loop = crashLoopAt(classified, i);
    if (loop) {
      out.push({
        key: `${head.event.atMs}-${head.event.name}-loop-${i}`,
        name: head.event.name,
        verb: t("quickMetrics.events.restarted"),
        kind: "restart",
        tone: "bg-amber-500",
        age: head.age,
        count: loop.starts,
        restartLoop: true,
      });
      i += loop.length;
      continue;
    }

    let end = i + 1;
    while (
      end < classified.length &&
      classified[end].event.name === head.event.name &&
      classified[end].verb === head.verb
    ) {
      end += 1;
    }
    out.push({
      key: `${head.event.atMs}-${head.event.name}-${head.i}`,
      name: head.event.name,
      verb: head.verb,
      kind: head.kind,
      tone: head.tone,
      age: head.age,
      count: end - i,
      restartLoop: false,
    });
    i = end;
  }
  return out;
});

const emptyText = computed(() => t("quickMetrics.events.empty"));
const unavailable = computed(() => t("quickMetrics.hostMetrics.unavailable"));

const visibleRows = computed(() => rows.value.slice(0, MAX_ROWS));
const hiddenRows = computed(() => Math.max(0, rows.value.length - MAX_ROWS));
</script>

<template>
  <WidgetSkeleton v-if="!loaded" />

  <div
    v-else
    class="yantra-card yantra-card-min group relative flex h-full w-full flex-col overflow-hidden p-7 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <div
      aria-hidden="true"
      class="pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-emerald-500/30 to-transparent dark:via-emerald-500/40"
    ></div>

    <!-- Header -->
    <div class="flex w-full items-start justify-between gap-4">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-emerald-200 bg-emerald-50 transition-transform duration-200 group-hover:scale-105 dark:border-emerald-900/50 dark:bg-emerald-900/20">
          <Activity class="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
        </div>
        <div>
          <h3 class="text-lg font-bold leading-none tracking-tight yantra-text-primary transition-colors group-hover:text-black dark:group-hover:text-white">
            {{ t("quickMetrics.events.label") }}
          </h3>
          <div class="mt-2 flex items-center gap-2">
            <span class="relative flex h-2 w-2">
              <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
              <span class="inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
            </span>
            <span class="text-[10px] font-semibold uppercase tracking-widest text-emerald-600 dark:text-emerald-500">
              {{ t("quickMetrics.events.subtitle") }}
            </span>
          </div>
        </div>
      </div>

      <!-- Right badge: degraded warning or live count -->
      <div class="flex shrink-0 items-center gap-2">
        <div
          v-if="degraded"
          class="flex items-center gap-1.5 rounded-full border border-amber-200 bg-amber-50 px-2.5 py-0.5 text-[9px] font-semibold uppercase tracking-wider text-amber-700 dark:border-amber-900/50 dark:bg-amber-900/20 dark:text-amber-400"
        >
          <span class="h-1.5 w-1.5 rounded-full bg-amber-500"></span>
          <span>{{ unavailable }}</span>
        </div>
        <div
          v-else-if="rows.length"
          class="flex items-center gap-1.5 rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-0.5 text-[9px] font-semibold uppercase tracking-wider text-emerald-700 dark:border-emerald-900/50 dark:bg-emerald-900/20 dark:text-emerald-400"
        >
          <span class="relative flex h-1.5 w-1.5">
            <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
            <span class="relative inline-flex h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
          </span>
          <span class="font-bold tabular-nums">{{ rows.length }}</span>
          <span>{{ rows.length === 1 ? 'event' : 'events' }}</span>
        </div>
        <div
          v-else
          class="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-widest yantra-text-secondary"
        >
          <span class="h-1.5 w-1.5 rounded-full bg-zinc-300 dark:bg-zinc-700"></span>
          <span>Quiet</span>
        </div>
      </div>
    </div>

    <!-- Ledger rows -->
    <div
      class="relative z-10 mt-auto pt-6 transition-opacity duration-200"
      :class="degraded ? 'opacity-50' : ''"
    >
      <template v-if="rows.length">
        <div class="grid grid-cols-1 gap-2.5 sm:grid-cols-2">
          <div
            v-for="row in visibleRows"
            :key="row.key"
            class="flex min-w-0 items-center justify-between gap-3 rounded-xl border border-zinc-200 bg-zinc-50 px-3.5 py-2.5 transition-colors duration-200 hover:border-zinc-300 hover:bg-white dark:border-zinc-800 dark:bg-zinc-900/50 dark:hover:border-zinc-700 dark:hover:bg-zinc-900"
          >
            <div class="flex min-w-0 items-center gap-2.5">
              <span class="relative flex h-2 w-2 shrink-0">
                <span class="h-2 w-2 rounded-full" :class="row.tone"></span>
              </span>

              <div class="flex min-w-0 items-center gap-2">
                <span class="truncate text-xs font-bold yantra-text-primary">
                  {{ row.name }}
                </span>

                <span
                  class="shrink-0 rounded-md px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-wider"
                  :class="badgeClassFor(row.kind)"
                >
                  {{ row.verb }}
                </span>

                <span
                  v-if="row.count > 1"
                  class="shrink-0 rounded-full border border-zinc-200 bg-zinc-100 px-1.5 py-0.5 text-[9px] font-black tabular-nums yantra-text-secondary dark:border-zinc-700 dark:bg-zinc-800"
                  :title="row.restartLoop
                    ? t('quickMetrics.events.restartBurst', { count: row.count })
                    : t('quickMetrics.events.repeat', { count: row.count })"
                >
                  ×{{ row.count }}
                </span>
              </div>
            </div>

            <span class="shrink-0 font-mono text-[11px] font-medium tabular-nums yantra-text-secondary">
              {{ row.age }}
            </span>
          </div>
        </div>

        <div
          v-if="hiddenRows > 0"
          class="mt-3 flex items-center justify-between border-t border-zinc-100 pt-3 dark:border-zinc-800"
        >
          <span class="text-[10px] font-semibold uppercase tracking-widest yantra-text-secondary">
            Last hour activity
          </span>
          <span class="rounded-full border border-zinc-200 bg-zinc-50 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider yantra-text-secondary dark:border-zinc-800 dark:bg-zinc-900/50">
            {{ t("quickMetrics.events.more", { count: hiddenRows }) }}
          </span>
        </div>
      </template>

      <!-- Empty state -->
      <div
        v-else
        class="flex flex-col items-center justify-center rounded-2xl border border-dashed border-zinc-200 bg-zinc-50 py-8 text-center dark:border-zinc-800 dark:bg-zinc-900/50"
      >
        <div class="flex h-10 w-10 items-center justify-center rounded-xl border border-zinc-200 bg-white text-zinc-400 dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-500">
          <Activity class="h-5 w-5" :class="degraded ? '' : 'animate-pulse text-emerald-500'" />
        </div>
        <p class="mt-3 text-xs font-semibold yantra-text-secondary">
          {{ degraded ? unavailable : emptyText }}
        </p>
      </div>
    </div>
  </div>
</template>
