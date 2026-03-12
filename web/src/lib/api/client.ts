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

type PayloadParser<T> = (payload: unknown) => T

export async function apiRequest(input: string, init?: RequestInit): Promise<void>
export async function apiRequest<T>(
  input: string,
  parser: PayloadParser<T>,
  init?: RequestInit,
): Promise<T>
export async function apiRequest<T>(
  input: string,
  parserOrInit: PayloadParser<T> | RequestInit = {},
  maybeInit: RequestInit = {},
): Promise<T | void> {
  const parser = typeof parserOrInit === 'function' ? parserOrInit : undefined
  const init = typeof parserOrInit === 'function' ? maybeInit : parserOrInit
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

  const response = await fetch(input, { ...init, headers })

  if (response.status === 204) {
    return
  }

  const payload = await readResponsePayload(response)
  if (!response.ok) {
    const errorPayload = isAPIErrorPayload(payload) ? payload : undefined
    throw new APIError(
      response.status,
      errorPayload?.error?.message ?? `Request failed with status ${response.status}`,
      errorPayload?.error?.code,
      payload,
    )
  }

  return parser ? parser(payload) : undefined
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
