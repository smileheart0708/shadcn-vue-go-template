import { http, HttpResponse } from 'msw'
import { MOCK_AUTH_TOKEN, MOCK_AUTH_USER, MOCK_LOGIN_CREDENTIALS, MOCK_LOGIN_RESPONSE, MOCK_REFRESH_RESPONSE } from '@/mocks/data/auth'

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
  (() => {
    let sessionActive = false
    let currentAccessToken = MOCK_AUTH_TOKEN

    const activateSession = () => {
      sessionActive = true
      currentAccessToken = MOCK_LOGIN_RESPONSE.accessToken
    }

    return [
      http.post('/api/auth/login', async ({ request }) => {
        const payload = await request.json().catch((): null => null)

        if (!isLoginRequestBody(payload)) {
          return HttpResponse.json(
            {
              success: false,
              error: { code: 'invalid_credentials', message: 'Invalid credentials.' },
            },
            { status: 401 },
          )
        }

        if (payload.identifier !== MOCK_LOGIN_CREDENTIALS.identifier || payload.password !== MOCK_LOGIN_CREDENTIALS.password) {
          return HttpResponse.json(
            {
              success: false,
              error: { code: 'invalid_credentials', message: 'Invalid credentials.' },
            },
            { status: 401 },
          )
        }

        activateSession()

        return HttpResponse.json({
          success: true,
          data: MOCK_LOGIN_RESPONSE,
        })
      }),
      http.post('/api/auth/refresh', () => {
        if (!sessionActive) {
          return createUnauthorizedResponse('Refresh session is invalid.')
        }

        currentAccessToken = MOCK_REFRESH_RESPONSE.accessToken
        return HttpResponse.json({
          success: true,
          data: {
            ...MOCK_REFRESH_RESPONSE,
            accessToken: currentAccessToken,
          },
        })
      }),
      http.post('/api/auth/logout', () => {
        sessionActive = false
        currentAccessToken = ''
        return HttpResponse.json({ success: true, data: { loggedOut: true } })
      }),
      http.get('/api/auth/me', ({ request }) => {
        const authorization = request.headers.get('Authorization')

        if (!sessionActive || authorization !== `Bearer ${currentAccessToken}`) {
          return createUnauthorizedResponse('Authentication required.')
        }

        return HttpResponse.json({ success: true, data: MOCK_AUTH_USER })
      }),
      http.post('/api/account/password', () => {
        sessionActive = false
        currentAccessToken = ''
        return HttpResponse.json({
          success: true,
          data: {
            ...MOCK_AUTH_USER,
            mustChangePassword: false,
          },
        })
      }),
    ]
  })(),
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
  http.delete('/api/account', () => HttpResponse.json({ success: true, data: { deleted: true } })),
].flat()

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
