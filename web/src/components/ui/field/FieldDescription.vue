<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <p
    data-slot="field-description"
    :class="
      cn(
        'text-sm/normal font-normal text-muted-foreground group-has-data-[orientation=horizontal]/field:text-balance',
        'last:mbs-0 nth-last-2:-mbs-1 [[data-variant=legend]+&]:-mbs-1.5',
        '[&>a]:underline [&>a]:underline-offset-4 [&>a:hover]:text-primary',
        props.class,
      )
    "
  >
    <slot />
  </p>
</template>
