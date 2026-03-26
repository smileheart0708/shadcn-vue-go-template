<script setup lang="ts">
import { RouterView } from 'vue-router'
import RouteLoadingBar from '@/components/RouteLoadingBar.vue'
import RouteLoadingErrorToast from '@/components/RouteLoadingErrorToast.vue'
import { Toaster } from '@/components/ui/sonner'
import { TooltipProvider } from '@/components/ui/tooltip'
import { useThemeStore } from '@/stores/theme'

const themeStore = useThemeStore()

themeStore.initialize()
</script>

<template>
  <TooltipProvider>
    <RouteLoadingBar />
    <RouteLoadingErrorToast />
    <RouterView />
    <Toaster />
  </TooltipProvider>
</template>
