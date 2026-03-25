import type { RouteRecordRaw } from 'vue-router'
import { createRouter, createWebHistory } from 'vue-router'
import { USER_ROLE } from '@/lib/auth/roles'
import { installAuthGuard } from '@/middleware/auth'
import type { RouteTitleKey } from '@/router/route-title'

const AppShellLayout = () => import('@/layouts/AppShellLayout.vue')
const BlankLayout = () => import('@/layouts/BlankLayout.vue')
const Dashboard = () => import('@/views/Dashboard.vue')
const Login = () => import('@/views/Login.vue')
const Register = () => import('@/views/Register.vue')
const Settings = () => import('@/views/Settings.vue')
const SystemLogs = () => import('@/views/SystemLogsView.vue')
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
  defineAppShellRoute({
    path: 'settings',
    name: 'settings',
    component: Settings,
    meta: { titleKey: 'route.settings', requiresAuth: true },
  }),
  defineAppShellRoute({
    path: 'system-logs',
    name: 'system-logs',
    component: SystemLogs,
    meta: { titleKey: 'route.systemLogs', requiresAuth: true, requiredRole: USER_ROLE.admin, maskUnauthorizedAsNotFound: true },
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
    { path: '/register', name: 'register', component: Register, meta: { titleKey: 'route.register', guestOnly: true } },
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
