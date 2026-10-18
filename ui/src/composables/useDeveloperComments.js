import { computed, ref } from "vue";
import {
  PROFILE_KIND,
  authorCommentFilter,
  commentAppIds,
  isAuthorYantrComment,
  isAuthentic,
  newestKind0Profiles,
  npubToHex,
  profileFilter,
  sanitizeCommentContent,
} from "../utils/nostr";
import { queryRelays } from "../utils/relay";
import { useNostrRelays } from "./useNostrRelays";
import {
  NOSTR_SCOPE,
  readNostrQuery,
  writeCachedProfile,
  writeNostrFailure,
  writeNostrQuery,
} from "../utils/nostrCache";

// One account's app-store comments, for the developer page.
//
// The app's own thread (useAppComments) is app-shaped: it asks the relays for
// notes on one per-app tag and trusts no author. This is author-shaped: it
// asks for the account's kind-1 notes under the bare `yantr` scope and keeps
// only the ones that name a per-app tag — so the list is every comment the
// account left anywhere in the catalog, with the app id each one carries read
// back out of the event for the page to map onto the catalog.
//
// The verdicts are the same as any relay read: signature-verified before
// display, sanitised after verification, and an unreachable relay set is the
// only hard failure — an empty list with relays answering means this account
// has signed nothing.

export function useDeveloperComments(developerKey) {
  // npubToHex is the one conversion the page needs: relays speak hex, the
  // route speaks npub, and an nprofile decodes to the same 32 bytes.
  const keyHex = computed(() => npubToHex(developerKey.value) || "");

  const { relays, load: loadRelays } = useNostrRelays();

  const comments = ref([]);
  const loading = ref(false);
  const loadFailed = ref(false);
  const relaysReached = ref(0);
  const relaysTotal = ref(relays.value.length);

  // The account's kind-0 profile, filled best-effort after the list paints so
  // the reader sees the account's own name rather than only the abbreviated
  // npub the page header may already carry.
  const profile = ref(null);

  const sortedComments = computed(() =>
    [...comments.value].sort((a, b) => (b.created_at || 0) - (a.created_at || 0))
  );

  function reset() {
    comments.value = [];
    loadFailed.value = false;
    relaysReached.value = 0;
  }

  async function load() {
    reset();
    // A key the page already vetted by shape (npub/nprofile) still has to
    // decode to hex before a filter can be built; there is nothing to query
    // otherwise, and the page's catalog half has already handled not-found.
    const filter = keyHex.value ? authorCommentFilter(keyHex.value) : null;
    if (!filter) {
      loadFailed.value = false;
      return;
    }

    loading.value = true;

    // The account's own thread, from the reader's own machine. Same contract
    // as the app thread's cache: verified, sanitised, and written only after a
    // relay actually answered.
    const cached = await readNostrQuery(NOSTR_SCOPE.developerComments, keyHex.value);
    if (cached?.failed) {
      loadFailed.value = true;
      loading.value = false;
      fetchProfile();
      return;
    }
    if (cached) {
      comments.value = Array.isArray(cached.comments) ? cached.comments : [];
      relaysReached.value = Number(cached.reached) || 0;
      relaysTotal.value = Number(cached.total) || relays.value.length;
      loadFailed.value = relaysReached.value === 0;
      loading.value = false;
      fetchProfile();
      return;
    }

    const asked = await loadRelays();
    const { events, reached, total } = await queryRelays(asked, filter, 1, {
      subIdPrefix: "yantr-developer-comments",
    });

    relaysReached.value = reached;
    relaysTotal.value = total;

    const merged = [];
    for (const event of events) {
      if (!event || typeof event.id !== "string") continue;
      if (!isAuthorYantrComment(event, keyHex.value)) continue;
      merged.push(event);
    }

    const verified = merged.filter((event) => isAuthentic(event));

    comments.value = verified
      .map((event) => ({
        ...event,
        content: sanitizeCommentContent(event.content),
        apps: commentAppIds(event),
      }))
      .filter((event) => event.content);

    loadFailed.value = relaysReached.value === 0;
    loading.value = false;

    if (relaysReached.value === 0) {
      await writeNostrFailure(NOSTR_SCOPE.developerComments, keyHex.value);
      fetchProfile();
      return;
    }

    await writeNostrQuery(
      NOSTR_SCOPE.developerComments,
      keyHex.value,
      { comments: comments.value, reached: relaysReached.value, total: relaysTotal.value },
      { answered: relaysReached.value }
    );

    fetchProfile();
  }

  // One account, so a single-author kind-0 REQ — the same batched shape
  // useAppComments uses for a whole thread's authors, just asked once. Fire
  // and forget: a slow relay must never hold the comments hostage, and the
  // page already shows the key before the name resolves.
  async function fetchProfile() {
    if (!keyHex.value) return;

    // Same record the comment threads use for their authors, so this page does
    // not ask for a kind-0 the reader's machine already answered today.
    const cached = await readNostrQuery(NOSTR_SCOPE.authorProfile, keyHex.value);
    if (cached) {
      profile.value = cached;
      return;
    }

    const { events } = await queryRelays(
      relays.value,
      profileFilter([keyHex.value]),
      PROFILE_KIND,
      { subIdPrefix: "yantr-developer-profile" }
    );
    const newest = newestKind0Profiles(events);
    if (newest[keyHex.value]) {
      profile.value = newest[keyHex.value];
      writeCachedProfile(keyHex.value, newest[keyHex.value], { answered: 1 });
    }
  }

  return {
    comments: sortedComments,
    loading,
    loadFailed,
    relaysReached,
    relaysTotal,
    profile,
    keyHex,
    load,
  };
}
