<script>
// Dashboard engine: two of the desktop grid's four columns — the ask and the
// CTA share a row, the facts the report carries get one thin strip, so the
// card never has to spend height on a landing-page stack. The .w-span-2
// utility in main.css clamps it per breakpoint; see Home.vue.
export const meta = { cols: 2 };
</script>
<script setup>
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { Bug, ArrowUpRight, ShieldCheck } from "@lucide/vue";
import { useApiUrl } from "../composables/useApiUrl";
import { buildYantrIssueReportUrl } from "../utils/developer";

// A report about Yantr itself, with the facts a maintainer would otherwise
// have to ask for already in the body: version and build, host OS and
// architecture, Podman's version, and how many stacks are running. Those are
// exactly what "it broke on my box" is missing, and they are the same facts
// the dashboard already shows two cards over.
//
// What is deliberately not attached: hostname, public IP, user, app list. A
// bug report should not carry the machine it came from. The card lists what
// goes in, because a report that fills itself in should be able to say so.
//
// It is deliberately its own card rather than a button on the repository
// card: reporting a bug and starring a repository are different intentions,
// and one of them deserves the reader's whole attention.

const { t } = useI18n();
const { apiUrl } = useApiUrl();

const version = ref("");
const buildDate = ref("");
const host = ref("");
const podmanVersion = ref("");
const stacksRunning = ref("");

// One fetch each, once. The dashboard is not the place for a poll: the numbers
// here are context for a button, not a live reading, and the cards that do
// track something already poll it.
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

// One cell per fact. A host we could not describe is left out rather than
// shown as a dash: an empty field in a list of real ones reads as a fault in
// the card.
const facts = computed(() =>
  [
    { label: t("issueCard.version"), value: [version.value, buildDate.value && `build ${buildDate.value}`].filter(Boolean).join(" · ") },
    { label: t("issueCard.host"), value: host.value },
    { label: t("issueCard.podman"), value: podmanVersion.value },
    { label: t("issueCard.stacks"), value: stacksRunning.value },
  ].filter((fact) => fact.value)
);

const hasFacts = computed(() => facts.value.length > 0);
</script>

<template>
  <div class="yantra-card group relative flex h-full w-full yantra-card-min flex-col overflow-hidden rounded-2xl p-7 text-left">
    <!-- Header: identity only. The ask lives with the CTA below it. -->
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

    <!-- Body: ask left, act right, evidence underneath. Held short so the row
         keeps its 320px like every neighbour card. -->
    <div class="relative z-10 mt-auto flex flex-col gap-4 pt-5">
      <div class="flex flex-wrap items-end justify-between gap-x-8 gap-y-4">
        <div class="min-w-0 max-w-[38ch] flex-1">
          <h3 class="yantra-text-primary mb-1.5 text-2xl font-black leading-[1.2] tracking-tight transition-transform duration-200 group-hover:-translate-y-[3px]">
            {{ t("issueCard.title") }}
          </h3>
          <p class="yantra-text-secondary line-clamp-2 text-xs leading-relaxed">
            {{ t("issueCard.description") }}
          </p>
        </div>

        <a
          :href="issueUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="report-cta group/cta relative flex h-11 shrink-0 items-center justify-between gap-3 rounded-xl px-5 transition-[border-color,box-shadow,transform] duration-200 active:scale-[0.98]"
        >
          <div class="relative z-10 flex items-center gap-2.5">
            <Bug class="h-4 w-4" />
            <span class="text-[11px] font-bold uppercase tracking-widest">{{ t("issueCard.cta") }}</span>
          </div>
          <ArrowUpRight class="relative z-10 h-4.5 w-4.5 opacity-70 transition-transform duration-200 group-hover/cta:-translate-y-1 group-hover/cta:translate-x-1 group-hover/cta:opacity-100" />
        </a>
      </div>

      <!-- What the report will carry, as one thin strip of cells: a reader who
           does not want their host described can check what leaves the machine
           without the card paying a stacked list for it. -->
      <div
        v-if="hasFacts"
        class="report-facts grid grid-cols-2 gap-x-4 gap-y-2 rounded-xl px-3.5 py-2.5 sm:grid-cols-4"
      >
        <div v-for="fact in facts" :key="fact.label" class="flex min-w-0 flex-col gap-0.5">
          <span class="yantra-text-secondary text-[9px] font-semibold uppercase tracking-wider">
            {{ fact.label }}
          </span>
          <span class="yantra-text-primary truncate font-mono text-[11px] font-medium" :title="fact.value">
            {{ fact.value }}
          </span>
        </div>
      </div>

      <div class="report-foot yantra-text-secondary flex items-center justify-between gap-4 pt-3 text-[10px] leading-relaxed">
        <span class="flex min-w-0 items-center gap-2">
          <ShieldCheck class="h-3 w-3 shrink-0" />
          <span class="truncate">{{ t("issueCard.footnote") }}</span>
        </span>
        <span v-if="hasFacts" class="yantra-text-secondary shrink-0 text-[9px] font-semibold uppercase tracking-widest">
          {{ t("issueCard.contextLabel") }}
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Card chrome, the icon tile and every text colour come from the shared
   helpers in main.css. This card owns its CTA, the facts strip and the
   hairline footer, and nothing else. */
.report-cta {
  background-color: var(--cta-bg);
  color: var(--cta-text);
}

.report-cta:hover {
  background-color: var(--cta-bg-hover);
}

.report-facts {
  background-color: var(--card-inset);
}

.report-foot {
  border-top: 1px solid var(--card-border);
}
</style>
