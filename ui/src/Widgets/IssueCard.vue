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

    <div class="relative z-10 mt-auto flex flex-col pt-6">
      <div class="yantra-text-primary mb-2 text-4xl font-black leading-[1.1] tracking-tighter transition-transform duration-200 group-hover:-translate-y-[3px] sm:text-5xl">
        {{ t("issueCard.title") }}
      </div>
      <div class="yantra-text-secondary mb-5 line-clamp-3 text-sm leading-relaxed">
        {{ t("issueCard.description") }}
      </div>

      <!-- What the report will carry. Shown rather than implied: a reader who
           does not want their host described should be able to see that before
           clicking, and these rows are the same facts the report body lists.
           Four rows, one line each — this card shares a dashboard row with
           cards that carry far more, and the tallest one sets the height for
           all of them. -->
      <div v-if="hasContext" class="issue-context mb-5 rounded-xl px-4 py-3">
        <p class="yantra-text-secondary mb-2 text-[9px] font-semibold uppercase tracking-widest">
          {{ t("issueCard.contextLabel") }}
        </p>
        <dl class="flex flex-col gap-1.5">
          <div v-for="row in contextRows" :key="row.label" class="flex items-baseline justify-between gap-3">
            <dt class="yantra-text-secondary shrink-0 text-[10px] font-semibold uppercase tracking-wider">
              {{ row.label }}
            </dt>
            <dd class="yantra-text-primary truncate font-mono text-[11px] font-medium" :title="row.value">
              {{ row.value }}
            </dd>
          </div>
        </dl>
      </div>

      <a
        :href="issueUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="issue-cta group/cta relative flex h-12 w-full items-center justify-between rounded-xl px-5 transition-[border-color,box-shadow,transform] duration-200 active:scale-[0.98]"
      >
        <div class="relative z-10 flex items-center gap-2.5">
          <Bug class="h-4 w-4" />
          <span class="text-[11px] font-bold uppercase tracking-widest">{{ t("issueCard.cta") }}</span>
        </div>
        <ArrowUpRight class="relative z-10 h-4.5 w-4.5 opacity-70 transition-transform duration-200 group-hover/cta:-translate-y-1 group-hover/cta:translate-x-1 group-hover/cta:opacity-100" />
      </a>

      <p class="issue-note yantra-text-secondary mt-4 flex items-center gap-2 text-[10px] leading-relaxed">
        <ShieldCheck class="h-3 w-3 shrink-0" />
        <span>{{ t("issueCard.footnote") }}</span>
      </p>
    </div>
  </div>
</template>

<style scoped>
/* Card chrome, the icon tile and every text colour come from the shared helpers
   in main.css. This card owns the context panel, the two calls to action and
   the footnote, and nothing else. */
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

.issue-ghost {
  border-color: var(--card-border);
  background-color: var(--card-bg);
  color: var(--text-secondary);
}

.issue-ghost:hover {
  border-color: var(--card-border-strong);
  background-color: var(--card-inset);
  color: var(--text-primary);
}

.issue-note {
  border-top: 1px solid var(--card-border);
  padding-top: 0.75rem;
}
</style>