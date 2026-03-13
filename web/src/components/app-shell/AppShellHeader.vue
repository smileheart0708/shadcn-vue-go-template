<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import ModeToggle from '@/components/ModeToggle.vue'
import { Separator } from '@/components/ui/separator'
import { SidebarTrigger } from '@/components/ui/sidebar'

const route = useRoute()
const { t } = useI18n()

const routeTitleMessages = {
  'route.dashboard': () => t('route.dashboard'),
  'route.login': () => t('route.login'),
  'route.notFound': () => t('route.notFound'),
} as const

function isRouteTitleKey(value: string): value is keyof typeof routeTitleMessages {
  return value in routeTitleMessages
}

const title = computed(() => {
  for (const record of [...route.matched].reverse()) {
    const routeTitle = record.meta.titleKey

    if (typeof routeTitle === 'string' && isRouteTitleKey(routeTitle)) {
      return routeTitleMessages[routeTitle]()
    }
  }

  return 'web'
})
</script>

<template>
  <header class="flex h-(--header-height) shrink-0 items-center gap-2 border-b transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-(--header-height)">
    <div class="flex w-full items-center gap-1 px-4 lg:gap-2 lg:px-6">
      <SidebarTrigger class="-ml-1" />
      <Separator
        orientation="vertical"
        class="mx-2 data-[orientation=vertical]:h-4"
      />
      <h1 class="text-base font-medium">
        {{ title }}
      </h1>
      <div class="ml-auto flex items-center gap-2">
        <ModeToggle />
      </div>
    </div>
  </header>
</template>
