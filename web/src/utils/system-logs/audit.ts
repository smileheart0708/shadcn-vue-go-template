import type { BadgeVariants } from '@/components/ui/badge'
import type { AuditLogEntry } from '@/lib/api/system-logs'

export function getAuditTotalPages(total: number, pageSize: number): number {
  return Math.max(1, Math.ceil(total / pageSize))
}

export function getAuditOutcomeVariant(outcome: AuditLogEntry['outcome']): NonNullable<BadgeVariants['variant']> {
  return outcome === 'success' ? 'outline' : 'destructive'
}

export function createAuditDateFormatter(locale: string): Intl.DateTimeFormat {
  return new Intl.DateTimeFormat(locale, {
    dateStyle: 'medium',
    timeStyle: 'short',
  })
}

export function formatAuditDate(value: string, formatter: Intl.DateTimeFormat): string {
  return formatter.format(new Date(value))
}

export function formatAuditActor(id: number | null, noneLabel: string): string {
  return id === null ? noneLabel : `#${String(id)}`
}

export function formatAuditReason(reason: string | null, noneLabel: string): string {
  return reason === null || reason.trim() === '' ? noneLabel : reason
}
