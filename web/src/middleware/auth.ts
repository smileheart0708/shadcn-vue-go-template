import type { RouteLocationNormalized, Router } from 'vue-router'
import { hasMinimumUserRole } from '@/lib/auth/roles'
import pinia from '@/stores/pinia'
import { useAuthStore } from '@/stores/auth'

function resolveNotFoundLocation(route: Pick<RouteLocationNormalized, 'path' | 'query' | 'hash'>) {
  return {
    name: 'not-found' as const,
    params: {
      pathMatch: route.path.split('/').filter(Boolean),
    },
    query: route.query,
    hash: route.hash || undefined,
  }
}

export function installAuthGuard(router: Router) {
  router.beforeEach(async (to) => {
    const authStore = useAuthStore(pinia)
    const requiresAuth = to.matched.some((record) => record.meta.requiresAuth)
    const guestOnly = to.matched.some((record) => record.meta.guestOnly)
    const maskUnauthorizedAsNotFound = to.matched.some((record) => record.meta.maskUnauthorizedAsNotFound)
    const requiredRole = to.matched.reduce<number | null>((highestRequiredRole, record) => {
      if (typeof record.meta.requiredRole !== 'number') {
        return highestRequiredRole
      }

      return highestRequiredRole === null ? record.meta.requiredRole : Math.max(highestRequiredRole, record.meta.requiredRole)
    }, null)

    if (!requiresAuth && !guestOnly && requiredRole === null) {
      return true
    }

    await authStore.initialize()

    if (guestOnly && authStore.isAuthenticated) {
      return { name: 'dashboard' }
    }

    if (requiresAuth && !authStore.isAuthenticated) {
      return {
        name: 'login',
        query: to.fullPath === '/login' ? undefined : { redirect: to.fullPath },
      }
    }

    if (requiredRole !== null && !hasMinimumUserRole(authStore.user?.role ?? 0, requiredRole)) {
      if (maskUnauthorizedAsNotFound) {
        return resolveNotFoundLocation(to)
      }

      return { name: 'dashboard' }
    }

    return true
  })
}
