<script setup>
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { RefreshCw, Send, KeyRound, AlertCircle, ExternalLink, Info } from "@lucide/vue";
import { useAppComments } from "../composables/useAppComments";
import { developerInitials, developerProfileUrl, developerShortKey, developersOf } from "../utils/developer";
import { hexToNpub, npubToHex } from "../utils/nostr";

const props = defineProps({
  app: { type: Object, required: true },
});

const { t, locale } = useI18n();

const appId = computed(() => String(props.app?.id ?? "").trim().toLowerCase());

// The app's credited Nostr keys as relay-hex. A comment whose pubkey lands in
// this set is the developer speaking, and renders with a badge. Decoded (not
// shape-checked): a badge on the wrong account is worse than no badge.
const developerHexes = computed(() => {
  const set = new Set();
  for (const d of developersOf(props.app)) {
    const hex = npubToHex(d.key);
    if (hex) set.add(hex);
  }
  return set;
});

function isDeveloperComment(event) {
  return developerHexes.value.has(String(event?.pubkey ?? "").toLowerCase());
}

// The connected signer is one of the app's developers: posting speaks with
// the badge.
const postingAsDeveloper = computed(
  () => !!pubkeyHex.value && developerHexes.value.has(pubkeyHex.value)
);

const {
  comments,
  loading,
  loadFailed,
  relaysReached,
  relaysTotal,
  publishing,
  publishError,
  pubkeyHex,
  signerMissing,
  shortAuthor,
  maxLength,
  load,
  connect,
  publish,
  profileOf,
  eventTag,
} = useAppComments(appId.value);

const draft = ref("");
const published = ref(false);

const remaining = computed(() => maxLength - draft.value.length);
const canPost = computed(
  () => draft.value.trim().length > 0 && draft.value.length <= maxLength && !publishing.value
);

const errorText = computed(() => {
  switch (publishError.value) {
    case "empty":
      return t("appComments.emptyComment");
    case "too-long":
      return t("appComments.tooLong", { max: maxLength });
    case "sign-cancelled":
      return t("appComments.signCancelled");
    case "sign-error":
    case "signer-error":
      return t("appComments.signerError");
    case "publish-failed":
      return t("appComments.publishFailed");
    default:
      return publishError.value ? t("appComments.publishFailed") : "";
  }
});

// The meta line is a caption, so the time in it is relative: "2 days ago" reads
// at a glance where a full locale timestamp ("10/15/2026, 9:26:46 AM") is a
// second sentence competing with the name. The exact stamp is kept in the
// element's title, so nothing is lost — only moved.
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

  const delta = seconds - Date.now() / 1000; // negative for the past
  for (const [unit, size] of TIME_UNITS) {
    if (Math.abs(delta) >= size) return formatter.format(Math.round(delta / size), unit);
  }
  return formatter.format(Math.round(delta), "second");
}

function exactTime(createdAt) {
  try {
    return new Date(Number(createdAt) * 1000).toLocaleString();
  } catch {
    return "";
  }
}

function authorNpubOf(event) {
  return hexToNpub(event?.pubkey);
}

function authorShortOf(event) {
  const npub = authorNpubOf(event);
  return npub ? developerShortKey(npub) : t("appComments.anonymous");
}

function authorUrlOf(event) {
  return developerProfileUrl(authorNpubOf(event));
}

// Author identity, progressively enriched: the relay's kind-0 name once it
// arrives, the abbreviated npub until then. Avatar likewise, with a
// per-author fallback to initials when the picture is missing or fails to
// load — a relay-supplied URL owes nothing.
const brokenAvatars = ref(new Set());

function displayNameOf(event) {
  const name = profileOf(event)?.name;
  if (name) return name;
  return authorShortOf(event);
}

function avatarOf(event) {
  const url = profileOf(event)?.picture || "";
  if (!url) return "";
  if (brokenAvatars.value.has(String(event?.pubkey ?? "").toLowerCase())) return "";
  return url;
}

function markAvatarBroken(event) {
  brokenAvatars.value = new Set([
    ...brokenAvatars.value,
    String(event?.pubkey ?? "").toLowerCase(),
  ]);
}

function nip05Of(event) {
  return profileOf(event)?.nip05 || "";
}

async function submit() {
  published.value = false;
  const ok = await publish(draft.value);
  if (ok) {
    draft.value = "";
    published.value = true;
  }
}

onMounted(() => {
  if (appId.value) load();
});

// Navigating between two app pages reuses the view, so the fetch cannot live
// only in onMounted.
watch(appId, () => {
  draft.value = "";
  published.value = false;
  pubkeyHex.value = null;
  brokenAvatars.value = new Set();
  if (appId.value) load();
});
</script>

