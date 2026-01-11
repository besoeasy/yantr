import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      redirect: '/containers'
    },
    {
      path: '/apps',
      name: 'apps',
      component: () => import('./views/Apps.vue')
    },
    {
      path: '/containers',
      name: 'containers',
      component: () => import('./views/Containers.vue')
    },
    {
      path: '/containers/:id',
      name: 'container-detail',
      component: () => import('./views/ContainerDetail.vue')
    },
    {
      path: '/images',
      name: 'images',
      component: () => import('./views/Images.vue')
    },
    {
      path: '/logs',
      name: 'logs',
      component: () => import('./views/Logs.vue')
    }
  ]
})

export default router
