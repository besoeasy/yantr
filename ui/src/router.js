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
      // One developer's profile and everything they maintain in the catalog.
      // :handle accepts either "besoeasy" (GitHub implied) or
      // "platform:handle" — see utils/developer.js.
      path: "/developer/:handle",
      name: "developer",
      component: () => import(/* webpackChunkName: "developer" */ "./views/DeveloperProfile.vue"),
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
    {
      path: "/logs",
      name: "logs",
      component: () => import(/* webpackChunkName: "logs" */ "./views/Logs.vue"),
    },
    {
      path: "/presence",
      name: "presence",
      component: () => import(/* webpackChunkName: "presence" */ "./views/Presence.vue"),
    },
  ],
});

export default router;
