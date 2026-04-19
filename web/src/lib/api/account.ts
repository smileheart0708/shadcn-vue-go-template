import { z } from 'zod'
import { authApi, normalizeAPIError } from '@/lib/api/client'
import { successEnvelopeSchema } from '@/lib/api/envelope'
import { viewerSchema } from '@/lib/api/auth'

export interface UpdateProfileInput {
  username: string
  email: string | null
}

export interface ChangePasswordInput {
  currentPassword: string
  newPassword: string
}

const viewerEnvelopeSchema = successEnvelopeSchema(viewerSchema)
const passwordChangedEnvelopeSchema = successEnvelopeSchema(
  z.object({
    passwordChanged: z.boolean(),
  }),
)
const deleteAccountEnvelopeSchema = successEnvelopeSchema(
  z.object({
    deleted: z.boolean(),
  }),
)

export async function updateProfile(input: UpdateProfileInput) {
  try {
    const payload = await authApi.patch('/api/account/profile', { json: input }).json<unknown>()
    return viewerEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function uploadAvatar(file: File) {
  try {
    const formData = new FormData()
    formData.set('avatar', file)

    const payload = await authApi.post('/api/account/avatar', { body: formData }).json<unknown>()
    return viewerEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function updatePassword(input: ChangePasswordInput) {
  try {
    const payload = await authApi.post('/api/account/password', { json: input }).json<unknown>()
    return passwordChangedEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function deleteAccount() {
  try {
    const payload = await authApi.delete('/api/account').json<unknown>()
    return deleteAccountEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}
