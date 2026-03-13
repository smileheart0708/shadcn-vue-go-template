import type { Component } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { IconChartBar, IconDashboard, IconDatabase, IconFileDescription, IconFolder, IconHelp, IconListDetails, IconReport, IconSearch, IconSettings, IconUsers } from '@tabler/icons-vue'

export interface AppShellNavItem {
  title: string
  icon?: Component
  to?: RouteLocationRaw
  disabled?: boolean
}

export interface AppShellUser {
  name: string
  email: string
  avatar: string
}

export interface AppShellNavigation {
  user: AppShellUser
  main: AppShellNavItem[]
  documents: AppShellNavItem[]
  secondary: AppShellNavItem[]
}

export function useAppShellNavigation(): AppShellNavigation {
  const { t } = useI18n()

  return {
    user: { name: 'shadcn', email: 'm@example.com', avatar: '' },
    main: [
      { title: t('nav.main.dashboard'), icon: IconDashboard, to: { name: 'dashboard' } },
      { title: t('nav.main.lifecycle'), icon: IconListDetails, disabled: true },
      { title: t('nav.main.analytics'), icon: IconChartBar, disabled: true },
      { title: t('nav.main.projects'), icon: IconFolder, disabled: true },
      { title: t('nav.main.team'), icon: IconUsers, disabled: true },
    ],
    documents: [
      { title: t('nav.documents.dataLibrary'), icon: IconDatabase, disabled: true },
      { title: t('nav.documents.reports'), icon: IconReport, disabled: true },
      { title: t('nav.documents.wordAssistant'), icon: IconFileDescription, disabled: true },
    ],
    secondary: [
      { title: t('nav.secondary.settings'), icon: IconSettings, disabled: true },
      { title: t('nav.secondary.getHelp'), icon: IconHelp, disabled: true },
      { title: t('nav.secondary.search'), icon: IconSearch, disabled: true },
    ],
  }
}
