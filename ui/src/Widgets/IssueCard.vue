<script>
// Dashboard engine: two of the desktop grid's four columns — the ask lives
// left and the facts the report will carry live right, so a reader can check
// what leaves the machine at a glance. See Home.vue for how meta.cols works.
export const meta = { cols: 2 };
</script>
<script setup>
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { Bug, ArrowUpRight, ShieldCheck } from "@lucide/vue";
import { useApiUrl } from "../composables/useApiUrl";
import { buildYantrIssueReportUrl } from "../utils/developer";

// A report about Yantr itself, with the facts a maintainer would otherwise have
// to ask for already in the body: version and build, host OS and architecture,
// Podman's version, and how many stacks are running. Those are exactly what
// "it broke on my box" is missing, and they are the same facts the dashboard
// already shows two cards over.
//
// What is deliberately not attached: hostname, public IP, user, app list. A bug
// report should not carry the machine it came from. The card lists what goes in,
// because a report that fills itself in should be able to say so.
//
// It is deliberately its own card rather than a button on the repository card:
// reporting a bug and starring a repository are different intentions, and one
// of them deserves the reader's whole attention.

const { t } = useI18n();
const { apiUrl } = useApiUrl();

const version = ref("");
const buildDate = ref("");
const host = ref("");
const podmanVersion = ref("");
const stacksRunning = ref("");

// One fetch each, once. The dashboard is not the place for a poll: the numbers
// here are context for a button, not a live reading, and the cards that do track
// something already poll it.
onMounted(async () => {
  const [versionResponse, infoResponse] = await Promise.allSettled([
    fetch(`${apiUrl.value}/api/version`),
    fetch(`${apiUrl.value}/api/system/info`),
  ]);

  if (versionResponse.status === "fulfilled" && versionResponse.value.ok) {
    const data = await versionResponse.value.json().catch(() => null);
    if (data?.version) version.value = String(data.version);
  }

  if (infoResponse.status === "fulfilled" && infoResponse.value.ok) {
    const data = await infoResponse.value.json().catch(() => null);
    const info = data?.info;
    if (info) {
      const os = [info.os?.name, info.os?.arch].filter(Boolean).join(" · ");
      const cores = info.cpu?.cores ? `${info.cpu.cores} cores` : "";
      host.value = [os, cores].filter(Boolean).join(" · ");
      if (info.podman?.version) podmanVersion.value = String(info.podman.version);
      if (Number.isFinite(info.podman?.containers?.running)) {
        stacksRunning.value = String(info.podman.containers.running);
      }
    }
  }

  const raw = import.meta.env.VITE_BUILD_TIMESTAMP;
  if (raw) buildDate.value = String(raw).split("T")[0];
});

const issueUrl = computed(() =>
  buildYantrIssueReportUrl({
    version: version.value,
    build: buildDate.value,
    host: host.value,
    podman: podmanVersion.value,
    stacks: stacksRunning.value,
  })
);

// Rows the report will actually carry. A host we could not describe is left out
// rather than shown as a dash: an empty field in a list of real ones reads as a
// fault in the card.
const contextRows = computed(() =>
  [
    { label: t("issueCard.version"), value: [version.value, buildDate.value && `build ${buildDate.value}`].filter(Boolean).join(" · ") },
    { label: t("issueCard.host"), value: host.value },
    { label: t("issueCard.podman"), value: podmanVersion.value },
    { label: t("issueCard.stacks"), value: stacksRunning.value },
  ].filter((row) => row.value)
);

const hasContext = computed(() => contextRows.value.length > 0);
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full min-h-[320px] flex-col overflow-hidden rounded-2xl p-7 text-left">
    <div class="flex w-full items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="yantra-tile yantra-tile--accent h-12 w-12 transition-transform duration-200 group-hover:scale-105">
          <Bug class="h-5 w-5" />
        </div>
        <div>
          <h3 class="yantra-text-primary text-lg font-bold leading-none tracking-tight">
            {{ t("issueCard.label") }}
          </h3>
          <p class="yantra-text-secondary mt-2 text-[10px] font-semibold uppercase tracking-widest">
            {{ t("issueCard.subtitle") }}
          </p>
        </div>
      </div>
    </div>

    <!-- Half-width body, held deliberately short: a compact headline, a
         two-line description and the facts as a 2x2 chip grid keep this card
         at its row's 320px instead of stretching the row taller than every
         neighbour. The ask sits left, the facts the report will carry sit
         right, so a reader can still check what leaves the machine. -->
    <div class="relative z-10 mt-auto grid gap-5 pt-5 sm:grid-cols-1" :class="hasContext ? 'sm:grid-cols-[minmax(0,1fr)_minmax(0,12rem)]' : ''">
      <div class="flex flex-col">
        <div class="yantra-text-primary mb-1.5 text-3xl font-black leading-[1.15] tracking-tighter transition-transform duration-200 group-hover:-translate-y-[3px]">
          {{ t("issueCard.title") }}
        </div>
        <div class="yantra-text-secondary mb-4 line-clamp-2 text-xs leading-relaxed">
          {{ t("issueCard.description") }}
        </div>

        <a
          :href="issueUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="issue-cta group/cta relative mt-auto flex h-11 w-full items-center justify-between rounded-xl px-4 transition-[border-color,box-shadow,transform] duration-200 active:scale-[0.98]"
        >
          <div class="relative z-10 flex items-center gap-2.5">
            <Bug class="h-4 w-4" />
            <span class="text-[11px] font-bold uppercase tracking-widest">{{ t("issueCard.cta") }}</span>
          </div>
          <ArrowUpRight class="relative z-10 h-4.5 w-4.5 opacity-70 transition-transform duration-200 group-hover/cta:-translate-y-1 group-hover/cta:translate-x-1 group-hover/cta:opacity-100" />
        </a>

        <p class="issue-note yantra-text-secondary mt-3 flex items-center gap-2 text-[10px] leading-relaxed">
          <ShieldCheck class="h-3 w-3 shrink-0" />
          <span>{{ t("issueCard.footnote") }}</span>
        </p>
      </div>

      <!-- What the report will carry. Shown rather than implied: a reader who
           does not want their host described should be able to see that before
           clicking. Two chip columns instead of four stacked rows — the facts
           stay one line each, the panel halves in height. -->
      <div v-if="hasContext" class="issue-context rounded-xl px-3.5 py-3">
        <p class="yantra-text-secondary mb-2 text-[9px] font-semibold uppercase tracking-widest">
          {{ t("issueCard.contextLabel") }}
        </p>
        <dl class="grid grid-cols-2 gap-x-3 gap-y-2">
          <div v-for="row in contextRows" :key="row.label" class="flex min-w-0 flex-col gap-0.5">
            <dt class="yantra-text-secondary text-[9px] font-semibold uppercase tracking-wider">
              {{ row.label }}
            </dt>
            <dd class="yantra-text-primary truncate font-mono text-[11px] font-medium" :title="row.value">
              {{ row.value }}
            </dd>
          </div>
        </dl>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Card chrome, the icon tile and every text colour come from the shared helpers
   in main.css. This card owns the context panel, the call to action and the
   footnote, and nothing else. */
.issue-context {
  background-color: var(--card-inset);
}

.issue-cta {
  background-color: var(--cta-bg);
  color: var(--cta-text);
}

.issue-cta:hover {
  background-color: var(--cta-bg-hover);
}

.issue-note {
  border-top: 1px solid var(--card-border);
  padding-top: 0.75rem;
}
</style>