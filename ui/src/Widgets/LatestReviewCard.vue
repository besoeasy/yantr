<script>
// Two columns: a quote needs width to breathe but not a full row — three
// full-width bands already lead the dashboard, and this card sits beside the
// sibling content slot as testimonial. Promotion, like the spotlight bands.
export const meta = { cols: 2 };
</script>
<script setup>
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { MessageSquareQuote, ArrowUpRight } from "@lucide/vue";
import AppLogo from "../components/AppLogo.vue";
import { useApiUrl } from "../composables/useApiUrl";
import { useNostrRelays } from "../composables/useNostrRelays";
import { expectApiSuccess } from "../composables/useApiResponse";
import { appDetail, apps as appsRoute } from "../utils/routes.js";
import { developerProfileUrl, developerShortKey } from "../utils/developer";
import { linesFor, showcaseReviews } from "../utils/reviewShowcase";
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

// Latest reviews across the whole catalog: the newest displayable yantr
// comments on any app, as many as the band's line budget lets them occupy —
// list enough short ones, showcase a long one nearly alone (utils/
// reviewShowcase). The feed-wide REQ (bare `yantr` scope, any author) catches
// the notes; everything the app pages' own threads require is required here
// too — per-app tag naming a catalog app, verified signature, sanitised
// content, sane timestamps — so the surfaces can never disagree about the
// same note.

const { t, locale } = useI18n();
const router = useRouter();
const { apiUrl } = useApiUrl();
const { relays, load: loadRelays } = useNostrRelays();

// Verified displayable reviews, newest first. Ten is enough backfill for the
// budget packing: the showcase takes at most four, and relays already cap the
// feed read at 200 — this slice only stops the sort from growing past use.
const candidates = ref([]);
// Relays answered but no comment exists: a real state, worth one line of
// copy pointing at the catalog — unlike a relay outage, which leaves the
// promotion slot to the other widgets (silent, same as the developer band).
const emptyKnown = ref(false);

// Kind-0 author profiles by lowercase hex pubkey, filled after the quotes
// paint — names pop in progressively, and a slow relay never holds the
// showcase hostage.
const profiles = ref({});

const showcase = computed(() => {
  const picked = new Set(showcaseReviews(candidates.value));
  return candidates.value.filter((r) => picked.has(r.id));
});

function authorNpubOf(review) {
  return hexToNpub(review?.pubkey);
}

function profileKey(review) {
  return String(review?.pubkey ?? "").toLowerCase();
}

function authorNameOf(review) {
  const profile = profiles.value[profileKey(review)];
  const name = profile?.name;
  if (name) return name.trim();
  const npub = authorNpubOf(review);
  return npub ? developerShortKey(npub) : t("appComments.anonymous");
}

function authorUrlOf(review) {
  return developerProfileUrl(authorNpubOf(review));
}

// Every quote is clamped to its measured line count, which is also what the
// budget counted — display and estimate must agree.
function clampClassFor(review) {
  return `line-clamp-${linesFor(review.content)}`;
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
  const wanted = [
    ...new Set(showcase.value.map(profileKey)),
  ].filter((hex) => /^[0-9a-f]{64}$/.test(hex) && !profiles.value[hex]);
  if (wanted.length === 0) return;

  const { events } = await queryRelays(
    relays.value,
    profileFilter(wanted),
    PROFILE_KIND,
    { subIdPrefix: "yantr-latest-review-profiles" }
  );

  const newest = newestKind0Profiles(events);
  if (Object.keys(newest).length > 0) {
    profiles.value = { ...profiles.value, ...newest };
  }
}

async function fetchData() {
  let catalogApps;
  try {
    const res = await fetch(`${apiUrl.value}/api/apps`);
    const data = await expectApiSuccess(res, "Failed to load apps");
    catalogApps = Array.isArray(data.apps) ? data.apps : [];
  } catch {
    // Catalog failure hides the card: without it no app id resolves to a
    // page, so no review could be shown honestly.
    return;
  }

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

  candidates.value = [...candidatesById.values()].slice(0, 10);

  if (candidates.value.length === 0) {
    emptyKnown.value = true;
    return;
  }

  fetchAuthorProfiles();
}

onMounted(fetchData);
</script>

