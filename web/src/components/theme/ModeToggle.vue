<script setup lang="ts">
import { Moon, Monitor, Sun } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { useThemeStore } from '@/stores/theme'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const themeStore = useThemeStore()

const icon = { light: Sun, dark: Moon, system: Monitor }
const themes = ['light', 'dark', 'system'] as const
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
      <DropdownMenuItem
        v-for="theme in themes"
        :key="theme"
        @click="themeStore.setTheme(theme)"
      >
        <component
          :is="icon[theme]"
          class="mr-2 h-4 w-4"
        />
        {{ t(`theme.${theme}`) }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>
