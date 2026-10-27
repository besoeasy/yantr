/**
 * routes.js — the route table, the route names, and the builders that produce
 * locations. Everything about *where a page lives* is written down here, and
 * nowhere else.
 *
 * This file deliberately does not import vue-router and never calls
 * createRouter — router.js does that. Keeping the table as pure data means
 * routes.test.js can assert things about it (no duplicated name, every route
 * titled, every detail route naming a real nav owner) in a plain Node
 * environment with no DOM, which a router instance would require.
 *
 * Views never write a path string. They push a builder's result instead:
 *
 *     router.push(containerDetail(container.id))
 *     <router-link :to="appDetail(app.id)">
 *
 * A renamed path or param is then one edit here rather than fifteen across the
 * views — which is how `/apps/:appname` drifted into holding an app id.
 */

/**
 * Route names. These are identities, not URLs: they survive a path change, so
 * they are the thing to compare against and the thing to navigate by.
 */
export const ROUTE = Object.freeze({
  DASHBOARD: "dashboard",
  APPS: "apps",
  APP_DETAIL: "app-detail",
  STACK_DETAIL: "stack-detail",
  CONTAINER_DETAIL: "container-detail",
  DEVELOPER_DETAIL: "developer-detail",
  STORAGE: "storage",
  LOGS: "logs",
  NOT_FOUND: "not-found",
});

/**
 * Sidebar nav ids. A route declares which item owns it via `meta.nav`, and the
 * shell highlights that item. Detail routes inherit their list's owner, so
 * /apps/some-app lights up "Apps" instead of leaving the sidebar with nothing
 * active — which is what the old `route.name === item.name` test did for every
 * route that is not itself a top-level page.
 */
export const NAV = Object.freeze({
  DASHBOARD: "dashboard",
  APPS: "apps",
  STORAGE: "storage",
  LOGS: "logs",
});

/**
 * The sidebar, in order. `id` is what a route's `meta.nav` is matched against
 * and must be a NAV value; `labelKey` names a key under the `nav` i18n
 * namespace. Adding an entry here is all it takes to add a nav item — the shell
 * builds itself from this list and no longer holds its own copy.
 */
export const NAV_ITEMS = [
  { id: NAV.DASHBOARD, labelKey: "nav.dashboard" },
  { id: NAV.APPS, labelKey: "nav.apps" },
  { id: NAV.STORAGE, labelKey: "nav.storage" },
  { id: NAV.LOGS, labelKey: "nav.logs" },
];

// ─── Builders ───────────────────────────────────────────────────────────────
// Every builder returns a vue-router location object, so `router.push` and
// `<router-link :to>` both take it unchanged.

/** The dashboard — everything running on this host. */
export function dashboard() {
  return { name: ROUTE.DASHBOARD };
}

/** The app catalog. */
export function apps() {
  return { name: ROUTE.APPS };
}

/** One app's page in the catalog: what it is, and the form that installs it. */
export function appDetail(appId) {
  return { name: ROUTE.APP_DETAIL, params: { appId } };
}

/** One deployed stack, addressed by its compose project. */
export function stackDetail(projectId) {
  return { name: ROUTE.STACK_DETAIL, params: { projectId } };
}

/** One container. */
export function containerDetail(containerId) {
  return { name: ROUTE.CONTAINER_DETAIL, params: { containerId } };
}

/** One developer's profile and the apps they are credited on. */
export function developerDetail(npub) {
  return { name: ROUTE.DEVELOPER_DETAIL, params: { npub } };
}

/** Storage, optionally opened on one of the four tabs. */
export function storageTab(tab) {
  return tab ? { name: ROUTE.STORAGE, query: { tab } } : { name: ROUTE.STORAGE };
}

/** The log stream. */
export function logs() {
  return { name: ROUTE.LOGS };
}

// ─── Table ──────────────────────────────────────────────────────────────────
//
// Entries are lazy so each view stays its own chunk. There are no
// `webpackChunkName` comments: Vite builds on Rollup, which ignores them, and
// the emitted chunks are named after the view files anyway.

export const routes = [
  { path: "/", redirect: { name: ROUTE.DASHBOARD } },

  // Legacy paths. A hash URL is bookmarkable and Yantr users bookmark their own
  // dashboard, so the paths that changed or were removed keep working instead
  // of dropping the user on the 404. Param renames are URL-neutral and need no
  // redirect. /stacks was removed as a page — the dashboard is the stack list
  // now — but the bookmark must survive, and stack detail lives on at
  // /stacks/:projectId below.
  { path: "/home", redirect: { name: ROUTE.DASHBOARD } },
  { path: "/stacks", redirect: { name: ROUTE.DASHBOARD } },
  { path: "/developer/:npub", redirect: (to) => developerDetail(to.params.npub) },

  {
    path: "/dashboard",
    name: ROUTE.DASHBOARD,
    component: () => import("../views/Home.vue"),
    meta: { title: "Dashboard", nav: NAV.DASHBOARD },
  },
  {
    path: "/apps",
    name: ROUTE.APPS,
    component: () => import("../views/Apps.vue"),
    meta: { title: "Apps", nav: NAV.APPS },
  },
  {
    path: "/apps/:appId",
    name: ROUTE.APP_DETAIL,
    component: () => import("../views/AppDetail.vue"),
    meta: { title: "App", nav: NAV.APPS },
  },
  {
    // No /stacks list route: the dashboard's Yantr Apps section is the stack
    // list. Detail pages keep their own paths and are owned by the dashboard
    // nav item, since that is where they are reached from.
    path: "/stacks/:projectId",
    name: ROUTE.STACK_DETAIL,
    component: () => import("../views/StackView.vue"),
    meta: { title: "Stack", nav: NAV.DASHBOARD },
  },
  {
    path: "/containers/:containerId",
    name: ROUTE.CONTAINER_DETAIL,
    component: () => import("../views/ContainerDetail.vue"),
    meta: { title: "Container", nav: NAV.DASHBOARD },
  },
  {
    // The param is a Nostr public key — see utils/developer.js.
    path: "/developers/:npub",
    name: ROUTE.DEVELOPER_DETAIL,
    component: () => import("../views/DeveloperProfile.vue"),
    meta: { title: "Developer", nav: NAV.APPS },
  },
  {
    path: "/storage",
    name: ROUTE.STORAGE,
    component: () => import("../views/Storage.vue"),
    meta: { title: "Storage", nav: NAV.STORAGE },
  },
  {
    path: "/logs",
    name: ROUTE.LOGS,
    component: () => import("../views/Logs.vue"),
    meta: { title: "Logs", nav: NAV.LOGS },
  },

  // Last: anything unmatched. Without this an unknown hash rendered an empty
  // <main> under a working sidebar — no message, no redirect.
  {
    path: "/:pathMatch(.*)*",
    name: ROUTE.NOT_FOUND,
    component: () => import("../views/NotFound.vue"),
    meta: { title: "Not found" },
  },
];
