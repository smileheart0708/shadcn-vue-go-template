import { z } from 'zod'
import { authApi, normalizeAPIError } from '@/lib/api/client'
import { successEnvelopeSchema } from '@/lib/api/envelope'

export const adminUserSchema = z.object({
  id: z.number().int().positive(),
  username: z.string(),
  email: z.email().nullable(),
  avatarUrl: z.string().nullable(),
  role: z.number().int(),
  status: z.enum(['active', 'banned']),
  createdAt: z.string(),
  bannedAt: z.string().nullable(),
  mustChangePassword: z.boolean(),
})

export type AdminUser = z.infer<typeof adminUserSchema>

export interface ListAdminUsersParams {
  q?: string
  role?: number | null
  status?: 'active' | 'banned' | null
  sort?: 'created_at_desc' | 'created_at_asc' | 'username_asc' | 'username_desc'
  page?: number
  pageSize?: number
}

export interface AdminUserUpsertInput {
  username: string
  email: string | null
  role?: number | null
}

export interface AdminCreateUserInput extends AdminUserUpsertInput {
  password: string
}

const adminUsersPageSchema = successEnvelopeSchema(
  z.object({
    items: z.array(adminUserSchema),
    page: z.number().int().positive(),
    pageSize: z.number().int().positive(),
    total: z.number().int().nonnegative(),
  }),
)

const adminUserEnvelopeSchema = successEnvelopeSchema(adminUserSchema)

export async function listAdminUsers(params: ListAdminUsersParams = {}) {
  try {
    const searchParams = new URLSearchParams()
    const query = params.q?.trim()

    if (query !== undefined && query.length > 0) {
      searchParams.set('q', query)
    }
    if (typeof params.role === 'number') {
      searchParams.set('role', String(params.role))
    }
    if (params.status) {
      searchParams.set('status', params.status)
    }
    if (params.sort) {
      searchParams.set('sort', params.sort)
    }
    if (typeof params.page === 'number') {
      searchParams.set('page', String(params.page))
    }
    if (typeof params.pageSize === 'number') {
      searchParams.set('pageSize', String(params.pageSize))
    }

    const payload = await authApi.get('/api/admin/users', { searchParams }).json<unknown>()
    return adminUsersPageSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function createAdminUser(input: AdminCreateUserInput) {
  try {
    const payload = await authApi.post('/api/admin/users', { json: input }).json<unknown>()
    return adminUserEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function updateAdminUser(id: number, input: AdminUserUpsertInput) {
  try {
    const payload = await authApi.patch(`/api/admin/users/${String(id)}`, { json: input }).json<unknown>()
    return adminUserEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function banAdminUser(id: number) {
  try {
    const payload = await authApi.post(`/api/admin/users/${String(id)}/ban`).json<unknown>()
    return adminUserEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}

export async function unbanAdminUser(id: number) {
  try {
    const payload = await authApi.post(`/api/admin/users/${String(id)}/unban`).json<unknown>()
    return adminUserEnvelopeSchema.parse(payload).data
  } catch (error) {
    return normalizeAPIError(error)
  }
}
