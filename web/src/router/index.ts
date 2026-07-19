import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'
import Layout from '../views/Layout.vue'

const routes: Array<RouteRecordRaw> = [
  {
    path: '/',
    component: Layout,
    redirect: '/projects',
    children: [
      {
        path: 'projects',
        name: 'Projects',
        component: () => import('../views/ProjectList.vue')
      },
      {
        path: 'projects/:id/environments',
        name: 'Environments',
        component: () => import('../views/EnvironmentConfig.vue')
      },
      {
        path: 'servers',
        name: 'Servers',
        component: () => import('../views/ServerList.vue')
      },
      {
        path: 'deployments/:id',
        name: 'DeploymentDetail',
        component: () => import('../views/DeploymentDetail.vue')
      }
    ]
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

export default router
