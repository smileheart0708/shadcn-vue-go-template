import type { RouteRecordRaw } from 'vue-router'
import { createRouter, createWebHistory } from 'vue-router'

const AppShellLayout = () => import('@/layouts/AppShellLayout.vue')
const BlankLayout = () => import('@/layouts/BlankLayout.vue')
const Dashboard = () => import('@/views/Dashboard.vue')
const NotFound = () => import('@/views/NotFound.vue')

function defineAppShellRoute<T extends RouteRecordRaw & { meta: { title: string } }>(route: T) {
  return route
}

const appShellRoutes = [
  defineAppShellRoute({
    path: 'dashboard',
    name: 'dashboard',
    component: Dashboard,
    meta: {
      title: 'Dashboard',
    },
  }),
] satisfies RouteRecordRaw[]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      component: AppShellLayout,
      children: [
        {
          path: '',
          redirect: { name: 'dashboard' },
        },
        ...appShellRoutes,
      ],
    },
    // Only routes nested under AppShellLayout render the sidebar/header shell.
    {
      path: '/:pathMatch(.*)*',
      component: BlankLayout,
      children: [
        {
          path: '',
          name: 'not-found',
          component: NotFound,
        },
      ],
    },
  ],
})

export default router
