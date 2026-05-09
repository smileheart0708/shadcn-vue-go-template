<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>

<template>
  <div
    data-slot="input-group"
    role="group"
    :class="
      cn(
        'group/input-group relative flex items-center rounded-md border border-input shadow-xs transition-[color,box-shadow] outline-none inline-full dark:bg-input/30',
        'block-9 min-inline-0 has-[>textarea]:block-auto',

        // Variants based on alignment.
        'has-[>[data-align=inline-start]]:[&>input]:ps-2',
        'has-[>[data-align=inline-end]]:[&>input]:pe-2',
        'has-[>[data-align=block-start]]:flex-col has-[>[data-align=block-start]]:block-auto has-[>[data-align=block-start]]:[&>input]:pbe-3',
        'has-[>[data-align=block-end]]:flex-col has-[>[data-align=block-end]]:block-auto has-[>[data-align=block-end]]:[&>input]:pbs-3',

        // Focus state.
        'has-[[data-slot=input-group-control]:focus-visible]:border-ring has-[[data-slot=input-group-control]:focus-visible]:ring-[3px] has-[[data-slot=input-group-control]:focus-visible]:ring-ring/50',

        // Error state.
        'has-[[data-slot][aria-invalid=true]]:border-destructive has-[[data-slot][aria-invalid=true]]:ring-destructive/20 dark:has-[[data-slot][aria-invalid=true]]:ring-destructive/40',

        props.class,
      )
    "
  >
    <slot />
  </div>
</template>
