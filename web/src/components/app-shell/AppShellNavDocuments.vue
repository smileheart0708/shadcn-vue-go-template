<script setup lang="ts">
import { IconDots } from '@tabler/icons-vue'
import { useI18n } from 'vue-i18n'
import type { AppShellNavItem } from '@/components/app-shell/navigation'
import AppShellNavButton from '@/components/app-shell/AppShellNavButton.vue'
import { SidebarGroup, SidebarGroupLabel, SidebarMenu, SidebarMenuItem } from '@/components/ui/sidebar'

defineProps<{ items: AppShellNavItem[] }>()

const { t } = useI18n()
</script>

<template>
  <SidebarGroup class="group-data-[collapsible=icon]:hidden">
    <SidebarGroupLabel>{{ t('nav.documents.label') }}</SidebarGroupLabel>
    <SidebarMenu>
      <SidebarMenuItem
        v-for="item in items"
        :key="item.title"
      >
        <AppShellNavButton :item="item" />
      </SidebarMenuItem>
      <SidebarMenuItem>
        <AppShellNavButton :item="{ title: t('nav.documents.more'), icon: IconDots, disabled: true }" />
      </SidebarMenuItem>
    </SidebarMenu>
  </SidebarGroup>
</template>
