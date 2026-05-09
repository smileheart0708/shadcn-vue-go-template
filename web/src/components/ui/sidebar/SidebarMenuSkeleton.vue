<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { computed } from 'vue'
import { cn } from '@/lib/utils'
import { Skeleton } from '@/components/ui/skeleton'

const props = defineProps<{ showIcon?: boolean; class?: HTMLAttributes['class'] }>()

const width = computed(() => {
  return String(Math.floor(Math.random() * 40) + 50) + '%'
})
</script>

<template>
  <div
    data-slot="sidebar-menu-skeleton"
    data-sidebar="menu-skeleton"
    :class="cn('flex items-center gap-2 rounded-md px-2 block-8', props.class)"
  >
    <Skeleton
      v-if="showIcon === true"
      class="rounded-md block-4 inline-4"
      data-sidebar="menu-skeleton-icon"
    />

    <Skeleton
      class="flex-1 block-4 max-inline-(--skeleton-width)"
      data-sidebar="menu-skeleton-text"
      :style="{ '--skeleton-width': width }"
    />
  </div>
</template>
