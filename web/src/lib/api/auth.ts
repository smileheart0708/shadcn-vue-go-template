import { z } from 'zod'
import { apiRequest } from '@/lib/api/client'

export const authUserSchema = z.object({
  id: z.string(),
  email: z.string().email(),
  name: z.string(),
})

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

export function login(credentials: LoginCredentials) {
  return apiRequest('/api/auth/login', loginResponseSchema.parse, {
    method: 'POST',
    body: JSON.stringify(credentials),
  })
}

export function getCurrentUser() {
  return apiRequest('/api/auth/me', currentUserResponseSchema.parse)
}
