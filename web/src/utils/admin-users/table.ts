import type { BadgeVariants } from '@/components/ui/badge'
import type { ManagedUser } from '@/lib/api/admin-users'

export function getAdminUsersTotalPages(
  total: number,
  pageSize: number,
): number {
  return Math.max(1, Math.ceil(total / pageSize))
}

export function createAdminUserDateFormatter(
  locale: string,
): Intl.DateTimeFormat {
  return new Intl.DateTimeFormat(locale, {
    dateStyle: 'medium',
    timeStyle: 'short',
  })
}

export function formatAdminUserDateTime(
  value: string,
  formatter: Intl.DateTimeFormat,
): string {
  return formatter.format(new Date(value))
}

export function formatNullableAdminUserDateTime(
  value: string | null,
  formatter: Intl.DateTimeFormat,
  fallback: string,
): string {
  return value === null ? fallback : formatAdminUserDateTime(value, formatter)
}

export function getAdminUserStatusBadgeVariant(
  status: ManagedUser['status'],
): NonNullable<BadgeVariants['variant']> {
  return status === 'active' ? 'success' : 'secondary'
}

export function hasAdminUserAction(
  user: ManagedUser,
  action: ManagedUser['actions'][number],
): boolean {
  return user.actions.includes(action)
}
