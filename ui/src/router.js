/**
 * router.js — vue-router wiring. The route table itself lives in
 * utils/routes.js, which holds no vue-router import so it stays testable
 * without a DOM.
 */
import { createRouter, createWebHashHistory } from "vue-router";
import { routes } from "./utils/routes.js";

const APP_NAME = "Yantr";

const router = createRouter({
  // Hash history, not HTML5 history: Yantr is served from whatever reverse
  // proxy the user already runs, and none of them can be assumed to rewrite
  // unknown paths to index.html.
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes,

  // Replaces the scroll-to-top that used to happen only as a side effect of
  // remounting the view. Back and forward restore the saved offset; a
  // query-only change (the storage tab) is not a new page and keeps its place.
  scrollBehavior(to, from, savedPosition) {
    if (savedPosition) return savedPosition;
    if (to.path === from.path) return false;
    return { top: 0 };
  },
});

// Every route declares meta.title, so history entries and browser tabs are
// distinguishable. Before this, index.html's <title> was the only one and all
// ten pages shared it.
router.afterEach((to) => {
  const title = to.meta?.title;
  document.title = title ? `${title} · ${APP_NAME}` : APP_NAME;
});

export default router;
