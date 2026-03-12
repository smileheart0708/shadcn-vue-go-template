import type { Router } from 'vue-router'
import { useAuth } from '@/composables/auth/useAuth'

export function installAuthGuard(router: Router) {
  router.beforeEach(async (to) => {
    const auth = useAuth()
    const requiresAuth = to.matched.some((record) => record.meta.requiresAuth)
    const guestOnly = to.matched.some((record) => record.meta.guestOnly)

    if (!requiresAuth && !guestOnly) {
      return true
    }

    await auth.initialize()

    if (guestOnly && auth.isAuthenticated.value) {
      return { name: 'dashboard' }
    }

    if (requiresAuth && !auth.isAuthenticated.value) {
      return {
        name: 'login',
        query: to.fullPath === '/login' ? undefined : { redirect: to.fullPath },
      }
    }

    return true
  })
}
