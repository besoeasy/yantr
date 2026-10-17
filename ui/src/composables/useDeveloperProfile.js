import { ref, shallowRef } from "vue";
import { useApiUrl } from "./useApiUrl";
import { expectApiSuccess } from "./useApiResponse";
import { useNostrRelays } from "./useNostrRelays";
import { queryRelays } from "../utils/relay";
import { npubToHex, profileFilter, newestKind0Profiles, PROFILE_KIND } from "../utils/nostr";

// Everything the #/developer/<key> route needs, in two independent halves.
//
// The half that matters is the catalogue: /api/developers/{key} answers from the
// local apps directory and works with no internet at all. The other half is the
// account's kind-0 metadata, and it is deliberately allowed to fail: Yantr is
// rootless and often runs where the relays are unreachable, and a developer page
// that dies with them would be a page about Yantr's dependencies rather than
// about the developer.
//
// That second half is a direct relay query from the browser (utils/relay.js),
// the same path comments take. It used to be proxied by the backend, which
// needed its own Nostr client, its own relay set and its own copy of the cache
// rules — none of which the API had any state to justify. In the browser it is
// one signed, verified kind-0 read, and the catalogue gate that endpoint needed
// (so Yantr would not be an open relay proxy) is no longer a concern: the query
// leaves from the reader's own machine.
//
// A factory rather than module-level state, unlike useRepoStats: the route param
// changes on every navigation and two developer pages must not share a cache.

// PROFILE_CACHE_TTL and PROFILE_MISS_TTL mirror the rules the backend's cache
// used, so moving the fetch did not change what a reader sees. Kind-0 is
// reference data — a name, a picture, an about line — so a week is right; a miss
// is cached far shorter because an account that publishes a profile this
// afternoon should not wait a week to show it.
const PROFILE_CACHE_TTL_MS = 7 * 24 * 60 * 60 * 1000;
const PROFILE_MISS_TTL_MS = 60 * 60 * 1000;

// Session-wide, not per-factory: the cache is reference data about an account,
// not about a page, so two developer pages in one session share it while a
// fresh load starts over. That is the same lifetime the backend's in-process
// cache had from the reader's point of view.
const profileCache = new Map();

// normaliseNostrProfile maps a kind-0 event onto the field names the view reads,
// because the view should not care where the metadata came from. Name falls back
// through the two fields clients disagree about, and leaves the key as the
// caller's last resort when neither is published.
function normaliseNostrProfile(raw) {
  if (!raw) return null;

  const name = String(raw.name ?? "").trim() || String(raw.displayName ?? "").trim();

  return {
    name,
    bio: String(raw.about ?? "").trim(),
    avatarUrl: String(raw.picture ?? "").trim(),
    blog: String(raw.website ?? "").trim(),
    nip05: String(raw.nip05 ?? "").trim(),
    lud16: String(raw.lud16 ?? "").trim(),
    key: String(raw.key ?? "").trim(),
    pubkey: String(raw.pubkey ?? "").trim(),
  };
}

// cachedProfile returns a cached kind-0 for one pubkey, if the entry is still
// fresh. A cached failure counts as fresh too, so a relay that was unreachable
// an hour ago is asked again now rather than remembered as "no profile".
function cachedProfile(pubkey) {
  const entry = profileCache.get(pubkey);
  if (!entry) return null;
  if (Date.now() >= entry.expires) return null;
  return entry.profile;
}

export function useDeveloperProfile() {
  const { apiUrl } = useApiUrl();
  const { load: loadRelays } = useNostrRelays();

  const developer = shallowRef(null);
  const apps = shallowRef([]);
  const profile = shallowRef(null);

  const loading = ref(false);
  const notFound = ref(false);
  const failed = ref(false);
  // Set when the catalogue answered but the relays did not. The page renders
  // normally and says so; it is not an error state.
  const profileUnavailable = ref(false);

  function reset() {
    developer.value = null;
    apps.value = [];
    profile.value = null;
    notFound.value = false;
    failed.value = false;
    profileUnavailable.value = false;
  }

  // fetchProfile reads the account's kind-0 straight from the relays, with the
  // browser as the only party involved.
  //
  // `key` is the catalog's own spelling (npub or nprofile); the filter is built
  // from the hex both decode to, which is what a relay matches authors on.
  async function fetchProfile(key) {
    const pubkey = npubToHex(key);
    if (!pubkey) {
      profileUnavailable.value = true;
      return;
    }

    const hit = cachedProfile(pubkey);
    if (hit) {
      profile.value = normaliseNostrProfile(hit);
      profileUnavailable.value = hit === null;
      return;
    }

    try {
      const relays = await loadRelays();
      const { events } = await queryRelays(relays, profileFilter([pubkey]), PROFILE_KIND, {
        subIdPrefix: "yantr-profile",
      });

      // Newest wins and unverified events are dropped inside this merge — a
      // kind-0 is a name, a picture and a NIP-05 the page is about to print, so
      // a relay must not be able to speak for the account.
      const found = newestKind0Profiles(events)[pubkey] || null;

      profileCache.set(pubkey, {
        profile: found,
        expires: Date.now() + (found ? PROFILE_CACHE_TTL_MS : PROFILE_MISS_TTL_MS),
      });

      profile.value = normaliseNostrProfile(found);
      profileUnavailable.value = !found;
    } catch {
      profileUnavailable.value = true;
    }
  }

  // load resolves one developer key. `key` is the route param; the server is
  // the authority on whether it names an account the catalog credits, so an
  // unknown key comes back as a 4xx rather than being guessed at here.
  async function load(ref) {
    reset();
    loading.value = true;

    try {
      const res = await fetch(`${apiUrl.value}/api/developers/${encodeURIComponent(ref)}`);

      // 404 means "no app credits this developer", which is a state the page
      // explains, not a failure. expectApiSuccess collapses every non-ok into one
      // bare Error, so the status has to be read before it.
      if (res.status === 404) {
        notFound.value = true;
        loading.value = false;
        return;
      }

      const data = await expectApiSuccess(res, "Failed to load developer");
      developer.value = data.developer ?? null;
      apps.value = Array.isArray(data.apps) ? data.apps : [];
    } catch {
      failed.value = true;
      loading.value = false;
      return;
    }

    loading.value = false;

    if (developer.value) {
      await fetchProfile(developer.value.key);
    }
  }

  return {
    developer,
    apps,
    profile,
    loading,
    notFound,
    failed,
    profileUnavailable,
    load,
  };
}