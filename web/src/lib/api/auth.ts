import { z } from 'zod'
import { api, normalizeAPIError } from '@/lib/api/client'

export const authUserSchema = z.object({ id: z.string(), email: z.email(), name: z.string() })

export type AuthUser = z.infer<typeof authUserSchema>

export interface LoginCredentials {
  email: string
  password: string
}

export const loginResponseSchema = z.object({
  accessToken: z.string(),
  tokenType: z.string(),
  expiresAt: z.string(),
  user: authUserSchema,
  success: z.boolean(),
  loginEvent: z.string(),
})

export type LoginResponse = z.infer<typeof loginResponseSchema>

export const currentUserResponseSchema = z.object({ user: authUserSchema })

export type CurrentUserResponse = z.infer<typeof currentUserResponseSchema>

export class LoginError extends Error {
  loginEvent: string
  constructor(loginEvent: string, message: string) {
    super(message)
    this.name = 'LoginError'
    this.loginEvent = loginEvent
  }
}

export async function login(credentials: LoginCredentials): Promise<LoginResponse> {
  const payload = await api.post('/api/auth/login', { json: credentials }).json<unknown>()
  const response = loginResponseSchema.parse(payload)

  if (!response.success) {
    throw new LoginError(response.loginEvent, response.loginEvent)
  }

  return response
}

export async function getCurrentUser(): Promise<CurrentUserResponse> {
  try {
    const payload = await api.get('/api/auth/me').json<unknown>()
    return currentUserResponseSchema.parse(payload)
  } catch (error) {
    return normalizeAPIError(error)
  }
}
