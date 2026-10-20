import { kvGet, kvIsFresh, kvRemove, kvSet } from "./kv";

// A twenty-hour cache for the answers of a Nostr query.
//
// Yantr's Nostr reads are made from the reader's browser (utils/relay.js), one
// signed REQ per relay per surface, and there are four surfaces on a normal
// visit: the app page's thread, the developer page's thread, the dashboard's
// review list, and a kind-0 for every author named in any of them. Without a
// cache that is a burst of websocket subscriptions every reload — the exact
// shape public relays rate-limit — and it is why the dashboard's promo cards
// spend their first moments behind a skeleton: the answer is in the reader's
// own machine, and nothing was keeping it there.
//
// Twenty hours, not a week, is the deliberate choice. A comment or a profile
// written this afternoon should be visible this afternoon; a reader who has
// had the app open all day is not re-reading the dashboard, and tomorrow's
// visit gets a fresh answer.
//
// Two rules the callers cannot bypass:
//
//   - only VERIFIED, display-ready data is ever written. The cache stores what
//     the page would have shown, so a hit cannot smuggle past the signature
//     check, the sanitiser or the app-tag filter that produced it.
//   - a record is written only when a relay actually answered. A transport
//     failure is not an answer, and caching one would turn a dead socket into
//     twenty hours of an empty page.
//
// Reads are best-effort in the other direction too: storage that refuses to
// open resolves as a miss, so the worst outcome of a broken cache is the query
// this module exists to avoid.
//
// A record whose shape this build does not know is ignored rather than
// misread, which is what CACHE_VERSION is for: bump it and the next read is a
// miss, not a card built from fields that no longer mean what they did.

export const NOSTR_QUERY_TTL_MS = 20 * 60 * 60 * 1000;

// "This account has published no profile" is worth remembering for an hour,
// not twenty: a name published this afternoon should not wait until tomorrow.
export const NOSTR_MISS_TTL_MS = 60 * 60 * 1000;

// A failure is remembered for ten minutes, and not one second longer.
//
// This is the one case where storing an outcome the relays never gave is the
// honest thing to do, because the alternative is worse: every page load
// re-opens three websockets, waits out the review card's four-second retry,
// and shows a skeleton for a network that is simply not there. Ten minutes is
// short enough that a host whose network comes back does not stay dark for
// long, and long enough that clicking around an app does not repeat the wait
// on every page.
export const NOSTR_FAILURE_TTL_MS = 10 * 60 * 1000;

const STORE = "nostr-queries";
const CACHE_VERSION = 1;

// One store, several surfaces. The scope keeps a review list, a thread and a
// set of author names from ever colliding on the same key.
export const NOSTR_SCOPE = {
  appComments: "app-comments",
  reviewFeed: "review-feed",
  authorProfile: "author-profile",
  relaySet: "relay-set",
};

function recordKey(scope, key) {
  return `${scope}:${key}`;
}

/**
 * The cached answer for one query, or null for a miss — which covers "never
 * asked", "expired", "written by an older build" and "storage said no". A
 * recorded empty answer is itself null, which reads as a miss; that is
 * deliberate, because both mean the same thing to a caller: ask the relays.
 */
export async function readNostrQuery(scope, key) {
  if (!scope || !key) return null;

  const entry = await kvGet(STORE, recordKey(scope, key));
  if (!entry || entry.v !== CACHE_VERSION) return null;
  if (!kvIsFresh(entry)) return null;
  return entry.data ?? null;
}

/**
 * Writes one query's answer.
 *
 * `null` is a legitimate value — it records "the relays answered, and there is
 * nothing here" — so only `undefined` is refused alongside an empty key.
 *
 * `answered` must be the count of relays that actually replied. It is not
 * optional in practice: passing a falsey value refuses the write, because the
 * one thing this cache must never store is "the network was down".
 */
export async function writeNostrQuery(scope, key, data, { answered = 0, ttlMs } = {}) {
  if (!scope || !key || data === undefined) return false;
  if (!answered) return false;

  const now = Date.now();
  return kvSet(STORE, recordKey(scope, key), {
    v: CACHE_VERSION,
    scope,
    key,
    data,
    fetchedAt: now,
    expires: now + (Number(ttlMs) > 0 ? Number(ttlMs) : NOSTR_QUERY_TTL_MS),
  });
}

export function forgetNostrQuery(scope, key) {
  if (!scope || !key) return Promise.resolve(null);
  return kvRemove(STORE, recordKey(scope, key));
}

/**
 * Records that a query was made and no relay answered, for ten minutes.
 *
 * Unlike writeNostrQuery this does not need an answer to store — the absence of
 * one is the content, kept short so it can never masquerade as a real result.
 * A reader who hits one gets their own failure state immediately instead of a
 * skeleton and a second round of sockets.
 */
export async function writeNostrFailure(scope, key) {
  if (!scope || !key) return false;
  const now = Date.now();
  return kvSet(STORE, recordKey(scope, key), {
    v: CACHE_VERSION,
    scope,
    key,
    data: { failed: true, at: now },
    fetchedAt: now,
    expires: now + NOSTR_FAILURE_TTL_MS,
  });
}

/**
 * Author names, in bulk, for the surfaces that resolve a thread's writers.
 *
 * One scope, one record per public key, because the same author signs notes in
 * several apps and each surface wants their name for the same reason.
 */
export async function readCachedProfiles(hexKeys) {
  const keys = [...new Set((hexKeys ?? []).filter((h) => /^[0-9a-f]{64}$/.test(String(h))))];
  if (keys.length === 0) return {};

  const found = {};
  const entries = await Promise.all(
    keys.map((hex) => readNostrQuery(NOSTR_SCOPE.authorProfile, hex))
  );
  keys.forEach((hex, i) => {
    const profile = entries[i];
    if (profile && typeof profile === "object") found[hex] = profile;
  });
  return found;
}

/**
 * Stores one author's name — or the fact that they have none, for an hour,
 * because an account that publishes a profile this afternoon should not have
 * to be asked about it twenty more times before it shows up.
 */
export async function writeCachedProfile(hex, profile, { answered = 0 } = {}) {
  if (!/^[0-9a-f]{64}$/.test(String(hex))) return false;
  const ttlMs = profile ? NOSTR_QUERY_TTL_MS : NOSTR_MISS_TTL_MS;
  return writeNostrQuery(NOSTR_SCOPE.authorProfile, hex, profile ?? null, {
    answered,
    ttlMs,
  });
}