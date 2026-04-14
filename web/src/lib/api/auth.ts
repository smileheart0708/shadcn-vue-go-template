import { z } from 'zod'
import { authApi, baseApi, normalizeAPIError } from '@/lib/api/client'
import { successEnvelopeSchema } from '@/lib/api/envelope'

export const authUserSchema = z.object({
  id: z.number().int().positive(),
  username: z.string(),
  email: z.email().nullable(),
  avatarUrl: z.string().nullable(),
  role: z.number().int(),
  mustChangePassword: z.boolean(),
})

export type AuthUser = z.infer<typeof authUserSchema>

export interface LoginCredentials {
  identifier: string
  password: string
}

export interface RegisterInput {
  username: string
  email: string | null
  password: string
}

export const registrationModeSchema = z.enum(['disabled', 'password'])
export type RegistrationMode = z.infer<typeof registrationModeSchema>

const loginPayloadSchema = z.object({
  accessToken: z.string(),
  tokenType: z.string(),
  expiresAt: z.string(),
  user: authUserSchema,
})

const logoutPayloadSchema = z.object({
  loggedOut: z.boolean(),
})

const registrationPolicyPayloadSchema = z.object({
  registrationMode: registrationModeSchema,
})

export const loginResponseSchema = successEnvelopeSchema(loginPayloadSchema)
const logoutResponseSchema = successEnvelopeSchema(logoutPayloadSchema)
const registrationPolicyResponseSchema = successEnvelopeSchema(registrationPolicyPayloadSchema)

export type LoginResponse = z.infer<typeof loginPayloadSchema>
export interface GetCurrentUserOptions {
  signal?: AbortSignal
  backgroundRequest?: boolean
}

const currentUserResponseSchema = successEnvelopeSchema(authUserSchema)

export async function login(credentials: LoginCredentials): Promise<LoginResponse> {
  try {
    const payload = await baseApi.post('/api/auth/login', { json: credentials }).json<unknown>()
    return loginResponseSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function register(input: RegisterInput): Promise<LoginResponse> {
  try {
    const payload = await baseApi.post('/api/auth/register', { json: input }).json<unknown>()
    return loginResponseSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function getRegistrationPolicy() {
  try {
    const payload = await baseApi.get('/api/auth/registration-policy').json<unknown>()
    return registrationPolicyResponseSchema.parse(payload).data
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

export async function getCurrentUser(options: GetCurrentUserOptions = {}): Promise<AuthUser> {
  try {
    const payload = await authApi
      .get('/api/auth/me', {
        signal: options.signal,
        context: {
          backgroundRequest: options.backgroundRequest === true,
        },
      })
      .json<unknown>()
    return currentUserResponseSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}
