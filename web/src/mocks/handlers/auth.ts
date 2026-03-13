import { http, HttpResponse } from 'msw'
import {
  MOCK_AUTH_TOKEN,
  MOCK_AUTH_USER,
  MOCK_LOGIN_CREDENTIALS,
  MOCK_LOGIN_RESPONSE,
} from '@/mocks/data/auth'

interface LoginRequestBody {
  email?: string
  password?: string
}

function isLoginRequestBody(value: unknown): value is LoginRequestBody {
  if (!isRecord(value)) {
    return false
  }

  const { email, password } = value
  return (
    (email === undefined || typeof email === 'string') &&
    (password === undefined || typeof password === 'string')
  )
}

function createUnauthorizedResponse(message: string) {
  return HttpResponse.json({ error: { code: 'UNAUTHORIZED', message } }, { status: 401 })
}

export const authHandlers = [
  http.post('/api/auth/login', async ({ request }) => {
    const payload = await request.json().catch(() => null)

    if (
      !isLoginRequestBody(payload) ||
      payload?.email !== MOCK_LOGIN_CREDENTIALS.email ||
      payload?.password !== MOCK_LOGIN_CREDENTIALS.password
    ) {
      return createUnauthorizedResponse('Invalid email or password.')
    }

    return HttpResponse.json(MOCK_LOGIN_RESPONSE)
  }),
  http.get('/api/auth/me', ({ request }) => {
    const authorization = request.headers.get('Authorization')

    if (authorization !== `Bearer ${MOCK_AUTH_TOKEN}`) {
      return createUnauthorizedResponse('Authentication required.')
    }

    return HttpResponse.json({ user: MOCK_AUTH_USER })
  }),
]

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
