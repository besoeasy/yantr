import { ref, shallowRef } from "vue";
import { useApiUrl } from "./useApiUrl";
import { expectApiSuccess } from "./useApiResponse";

// Repository counters for the GitHub card, fetched from the backend's cached
// /api/github/stats proxy rather than straight from api.github.com.
//
// Two reasons it is proxied rather than called from the browser: the
// unauthenticated GitHub API allows 60 requests an hour per source IP, which
// a NAT'd homelab would exhaust quickly, and the card should not be handing
// every visitor's IP to GitHub. The backend caches for 30 minutes, so this
// composable only needs to avoid duplicating the request per card instance.

const stats = shallowRef(null);
const loading = ref(false);
const failed = ref(false);

let inFlight = null;

const { apiUrl } = useApiUrl();

function fetchStats() {
  if (stats.value) return Promise.resolve(stats.value);
  if (inFlight) return inFlight;

  loading.value = true;
  inFlight = (async () => {
    try {
      const res = await fetch(`${apiUrl.value}/api/github/stats`);
      const data = await expectApiSuccess(res, "Failed to fetch repository stats");
      stats.value = data.stats ?? null;
      failed.value = false;
    } catch {
      // The card deliberately shows no counter, so a failure here has nothing
      // to degrade — it only means the cache stays cold until the next load.
      failed.value = true;
    } finally {
      loading.value = false;
      inFlight = null;
    }
    return stats.value;
  })();

  return inFlight;
}

export function useRepoStats() {
  return { stats, loading, failed, fetchStats };
}