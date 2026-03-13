import type { RouteRecordRaw } from 'vue-router'
import { createRouter, createWebHistory } from 'vue-router'
import { installAuthGuard } from '@/middleware/auth'
import type { RouteTitleKey } from '@/router/route-title'

const AppShellLayout = () => import('@/layouts/AppShellLayout.vue')
const BlankLayout = () => import('@/layouts/BlankLayout.vue')
const Dashboard = () => import('@/views/Dashboard.vue')
const Login = () => import('@/views/Login.vue')
const NotFound = () => import('@/views/NotFound.vue')

function defineAppShellRoute<T extends RouteRecordRaw & { meta: { titleKey: RouteTitleKey } }>(route: T) {
  return route
}

const appShellRoutes = [
  defineAppShellRoute({
    path: 'dashboard',
    name: 'dashboard',
    component: Dashboard,
    meta: { titleKey: 'route.dashboard', requiresAuth: true },
  }),
] satisfies RouteRecordRaw[]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      component: AppShellLayout,
      children: [{ path: '', redirect: { name: 'dashboard' } }, ...appShellRoutes],
    },
    { path: '/login', name: 'login', component: Login, meta: { titleKey: 'route.login', guestOnly: true } },
    // Only routes nested under AppShellLayout render the sidebar/header shell.
    {
      path: '/:pathMatch(.*)*',
      component: BlankLayout,
      children: [{ path: '', name: 'not-found', component: NotFound, meta: { titleKey: 'route.notFound' } }],
    },
  ],
})

installAuthGuard(router)

export default router
