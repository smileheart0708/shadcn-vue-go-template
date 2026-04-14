import type { RouteRecordRaw } from 'vue-router'
import { createRouter, createWebHistory } from 'vue-router'
import { USER_ROLE } from '@/lib/auth/roles'
import { installAuthGuard } from '@/middleware/auth'
import { installRequestLoadingTracking, installRouteLoading } from '@/router/route-loading'
import type { RouteTitleKey } from '@/router/route-title'

const AppShellLayout = async () => import('@/layouts/AppShellLayout.vue')
const BlankLayout = async () => import('@/layouts/BlankLayout.vue')
const Dashboard = async () => import('@/views/DashboardView.vue')
const Home = async () => import('@/views/HomeView.vue')
const Login = async () => import('@/views/LoginView.vue')
const Register = async () => import('@/views/RegisterView.vue')
const Settings = async () => import('@/views/SettingsView.vue')
const AdminSettings = async () => import('@/views/AdminSettingsView.vue')
const AdminUsers = async () => import('@/views/AdminUsersView.vue')
const SystemLogs = async () => import('@/views/SystemLogsView.vue')
const Tasks = async () => import('@/views/TasksView.vue')
const NotFound = async () => import('@/views/NotFoundView.vue')

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
    path: 'system-configs',
    name: 'system-config',
    component: AdminSettings,
    meta: { titleKey: 'route.systemConfig', requiresAuth: true, requiredRole: USER_ROLE.admin, maskUnauthorizedAsNotFound: true },
  }),
  defineAppShellRoute({
    path: 'admin-users',
    name: 'admin-users',
    component: AdminUsers,
    meta: { titleKey: 'route.adminUsers', requiresAuth: true, requiredRole: USER_ROLE.admin, maskUnauthorizedAsNotFound: true },
  }),
  {
    path: 'tasks',
    name: 'tasks',
    component: Tasks,
    meta: { titleKey: 'route.tasks', requiresAuth: true },
  },
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
      name: 'home',
      component: Home,
    },
    {
      path: '/',
      component: AppShellLayout,
      children: appShellRoutes,
    },
    { path: '/login', name: 'login', component: Login, meta: { titleKey: 'route.login', guestOnly: true } },
    { path: '/register', name: 'register', component: Register, meta: { titleKey: 'route.register', guestOnly: true } },
    {
      path: '/:pathMatch(.*)*',
      component: BlankLayout,
      children: [{ path: '', name: 'not-found', component: NotFound, meta: { titleKey: 'route.notFound' } }],
    },
  ],
})

installAuthGuard(router)
installRouteLoading(router)
installRequestLoadingTracking()

export default router
