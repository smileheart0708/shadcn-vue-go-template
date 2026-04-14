<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { defaultDocument, useEventListener, useMediaQuery } from '@vueuse/core'
import { TooltipProvider } from 'reka-ui'
import { computed, ref, watch } from 'vue'
import { cn } from '@/lib/utils'
import { provideSidebarContext, SIDEBAR_COOKIE_MAX_AGE, SIDEBAR_COOKIE_NAME, SIDEBAR_KEYBOARD_SHORTCUT, SIDEBAR_WIDTH, SIDEBAR_WIDTH_ICON } from './utils'

const props = defineProps<{ defaultOpen?: boolean | null; open?: boolean | null; class?: HTMLAttributes['class'] }>()

const emits = defineEmits<{ 'update:open': [open: boolean] }>()

const isMobile = useMediaQuery('(max-width: 768px)')
const openMobile = ref(false)
const uncontrolledOpen = ref(readStoredSidebarOpen())

watch(
  () => props.defaultOpen,
  (defaultOpen) => {
    if (defaultOpen !== null && defaultOpen !== undefined) {
      uncontrolledOpen.value = defaultOpen
    }
  },
  { immediate: true, once: true },
)

const open = computed({
  get: () => props.open ?? uncontrolledOpen.value,
  set: (value: boolean) => {
    if (props.open === null || props.open === undefined) {
      uncontrolledOpen.value = value
    }

    emits('update:open', value)
  },
})

function setOpen(value: boolean) {
  open.value = value // emits('update:open', value)
  const cookieValue = value ? 'true' : 'false'

  // This sets the cookie to keep the sidebar state.
  document.cookie = `${SIDEBAR_COOKIE_NAME}=${cookieValue}; path=/; max-age=${String(SIDEBAR_COOKIE_MAX_AGE)}`
}

function setOpenMobile(value: boolean) {
  openMobile.value = value
}

// Helper to toggle the sidebar.
function toggleSidebar() {
  if (isMobile.value) {
    setOpenMobile(!openMobile.value)
    return
  }

  setOpen(!open.value)
}

useEventListener('keydown', (event: KeyboardEvent) => {
  if (event.key === SIDEBAR_KEYBOARD_SHORTCUT && (event.metaKey || event.ctrlKey)) {
    event.preventDefault()
    toggleSidebar()
  }
})

// We add a state so that we can do data-state="expanded" or "collapsed".
// This makes it easier to style the sidebar with Tailwind classes.
const state = computed(() => (open.value ? 'expanded' : 'collapsed'))

provideSidebarContext({ state, open, setOpen, isMobile, openMobile, setOpenMobile, toggleSidebar })

function readStoredSidebarOpen(): boolean {
  if (defaultDocument === undefined) {
    return true
  }

  return !defaultDocument.cookie.includes(`${SIDEBAR_COOKIE_NAME}=false`)
}
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
