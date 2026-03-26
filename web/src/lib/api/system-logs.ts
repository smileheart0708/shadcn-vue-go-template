import { createParser, type EventSourceMessage } from 'eventsource-parser'
import { z } from 'zod'
import { APIError, authApi, normalizeAPIError } from '@/lib/api/client'

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
  onOpen?: () => void
  onEntry: (entry: SystemLogEntry) => void
}

export async function openSystemLogsStream(options: OpenSystemLogsStreamOptions): Promise<void> {
  try {
    const response = await authApi.get('/api/admin/system-logs/stream', {
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

    while (true) {
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
