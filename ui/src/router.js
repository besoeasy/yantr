import { createRouter, createWebHashHistory } from "vue-router";

const router = createRouter({
  history: createWebHashHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: "/",
      redirect: "/home",
    },
    {
      path: "/home",
      name: "home",
      component: () => import(/* webpackChunkName: "home" */ "./views/Home.vue"),
    },
    {
      path: "/apps",
      name: "apps",
      component: () => import(/* webpackChunkName: "apps" */ "./views/Apps.vue"),
    },
    {
      path: "/apps/:appname",
      name: "app-install",
      component: () => import(/* webpackChunkName: "app-install" */ "./views/AppDetail.vue"),
    },
    {
      path: "/containers/:id",
      name: "container-inspect",
      component: () => import(/* webpackChunkName: "container-inspect" */ "./views/ContainerDetail.vue"),
    },
    {
      path: "/stacks/:projectId",
      name: "stack-detail",
      component: () => import(/* webpackChunkName: "stack-detail" */ "./views/StackView.vue"),
    },
    {
      path: "/storage",
      name: "storage",
      component: () => import(/* webpackChunkName: "storage" */ "./views/Storage.vue"),
    },
    // Images and volumes were separate pages with identical chrome and their own
    // nested active/unused tabs. They now share one tabbed page; these redirects
    // keep deep links, bookmarks, and the widget buttons working.
    {
      path: "/images",
      redirect: () => ({ path: "/storage", query: { tab: "images-active" } }),
    },
    {
      path: "/volumes",
      redirect: () => ({ path: "/storage", query: { tab: "volumes-active" } }),
    },
    {
      path: "/logs",
      name: "logs",
      component: () => import(/* webpackChunkName: "logs" */ "./views/Logs.vue"),
    },
    {
      path: "/telemetry",
      name: "telemetry",
      component: () => import(/* webpackChunkName: "telemetry" */ "./views/Telemetry.vue"),
    },
  ],
});

export default router;
