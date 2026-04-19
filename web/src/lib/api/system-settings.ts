import { z } from 'zod'
import { authApi, normalizeAPIError } from '@/lib/api/client'
import { successEnvelopeSchema } from '@/lib/api/envelope'
import { authModeSchema, registrationModeSchema } from '@/lib/api/auth'

export const systemSettingsSchema = z.object({
  authMode: authModeSchema,
  registrationMode: registrationModeSchema,
  passwordLoginEnabled: z.boolean(),
  adminUserCreateEnabled: z.boolean(),
  selfServiceAccountDeletionEnabled: z.boolean(),
  updatedAt: z.string(),
})

export type SystemSettings = z.infer<typeof systemSettingsSchema>

export interface UpdateSystemSettingsInput {
  authMode?: z.infer<typeof authModeSchema>
  registrationMode?: z.infer<typeof registrationModeSchema>
  adminUserCreateEnabled?: boolean
  selfServiceAccountDeletionEnabled?: boolean
}

const systemSettingsEnvelopeSchema = successEnvelopeSchema(systemSettingsSchema)

export async function getAdminSystemSettings() {
  try {
    const payload = await authApi.get('/api/system/settings').json<unknown>()
    return systemSettingsEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function updateSystemSettings(input: UpdateSystemSettingsInput) {
  try {
    const payload = await authApi.patch('/api/system/settings', { json: input }).json<unknown>()
    return systemSettingsEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}
