<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { AppShellNavItem } from '@/components/app-shell/navigation'
import AppShellNavButton from '@/components/app-shell/AppShellNavButton.vue'
import { SidebarGroup, SidebarGroupContent, SidebarGroupLabel, SidebarMenu, SidebarMenuItem } from '@/components/ui/sidebar'

defineProps<{ items: AppShellNavItem[] }>()

const { t } = useI18n()

</script>

<template>
  <SidebarGroup v-if="items.length > 0">
    <SidebarGroupLabel>{{ t('nav.management.label') }}</SidebarGroupLabel>
    <SidebarGroupContent>
      <SidebarMenu>
        <SidebarMenuItem
          v-for="item in items"
          :key="item.title"
        >
          <AppShellNavButton :item="item" />
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarGroupContent>
  </SidebarGroup>
</template>
