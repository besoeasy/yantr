<script setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { ArrowRight, ExternalLink } from "@lucide/vue";
import AppLogo from "./AppLogo.vue";
import { stackDetail } from "../utils/routes.js";
import { appUrl, isNavigableProtocol } from "../utils/url.js";

const { t } = useI18n();
const props = defineProps({
  containers: { type: Array, default: () => [] },
  showHeader: { type: Boolean, default: false },
});

const router = useRouter();

// Group containers into stacks by projectId, and lift the app's catalog facts
// onto the group. The card used to be titled "Yantr App" — a category label
// repeated on every tile, which said nothing about the app under it. It now
// carries the app's own name, its description, its tags and the port that opens
// it: the catalog already knows all of that, and the dashboard is where a user
// asks "what is this and how do I get in".
const appGroups = computed(() => {
  const map = new Map();
  for (const c of props.containers) {
    const projectId = c.app?.projectId || c.id;
    if (!map.has(projectId)) {
      map.set(projectId, {
        projectId,
        appId: c.app?.id,
        name: c.app?.name || projectId,
        logo: c.app?.logo || null,
        description: c.app?.short_description || "",
        tags: Array.isArray(c.app?.tags) ? c.app.tags.slice(0, 3) : [],
        catalogPorts: Array.isArray(c.app?.ports) ? c.app.ports : [],
        containers: [],
      });
    }
    map.get(projectId).containers.push(c);
  }
  for (const group of map.values()) group.ports = accessPorts(group);
  return [...map.values()];
});

// A published port is only worth an "open" affordance when the catalog says a
// human is meant to look at it (`show: true`) and the protocol is one a browser
// can navigate to. The container's published port is what the host actually
// exposes; the catalog entry supplies the label ("Web UI") and that intent.
// Work ports — torrent peers, databases — are skipped rather than linked to a
// URL that cannot answer.
function accessPorts(group) {
  const ports = [];
  for (const c of group.containers) {
    const service = c.app?.service;
    for (const published of Array.isArray(c.ports) ? c.ports : []) {
      const hostPort = published?.PublicPort;
      if (!hostPort) continue;
      const spec = group.catalogPorts.find(
        (p) =>
          Number(p.port) === Number(published.PrivatePort) &&
          (!p.service || !service || p.service === service)
      );
      if (!spec || spec.show !== true || !isNavigableProtocol(spec.protocol)) continue;
      ports.push({
        key: `${c.id}-${published.PrivatePort}`,
        label: spec.label || spec.service || `:${hostPort}`,
        hostPort,
        href: appUrl(hostPort, spec.protocol),
      });
    }
  }

  // Nothing published — the stack is stopped, or it runs on the host network.
  // The catalog still knows the app has a UI, so the card names it without a
  // port and without a link, instead of claiming the app has no interface at
  // all. "Internal only" is then reserved for apps that really have none.
  if (!ports.length) {
    for (const spec of group.catalogPorts) {
      if (spec.show !== true || !isNavigableProtocol(spec.protocol)) continue;
      ports.push({
        key: `catalog-${spec.port}`,
        label: spec.label || spec.service || `:${spec.port}`,
        hostPort: null,
        href: null,
      });
    }
  }

  return ports;
}

function groupState(group) {
  const states = group.containers.map((c) => c.state);
  if (states.every((s) => s === "running")) return "running";
  if (states.some((s) => s === "running")) return "partial";
  return "stopped";
}

function stateClass(group) {
  const state = groupState(group);
  if (state === "running") return "text-emerald-600 dark:text-emerald-500";
  if (state === "partial") return "text-amber-600 dark:text-amber-500";
  return "text-zinc-500 dark:text-zinc-400";
}

function stateLabel(group) {
  const state = groupState(group);
  return t(state === "partial" ? "stackView.partial" : `stackView.${state}`);
}

function hasTemporary(group) {
  return group.containers.some((c) => c?.labels?.["yantr.expireAt"]);
}

// Without a catalog entry there is no description to show, so the services the
// stack is made of stand in for it rather than leaving the card with a hole.
function serviceSummary(group) {
  return group.containers.map((c) => c.app?.service || c.name).join(" · ");
}

function navigate(group) {
  router.push(stackDetail(group.projectId));
}
</script>

