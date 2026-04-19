import type { Capability, RoleKey } from '@/lib/api/auth'

export const ROLE_KEY = {
  owner: 'owner',
  admin: 'admin',
  user: 'user',
} as const satisfies Record<string, RoleKey>

export const CAPABILITY = {
  systemSettingsRead: 'system.settings.read',
  systemSettingsUpdate: 'system.settings.update',
  managementUsersRead: 'management.users.read',
  managementUsersCreate: 'management.users.create',
  managementUsersUpdate: 'management.users.update',
  managementUsersDisable: 'management.users.disable',
  managementUsersEnable: 'management.users.enable',
  managementAuditLogsRead: 'management.audit_logs.read',
  managementSystemLogsRead: 'management.system_logs.read',
  accountDeleteSelf: 'account.delete_self',
} as const satisfies Record<string, Capability>

export function roleLabel(roleKey: RoleKey | null | undefined) {
  switch (roleKey) {
    case ROLE_KEY.owner:
      return 'owner'
    case ROLE_KEY.admin:
      return 'admin'
    default:
      return 'user'
  }
}

export function getPrimaryRoleKey(roleKeys: readonly unknown[] | null | undefined): RoleKey | null {
  if (!Array.isArray(roleKeys) || roleKeys.length === 0) {
    return null
  }

  for (const roleKey of roleKeys) {
    if (isRoleKey(roleKey)) {
      return roleKey
    }
  }

  return null
}

export function getUserRoleLabelKey(role: RoleKey | readonly string[] | null | undefined) {
  const roleKey = Array.isArray(role) ? getPrimaryRoleKey(role) : role
  switch (roleKey) {
    case ROLE_KEY.owner:
      return 'common.role.owner'
    case ROLE_KEY.admin:
      return 'common.role.admin'
    default:
      return 'common.role.user'
  }
}

export function getUserRoleBadgeVariant(role: RoleKey | readonly string[] | null | undefined): 'warning' | 'secondary' | 'outline' {
  const roleKey = Array.isArray(role) ? getPrimaryRoleKey(role) : role
  switch (roleKey) {
    case ROLE_KEY.owner:
      return 'warning'
    case ROLE_KEY.admin:
      return 'secondary'
    default:
      return 'outline'
  }
}

export function hasCapability(capabilities: readonly string[] | null | undefined, capability: Capability) {
  return Array.isArray(capabilities) && capabilities.includes(capability)
}

function isRoleKey(value: unknown): value is RoleKey {
  return value === ROLE_KEY.owner || value === ROLE_KEY.admin || value === ROLE_KEY.user
}
