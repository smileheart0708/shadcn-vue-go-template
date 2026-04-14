<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <tfoot
    data-slot="table-footer"
    :class="cn('border-bs bg-muted/50 font-medium [&>tr]:last:border-be-0', props.class)"
  >
    <slot />
  </tfoot>
</template>
