<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { useEventListener } from '@vueuse/core'
import { TooltipProvider } from 'reka-ui'
import { cn } from '@/lib/utils'
import { useSidebarController } from './controller'
import { provideSidebarContext, SIDEBAR_KEYBOARD_SHORTCUT, SIDEBAR_WIDTH, SIDEBAR_WIDTH_ICON } from './utils'

const props = defineProps<{ defaultOpen?: boolean | null; open?: boolean | null; class?: HTMLAttributes['class'] }>()

const emits = defineEmits<{ 'update:open': [open: boolean] }>()

const sidebar = useSidebarController(props, (value) => {
  emits('update:open', value)
})

useEventListener('keydown', (event: KeyboardEvent) => {
  if (event.key === SIDEBAR_KEYBOARD_SHORTCUT && (event.metaKey || event.ctrlKey)) {
    event.preventDefault()
    sidebar.toggleSidebar()
  }
})

provideSidebarContext(sidebar)
</script>

<template>
  <TooltipProvider :delay-duration="0">
    <div
      data-slot="sidebar-wrapper"
      :style="{ '--sidebar-width': SIDEBAR_WIDTH, '--sidebar-width-icon': SIDEBAR_WIDTH_ICON }"
      :class="cn('group/sidebar-wrapper flex h-svh w-full has-data-[variant=inset]:bg-sidebar', props.class)"
      v-bind="$attrs"
    >
      <slot />
    </div>
  </TooltipProvider>
</template>