<template>
  <div style="display: contents">
    <!-- Section Header -->
    <div v-if="props.showHeader" class="col-span-full flex items-center gap-2 pt-2 pb-1">
      <span class="text-[10px] font-bold uppercase tracking-[0.2em] text-zinc-500 dark:text-zinc-400">{{ t("home.yantrApps") }}</span>
      <span class="text-[10px] font-bold text-zinc-400 dark:text-zinc-500">{{ appGroups.length }}</span>
    </div>

    <div
      v-for="(group, index) in appGroups"
      :key="group.projectId"
      :style="{ animationDelay: `${index * 50}ms` }"
      @click="navigate(group)"
      @keydown.enter.prevent="navigate(group)"
      @keydown.space.prevent="navigate(group)"
      role="button"
      tabindex="0"
      class="yantra-card group relative flex h-full w-full cursor-pointer flex-col overflow-hidden p-6 text-left transition-[border-color,box-shadow,transform] duration-200 hover:-translate-y-[3px] hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)] anim-enter focus-visible:outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-zinc-500"
    >
      <!-- Identity: the logo and the app's state. The arrow keeps its own slot
           so the state does not shift when it fades in. -->
      <div class="flex w-full items-start justify-between gap-3">
        <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 p-2.5 transition-transform duration-200 group-hover:scale-105 dark:border-zinc-800 dark:bg-zinc-900">
          <AppLogo
            :logo="group.logo"
            :name="group.name"
            :seed="group.appId || group.projectId"
            img-class="h-full w-full object-contain"
            icon-class="h-5 w-5 text-zinc-900 dark:text-zinc-100 transition-colors"
          />
        </div>

        <div class="flex min-w-0 items-center gap-2">
          <div class="flex min-w-0 items-center gap-1.5" :class="stateClass(group)">
            <span v-if="groupState(group) === 'running'" class="relative flex h-1.5 w-1.5">
              <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
              <span class="relative inline-flex h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
            </span>
            <span
              v-else
              class="h-1.5 w-1.5 rounded-full"
              :class="groupState(group) === 'partial' ? 'bg-amber-500' : 'bg-zinc-300 dark:bg-zinc-700'"
            ></span>

            <span class="truncate text-[10px] font-semibold uppercase tracking-widest">{{ stateLabel(group) }}</span>

            <span
              v-if="group.containers.length > 1"
              class="shrink-0 text-[10px] font-medium text-zinc-400 dark:text-zinc-500"
            >
              {{ t("home.yantraContainersGrid.serviceCount", { count: group.containers.length }) }}
            </span>

            <span
              v-if="hasTemporary(group)"
              class="text-[10px] font-semibold uppercase tracking-widest text-amber-600 dark:text-amber-500"
            >
              · {{ t("appOverview.expiresIn") }}
            </span>
          </div>

          <span class="flex h-6 w-6 shrink-0 items-center justify-center">
            <ArrowRight class="h-4 w-4 text-zinc-400 opacity-0 transition-opacity duration-200 group-hover:opacity-100 dark:text-zinc-500" />
          </span>
        </div>
      </div>

      <!-- The app itself: name, what it is, what it is tagged as, and how to
           reach it. -->
      <div class="relative z-10 mt-auto pt-6">
        <h3 class="line-clamp-2 text-3xl font-black leading-tight tracking-tighter text-zinc-900 transition-colors duration-200 group-hover:text-black sm:text-4xl dark:text-white">
          {{ group.name }}
        </h3>

        <p v-if="group.description" class="mt-2 line-clamp-2 text-xs leading-relaxed text-zinc-500 dark:text-zinc-400">
          {{ group.description }}
        </p>
        <p v-else class="mt-2 line-clamp-1 text-xs leading-relaxed text-zinc-500 dark:text-zinc-400">
          {{ serviceSummary(group) }}
        </p>

        <div v-if="group.tags.length" class="mt-3 flex flex-wrap gap-1.5">
          <span
            v-for="tag in group.tags"
            :key="tag"
            class="rounded-full border border-zinc-200 px-2 py-0.5 text-[9px] font-bold uppercase tracking-wider text-zinc-500 dark:border-zinc-800 dark:text-zinc-400"
          >
            {{ tag }}
          </span>
        </div>

        <div class="mt-4 flex items-center justify-between border-t border-zinc-100 pt-3 dark:border-zinc-800">
          <a
            v-if="group.ports.length"
            :href="group.ports[0].href || undefined"
            :target="group.ports[0].href ? '_blank' : undefined"
            :rel="group.ports[0].href ? 'noopener noreferrer' : undefined"
            @click.stop
            class="flex w-full items-center justify-between gap-2"
          >
            <span class="flex min-w-0 items-center gap-1.5 text-[10px] font-bold uppercase tracking-widest text-zinc-900 dark:text-white">
              <span class="truncate">{{ group.ports[0].label }}</span>
              <span v-if="group.ports.length > 1" class="shrink-0 text-zinc-400 dark:text-zinc-500">
                +{{ group.ports.length - 1 }}
              </span>
            </span>
            <span class="flex shrink-0 items-center gap-1.5 font-mono text-[11px] font-medium text-zinc-500 dark:text-zinc-400">
              <template v-if="group.ports[0].hostPort">:{{ group.ports[0].hostPort }}</template>
              <template v-else>—</template>
              <ExternalLink v-if="group.ports[0].href" class="h-3 w-3" />
            </span>
          </a>
          <span v-else class="text-[10px] font-semibold uppercase tracking-widest text-zinc-400 dark:text-zinc-500">
            {{ t("home.yantraContainersGrid.internalOnly") }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>
