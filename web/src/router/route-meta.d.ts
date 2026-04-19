import 'vue-router'
import type { Capability } from '@/lib/api/auth'
import type { RouteTitleKey } from '@/router/route-title'

declare module 'vue-router' {
  interface RouteMeta {
    titleKey?: RouteTitleKey
    requiresAuth?: boolean
    guestOnly?: boolean
    requiredCapabilities?: Capability[]
    maskUnauthorizedAsNotFound?: boolean
  }
}

export {}
