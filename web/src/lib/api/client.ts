import ky, { HTTPError, type Options } from 'ky'
import { readAuthToken } from '@/lib/auth/token'

interface APIErrorPayload {
  error?: { code?: string; message?: string }
}

interface AuthClientContext {
  authRetryAttempted?: boolean
  skipAuthRefresh?: boolean
}

type RefreshAccessTokenHandler = () => Promise<string>
type UnauthorizedHandler = () => Promise<void> | void

let refreshAccessTokenHandler: RefreshAccessTokenHandler | null = null
let unauthorizedHandler: UnauthorizedHandler | null = null
let refreshPromise: Promise<string> | null = null

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

export function registerAuthClientHandlers(handlers: {
  refreshAccessToken: RefreshAccessTokenHandler
  onUnauthorized: UnauthorizedHandler
}) {
  refreshAccessTokenHandler = handlers.refreshAccessToken
  unauthorizedHandler = handlers.onUnauthorized
}

export function clearAuthClientHandlers() {
  refreshAccessTokenHandler = null
  unauthorizedHandler = null
  refreshPromise = null
}

const sharedHooks = {
  beforeRequest: [
    (request: Request, options: Options) => {
      if (!request.headers.has('Accept')) {
        request.headers.set('Accept', 'application/json')
      }

      const requestHasJSONBody = 'json' in options && options.json !== undefined && !request.headers.has('Content-Type')
      if (requestHasJSONBody) {
        request.headers.set('Content-Type', 'application/json')
      }
    },
  ],
}

export const baseApi = ky.create({
  retry: 0,
  credentials: 'same-origin',
  hooks: sharedHooks,
})

export const authApi = ky.create({
  retry: 0,
  credentials: 'same-origin',
  hooks: {
    ...sharedHooks,
    beforeRequest: [
      ...sharedHooks.beforeRequest,
      (request) => {
        const token = readAuthToken()
        if (token && !request.headers.has('Authorization')) {
          request.headers.set('Authorization', `Bearer ${token}`)
        }
      },
    ],
    afterResponse: [
      async (request, options, response) => {
        const context = readAuthClientContext(options)
        if (response.status !== 401 || context.skipAuthRefresh || context.authRetryAttempted || isAuthLifecycleRequest(request)) {
          return response
        }

        if (!refreshAccessTokenHandler) {
          return response
        }

        try {
          await refreshAccessTokenSingleFlight()
        } catch {
          await unauthorizedHandler?.()
          return response
        }

        const headers = new Headers(options.headers ?? request.headers)
        const nextToken = readAuthToken()
        if (nextToken) {
          headers.set('Authorization', `Bearer ${nextToken}`)
        }

        return authApi(request.url, {
          ...options,
          headers,
          context: {
            ...context,
            authRetryAttempted: true,
          } satisfies AuthClientContext,
        })
      },
    ],
  },
})

export async function toAPIError(error: HTTPError): Promise<APIError> {
  const payload = await readResponsePayload(error.response.clone())
  const errorPayload = isAPIErrorPayload(payload) ? payload : undefined

  return new APIError(error.response.status, errorPayload?.error?.message ?? `Request failed with status ${String(error.response.status)}`, errorPayload?.error?.code, payload)
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

async function refreshAccessTokenSingleFlight(): Promise<string> {
  if (refreshPromise) {
    return refreshPromise
  }

  if (!refreshAccessTokenHandler) {
    throw new Error('No refresh handler registered.')
  }

  refreshPromise = refreshAccessTokenHandler().finally(() => {
    refreshPromise = null
  })

  return refreshPromise
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

function readAuthClientContext(options: Options): AuthClientContext {
  if (!isRecord(options.context)) {
    return {}
  }

  const { authRetryAttempted, skipAuthRefresh } = options.context
  return {
    authRetryAttempted: authRetryAttempted === true,
    skipAuthRefresh: skipAuthRefresh === true,
  }
}

function isAuthLifecycleRequest(request: Request): boolean {
  const pathname = new URL(request.url, window.location.origin).pathname
  return pathname === '/api/auth/login' || pathname === '/api/auth/refresh' || pathname === '/api/auth/logout'
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
  return (code === undefined || typeof code === 'string') && (message === undefined || typeof message === 'string')
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
