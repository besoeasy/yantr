import { onUnmounted, ref } from "vue";
import { useApiUrl } from "./useApiUrl";
import { expectApiSuccess } from "./useApiResponse";

// Shared dashboard data.
//
// Four cards want the container list, three want the catalog, one wants image
// and volume counts — and each used to fetch its own copy. On the dashboard
// that meant four identical requests at mount and three independent 15-second
// pollers all asking the same question, on a host where every request is real
// socket work. Three of those pollers also disagreed about failure: one card
// showed an error while three others quietly rendered "nothing running".
//
// So the lists live here, once. The state is module-scoped and the pollers are
// reference-counted, exactly like useHostMetrics: the first card to ask starts
// the fetch, the first to want live data starts the one ticker, and the last
// to unmount leaves the cached list in place for the next visitor without a
// second round trip.
//
// Failure is reported, not swallowed: `loadFailed` distinguishes "the request
// failed" from "the host has nothing running", which is the distinction the
// individual cards used to lose.

// The API base is read per request rather than captured at module load: the
// resources below are created when this module is imported, and useApiUrl
// touches `window`. Import-time side effects are also what would break any
// non-browser context that imports a widget (the router tests do).
function apiBase() {
  return useApiUrl().apiUrl.value;
}

// The default reader is one endpoint and a projection; a resource may bring
// its own (the storage counts pair two endpoints), and is then handed the
// resolved base URL rather than reaching for one.
async function defaultRead(apiUrl, { key, path, pick }) {
  const response = await fetch(`${apiUrl}${path}`);
  const body = await expectApiSuccess(response, `Failed to load ${key}`);
  return pick(body);
}

function createResource({ key, empty, interval, path, pick, read }) {
  // The exposed value is always the shape a widget expects — an array, or the
  // counts object — starting empty, because every consumer walks it with
  // filter/map and a `null` would put a TypeError on the dashboard for the
  // frames before the first answer lands. "Not loaded yet" is carried by
  // `loaded`, which is what a skeleton needs and an array cannot say.
  const data = ref(structuredClone(empty));
  const loaded = ref(false);
  const loading = ref(false);
  const loadFailed = ref(false);

  let inFlight = null;
  let liveSubscribers = 0;
  let ticker = null;

  const reader = read || ((apiUrl) => defaultRead(apiUrl, { key, path, pick }));

  async function loadOnce() {
    return reader(apiBase());
  }

  // One in-flight request per resource: four cards mounting in the same tick
  // ask once, and a slow response cannot be overtaken by a newer one.
  function refresh() {
    if (inFlight) return inFlight;
    loading.value = true;
    inFlight = loadOnce()
      .then((value) => {
        data.value = value;
        loaded.value = true;
        loadFailed.value = false;
        return value;
      })
      .catch(() => {
        loadFailed.value = true;
        return null;
      })
      .finally(() => {
        loading.value = false;
        inFlight = null;
      });
    return inFlight;
  }

  // ensure is what a card calls: fetch now unless someone already is. The
  // cached list stays valid for the rest of the visit, so navigating back to
  // the dashboard does not refetch.
  function ensure() {
    if (!loaded.value && !inFlight) refresh();
    return data;
  }

  function visible() {
    return typeof document === "undefined" || document.visibilityState !== "hidden";
  }

  function tick() {
    if (!visible()) return;
    refresh();
  }

  function startTicker() {
    if (ticker) return;
    ticker = setInterval(tick, interval);
  }

  function stopTicker() {
    if (!ticker) return;
    clearInterval(ticker);
    ticker = null;
  }

  // use joins the live set: the card gets live data and unmounting returns its
  // share, so the ticker stops when the last live card leaves rather than
  // polling a page nobody is looking at.
  function use(live) {
    ensure();
    if (!live) return;
    liveSubscribers += 1;
    startTicker();
    onUnmounted(() => {
      liveSubscribers -= 1;
      if (liveSubscribers <= 0) {
        liveSubscribers = 0;
        stopTicker();
      }
    });
  }

  return { data, loaded, loading, loadFailed, refresh, ensure, use };
}

const containers = createResource({
  key: "containers",
  empty: [],
  path: "/api/containers",
  interval: 15000,
  pick: (body) => (Array.isArray(body?.containers) ? body.containers : []),
});

const catalog = createResource({
  key: "apps",
  empty: [],
  path: "/api/apps",
  // The catalog changes when the host's apps/ tree does, not on a timer, so
  // it is fetched once and kept — a live subscriber still gets a refresh on
  // mount, which is what picks up a newly deployed app.
  interval: 30000,
  pick: (body) => (Array.isArray(body?.apps) ? body.apps : []),
});

const storageCounts = createResource({
  key: "storage counts",
  empty: { imageCount: null, volumeCount: null },
  // Both counts come from two endpoints; this resource reads the pair so the
  // card that shows them makes one logical fetch instead of two lifecycles.
  read: async (apiUrl) => {
    const [imagesRes, volumesRes] = await Promise.all([
      fetch(`${apiUrl}/api/images`),
      fetch(`${apiUrl}/api/volumes`),
    ]);
    const [images, volumes] = await Promise.all([
      expectApiSuccess(imagesRes, "images"),
      expectApiSuccess(volumesRes, "volumes"),
    ]);
    return {
      imageCount: Array.isArray(images?.images) ? images.images.length : null,
      volumeCount: Array.isArray(volumes?.volumes) ? volumes.volumes.length : null,
    };
  },
  interval: 30000,
});

/** Every container on this host — running and stopped. `live` keeps it fresh. */
export function useContainers({ live = false } = {}) {
  containers.use(live);
  return {
    containers: containers.data,
    containersLoaded: containers.loaded,
    containersLoading: containers.loading,
    containersLoadFailed: containers.loadFailed,
    refreshContainers: containers.refresh,
  };
}

/** The app catalog. */
export function useCatalog({ live = false } = {}) {
  catalog.use(live);
  return {
    apps: catalog.data,
    appsLoaded: catalog.loaded,
    appsLoading: catalog.loading,
    appsLoadFailed: catalog.loadFailed,
    refreshApps: catalog.refresh,
  };
}

/** How many images and volumes this host holds. */
export function useStorageCounts({ live = false } = {}) {
  storageCounts.use(live);
  return {
    counts: storageCounts.data,
    countsLoaded: storageCounts.loaded,
    countsLoading: storageCounts.loading,
    countsLoadFailed: storageCounts.loadFailed,
    refreshCounts: storageCounts.refresh,
  };
}