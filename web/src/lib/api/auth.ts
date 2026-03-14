import { z } from 'zod'
import { api, normalizeAPIError } from '@/lib/api/client'
import { successEnvelopeSchema } from '@/lib/api/envelope'

export const authUserSchema = z.object({
  id: z.number().int().positive(),
  username: z.string(),
  email: z.string().email().nullable(),
  avatarUrl: z.string().nullable(),
  role: z.number().int(),
  mustChangePassword: z.boolean(),
})

export type AuthUser = z.infer<typeof authUserSchema>

export interface LoginCredentials {
  identifier: string
  password: string
}

const loginPayloadSchema = z.object({
  accessToken: z.string(),
  tokenType: z.string(),
  expiresAt: z.string(),
  user: authUserSchema,
})

export const loginResponseSchema = successEnvelopeSchema(loginPayloadSchema)

export type LoginResponse = z.infer<typeof loginPayloadSchema>

const currentUserResponseSchema = successEnvelopeSchema(authUserSchema)

export async function login(credentials: LoginCredentials): Promise<LoginResponse> {
  try {
    const payload = await api.post('/api/auth/login', { json: credentials }).json<unknown>()
    return loginResponseSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function getCurrentUser(): Promise<AuthUser> {
  try {
    const payload = await api.get('/api/auth/me').json<unknown>()
    return currentUserResponseSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}
