import { ref, shallowRef } from "vue";
import { useApiUrl } from "./useApiUrl";
import { expectApiSuccess } from "./useApiResponse";
import { canFetchProfile } from "../utils/developer";

// Everything the #/developer/<handle> route needs, in two independent halves.
//
// The half that matters is the catalogue: /api/developers/{handle} answers from
// the local apps directory and works with no internet at all. The other half is
// the platform profile — GitHub's API or a Nostr relay — and it is deliberately
// allowed to fail: Yantr is rootless and often runs where neither is reachable,
// and a developer page that dies with them would be a page about Yantr's
// dependencies rather than about the developer.
//
// A factory rather than module-level state, unlike useRepoStats: the route param
// changes on every navigation and two developer pages must not share a cache.

// normaliseNostrProfile maps a kind-0 event onto the same field names a GitHub
// user has, because the view renders one hero for both platforms. Doing it here
// rather than with a ternary on every line of the template also puts the two
// platforms' real differences in one place: Nostr has no repository or follower
// counts, so those stay undefined and the counters that need them stay hidden.
function normaliseNostrProfile(raw) {
  if (!raw) return null;

  const name = String(raw.name ?? "").trim() || String(raw.displayName ?? "").trim();

  return {
    // Clients disagree about which field holds the human name, and either may
    // be absent. The handle is the caller's fallback.
    name,
    bio: String(raw.about ?? "").trim(),
    avatarUrl: String(raw.picture ?? "").trim(),
    blog: String(raw.website ?? "").trim(),
    nip05: String(raw.nip05 ?? "").trim(),
    lud16: String(raw.lud16 ?? "").trim(),
    key: String(raw.key ?? "").trim(),
    pubkey: String(raw.pubkey ?? "").trim(),
    source: "nostr",
  };
}

export function useDeveloperProfile() {
  const { apiUrl } = useApiUrl();

  const developer = shallowRef(null);
  const apps = shallowRef([]);
  const profile = shallowRef(null);
  const repos = shallowRef([]);

  const loading = ref(false);
  const notFound = ref(false);
  const failed = ref(false);
  // Set when the catalogue answered but the platform profile did not. The page
  // renders normally and says so; it is not an error state.
  const profileUnavailable = ref(false);

  function reset() {
    developer.value = null;
    apps.value = [];
    profile.value = null;
    repos.value = [];
    notFound.value = false;
    failed.value = false;
    profileUnavailable.value = false;
  }

  // fetchNostrProfile loads the kind-0 metadata the server proxy caches. Unlike
  // GitHub there is only one request, and no repo list to go with it.
  async function fetchNostrProfile(key) {
    try {
      const res = await fetch(`${apiUrl.value}/api/nostr/profile/${encodeURIComponent(key)}`);

      // 404 covers both "the relay has no metadata for this key" and "the
      // catalog declares no such key". Neither is a failure: there is simply no
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

  // fetchEnrichment loads whatever profile data the platform offers. It is
  // best-effort: a failure flips profileUnavailable and nothing else, because
  // the app list below is already rendered and correct.
  async function fetchEnrichment(platform, handle) {
    if (!canFetchProfile(platform)) return;

    if (platform === "nostr") {
      await fetchNostrProfile(handle);
      return;
    }

    const login = encodeURIComponent(handle);
    const [profileResult, reposResult] = await Promise.allSettled([
      fetch(`${apiUrl.value}/api/github/user/${login}`).then((res) =>
        expectApiSuccess(res, "profile unavailable")
      ),
      fetch(`${apiUrl.value}/api/github/user/${login}/repos`).then((res) =>
        expectApiSuccess(res, "repos unavailable")
      ),
    ]);

    if (profileResult.status === "fulfilled") {
      profile.value = profileResult.value.user ?? null;
    }
    if (reposResult.status === "fulfilled") {
      repos.value = Array.isArray(reposResult.value.repos) ? reposResult.value.repos : [];
    }

    // Only call it unavailable when there is genuinely nothing to show. A
    // missing profile with a successful repo list is still a richer page.
    profileUnavailable.value = profile.value === null && repos.value.length === 0;
  }

  // load resolves one developer reference. `ref` is the route param, either
  // "handle" or "platform:handle"; the server is the authority on whether it is
  // valid, so an unknown platform comes back as a 4xx rather than being guessed
  // at here.
  async function load(ref) {
    reset();
    loading.value = true;

    try {
      const res = await fetch(`${apiUrl.value}/api/developers/${encodeURIComponent(ref)}`);

      // 404 means "no app credits this developer", which is a state the page
      // explains, not a failure. expectApiSuccess collapses every non-ok into
      // one bare Error, so the status has to be read before it.
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
      await fetchEnrichment(developer.value.platform, developer.value.handle);
    }
  }

  return {
    developer,
    apps,
    profile,
    repos,
    loading,
    notFound,
    failed,
    profileUnavailable,
    load,
  };
}
