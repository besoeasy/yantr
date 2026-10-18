import { computed, ref } from "vue";
import {
  COMMENT_KIND,
  COMMENT_MAX_LENGTH,
  PROFILE_KIND,
  appCommentTag,
  buildCommentTags,
  commentFilter,
  hexToNpub,
  isAppComment,
  isAuthentic,
  newestKind0Profiles,
  profileFilter,
  sanitizeCommentContent,
} from "../utils/nostr";
import { publishToRelay, queryRelays } from "../utils/relay";
import { useNostrRelays } from "./useNostrRelays";
import {
  NOSTR_SCOPE,
  readCachedProfiles,
  readNostrQuery,
  writeCachedProfile,
  writeNostrFailure,
  writeNostrQuery,
} from "../utils/nostrCache";
import { developerShortKey } from "../utils/developer";

// Everything the comments section needs: one relay fan-out for reading, and
// the reader's own NIP-07 signer for writing. The section shows at the foot of
// an app's page and at the foot of a running stack's page; both address the
// same app by the same tag, so both see the same notes.
//
// Comments live on public relays, never on this host. Reading opens one
// NIP-01 REQ per relay and merges what arrives; publishing signs locally
// through `window.nostr` (Alby, nos2x, …) and broadcasts the signed event.
// No key material touches Yantr either way.
//
// A factory rather than module-level state, like useDeveloperProfile: the app
// id changes on every navigation and two mounted sections must not share a
// list.

