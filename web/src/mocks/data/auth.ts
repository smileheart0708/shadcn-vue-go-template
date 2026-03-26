export const MOCK_AUTH_USER = {
  id: 1,
  username: 'demo',
  email: 'demo@example.com',
  avatarUrl: null,
  role: 0,
  mustChangePassword: false,
} as const

export const MOCK_AUTH_TOKEN = 'mock-access-token'

export const MOCK_LOGIN_CREDENTIALS = { identifier: 'demo', password: 'demo123456' } as const

export const MOCK_LOGIN_RESPONSE = {
  accessToken: MOCK_AUTH_TOKEN,
  tokenType: 'Bearer',
  expiresAt: '2099-01-01T00:00:00.000Z',
  user: MOCK_AUTH_USER,
} as const

export const MOCK_REFRESH_RESPONSE = {
  ...MOCK_LOGIN_RESPONSE,
  accessToken: 'mock-access-token-refreshed',
} as const
