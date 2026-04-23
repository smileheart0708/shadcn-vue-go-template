import { computed, type Component, type ComputedRef } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { IconChartBar, IconChecklist, IconDashboard, IconFolder, IconHelp, IconListDetails, IconSearch, IconSettings } from '@tabler/icons-vue'
import { Cog, Logs, Users } from 'lucide-vue-next'
import { CAPABILITY } from '@/lib/auth/roles'
import { useAuthStore } from '@/stores/auth'

export interface AppShellNavItem {
  title: string
  icon?: Component
  to?: RouteLocationRaw
  disabled?: boolean
}

export interface AppShellNavigation {
  main: AppShellNavItem[]
  management: AppShellNavItem[]
  secondary: AppShellNavItem[]
}

export function useAppShellNavigation(): ComputedRef<AppShellNavigation> {
  const { t } = useI18n()
  const authStore = useAuthStore()

  const canReadSystemSettings = computed(() => authStore.can(CAPABILITY.systemSettingsRead))
  const canReadUsers = computed(() => authStore.can(CAPABILITY.managementUsersRead))
  const canReadSystemLogs = computed(() => authStore.can(CAPABILITY.managementSystemLogsRead))

  return computed(() => ({
    main: [
      { title: t('nav.main.dashboard'), icon: IconDashboard, to: { name: 'dashboard' } },
      { title: t('nav.main.tasks'), icon: IconChecklist, to: { name: 'tasks' } },
      { title: t('nav.main.lifecycle'), icon: IconListDetails, disabled: true },
      { title: t('nav.main.analytics'), icon: IconChartBar, disabled: true },
      { title: t('nav.main.projects'), icon: IconFolder, disabled: true },
    ],
    management: [
      ...(canReadSystemSettings.value ? [{ title: t('nav.management.systemConfig'), icon: Cog, to: { name: 'system-config' } }] : []),
      ...(canReadUsers.value ? [{ title: t('nav.management.users'), icon: Users, to: { name: 'users' } }] : []),
      ...(canReadSystemLogs.value ? [{ title: t('nav.management.systemLogs'), icon: Logs, to: { name: 'system-logs' } }] : []),
    ],
    secondary: [
      { title: t('nav.secondary.settings'), icon: IconSettings, to: { name: 'settings' } },
      { title: t('nav.secondary.getHelp'), icon: IconHelp, disabled: true },
      { title: t('nav.secondary.search'), icon: IconSearch, disabled: true },
    ],
  }))
}
