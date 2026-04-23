import type { Capability, RoleKey } from '@/lib/api/auth'

export const ROLE_KEY = {
  owner: 'owner',
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
    default:
      return 'user'
  }
}

export function getUserRoleLabelKey(role: RoleKey | null | undefined) {
  switch (role) {
    case ROLE_KEY.owner:
      return 'common.role.owner'
    default:
      return 'common.role.user'
  }
}

export function getUserRoleBadgeVariant(role: RoleKey | null | undefined): 'warning' | 'outline' {
  switch (role) {
    case ROLE_KEY.owner:
      return 'warning'
    default:
      return 'outline'
  }
}

export function hasCapability(capabilities: readonly string[] | null | undefined, capability: Capability) {
  return Array.isArray(capabilities) && capabilities.includes(capability)
}
