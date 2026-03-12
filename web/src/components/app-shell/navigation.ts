import type { Component } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import {
  IconChartBar,
  IconDashboard,
  IconDatabase,
  IconFileDescription,
  IconFolder,
  IconHelp,
  IconListDetails,
  IconReport,
  IconSearch,
  IconSettings,
  IconUsers,
} from '@tabler/icons-vue'

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

export const APP_SHELL_NAVIGATION = {
  user: { name: 'shadcn', email: 'm@example.com', avatar: '' },
  main: [
    { title: 'Dashboard', icon: IconDashboard, to: { name: 'dashboard' } },
    { title: 'Lifecycle', icon: IconListDetails, disabled: true },
    { title: 'Analytics', icon: IconChartBar, disabled: true },
    { title: 'Projects', icon: IconFolder, disabled: true },
    { title: 'Team', icon: IconUsers, disabled: true },
  ],
  documents: [
    { title: 'Data Library', icon: IconDatabase, disabled: true },
    { title: 'Reports', icon: IconReport, disabled: true },
    { title: 'Word Assistant', icon: IconFileDescription, disabled: true },
  ],
  secondary: [
    { title: 'Settings', icon: IconSettings, disabled: true },
    { title: 'Get Help', icon: IconHelp, disabled: true },
    { title: 'Search', icon: IconSearch, disabled: true },
  ],
} satisfies AppShellNavigation
