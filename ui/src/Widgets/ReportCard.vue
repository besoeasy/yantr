<script>
// Dashboard engine: one of the desktop grid's four columns, spent entirely on
// the ask and the CTA. The facts the report carries are not rendered at all —
// they ride in the button's href, pre-filled into the issue body, so the card
// spends its pixels on the ask instead of restating what the link already
// embeds. The ask and the CTA stack rather than sit side by side — at one
// column they are a sentence followed by the button that answers it; see
// Home.vue.
export const meta = { cols: 1 };
</script>
<script setup>
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { Bug, ArrowUpRight, ShieldCheck } from "@lucide/vue";
import { useApiUrl } from "../composables/useApiUrl";
import { buildYantrIssueReportUrl } from "../utils/developer";

// A report about Yantr itself, with the facts a maintainer would otherwise
// have to ask for already in the body of the issue the button opens: version
// and build, host OS and architecture, Podman's version, and how many stacks
// are running. Those are exactly what "it broke on my box" is missing, and
// the built report is the only place they appear — the card does not restate
// them, because a list of machine facts is display space the ask deserves.
//
// What is deliberately not attached: hostname, public IP, user, app list. A
// bug report should not carry the machine it came from, and nothing leaves
// until the reader clicks through — the footnote on the card says so, which
// is the consent story a filled-in report still has to carry.
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

// One fetch each, once, purely so the link can quote them. The dashboard is
// not the place for a poll: these are context for a report body, not a live
// reading, and the cards that do track something already poll it.
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
</script>

<template>
  <div class="yantra-card yantra-card-min group relative flex h-full w-full flex-col items-center overflow-hidden p-7 text-center">
    <!-- A quiet ask: one icon, one line, one action. The facts the report
         carries ride in the button's href, so nothing else earns pixels. -->
    <div class="yantra-tile yantra-tile--accent h-12 w-12 transition-transform duration-200 group-hover:scale-105">
      <Bug class="h-5 w-5" />
    </div>

    <h3 class="yantra-text-primary mt-4 text-xl font-black leading-tight tracking-tight">
      {{ t("issueCard.title") }}
    </h3>
    <p class="yantra-text-secondary mt-2 max-w-[250px] text-[11px] leading-relaxed line-clamp-2">
      {{ t("issueCard.description") }}
    </p>

    <!-- CTA spans the card: the one action, no competing button. -->
    <a
      :href="issueUrl"
      target="_blank"
      rel="noopener noreferrer"
      class="report-cta group/cta relative mt-5 mt-auto flex h-10 w-full items-center justify-center gap-2 rounded-xl px-5 transition-[border-color,box-shadow,transform] duration-200 active:scale-[0.98]"
    >
      <Bug class="h-4 w-4" />
      <span class="text-[10px] font-bold uppercase tracking-widest">{{ t("issueCard.cta") }}</span>
      <ArrowUpRight class="h-4 w-4 opacity-70 transition-transform duration-200 group-hover/cta:-translate-y-1 group-hover/cta:translate-x-1 group-hover/cta:opacity-100" />
    </a>

    <div class="report-foot yantra-text-secondary mt-4 flex items-center justify-center gap-2 text-[9px] uppercase tracking-widest">
      <ShieldCheck class="h-3 w-3 shrink-0" />
      <span>{{ t("issueCard.footnote") }}</span>
    </div>
  </div>
</template>

<style scoped>
/* Card chrome, the icon tile and every text colour come from the shared
   helpers in main.css. This card owns its CTA and the hairline footer, and
   nothing else. */
.report-cta {
  background-color: var(--cta-bg);
  color: var(--cta-text);
}

.report-cta:hover {
  background-color: var(--cta-bg-hover);
}

.report-foot {
  border-top: 1px solid var(--card-border);
}
</style>
