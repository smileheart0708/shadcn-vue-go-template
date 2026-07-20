<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <div
    data-slot="sidebar-group"
    data-sidebar="group"
    :class="
      cn('relative flex flex-col p-2 inline-full min-inline-0', props.class)
    "
  >
    <slot />
  </div>
</template>
