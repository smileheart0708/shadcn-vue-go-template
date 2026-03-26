import { z } from 'zod'
import { authApi, baseApi, normalizeAPIError } from '@/lib/api/client'
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

const logoutPayloadSchema = z.object({
  loggedOut: z.boolean(),
})

export const loginResponseSchema = successEnvelopeSchema(loginPayloadSchema)
const logoutResponseSchema = successEnvelopeSchema(logoutPayloadSchema)

export type LoginResponse = z.infer<typeof loginPayloadSchema>

const currentUserResponseSchema = successEnvelopeSchema(authUserSchema)

export async function login(credentials: LoginCredentials): Promise<LoginResponse> {
  try {
    const payload = await baseApi.post('/api/auth/login', { json: credentials }).json<unknown>()
    return loginResponseSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function refreshSession(): Promise<LoginResponse> {
  try {
    const payload = await baseApi
      .post('/api/auth/refresh', {
        context: {
          skipAuthRefresh: true,
        },
      })
      .json<unknown>()
    return loginResponseSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function logout() {
  try {
    const payload = await baseApi
      .post('/api/auth/logout', {
        context: {
          skipAuthRefresh: true,
        },
      })
      .json<unknown>()
    return logoutResponseSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function getCurrentUser(): Promise<AuthUser> {
  try {
    const payload = await authApi.get('/api/auth/me').json<unknown>()
    return currentUserResponseSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}
