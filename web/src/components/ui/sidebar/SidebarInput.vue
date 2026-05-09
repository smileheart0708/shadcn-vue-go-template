<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@/lib/utils'
import { Input } from '@/components/ui/input'

const props = defineProps<{ class?: HTMLAttributes['class'] }>()
</script>

<template>
  <Input
    data-slot="sidebar-input"
    data-sidebar="input"
    :class="cn('bg-background shadow-none block-8 inline-full', props.class)"
  >
    <slot />
  </Input>
</template>
