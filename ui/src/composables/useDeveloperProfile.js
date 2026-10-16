import { ref, shallowRef } from "vue";
import { useApiUrl } from "./useApiUrl";
import { expectApiSuccess } from "./useApiResponse";
import { canFetchProfile } from "../utils/developer";

// Everything the #/developer/<handle> route needs, in two independent halves.
//
// The half that matters is the catalogue: /api/developers/{handle} answers from
// the local apps directory and works with no internet at all. The other half is
// GitHub, and it is deliberately allowed to fail — Yantr is rootless and often
// runs where api.github.com is unreachable or rate-limited, and a developer page
// that dies with it would be a page about Yantr's dependencies rather than about
// the developer.
//
// A factory rather than module-level state, unlike useRepoStats: the route param
// changes on every navigation and two developer pages must not share a cache.

export function useDeveloperProfile() {
  const { apiUrl } = useApiUrl();

  const developer = shallowRef(null);
  const apps = shallowRef([]);
  const profile = shallowRef(null);
  const repos = shallowRef([]);

  const loading = ref(false);
  const notFound = ref(false);
  const failed = ref(false);
  // Set when the catalogue answered but GitHub did not. The page renders
  // normally and says so; it is not an error state.
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

  // fetchEnrichment loads the GitHub halves. Both are best-effort: a failure
  // flips profileUnavailable and nothing else, because the app list below is
  // already rendered and correct.
  async function fetchEnrichment(platform, handle) {
    if (!canFetchProfile(platform)) return;

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
