import type { SystemLogEntry, SystemLogLevelFilter } from '@/lib/api/system-logs'

export const SYSTEM_LOG_EXPORT_COUNT_VALUES = ['ALL', 10, 20, 50, 100] as const
export const SYSTEM_LOG_EXPORT_FORMAT_VALUES = ['csv', 'txt', 'json'] as const

export type SystemLogExportCount = (typeof SYSTEM_LOG_EXPORT_COUNT_VALUES)[number]
export type SystemLogExportFormat = (typeof SYSTEM_LOG_EXPORT_FORMAT_VALUES)[number]

interface ExportSystemLogsOptions {
  entries: SystemLogEntry[]
  count: SystemLogExportCount
  level: SystemLogLevelFilter
  format: SystemLogExportFormat
}

export function formatSystemLogTimestamp(timestamp: number | null | undefined): string {
  if (!timestamp) {
    return ''
  }

  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  }).format(new Date(timestamp * 1000))
}

export function selectSystemLogEntries(options: ExportSystemLogsOptions): SystemLogEntry[] {
  const filteredEntries = options.entries.filter((entry) => options.level === 'ALL' || entry.level === options.level)

  if (options.count === 'ALL') {
    return filteredEntries
  }

  return filteredEntries.slice(-options.count)
}

export function downloadSystemLogs(options: ExportSystemLogsOptions): number {
  const selectedEntries = selectSystemLogEntries(options)
  const blob = createSystemLogExportBlob(selectedEntries, options.format)
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')

  link.href = url
  link.download = `system-logs-${formatExportFileDate(new Date())}.${options.format}`
  link.click()
  window.URL.revokeObjectURL(url)

  return selectedEntries.length
}

function createSystemLogExportBlob(entries: SystemLogEntry[], format: SystemLogExportFormat): Blob {
  switch (format) {
    case 'json':
      return new Blob([JSON.stringify(entries, null, 2)], { type: 'application/json;charset=utf-8' })
    case 'txt':
      return new Blob([serializeSystemLogsTXT(entries)], { type: 'text/plain;charset=utf-8' })
    case 'csv':
    default:
      return new Blob([`\uFEFF${serializeSystemLogsCSV(entries)}`], { type: 'text/csv;charset=utf-8' })
  }
}

function serializeSystemLogsCSV(entries: SystemLogEntry[]): string {
  const header = ['timestamp', 'level', 'source', 'message', 'text']
  const rows = entries.map((entry) => [formatSystemLogTimestamp(entry.timestamp), entry.level, entry.source, entry.message, entry.text])

  return [header, ...rows].map((row) => row.map(escapeCSVValue).join(',')).join('\r\n')
}

function serializeSystemLogsTXT(entries: SystemLogEntry[]): string {
  return entries.map((entry) => `[${formatSystemLogTimestamp(entry.timestamp)}] [${entry.level}] [${entry.source}] ${entry.text}`).join('\r\n')
}

function escapeCSVValue(value: string): string {
  return `"${value.replace(/"/g, '""')}"`
}

function formatExportFileDate(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')

  return `${year}${month}${day}-${hours}${minutes}${seconds}`
}
