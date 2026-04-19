import { createParser, type EventSourceMessage } from 'eventsource-parser'
import { z } from 'zod'
import { APIError, authApi, normalizeAPIError } from '@/lib/api/client'
import { successEnvelopeSchema } from '@/lib/api/envelope'

export const systemLogEntrySchema = z.object({
  id: z.number().int().positive(),
  timestamp: z.number().int().nonnegative(),
  level: z.string().min(1),
  message: z.string(),
  text: z.string(),
  source: z.string().min(1),
})

export type SystemLogEntry = z.infer<typeof systemLogEntrySchema>

export const auditLogEntrySchema = z.object({
  id: z.number().int().positive(),
  actorUserID: z.number().int().positive().nullable(),
  subjectUserID: z.number().int().positive().nullable(),
  authSessionID: z.string().nullable(),
  eventType: z.string(),
  outcome: z.enum(['success', 'failure']),
  reason: z.string().nullable(),
  ip: z.string().nullable(),
  userAgent: z.string().nullable(),
  metadata: z.record(z.string(), z.unknown()).nullish(),
  occurredAt: z.string(),
})

export type AuditLogEntry = z.infer<typeof auditLogEntrySchema>

export const SYSTEM_LOG_LEVEL_VALUES = ['DEBUG', 'INFO', 'WARN', 'ERROR'] as const
export type SystemLogLevelFilter = 'ALL' | (typeof SYSTEM_LOG_LEVEL_VALUES)[number]

const auditLogsEnvelopeSchema = successEnvelopeSchema(
  z.object({
    items: z.array(auditLogEntrySchema),
    page: z.number().int().positive(),
    pageSize: z.number().int().positive(),
    total: z.number().int().nonnegative(),
  }),
)

interface OpenSystemLogsStreamOptions {
  tail?: number
  signal: AbortSignal
  onOpen?: () => void
  onEntry: (entry: SystemLogEntry) => void
}

export async function listAuditLogs(page = 1, pageSize = 50) {
  try {
    const payload = await authApi
      .get('/api/management/audit-logs', {
        searchParams: {
          page: String(page),
          pageSize: String(pageSize),
        },
      })
      .json<unknown>()
    return auditLogsEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function openSystemLogsStream(options: OpenSystemLogsStreamOptions): Promise<void> {
  try {
    const response = await authApi.get('/api/management/system-logs/stream', {
      headers: {
        Accept: 'text/event-stream',
      },
      searchParams: {
        tail: String(options.tail ?? 200),
      },
      signal: options.signal,
    })

    if (!response.body) {
      throw new APIError(500, 'System log stream is unavailable.', 'system_log_stream_unavailable')
    }

    options.onOpen?.()

    const parser = createParser({
      onEvent(event) {
        handleStreamEvent(event, options.onEntry)
      },
    })

    const reader = response.body.getReader()
    const decoder = new TextDecoder()

    for (;;) {
      const { done, value } = await reader.read()
      if (done) {
        break
      }

      parser.feed(decoder.decode(value, { stream: true }))
    }

    parser.feed(decoder.decode())
  } catch (error) {
    return normalizeAPIError(error)
  }
}

function handleStreamEvent(event: EventSourceMessage, onEntry: (entry: SystemLogEntry) => void) {
  if (event.event !== 'log') {
    return
  }

  const payload: unknown = JSON.parse(event.data)
  onEntry(systemLogEntrySchema.parse(payload))
}
