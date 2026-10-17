<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import {
  DEVELOPER_LABEL,
  developersOf,
  developerInitials,
  developerProfileUrl,
  developerRouteRef,
  developerShortKey,
} from "../utils/developer";

const props = defineProps({
  app: {
    type: Object,
    default: null,
  },
});

const { t } = useI18n();

// Who writes this app, if the catalog knows. Both pages that show an app —
// the catalog page and a deployed stack's page — render this, and they render
// the same component on purpose: the credit is one fact, so the two cannot
// drift into showing it differently.
//
// Each entry links into the developer route, so a reader can see the rest of
// that person's work without leaving Yantr. `url` is that account's page, for
// the chip's tooltip.
const developers = computed(() =>
  developersOf(props.app).map((d) => ({
    ...d,
    href: `#/developer/${developerRouteRef(d.key)}`,
    url: d.url || developerProfileUrl(d.key),
    initials: developerInitials(d.key),
    shortKey: developerShortKey(d.key),
    platformLabel: DEVELOPER_LABEL,
  }))
);
</script>

<template>
  <!-- v-if sits on the root so a caller can hand this component the separator
       it wants (a rule at the end of a card) and get nothing at all — not an
       empty bordered row — when the catalog credits nobody. -->
  <div v-if="developers.length > 0" class="flex flex-wrap items-center gap-2">
    <span class="text-[10px] font-bold uppercase tracking-widest text-zinc-500">{{ t('appDetail.developer') }}</span>
    <a
      v-for="dev in developers"
      :key="dev.key"
      :href="dev.href"
      :title="`${dev.platformLabel}: ${dev.key}`"
      class="inline-flex items-center gap-1.5 rounded-full border border-zinc-200 py-0.5 pl-0.5 pr-2.5 text-[11px] font-semibold text-zinc-600 transition-colors hover:border-zinc-300 hover:text-zinc-900 dark:border-zinc-800 dark:text-zinc-400 dark:hover:border-zinc-700 dark:hover:text-zinc-100"
    >
      <span class="flex h-5 w-5 items-center justify-center rounded-full bg-zinc-900 text-[9px] font-black text-white dark:bg-zinc-100 dark:text-zinc-900">
        {{ dev.initials }}
      </span>
      {{ dev.shortKey }}
    </a>
  </div>
</template>
