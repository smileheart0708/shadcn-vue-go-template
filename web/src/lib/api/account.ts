import { z } from 'zod'
import { authApi, normalizeAPIError } from '@/lib/api/client'
import { authUserSchema } from '@/lib/api/auth'
import { successEnvelopeSchema } from '@/lib/api/envelope'

export interface UpdateProfileInput {
  username: string
  email: string | null
}

export interface ChangePasswordInput {
  currentPassword: string
  newPassword: string
}

const authUserEnvelopeSchema = successEnvelopeSchema(authUserSchema)
const deleteAccountResponseSchema = successEnvelopeSchema(z.object({ deleted: z.boolean() }))

export async function updateProfile(input: UpdateProfileInput) {
  try {
    const payload = await authApi.patch('/api/account/profile', { json: input }).json<unknown>()
    return authUserEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function uploadAvatar(file: File) {
  try {
    const formData = new FormData()
    formData.set('avatar', file)

    const payload = await authApi.post('/api/account/avatar', { body: formData }).json<unknown>()
    return authUserEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function updatePassword(input: ChangePasswordInput) {
  try {
    const payload = await authApi.post('/api/account/password', { json: input }).json<unknown>()
    return authUserEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function deleteAccount() {
  try {
    const payload = await authApi.delete('/api/account').json<unknown>()
    return deleteAccountResponseSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}
