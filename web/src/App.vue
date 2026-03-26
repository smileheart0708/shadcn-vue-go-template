<script setup lang="ts">
import { RouterView } from 'vue-router'
import { Toaster } from '@/components/ui/sonner'
import { useThemeStore } from '@/stores/theme'

const themeStore = useThemeStore()

themeStore.initialize()
</script>

<template>
  <RouterView />
  <Toaster />
</template>