<template>
  <!-- No card. The section is separated from the deploy form by one hairline,
       and everything inside it is type and space: hairline dividers between
       comments would restate a boundary the gaps already draw, and a bordered
       box around the composer would draw a second one. -->
  <section class="border-t border-zinc-200 pt-6 dark:border-zinc-800">
    <!-- One measured column. The hairline separates the section from the deploy
         form above it; the measure is what keeps a comment readable instead of
         letting prose run the width of the page. -->
    <div class="max-w-2xl">
      <div class="flex items-center justify-between gap-3">
        <div class="flex items-baseline gap-2">
          <h3 class="text-sm font-semibold tracking-tight text-zinc-900 dark:text-zinc-100">
            {{ t("appComments.title") }}
          </h3>
          <span v-if="!loading" class="text-[11px] tabular-nums text-zinc-400">
            {{ comments.length }}
          </span>
        </div>
        <button
          @click="load"
          :disabled="loading"
          class="inline-flex items-center gap-1.5 rounded-lg px-2 py-1 text-[11px] font-medium text-zinc-500 transition-colors hover:bg-zinc-100 hover:text-zinc-900 disabled:opacity-50 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-white"
        >
          <RefreshCw :size="13" :class="{ 'animate-spin': loading }" />
          {{ t("appComments.refresh") }}
        </button>
      </div>

      <p class="mt-2 text-[11px] leading-relaxed text-zinc-400">
        {{ t("appComments.subtitle", { tag: `#yantr #${eventTag()}` }) }}
      </p>

      <!-- Loading. Two skeleton rows rather than a spinner: the list keeps its
           shape, so nothing jumps when the relays answer. -->
      <div v-if="loading" class="mt-6 flex flex-col gap-6">
        <span class="a11y-visually-hidden" aria-live="polite">{{ t("appComments.loading") }}</span>
        <div v-for="i in 2" :key="i" class="flex animate-pulse gap-3" aria-hidden="true">
          <span class="h-7 w-7 shrink-0 rounded-full bg-zinc-100 dark:bg-zinc-800"></span>
          <span class="mt-0.5 flex-1 space-y-2">
            <span class="block h-2.5 w-2/5 rounded-full bg-zinc-100 dark:bg-zinc-800"></span>
            <span class="block h-2.5 w-full rounded-full bg-zinc-100 dark:bg-zinc-800"></span>
          </span>
        </div>
      </div>

      <template v-else>
        <!-- Relays unreachable -->
        <p v-if="loadFailed" class="mt-6 flex items-start gap-2 text-xs leading-relaxed text-zinc-500">
          <AlertCircle :size="14" class="mt-0.5 shrink-0" />
          <span>{{ t("appComments.loadFailed") }}</span>
        </p>

        <!-- Empty state -->
        <p v-else-if="comments.length === 0" class="mt-6 text-sm leading-relaxed text-zinc-500">
          {{ t("appComments.empty") }}
        </p>

        <!-- Comment list. Content renders as plain text (interpolation escapes),
             never v-html — and it arrives pre-sanitised from useAppComments
             (controls, bidi spoofing and overlong payloads removed), so a relay
             serving hostile bytes has no path to script or to visual spoofing. -->
        <ul v-else class="mt-6 flex flex-col gap-6">
          <li v-for="c in comments" :key="c.id" class="flex gap-3">
            <img
              v-if="avatarOf(c)"
              :src="avatarOf(c)"
              :alt="displayNameOf(c)"
              loading="lazy"
              referrerpolicy="no-referrer"
              @error="markAvatarBroken(c)"
              class="h-7 w-7 shrink-0 rounded-full bg-zinc-100 object-cover dark:bg-zinc-800"
            />
            <span
              v-else
              class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-zinc-100 text-[10px] font-semibold text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400"
            >
              {{ developerInitials(displayNameOf(c)) }}
            </span>
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-baseline gap-2">
                <a
                  v-if="authorUrlOf(c)"
                  :href="authorUrlOf(c)"
                  target="_blank"
                  rel="noopener noreferrer"
                  :title="authorNpubOf(c)"
                  class="inline-flex min-w-0 items-center gap-1 text-[13px] font-semibold text-zinc-900 dark:text-zinc-100"
                >
                  <span class="truncate">{{ displayNameOf(c) }}</span>
                  <ExternalLink :size="11" class="shrink-0 text-zinc-400" />
                </a>
                <span v-else class="min-w-0 truncate text-[13px] font-semibold text-zinc-900 dark:text-zinc-100">
                  {{ displayNameOf(c) }}
                </span>
                <!-- The developer credit is a dot and a word, not a filled pill:
                     it should read as a caption on the name, not as a second
                     heading competing with it. -->
                <span
                  v-if="isDeveloperComment(c)"
                  :title="t('appComments.developerBadgeHint')"
                  class="inline-flex shrink-0 items-center gap-1.5 text-[10px] font-semibold uppercase tracking-widest text-zinc-500"
                >
                  <span class="h-1 w-1 shrink-0 rounded-full bg-sky-500"></span>
                  {{ t("appComments.developerBadge") }}
                </span>
                <span v-if="nip05Of(c)" class="truncate font-mono text-[10px] text-zinc-400">
                  {{ nip05Of(c) }}
                </span>
                <span class="shrink-0 text-[11px] tabular-nums text-zinc-400" :title="exactTime(c.created_at)">
                  {{ formatTime(c.created_at) }}
                </span>
              </div>
              <p class="mt-1.5 whitespace-pre-wrap break-words text-sm leading-relaxed text-zinc-600 dark:text-zinc-400">
                {{ c.content }}
              </p>
            </div>
          </li>
        </ul>
      </template>

      <!-- Composer. The bubble is the field: a borderless textarea on a tonal
           surface, so the only thing drawn is the surface itself. -->
      <div class="mt-6">
        <div v-if="!pubkeyHex">
          <button
            @click="connect"
            class="inline-flex items-center gap-1.5 rounded-full border border-zinc-200 px-3.5 py-1.5 text-[11px] font-semibold text-zinc-700 hover:bg-zinc-50 dark:border-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-900"
          >
            <KeyRound :size="13" />
            {{ t("appComments.connect") }}
          </button>
          <p class="mt-2 text-[11px] leading-relaxed text-zinc-500">
            {{ t("appComments.connectHint") }}
          </p>
          <p
            v-if="signerMissing"
            class="mt-2 flex items-start gap-2 text-[11px] leading-relaxed text-zinc-500"
          >
            <AlertCircle :size="13" class="mt-0.5 shrink-0" />
            <span>{{ t("appComments.noSigner") }}</span>
          </p>
        </div>

        <div v-else class="composer rounded-2xl bg-zinc-50 p-4 dark:bg-zinc-900/50">
          <p class="font-mono text-[10px] text-zinc-400">
            {{ t("appComments.connectedAs", { key: shortAuthor }) }}
          </p>
          <p
            v-if="postingAsDeveloper"
            class="mt-1.5 flex items-center gap-1.5 text-[11px] font-medium text-zinc-500"
          >
            <span class="h-1 w-1 shrink-0 rounded-full bg-sky-500"></span>
            {{ t("appComments.postingAsDeveloper") }}
          </p>
          <label :for="`comment-${appId}`" class="a11y-visually-hidden">
            {{ t("appComments.placeholder") }}
          </label>
          <textarea
            :id="`comment-${appId}`"
            v-model="draft"
            :maxlength="maxLength"
            rows="3"
            :placeholder="t('appComments.placeholder')"
            class="mt-3 w-full bg-transparent text-sm leading-relaxed text-zinc-900 placeholder:text-zinc-400 focus:outline-none dark:text-zinc-100"
          />
          <div class="mt-3 flex flex-wrap items-center justify-between gap-3">
            <span class="font-mono text-[10px] text-zinc-400">{{ remaining }}</span>
            <button
              @click="submit"
              :disabled="!canPost"
              class="inline-flex h-8 items-center gap-1.5 rounded-full bg-zinc-900 px-3.5 text-[11px] font-semibold text-white transition-colors hover:opacity-90 disabled:opacity-40 dark:bg-zinc-100 dark:text-zinc-900"
            >
              <Send :size="13" />
              {{ publishing ? t("appComments.posting") : t("appComments.post") }}
            </button>
          </div>
          <p v-if="errorText" class="mt-3 flex items-start gap-2 text-[11px] leading-relaxed text-red-500">
            <AlertCircle :size="13" class="mt-0.5 shrink-0" />
            <span>{{ errorText }}</span>
          </p>
          <p v-if="published" class="mt-3 text-[11px] leading-relaxed text-green-600 dark:text-green-400">
            {{ t("appComments.published") }}
          </p>
        </div>

        <!-- The fine print, in one quiet line rather than three. -->
        <p class="mt-5 flex items-start gap-2 text-[11px] leading-relaxed text-zinc-400">
          <Info :size="13" class="mt-0.5 shrink-0" />
          <span>
            {{ t("appComments.expiryNote") }}
            <template v-if="!loading && !loadFailed">
              <span class="text-zinc-300 dark:text-zinc-600"> · </span>
              {{ t("appComments.relayStatus", { reached: relaysReached, total: relaysTotal }) }}
            </template>
          </span>
        </p>
      </div>
    </div>
  </section>
</template>

<style scoped>
/* The bubble is the field, so the textarea inside it must not draw one of its
   own: main.css rings every focused textarea, which would outline a control
   the reader is not looking at. The wrapper answers the focus instead. */
.composer textarea {
  resize: none;
}

.composer textarea:focus {
  border-color: transparent;
  box-shadow: none;
}

.composer:focus-within {
  background-color: var(--color-zinc-100);
}

.dark .composer:focus-within {
  background-color: var(--color-zinc-800);
}
</style>
