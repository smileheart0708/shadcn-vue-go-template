<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

const props = defineProps<{
  title: string
  description?: string
  titleClass?: HTMLAttributes['class']
  contentClass?: HTMLAttributes['class']
  footerClass?: HTMLAttributes['class']
  class?: HTMLAttributes['class']
}>()
</script>

<template>
  <Card :class="props.class">
    <CardHeader>
      <CardTitle :class="props.titleClass">{{ props.title }}</CardTitle>
      <CardDescription v-if="props.description !== undefined">{{
        props.description
      }}</CardDescription>
    </CardHeader>
    <CardContent :class="props.contentClass">
      <slot />
    </CardContent>
    <CardFooter
      v-if="$slots.footer !== undefined"
      :class="props.footerClass"
    >
      <slot name="footer" />
    </CardFooter>
  </Card>
</template>
