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
})

export type LoginResponse = z.infer<typeof loginResponseSchema>

export const currentUserResponseSchema = z.object({ user: authUserSchema })

export type CurrentUserResponse = z.infer<typeof currentUserResponseSchema>

export async function login(credentials: LoginCredentials): Promise<LoginResponse> {
  try {
    const payload = await api.post('/api/auth/login', { json: credentials }).json<unknown>()
    return loginResponseSchema.parse(payload)
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function getCurrentUser(): Promise<CurrentUserResponse> {
  try {
    const payload = await api.get('/api/auth/me').json<unknown>()
    return currentUserResponseSchema.parse(payload)
  } catch (error) {
    return normalizeAPIError(error)
  }
}
