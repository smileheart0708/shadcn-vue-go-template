<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <div
    data-slot="empty-content"
    :class="cn('flex flex-col items-center gap-4 text-sm text-balance inline-full max-inline-sm min-inline-0', props.class)"
  >
    <slot />
  </div>
</template>