export function useAppComments(appId) {
  const id = computed(() => String(appId ?? "").trim().toLowerCase());
  const { relays, load: loadRelays } = useNostrRelays();

  const comments = ref([]);
  const loading = ref(false);
  const loadFailed = ref(false);
  const relaysReached = ref(0);
  // How many relays were asked, which is whatever the host is configured with
  // (useNostrRelays) rather than a number baked into the bundle.
  const relaysTotal = ref(relays.value.length);

  // Kind-0 author profiles by lowercase hex pubkey. Filled after the comments
  // render, so names pop in progressively; cached per session so navigating
  // between apps never refetches an author twice.
  const profiles = ref({});

  const publishing = ref(false);
  const publishError = ref("");
  const justPublishedId = ref("");

  // The connected NIP-07 account, as hex from the signer. Null until the
  // reader connects; the composer renders the connect button instead.
  const pubkeyHex = ref(null);
  const signerMissing = ref(false);

  const npub = computed(() => (pubkeyHex.value ? hexToNpub(pubkeyHex.value) : ""));
  const shortAuthor = computed(() =>
    npub.value ? developerShortKey(npub.value) : ""
  );
  const sortedComments = computed(() =>
    [...comments.value].sort((a, b) => (b.created_at || 0) - (a.created_at || 0))
  );

  function reset() {
    comments.value = [];
    loadFailed.value = false;
    relaysReached.value = 0;
    publishError.value = "";
    justPublishedId.value = "";
  }

  // load fans the comment filter out to every relay and merges what arrives.
  // First answer does not win here, unlike the kind-0 profile fetch: comments
  // are a set, and every relay may hold ones the others do not.
  async function load() {
    if (!id.value) return;
    reset();
    loading.value = true;

    // A fresh record from this reader's own machine is the answer: the notes
    // were signature-verified and sanitised before they were written, and the
    // catalogue half of this page is already on screen. Asking three relays
    // again for it would buy nothing and re-show the composer's skeleton.
    const cached = await readNostrQuery(NOSTR_SCOPE.appComments, id.value);
    if (cached?.failed) {
      // Asked recently, nobody answered. Asking again would only repeat the
      // wait — the page says the relays are unreachable, which is true.
      loadFailed.value = true;
      loading.value = false;
      return;
    }
    if (cached) {
      comments.value = Array.isArray(cached.comments) ? cached.comments : [];
      profiles.value = cached.profiles && typeof cached.profiles === "object"
        ? cached.profiles
        : {};
      relaysReached.value = Number(cached.reached) || 0;
      relaysTotal.value = Number(cached.total) || relays.value.length;
      loadFailed.value = relaysReached.value === 0;
      loading.value = false;
      return;
    }

    const filter = commentFilter(id.value);
    const nowSec = Math.floor(Date.now() / 1000);

    const asked = await loadRelays();
    const { events, reached, total } = await queryRelays(asked, filter, COMMENT_KIND, {
      subIdPrefix: "yantr-comments",
    });

    relaysReached.value = reached;
    relaysTotal.value = total;

    // Filter before verifying: a relay may hold any note at all under the tag,
    // and only one that is a comment for this app belongs in the list.
    const merged = [];
    for (const event of events) {
      if (!event || typeof event.id !== "string") continue;
      if (isAppComment(event, id.value, nowSec)) merged.push(event);
    }

    // Authorship check on the RAW event, before sanitising: the signature
    // covers exactly the bytes the author signed, so cleaning first would fail
    // every legitimately tidied note. A failure drops that one event, never the
    // list — and a forged comment is exactly what this filters out.
    const verified = merged.filter((event) => isAuthentic(event));

    // Sanitise for display last: strip relay-served controls and spoof
    // characters, truncate overlong backfill, and drop anything with nothing
    // displayable left (e.g. a note of only zero-width characters).
    comments.value = verified
      .map((event) => ({ ...event, content: sanitizeCommentContent(event.content) }))
      .filter((event) => event.content);

    // No relay reachable is the only hard failure: an empty list with relays
    // answering is a real answer — nobody commented yet.
    loadFailed.value = relaysReached.value === 0;
    loading.value = false;

    if (relaysReached.value === 0) {
      await writeNostrFailure(NOSTR_SCOPE.appComments, id.value);
      return;
    }

    // Written only when a relay answered — `reached` is the count that did, and
    // an empty list from a working relay is an answer worth remembering: it is
    // what stops this page re-querying every visit for a thread nobody has
    // commented on.
    await writeNostrQuery(
      NOSTR_SCOPE.appComments,
      id.value,
      { comments: comments.value, reached: relaysReached.value, total: relaysTotal.value },
      { answered: relaysReached.value }
    );

    // Author names resolve after the thread paints.
    fetchProfiles();
  }

  function profileOf(event) {
    return profiles.value[String(event?.pubkey ?? "").toLowerCase()] || null;
  }

  // fetchProfiles resolves display names for the thread's authors. One batched
  // kind-0 REQ per relay, newest event per pubkey wins (kind-0 replaces rather
  // than chains) and unverified events are dropped — a forged name is the same
  // impersonation a forged comment would be, just cheaper to pull off.
  // Fire-and-forget from load(): names pop in after the thread renders, and a
  // slow relay never holds the comments hostage.
  async function fetchProfiles() {
    const wanted = [
      ...new Set(
        comments.value.map((c) => String(c.pubkey || "").toLowerCase())
      ),
    ].filter(
      (h) => /^[0-9a-f]{64}$/.test(h) && !profiles.value[h]
    );
    if (wanted.length === 0) return;

    // A name is a name whoever asked for it: the same author signs notes in
    // several apps, so one record per public key serves every thread.
    const known = await readCachedProfiles(wanted);
    const unknown = wanted.filter((hex) => !known[hex]);
    if (Object.keys(known).length > 0) {
      profiles.value = { ...profiles.value, ...known };
    }
    if (unknown.length === 0) return;

    const { events } = await queryRelays(
      relays.value,
      profileFilter(unknown),
      PROFILE_KIND,
      { subIdPrefix: "yantr-profiles" }
    );

    const newest = newestKind0Profiles(events);
    if (Object.keys(newest).length > 0) {
      profiles.value = { ...profiles.value, ...newest };
      for (const [hex, profile] of Object.entries(newest)) {
        writeCachedProfile(hex, profile, { answered: 1 });
      }
    }
  }

  function hasSigner() {
    return (
      typeof window !== "undefined" &&
      !!window.nostr &&
      typeof window.nostr.getPublicKey === "function" &&
      typeof window.nostr.signEvent === "function"
    );
  }

  // connect asks the NIP-07 extension for the account that will sign. The
  // public key is the only thing that crosses into the page; signing stays
  // inside the extension.
  async function connect() {
    signerMissing.value = false;
    publishError.value = "";
    if (!hasSigner()) {
      signerMissing.value = true;
      return false;
    }
    try {
      const hex = await window.nostr.getPublicKey();
      if (!hex || !/^[0-9a-fA-F]{64}$/.test(String(hex))) {
        publishError.value = "signer-error";
        return false;
      }
      pubkeyHex.value = String(hex).toLowerCase();
      return true;
    } catch {
      publishError.value = "signer-error";
      return false;
    }
  }

  // publish signs one comment with the reader's extension and broadcasts it.
  // One positive OK is success: relays differ in auth policy and retention,
  // and unanimity was never on offer.
  async function publish(text) {
    publishError.value = "";
    justPublishedId.value = "";

    // Sanitise before signing, not just before display: what is shown must be
    // exactly what was signed, and no code path may sign the raw bytes.
    const raw = String(text ?? "").trim();
    if (!raw) {
      publishError.value = "empty";
      return false;
    }
    if ([...raw].length > COMMENT_MAX_LENGTH) {
      publishError.value = "too-long";
      return false;
    }
    const content = sanitizeCommentContent(raw);
    if (!content) {
      publishError.value = "empty";
      return false;
    }

    if (!pubkeyHex.value) {
      const ok = await connect();
      if (!ok) return false;
    }

    publishing.value = true;
    try {
      const created_at = Math.floor(Date.now() / 1000);
      const unsigned = {
        kind: COMMENT_KIND,
        pubkey: pubkeyHex.value,
        created_at,
        tags: buildCommentTags(id.value, created_at),
        content,
      };

      let signed;
      try {
        signed = await window.nostr.signEvent(unsigned);
      } catch {
        publishError.value = "sign-cancelled";
        return false;
      }
      if (!signed || typeof signed.id !== "string" || typeof signed.sig !== "string") {
        publishError.value = "sign-error";
        return false;
      }

      const oks = await Promise.all(
        relays.value.map((relay) => publishToRelay(relay, signed))
      );
      if (!oks.some(Boolean)) {
        publishError.value = "publish-failed";
        return false;
      }

      comments.value = [
        { ...signed, content: signed.content ?? content },
        ...comments.value,
      ];
      justPublishedId.value = signed.id;
      // Keep the stored thread equal to what is on screen, or the next visit
      // would open a twenty-hour-old copy of a list the reader just extended.
      writeNostrQuery(
        NOSTR_SCOPE.appComments,
        id.value,
        { comments: comments.value, reached: relaysReached.value, total: relaysTotal.value },
        { answered: relaysReached.value }
      );
      return true;
    } finally {
      publishing.value = false;
    }
  }

  function authorNpub(event) {
    return hexToNpub(event?.pubkey);
  }

  function eventTag() {
    return appCommentTag(id.value);
  }

  return {
    comments: sortedComments,
    loading,
    loadFailed,
    relaysReached,
    relaysTotal,
    publishing,
    publishError,
    justPublishedId,
    pubkeyHex,
    signerMissing,
    npub,
    shortAuthor,
    maxLength: COMMENT_MAX_LENGTH,
    load,
    connect,
    publish,
    profileOf,
    authorNpub,
    eventTag,
  };
}
