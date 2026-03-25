import { createParser, type EventSourceMessage } from 'eventsource-parser'
import { z } from 'zod'
import { APIError } from '@/lib/api/client'
import { readAuthToken } from '@/lib/auth/token'

export const systemLogEntrySchema = z.object({
  id: z.number().int().positive(),
  timestamp: z.number().int().nonnegative(),
  level: z.string().min(1),
  message: z.string(),
  text: z.string(),
  source: z.string().min(1),
})

export type SystemLogEntry = z.infer<typeof systemLogEntrySchema>

export const SYSTEM_LOG_LEVEL_VALUES = ['DEBUG', 'INFO', 'WARN', 'ERROR'] as const
export type SystemLogLevelFilter = 'ALL' | (typeof SYSTEM_LOG_LEVEL_VALUES)[number]

interface OpenSystemLogsStreamOptions {
  tail?: number
  signal: AbortSignal
  onEntry: (entry: SystemLogEntry) => void
}

export async function openSystemLogsStream(options: OpenSystemLogsStreamOptions): Promise<void> {
  const token = readAuthToken()
  if (!token) {
    throw new APIError(401, 'Authentication is required.', 'unauthorized')
  }

  const url = new URL('/api/admin/system-logs/stream', window.location.origin)
  url.searchParams.set('tail', String(options.tail ?? 200))

  const response = await fetch(url, {
    method: 'GET',
    headers: {
      Accept: 'text/event-stream',
      Authorization: `Bearer ${token}`,
    },
    credentials: 'same-origin',
    signal: options.signal,
  })

  if (!response.ok) {
    throw await toStreamAPIError(response)
  }

  if (!response.body) {
    throw new APIError(500, 'System log stream is unavailable.', 'system_log_stream_unavailable')
  }

  const parser = createParser({
    onEvent(event) {
      handleStreamEvent(event, options.onEntry)
    },
  })

  const reader = response.body.getReader()
  const decoder = new TextDecoder()

  while (true) {
    const { done, value } = await reader.read()
    if (done) {
      break
    }

    parser.feed(decoder.decode(value, { stream: true }))
  }

  parser.feed(decoder.decode())
}

function handleStreamEvent(event: EventSourceMessage, onEntry: (entry: SystemLogEntry) => void) {
  if (event.event !== 'log') {
    return
  }

  const payload: unknown = JSON.parse(event.data)
  onEntry(systemLogEntrySchema.parse(payload))
}

async function toStreamAPIError(response: Response): Promise<APIError> {
  let payload: unknown

  try {
    payload = response.headers.get('Content-Type')?.includes('application/json') ? await response.json() : await response.text()
  } catch {
    payload = undefined
  }

  const errorRecord = isRecord(payload) && isRecord(payload.error) ? payload.error : undefined
  const code = typeof errorRecord?.code === 'string' ? errorRecord.code : undefined
  const message = typeof errorRecord?.message === 'string' ? errorRecord.message : `Request failed with status ${String(response.status)}`

  return new APIError(response.status, message, code, payload)
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
