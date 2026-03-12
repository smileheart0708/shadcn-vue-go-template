<script setup lang="ts">
import AppShellNavDocuments from '@/components/app-shell/AppShellNavDocuments.vue'
import AppShellNavMain from '@/components/app-shell/AppShellNavMain.vue'
import AppShellNavSecondary from '@/components/app-shell/AppShellNavSecondary.vue'
import AppShellNavUser from '@/components/app-shell/AppShellNavUser.vue'
import { APP_SHELL_NAVIGATION } from '@/components/app-shell/navigation'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
} from '@/components/ui/sidebar'
</script>

<template>
  <Sidebar collapsible="offcanvas">
    <SidebarHeader>
      <div class="flex items-center gap-2 px-2">
        <img src="/logo.svg" class="size-5" alt="Logo" />
        <span class="text-base font-semibold">web</span>
      </div>
    </SidebarHeader>
    <SidebarContent>
      <AppShellNavMain :items="APP_SHELL_NAVIGATION.main" />
      <AppShellNavDocuments :items="APP_SHELL_NAVIGATION.documents" />
      <AppShellNavSecondary
        :items="APP_SHELL_NAVIGATION.secondary"
        class="mt-auto"
      />
    </SidebarContent>
    <SidebarFooter>
      <AppShellNavUser :user="APP_SHELL_NAVIGATION.user" />
    </SidebarFooter>
  </Sidebar>
</template>
