import type { Router } from 'vue-router'
import pinia from '@/stores/pinia'
import { useAuthStore } from '@/stores/auth'

export function installAuthGuard(router: Router) {
  router.beforeEach(async (to) => {
    const authStore = useAuthStore(pinia)
    const requiresAuth = to.matched.some((record) => record.meta.requiresAuth)
    const guestOnly = to.matched.some((record) => record.meta.guestOnly)

    if (!requiresAuth && !guestOnly) {
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

    return true
  })
}
