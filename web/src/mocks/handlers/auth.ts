import { http, HttpResponse } from 'msw'
import { MOCK_AUTH_TOKEN, MOCK_AUTH_USER, MOCK_LOGIN_CREDENTIALS, MOCK_LOGIN_RESPONSE } from '@/mocks/data/auth'

interface LoginRequestBody {
  identifier?: string
  password?: string
}

function isLoginRequestBody(value: unknown): value is LoginRequestBody {
  if (!isRecord(value)) {
    return false
  }

  const { identifier, password } = value
  return (identifier === undefined || typeof identifier === 'string') && (password === undefined || typeof password === 'string')
}

function createUnauthorizedResponse(message: string) {
  return HttpResponse.json({ success: false, error: { code: 'unauthorized', message } }, { status: 401 })
}

export const authHandlers = [
  http.post('/api/auth/login', async ({ request }) => {
    const payload = await request.json().catch((): null => null)

    if (!isLoginRequestBody(payload) || payload?.identifier !== MOCK_LOGIN_CREDENTIALS.identifier || payload?.password !== MOCK_LOGIN_CREDENTIALS.password) {
      return HttpResponse.json(
        {
          success: false,
          error: { code: 'invalid_credentials', message: 'Invalid credentials.' },
        },
        { status: 401 },
      )
    }

    return HttpResponse.json({
      success: true,
      data: MOCK_LOGIN_RESPONSE,
    })
  }),
  http.get('/api/auth/me', ({ request }) => {
    const authorization = request.headers.get('Authorization')

    if (authorization !== `Bearer ${MOCK_AUTH_TOKEN}`) {
      return createUnauthorizedResponse('Authentication required.')
    }

    return HttpResponse.json({ success: true, data: MOCK_AUTH_USER })
  }),
  http.patch('/api/account/profile', async ({ request }) => {
    const payload = await request.json().catch((): null => null)
    if (!isRecord(payload)) {
      return HttpResponse.json({ success: false, error: { code: 'invalid_request', message: 'Invalid request.' } }, { status: 400 })
    }

    return HttpResponse.json({
      success: true,
      data: {
        ...MOCK_AUTH_USER,
        username: typeof payload.username === 'string' ? payload.username : MOCK_AUTH_USER.username,
        email: typeof payload.email === 'string' ? payload.email : null,
      },
    })
  }),
  http.post('/api/account/avatar', () => {
    return HttpResponse.json({
      success: true,
      data: {
        ...MOCK_AUTH_USER,
        avatarUrl: '/api/avatars/mock-avatar.png',
      },
    })
  }),
  http.post('/api/account/password', () => {
    return HttpResponse.json({
      success: true,
      data: {
        ...MOCK_AUTH_USER,
        mustChangePassword: false,
      },
    })
  }),
  http.delete('/api/account', () => HttpResponse.json({ success: true, data: { deleted: true } })),
]

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
