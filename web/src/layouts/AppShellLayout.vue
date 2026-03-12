<script setup lang="ts">
import AppShellHeader from '@/components/app-shell/AppShellHeader.vue'
import AppShellSidebar from '@/components/app-shell/AppShellSidebar.vue'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
import { RouterView } from 'vue-router'
</script>

<template>
  <SidebarProvider class="bg-background">
    <AppShellSidebar />
    <SidebarInset>
      <AppShellHeader />
      <div class="flex flex-1 flex-col">
        <RouterView />
      </div>
    </SidebarInset>
  </SidebarProvider>
</template>
