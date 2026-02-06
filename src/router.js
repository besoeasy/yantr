import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      redirect: '/home'
    },
    {
      path: '/home',
      name: 'home',
      component: () => import(/* webpackChunkName: "home" */ './views/Home.vue')
    },
    {
      path: '/apps',
      name: 'apps',
      component: () => import(/* webpackChunkName: "apps" */ './views/Apps.vue')
    },
    {
      path: '/apps/:appname',
      name: 'app-detail',
      component: () => import(/* webpackChunkName: "app-detail" */ './views/AppDetail.vue')
    },
  
    {
      path: '/containers/:id',
      name: 'container-detail',
      component: () => import(/* webpackChunkName: "container-detail" */ './views/ContainerDetail.vue')
    },
    {
      path: '/images',
      name: 'images',
      component: () => import(/* webpackChunkName: "images" */ './views/Images.vue')
    },
    {
      path: '/volumes',
      name: 'volumes',
      component: () => import(/* webpackChunkName: "volumes" */ './views/Volumes.vue')
    },
    {
      path: '/backup',
      name: 'backup',
      component: () => import(/* webpackChunkName: "backup" */ './views/Backup.vue')
    },
    {
      path: '/logs',
      name: 'logs',
      component: () => import(/* webpackChunkName: "logs" */ './views/Logs.vue')
    },
    {
      path: '/extra',
      name: 'extra',
      component: () => import(/* webpackChunkName: "extra" */ './views/Extra.vue')
    }
  ]
})

export default router
