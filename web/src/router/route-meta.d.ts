import 'vue-router'
import type { RouteTitleKey } from '@/router/route-title'

declare module 'vue-router' {
  interface RouteMeta {
    titleKey?: RouteTitleKey
    requiresAuth?: boolean
    guestOnly?: boolean
    requiredRole?: number
    maskUnauthorizedAsNotFound?: boolean
  }
}

export {}
