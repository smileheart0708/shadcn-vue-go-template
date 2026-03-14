export const USER_ROLE = {
  user: 0,
  admin: 1,
  superAdmin: 2,
} as const

export type UserRole = (typeof USER_ROLE)[keyof typeof USER_ROLE]

export function getUserRoleLabelKey(role: number): `common.userRole.${0 | 1 | 2}` {
  switch (role) {
    case USER_ROLE.admin:
      return 'common.userRole.1'
    case USER_ROLE.superAdmin:
      return 'common.userRole.2'
    default:
      return 'common.userRole.0'
  }
}

export function getUserRoleBadgeVariant(role: number): 'secondary' | 'outline' | 'warning' {
  switch (role) {
    case USER_ROLE.admin:
      return 'outline'
    case USER_ROLE.superAdmin:
      return 'warning'
    default:
      return 'secondary'
  }
}

export function canDeleteOwnAccount(role: number): boolean {
  return role !== USER_ROLE.superAdmin
}
