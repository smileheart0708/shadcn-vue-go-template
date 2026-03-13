<script setup lang="ts">
import { Moon, Monitor, Sun } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { useThemeStore } from '@/stores/theme'

const themeStore = useThemeStore()

const icon = { light: Sun, dark: Moon, system: Monitor }
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button
        variant="ghost"
        size="icon"
      >
        <component
          :is="icon[themeStore.theme]"
          class="h-[1.2rem] w-[1.2rem]"
        />
        <span class="sr-only">Toggle theme</span>
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end">
      <DropdownMenuItem @click="themeStore.setTheme('light')"> Light </DropdownMenuItem>
      <DropdownMenuItem @click="themeStore.setTheme('dark')"> Dark </DropdownMenuItem>
      <DropdownMenuItem @click="themeStore.setTheme('system')"> System </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
