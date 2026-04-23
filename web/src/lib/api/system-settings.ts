import { z } from 'zod'
import { authApi, normalizeAPIError } from '@/lib/api/client'
import { successEnvelopeSchema } from '@/lib/api/envelope'

export const systemSettingsSchema = z.object({
  publicRegistrationEnabled: z.boolean(),
  selfServiceAccountDeletionEnabled: z.boolean(),
})

export type SystemSettings = z.infer<typeof systemSettingsSchema>

export interface UpdateSystemSettingsInput {
  publicRegistrationEnabled?: boolean
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
