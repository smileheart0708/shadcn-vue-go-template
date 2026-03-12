import type { RouteRecordRaw } from 'vue-router'
import { createRouter, createWebHistory } from 'vue-router'
import AppShellLayout from '@/layouts/AppShellLayout.vue'
import BlankLayout from '@/layouts/BlankLayout.vue'
import Dashboard from '@/views/Dashboard.vue'
import NotFound from '@/views/NotFound.vue'

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
