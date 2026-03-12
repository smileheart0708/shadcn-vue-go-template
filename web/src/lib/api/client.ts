import { readAuthToken } from '@/lib/auth/token'

interface APIErrorPayload {
  error?: {
    code?: string
    message?: string
  }
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

export async function apiRequest<T>(input: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  const token = readAuthToken()
  const isFormData = init.body instanceof FormData

  if (!headers.has('Accept')) {
    headers.set('Accept', 'application/json')
  }

  if (init.body !== undefined && !isFormData && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  if (token && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const response = await fetch(input, {
    ...init,
    headers,
  })

  if (response.status === 204) {
    return undefined as T
  }

  const payload = await readResponsePayload(response)
  if (!response.ok) {
    const errorPayload = payload as APIErrorPayload | undefined
    throw new APIError(
      response.status,
      errorPayload?.error?.message ?? `Request failed with status ${response.status}`,
      errorPayload?.error?.code,
      payload,
    )
  }

  return payload as T
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
