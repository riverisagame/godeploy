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
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue')
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('../views/NotFound.vue')
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

router.beforeEach((to, _from, next) => {
  const token = localStorage.getItem('token')
  if (to.path !== '/login' && !token) {
    next('/login')
  } else if (to.path === '/login' && token) {
    next('/')
  } else {
    next()
  }
})

export default router
