import { ref, shallowRef } from "vue";
import { useApiUrl } from "./useApiUrl";
import { expectApiSuccess } from "./useApiResponse";

// Everything the #/developer/<key> route needs, in two independent halves.
//
// The half that matters is the catalogue: /api/developers/{key} answers from the
// local apps directory and works with no internet at all. The other half is the
// account's kind-0 metadata from the relays, and it is deliberately allowed to
// fail: Yantr is rootless and often runs where the relays are unreachable, and a
// developer page that dies with them would be a page about Yantr's dependencies
// rather than about the developer.
//
// A factory rather than module-level state, unlike useRepoStats: the route param
// changes on every navigation and two developer pages must not share a cache.

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

export function useDeveloperProfile() {
  const { apiUrl } = useApiUrl();

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

  // fetchProfile loads the kind-0 metadata the server proxy caches.
  async function fetchProfile(key) {
    try {
      const res = await fetch(`${apiUrl.value}/api/nostr/profile/${encodeURIComponent(key)}`);

      // 404 covers both "no relay holds this account's metadata" and "the
      // catalog credits no such key". Neither is a failure: there is simply no
      // profile card, and the QR code is drawn from the catalog regardless.
      if (res.status === 404) {
        profileUnavailable.value = true;
        return;
      }

      const data = await expectApiSuccess(res, "nostr profile unavailable");
      profile.value = normaliseNostrProfile(data.profile);
      profileUnavailable.value = profile.value === null;
    } catch {
      profileUnavailable.value = true;
    }
  }

  // load resolves one developer key. `key` is the route param; the server is the
  // authority on whether it names an account the catalog credits, so an unknown
  // key comes back as a 4xx rather than being guessed at here.
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