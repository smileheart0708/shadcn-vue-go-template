import {
  isSystemLogLevel,
  type SystemLogEntry,
  type SystemLogHistoryLimit,
  type SystemLogLevel,
} from '@/lib/api/system-logs'
import type { SystemLogExportFormat } from '@/stores/system-logs-preferences'
import { applyHistoryLimit, formatSystemLogTimestamp } from './console'

interface ExportSystemLogsOptions {
  entries: readonly SystemLogEntry[]
  historyLimit: SystemLogHistoryLimit
  levels: readonly SystemLogLevel[]
}

export function selectSystemLogEntries(
  options: ExportSystemLogsOptions,
): SystemLogEntry[] {
  const selectedLevels = new Set(options.levels)
  const filteredEntries = options.entries.filter(
    (entry) => isSystemLogLevel(entry.level) && selectedLevels.has(entry.level),
  )
  return applyHistoryLimit(filteredEntries, options.historyLimit)
}

export function createSystemLogExportBlob(
  entries: readonly SystemLogEntry[],
  format: SystemLogExportFormat,
): Blob {
  switch (format) {
    case 'json':
      return new Blob([JSON.stringify(entries, null, 2)], {
        type: 'application/json;charset=utf-8',
      })
    case 'txt':
      return new Blob([serializeSystemLogsTXT(entries)], {
        type: 'text/plain;charset=utf-8',
      })
    case 'csv':
    default:
      return new Blob([`\uFEFF${serializeSystemLogsCSV(entries)}`], {
        type: 'text/csv;charset=utf-8',
      })
  }
}

export function getSystemLogExportFileName(
  format: SystemLogExportFormat,
): string {
  return `system-logs-${formatExportFileDate(new Date())}.${format}`
}

function serializeSystemLogsCSV(entries: readonly SystemLogEntry[]): string {
  const header = ['timestamp', 'level', 'source', 'message', 'text']
  const rows = entries.map((entry) => [
    formatSystemLogTimestamp(entry.timestamp),
    entry.level,
    entry.source,
    entry.message,
    entry.text,
  ])

  return [header, ...rows]
    .map((row) => row.map(escapeCSVValue).join(','))
    .join('\r\n')
}

function serializeSystemLogsTXT(entries: readonly SystemLogEntry[]): string {
  return entries
    .map(
      (entry) =>
        `[${formatSystemLogTimestamp(entry.timestamp)}] [${entry.level}] [${entry.source}] ${entry.text}`,
    )
    .join('\r\n')
}

function escapeCSVValue(value: string): string {
  return `"${value.replace(/"/g, '""')}"`
}

function formatExportFileDate(date: Date): string {
  const year = String(date.getFullYear())
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')

  return `${year}${month}${day}-${hours}${minutes}${seconds}`
}
