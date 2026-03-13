import type { RouteLocationNormalizedLoaded } from 'vue-router'
import type { MessageSchema } from '@/locales/schema'

export type RouteTitleKey = `route.${Extract<keyof MessageSchema['route'], string>}`

export function getRouteTitleKey(route: Pick<RouteLocationNormalizedLoaded, 'matched'>): RouteTitleKey | null {
  for (const record of [...route.matched].reverse()) {
    if (record.meta.titleKey) {
      return record.meta.titleKey
    }
  }

  return null
}
