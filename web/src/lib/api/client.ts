import ky, { HTTPError } from 'ky'
import { readAuthToken } from '@/lib/auth/token'

interface APIErrorPayload {
  error?: { code?: string; message?: string }
}

export class APIError extends Error {
  status: number
  code?: string
  payload?: unknown

  constructor(status: number, message: string, code?: string, payload?: unknown) {
    super(message)
    this.name = 'APIError'
    this.status = status
    this.code = code
    this.payload = payload
  }
}

export const api = ky.create({
  retry: 0,
  hooks: {
    beforeRequest: [
      (request, options) => {
        if (!request.headers.has('Accept')) {
          request.headers.set('Accept', 'application/json')
        }

        const requestMayHaveJsonBody =
          options.body !== undefined &&
          request.body !== null &&
          !request.headers.has('Content-Type')
        if (requestMayHaveJsonBody) {
          request.headers.set('Content-Type', 'application/json')
        }

        const token = readAuthToken()
        if (token && !request.headers.has('Authorization')) {
          request.headers.set('Authorization', `Bearer ${token}`)
        }
      },
    ],
  },
})

export async function toAPIError(error: HTTPError): Promise<APIError> {
  const payload = await readResponsePayload(error.response.clone())
  const errorPayload = isAPIErrorPayload(payload) ? payload : undefined

  return new APIError(
    error.response.status,
    errorPayload?.error?.message ?? `Request failed with status ${String(error.response.status)}`,
    errorPayload?.error?.code,
    payload,
  )
}

export async function normalizeAPIError(error: unknown): Promise<never> {
  if (error instanceof APIError) {
    throw error
  }

  if (error instanceof HTTPError) {
    throw await toAPIError(error)
  }

  throw error
}

async function readResponsePayload(response: Response): Promise<unknown> {
  const contentType = response.headers.get('Content-Type') ?? ''
  if (contentType.includes('application/json')) {
    return response.json()
  }

  const text = await response.text()
  if (text.length === 0) {
    return undefined
  }

  return text
}

function isAPIErrorPayload(payload: unknown): payload is APIErrorPayload {
  if (!isRecord(payload)) {
    return false
  }

  const { error } = payload
  if (error === undefined) {
    return true
  }

  if (!isRecord(error)) {
    return false
  }

  const { code, message } = error
  return (
    (code === undefined || typeof code === 'string') &&
    (message === undefined || typeof message === 'string')
  )
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
