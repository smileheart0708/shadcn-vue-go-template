<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <div
    data-slot="empty"
    :class="cn('flex flex-1 flex-col items-center justify-center gap-6 rounded-lg border-dashed p-6 text-center text-balance min-inline-0 md:p-12', props.class)"
  >
    <slot />
  </div>
</template>
