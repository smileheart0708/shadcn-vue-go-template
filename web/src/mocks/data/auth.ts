export const MOCK_AUTH_USER = {
  id: 'mock-user-1',
  email: 'demo@example.com',
  name: 'Demo User',
} as const

export const MOCK_AUTH_TOKEN = 'mock-access-token'

export const MOCK_LOGIN_CREDENTIALS = { email: 'demo@example.com', password: 'demo123456' } as const

export const MOCK_LOGIN_RESPONSE = {
  accessToken: MOCK_AUTH_TOKEN,
  tokenType: 'Bearer',
  expiresAt: '2099-01-01T00:00:00.000Z',
  user: MOCK_AUTH_USER,
} as const