<template>
  <section
    v-if="showcase.length"
    class="group relative flex h-full w-full flex-col overflow-hidden rounded-2xl border border-zinc-200 bg-white p-7 text-left transition-[border-color,box-shadow] duration-200 hover:border-zinc-300 hover:shadow-[0_8px_30px_rgb(0,0,0,0.04)] dark:border-zinc-800 dark:bg-black dark:hover:border-zinc-700 dark:hover:shadow-[0_8px_30px_rgb(255,255,255,0.02)]"
  >
    <!-- Header: what this card is -->
    <div class="flex w-full items-start justify-between gap-4">
      <div class="flex items-center gap-4">
        <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-sky-200 bg-sky-50 transition-transform duration-200 group-hover:scale-105 dark:border-sky-900/50 dark:bg-sky-900/20">
          <MessageSquareQuote class="h-5 w-5 text-sky-600 dark:text-sky-400" />
        </div>
        <h3 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500 dark:text-zinc-400">
          {{ t("home.latestReview.title") }}
        </h3>
      </div>
    </div>

    <!-- One row per review, sharing the band's height evenly. Hairlines
         separate the quotes; each keeps the shape from the sketch — app
         chip first, then the quote, then who said it. -->
    <div class="flex flex-1 flex-col pt-1">
      <article
        v-for="(review, i) in showcase"
        :key="review.id"
        class="flex flex-1 flex-col justify-center"
        :class="i > 0 ? 'mt-5 border-t border-zinc-100 pt-5 dark:border-zinc-900' : ''"
      >
        <div class="flex w-full items-center justify-between gap-4">
          <button
            type="button"
            @click="viewApp(review.app.id)"
            class="inline-flex min-w-0 items-center gap-2 rounded-full border border-zinc-200 bg-zinc-50 py-1 pl-1 pr-3 text-xs font-medium text-zinc-700 transition-colors hover:border-zinc-300 hover:bg-white hover:text-zinc-900 dark:border-zinc-800 dark:bg-zinc-900/50 dark:text-zinc-300 dark:hover:border-zinc-700 dark:hover:bg-zinc-900 dark:hover:text-zinc-100"
            :title="review.app.name"
          >
            <span class="flex h-5 w-5 shrink-0 items-center justify-center overflow-hidden rounded-full">
              <AppLogo
                :logo="review.app.logo"
                :name="review.app.name"
                :seed="review.app.id"
                img-class="h-full w-full object-contain"
                icon-class="h-4 w-4 text-zinc-300 dark:text-zinc-700"
              />
            </span>
            <span class="truncate">{{ review.app.name }}</span>
          </button>

          <time
            v-if="formatTime(review.created_at)"
            :datetime="new Date(Number(review.created_at) * 1000).toISOString()"
            class="shrink-0 font-mono text-[11px] text-zinc-400"
          >
            {{ formatTime(review.created_at) }}
          </time>
        </div>

        <!-- The quote. whitespace-pre-wrap keeps the author's line breaks;
             the content was sanitised at fetch, before it touched the DOM. -->
        <blockquote
          class="mt-3 font-medium tracking-tight text-zinc-900 whitespace-pre-wrap dark:text-white"
          :class="[showcase.length === 1 ? 'text-lg leading-snug' : 'text-sm leading-relaxed', clampClassFor(review)]"
        >
          {{ review.content }}
        </blockquote>

        <a
          :href="authorUrlOf(review)"
          target="_blank"
          rel="noopener noreferrer"
          class="mt-2 inline-flex w-fit items-center gap-1.5 text-[11px] font-medium text-zinc-500 transition-colors hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
          :title="t('home.latestReview.byAuthor')"
        >
          <ArrowUpRight :size="12" class="shrink-0" />
          {{ authorNameOf(review) }}
        </a>
      </article>
    </div>
  </section>

  <!-- Empty: relays answered, the catalog just has no comments yet -->
  <section
    v-else-if="emptyKnown"
    class="flex h-full w-full flex-col overflow-hidden rounded-2xl border border-dashed border-zinc-200 bg-white p-7 text-left dark:border-zinc-800 dark:bg-black"
  >
    <div class="flex items-center gap-4">
      <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-zinc-200 bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-900">
        <MessageSquareQuote class="h-5 w-5 text-zinc-400 dark:text-zinc-600" />
      </div>
      <h3 class="text-[10px] font-bold uppercase tracking-widest text-zinc-500 dark:text-zinc-400">
        {{ t("home.latestReview.title") }}
      </h3>
    </div>

    <div class="mt-auto flex flex-col pb-1">
      <p class="max-w-sm text-sm leading-relaxed text-zinc-500 dark:text-zinc-400">
        {{ t("home.latestReview.empty") }}
      </p>
      <div class="mt-4">
        <button
          type="button"
          @click="router.push(appsRoute())"
          class="inline-flex items-center gap-1.5 rounded-full bg-zinc-900 px-3.5 py-1.5 text-[11px] font-semibold text-white transition-opacity hover:opacity-90 dark:bg-zinc-100 dark:text-zinc-900"
        >
          {{ t("nav.apps") }}
          <ArrowUpRight :size="13" />
        </button>
      </div>
    </div>
  </section>
</template>
