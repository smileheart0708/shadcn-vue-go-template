<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <div
    data-slot="table-container"
    class="relative overflow-auto inline-full"
  >
    <table
      data-slot="table"
      :class="cn('caption-bottom text-sm inline-full', props.class)"
    >
      <slot />
    </table>
  </div>
</template>
