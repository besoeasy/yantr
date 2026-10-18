<script>
// Two columns. The card is a list, built like one: a label, hairline-separated
// entries, nothing else — and no pictures. Every entry is two lines (the app
// with the words, then who said it and when) and there are three of them, so
// the card's shape no longer depends on how much a reviewer wrote.
export const meta = { cols: 2 };
</script>
<script setup>
import { computed, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { MessageSquareQuote, ArrowUpRight } from "@lucide/vue";
import { useNostrRelays } from "../composables/useNostrRelays";
import { appDetail, apps as appsRoute } from "../utils/routes.js";
import { useCatalog } from "../composables/useDashboardData";
import WidgetSkeleton from "../components/WidgetSkeleton.vue";
import { developerProfileUrl, developerShortKey } from "../utils/developer";
import {
  COMMENT_KIND,
  PROFILE_KIND,
  commentAppIds,
  hexToNpub,
  isAuthentic,
  isYantrComment,
  newestKind0Profiles,
  profileFilter,
  sanitizeCommentContent,
  yantrFeedFilter,
} from "../utils/nostr";
import { queryRelays } from "../utils/relay";

// The newest displayable yantr comments on any app in the catalog, as many as
// fit. The feed-wide REQ (bare `yantr` scope, any author) catches the notes;
// everything the app pages' own threads require is required here too — per-app
// tag naming a catalog app, verified signature, sanitised content, sane
// timestamps — so the surfaces can never disagree about the same note.

const { t, locale } = useI18n();
const router = useRouter();
const { apps, appsLoaded } = useCatalog();
const { relays, load: loadRelays } = useNostrRelays();

// The three newest verified displayable reviews, in order. Three is the
// card's whole shape — six lines — so there is nothing to pack and nothing to
// fill: the sort and the slice are the layout.
const SHOWCASE_COUNT = 3;
const candidates = ref([]);
// Relays answered but no comment exists: a real state, worth one line of
// copy pointing at the catalog — unlike a relay outage, which leaves the
// promotion slot to the other widgets (silent, same as the developer band).
const emptyKnown = ref(false);

// Kind-0 author profiles by lowercase hex pubkey, filled after the list
// paints — names pop in progressively, and a slow relay never holds the
// reviews hostage. The key itself is always there, so an author stays
// identifiable even if their profile never arrives.
const profiles = ref({});

function profileKey(review) {
  return String(review?.pubkey ?? "").toLowerCase();
}

function authorNpubOf(review) {
  return hexToNpub(review?.pubkey);
}

function authorNameOf(review) {
  const name = profiles.value[profileKey(review)]?.name;
  if (name) return name.trim();
  const npub = authorNpubOf(review);
  return npub ? developerShortKey(npub) : t("appComments.anonymous");
}

// The key in the author's own terms: the card names them and shows the key,
// so a reader can check who signed a note without leaving the page.
function authorShortKeyOf(review) {
  return developerShortKey(authorNpubOf(review));
}

function authorUrlOf(review) {
  return developerProfileUrl(authorNpubOf(review));
}

const relativeTime = computed(() => {
  try {
    return new Intl.RelativeTimeFormat(locale.value, { numeric: "auto" });
  } catch {
    return null;
  }
});

const TIME_UNITS = [
  ["year", 31536000],
  ["month", 2592000],
  ["week", 604800],
  ["day", 86400],
  ["hour", 3600],
  ["minute", 60],
];

// Same formatting as the app pages' comment threads — the same note must not
// say "3h ago" in one place and "3 hours ago" in the other.
function formatTime(createdAt) {
  const seconds = Number(createdAt);
  if (!Number.isFinite(seconds) || seconds <= 0) return "";

  const formatter = relativeTime.value;
  if (!formatter) {
    try {
      return new Date(seconds * 1000).toLocaleDateString();
    } catch {
      return "";
    }
  }

  const delta = seconds - Date.now() / 1000;
  for (const [unit, size] of TIME_UNITS) {
    if (Math.abs(delta) >= size) return formatter.format(Math.round(delta / size), unit);
  }
  return formatter.format(Math.round(delta), "second");
}

function viewApp(appId) {
  router.push(appDetail(appId));
}

// Author names arrive after the quotes paint: one batched kind-0 REQ, newest
// event per pubkey wins, unverified events dropped by newestKind0Profiles —
// a forged name is the same impersonation a forged comment would be.
async function fetchAuthorProfiles() {
  const wanted = [...new Set(candidates.value.map(profileKey))].filter(
    (hex) => /^[0-9a-f]{64}$/.test(hex) && !profiles.value[hex]
  );
  if (wanted.length === 0) return;

  const { events } = await queryRelays(relays.value, profileFilter(wanted), PROFILE_KIND, {
    subIdPrefix: "yantr-latest-review-profiles",
  });

  const newest = newestKind0Profiles(events);
  if (Object.keys(newest).length > 0) {
    profiles.value = { ...profiles.value, ...newest };
  }
}

async function fetchData(catalogApps) {
  const asked = await loadRelays();
  let { events, reached } = await queryRelays(asked, yantrFeedFilter(200), COMMENT_KIND, {
    subIdPrefix: "yantr-latest-review",
  });

  // One quiet retry before giving the slot up. Public relays rate-limit
  // bursts of new subscriptions, and front-loading a dashboard looks like
  // one — the same REQ a minute later answers. One attempt more, not endless
  // polling: a relay that is truly down stays down and the band stays silent.
  if (reached === 0) {
    await new Promise((resolve) => setTimeout(resolve, 4000));
    ({ events, reached } = await queryRelays(asked, yantrFeedFilter(200), COMMENT_KIND, {
      subIdPrefix: "yantr-latest-review",
    }));
  }

  // Nothing reachable is the only hard failure — silent, like the rest of
  // the promo bands. An empty merge with relays answering is a real answer.
  if (reached === 0) return;

  const appById = new Map(catalogApps.map((app) => [app?.id, app].filter(Boolean)));
  const nowSec = Math.floor(Date.now() / 1000);

  // Newest first, in display order. The app-pages rules filter each one:
  // per-app tag naming a catalog app (a note for a removed app has no page
  // to link its quote to), verified signature on the RAW event, displayable
  // sanitised content. Duplicates by content per author collapse too — two
  // relays serving the same repost would otherwise occupy two slots.
  const candidatesById = new Map();
  const seenByAuthor = new Set();
  for (const event of [...events].sort((a, b) => (b.created_at || 0) - (a.created_at || 0))) {
    if (!event || typeof event.id !== "string") continue;
    if (candidatesById.has(event.id)) continue;

    const appId = commentAppIds(event).find((id) => appById.has(id));
    if (!appId) continue;
    if (!isYantrComment(event, nowSec)) continue;
    if (!isAuthentic(event)) continue;

    const content = sanitizeCommentContent(event.content);
    if (!content) continue;

    const sameNote = `${event.pubkey}\u0000${content}`;
    if (seenByAuthor.has(sameNote)) continue;
    seenByAuthor.add(sameNote);

    candidatesById.set(event.id, { ...event, content, app: appById.get(appId) });
  }

  candidates.value = [...candidatesById.values()].slice(0, SHOWCASE_COUNT);

  if (candidates.value.length === 0) {
    emptyKnown.value = true;
    return;
  }

  fetchAuthorProfiles();
}

// The catalog is shared (useCatalog) and arrives asynchronously, so the relay
// query waits for it rather than racing it: without the catalog no app id
// resolves to a page, and a review could not be shown honestly anyway.
watch(
  [appsLoaded, apps],
  ([isLoaded, catalogApps]) => {
    if (!isLoaded) return;
    fetchData(catalogApps);
  },
  { immediate: true }
);
</script>

<template>
  <WidgetSkeleton v-if="!appsLoaded && !appsLoadFailed" />

  <section
    v-else-if="candidates.length"
    class="yantra-card yantra-card-min flex h-full w-full flex-col overflow-hidden rounded-2xl p-7 text-left transition-[border-color,box-shadow] duration-200 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <h3 class="flex items-center gap-2 text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
      <MessageSquareQuote :size="13" class="yantra-text-secondary" />
      {{ t("home.latestReview.title") }}
    </h3>

    <!-- Two lines per review, and no pictures: the app's name with the words
         on the first, who said it and when on the second. Three reviews, six
         lines — the whole list reads in one glance. Hairlines do the
         separating; there is no card inside the card. -->
    <div class="mt-4 flex flex-1 flex-col justify-center">
      <article
        v-for="(review, i) in candidates"
        :key="review.id"
        class="py-2.5"
        :class="i > 0 ? 'border-t border-zinc-100 dark:border-zinc-900' : ''"
      >
        <div class="flex min-w-0 items-baseline gap-2">
          <button
            type="button"
            @click="viewApp(review.app.id)"
            class="shrink-0 text-sm font-semibold yantra-text-primary transition-opacity hover:opacity-70"
            :title="review.app.name"
          >
            {{ review.app.name }}
          </button>
          <span class="shrink-0 yantra-text-secondary">-</span>
          <!-- One line, ellipsised: the shape of the card no longer depends
               on how long a reviewer wrote. The content was sanitised at
               fetch, before it touched the DOM. -->
          <p class="min-w-0 flex-1 truncate text-sm yantra-text-secondary">
            {{ review.content }}
          </p>
        </div>

        <div class="mt-1 flex min-w-0 items-center gap-1.5 text-[11px] yantra-text-secondary">
          <time
            v-if="formatTime(review.created_at)"
            :datetime="new Date(Number(review.created_at) * 1000).toISOString()"
            class="shrink-0"
          >
            {{ formatTime(review.created_at) }}
          </time>
          <span class="shrink-0">·</span>
          <span class="shrink-0">{{ t("home.latestReview.by") }}</span>
          <a
            :href="authorUrlOf(review)"
            target="_blank"
            rel="noopener noreferrer"
            class="min-w-0 truncate transition-colors hover:text-zinc-700 dark:hover:text-zinc-200"
            :title="t('home.latestReview.byAuthor')"
          >
            {{ authorNameOf(review) }}
            <span v-if="authorShortKeyOf(review)" class="font-mono">({{ authorShortKeyOf(review) }})</span>
          </a>
        </div>
      </article>
    </div>
  </section>

  <!-- Empty: relays answered, the catalog just has no comments yet -->
  <section
    v-else-if="emptyKnown"
    class="flex h-full w-full flex-col overflow-hidden rounded-2xl border-dashed p-7 text-left dark:border-zinc-800"
  >
    <h3 class="flex items-center gap-2 text-[10px] font-bold uppercase tracking-widest yantra-text-secondary">
      <MessageSquareQuote :size="13" />
      {{ t("home.latestReview.title") }}
    </h3>

    <p class="mt-auto max-w-sm text-sm leading-relaxed yantra-text-secondary">
      {{ t("home.latestReview.empty") }}
    </p>

    <div class="mt-4">
      <button
        type="button"
        @click="router.push(appsRoute())"
        class="inline-flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wider yantra-text-secondary transition-colors hover:text-zinc-900 dark:hover:text-zinc-100"
      >
        {{ t("nav.apps") }}
        <ArrowUpRight :size="13" />
      </button>
    </div>
  </section>
</template>