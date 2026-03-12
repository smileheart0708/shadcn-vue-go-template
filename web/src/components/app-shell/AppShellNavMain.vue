<script setup lang="ts">
import { IconCirclePlusFilled, IconMail } from '@tabler/icons-vue'
import type { AppShellNavItem } from '@/components/app-shell/navigation'
import AppShellNavButton from '@/components/app-shell/AppShellNavButton.vue'
import { Button } from '@/components/ui/button'
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar'

defineProps<{ items: AppShellNavItem[] }>()
</script>

<template>
  <SidebarGroup>
    <SidebarGroupContent class="flex flex-col gap-2">
      <SidebarMenu>
        <SidebarMenuItem class="flex items-center gap-2">
          <SidebarMenuButton
            tooltip="Quick Create"
            class="bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground active:bg-primary/90 active:text-primary-foreground min-w-8 duration-200 ease-linear"
          >
            <IconCirclePlusFilled />
            <span>Quick Create</span>
          </SidebarMenuButton>
          <Button
            size="icon"
            class="size-8 group-data-[collapsible=icon]:opacity-0"
            variant="outline"
          >
            <IconMail />
            <span class="sr-only">Inbox</span>
          </Button>
        </SidebarMenuItem>
      </SidebarMenu>
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
