<script setup lang="ts">
import AppSidebar from '@/components/AppSidebar.vue'
import SiteHeader from '@/components/SiteHeader.vue'
import SectionCards from '@/components/SectionCards.vue'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
</script>

<template>
  <SidebarProvider>
    <div class="flex min-h-screen w-full bg-background">
      <AppSidebar class="hidden lg:block" />
      <SidebarInset>
        <SiteHeader />
        <main class="flex flex-1 flex-col gap-4 p-4 lg:gap-6 lg:p-6">
          <div class="flex items-center justify-between">
            <h1 class="text-2xl font-bold tracking-tight">Dashboard</h1>
          </div>
          <SectionCards />
        </main>
      </SidebarInset>
    </div>
  </SidebarProvider>
</template>
