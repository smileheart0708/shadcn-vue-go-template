<script setup lang="ts">
import AppSidebar from '@/components/AppSidebar.vue'
import SiteHeader from '@/components/SiteHeader.vue'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
import { RouterView } from 'vue-router'
</script>

<template>
  <SidebarProvider>
    <div class="flex min-h-screen w-full bg-background">
      <AppSidebar class="hidden lg:block" />
      <SidebarInset>
        <SiteHeader />
        <RouterView />
      </SidebarInset>
    </div>
  </SidebarProvider>
</template>
