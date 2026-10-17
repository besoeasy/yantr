import { ref } from "vue";
import { useApiUrl } from "./useApiUrl";
import { NOSTR_RELAYS } from "../utils/nostr";

// The relay set the backend is configured with, asked once per session.
//
// Why the backend at all, when every Nostr query is made from the browser
// (utils/relay.js)? Because the set is configuration, not code: YANTR_NOSTR_RELAYS
// on the host is how a user behind a network that blocks these three, or one
// running their own relay, redirects every Nostr read and write in Yantr at
// once. Serving it from the API keeps that override in one place and stops the
// frontend list from drifting into a second, unconfigurable copy — which is
// what the constant in utils/nostr.js is: the fallback for a backend that
// cannot answer.
//
// The set is memoised at module level for the session: it does not change while
// Yantr runs, so a second page asking for it costs nothing, and two pages in the
// same session can never disagree about which relays are in play.

// TODO(presence-fanout): if a future feature reads from relays without a
// backend to ask (the website's static build, for instance), this is the module
// that would need a hardcoded-set fallback of its own — the backend would not
// be there to answer.
let sessionRelays = null;
let sessionPromise = null;

export function useNostrRelays() {
  const { apiUrl } = useApiUrl();

  // Seeded with the fallback so a caller that reads it before load() resolves
  // still has a usable set rather than an empty one.
  const relays = ref(sessionRelays || NOSTR_RELAYS);
  const loaded = ref(!!sessionRelays);

  async function load() {
    if (sessionRelays) {
      relays.value = sessionRelays;
      loaded.value = true;
      return sessionRelays;
    }
    if (!sessionPromise) {
      sessionPromise = fetch(`${apiUrl.value}/api/nostr/relays`)
        .then((res) => (res.ok ? res.json() : null))
        .then((data) => {
          const list = Array.isArray(data?.relays) ? data.relays : [];
          // An empty or malformed answer is treated as "no override", not as
          // "use no relays": a broken endpoint must not silently stop every
          // Nostr read in the UI.
          if (list.length === 0) return NOSTR_RELAYS;
          return list.filter((r) => typeof r === "string" && /^wss?:\/\//i.test(r));
        })
        .catch(() => NOSTR_RELAYS)
        .then((list) => {
          sessionRelays = list.length > 0 ? list : NOSTR_RELAYS;
          return sessionRelays;
        });
    }

    try {
      sessionRelays = await sessionPromise;
    } finally {
      loaded.value = true;
    }
    relays.value = sessionRelays;
    return sessionRelays;
  }

  return { relays, loaded, load };
}