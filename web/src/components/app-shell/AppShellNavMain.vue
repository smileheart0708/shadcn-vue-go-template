<script setup lang="ts">
import type { AppShellNavItem } from '@/components/app-shell/navigation'
import AppShellNavButton from '@/components/app-shell/AppShellNavButton.vue'
import { SidebarGroup, SidebarGroupContent, SidebarMenu, SidebarMenuItem } from '@/components/ui/sidebar'

defineProps<{ items: AppShellNavItem[] }>()
</script>

<template>
  <SidebarGroup>
    <SidebarGroupContent class="flex flex-col gap-2">
      <SidebarMenu>
        <SidebarMenuItem
          v-for="item in items"
          :key="item.title"
        >
          <AppShellNavButton
            :item="item"
            :tooltip="item.title"
          />
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarGroupContent>
  </SidebarGroup>
</template>
