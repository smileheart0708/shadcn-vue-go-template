<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <div
    data-slot="field-label"
    :class="cn('flex items-center gap-2 text-sm/snug font-medium inline-fit group-data-[disabled=true]/field:opacity-50', props.class)"
  >
    <slot />
  </div>
</template>
