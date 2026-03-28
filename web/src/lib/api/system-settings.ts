import { z } from 'zod'
import { authApi, normalizeAPIError } from '@/lib/api/client'
import { registrationModeSchema } from '@/lib/api/auth'
import { successEnvelopeSchema } from '@/lib/api/envelope'

export const adminSystemSettingsSchema = z.object({
  registrationMode: registrationModeSchema,
  updatedAt: z.string(),
})

export type AdminSystemSettings = z.infer<typeof adminSystemSettingsSchema>

const adminSystemSettingsEnvelopeSchema = successEnvelopeSchema(adminSystemSettingsSchema)

export async function getAdminSystemSettings() {
  try {
    const payload = await authApi.get('/api/admin/system-settings').json<unknown>()
    return adminSystemSettingsEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function updateRegistrationMode(registrationMode: 'disabled' | 'password') {
  try {
    const payload = await authApi.patch('/api/admin/system-settings/registration', { json: { registrationMode } }).json<unknown>()
    return adminSystemSettingsEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}
