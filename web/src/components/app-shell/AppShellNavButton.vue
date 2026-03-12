<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import type { AppShellNavItem } from '@/components/app-shell/navigation'
import { SidebarMenuButton } from '@/components/ui/sidebar'

const props = defineProps<{ item: AppShellNavItem; tooltip?: string }>()

const route = useRoute()
const router = useRouter()

const isActive = computed(() => {
  if (!props.item.to || props.item.disabled) {
    return false
  }

  const target = router.resolve(props.item.to)

  return (
    route.path === target.path || (target.path !== '/' && route.path.startsWith(`${target.path}/`))
  )
})

const isDisabled = computed(() => props.item.disabled ?? !props.item.to)
</script>

<template>
  <SidebarMenuButton
    v-if="item.to && !isDisabled"
    as-child
    :tooltip="tooltip"
    :is-active="isActive"
  >
    <RouterLink :to="item.to">
      <component
        :is="item.icon"
        v-if="item.icon"
      />
      <span>{{ item.title }}</span>
    </RouterLink>
  </SidebarMenuButton>

  <SidebarMenuButton
    v-else
    :tooltip="tooltip"
    :disabled="isDisabled"
    class="text-sidebar-foreground/70 disabled:pointer-events-none disabled:opacity-50"
  >
    <component
      :is="item.icon"
      v-if="item.icon"
    />
    <span>{{ item.title }}</span>
  </SidebarMenuButton>
</template>
