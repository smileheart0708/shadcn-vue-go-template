<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { computed } from 'vue'
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: HTMLAttributes['class']
  errors?: (string | { message: string | undefined } | undefined)[]
}>()

const content = computed(() => {
  if (props.errors === undefined || props.errors.length === 0) {
    return null
  }

  const uniqueMessages = new Map<string, true>()

  for (const error of props.errors) {
    if (error === undefined) {
      continue
    }

    const message = typeof error === 'string' ? error : error.message

    if (message === undefined) {
      continue
    }

    uniqueMessages.set(message, true)
  }

  const messages = [...uniqueMessages.keys()]

  if (messages.length === 0) {
    return null
  }

  return messages.length === 1 ? messages[0] : messages
})
</script>

<template>
  <div
    v-if="$slots.default !== undefined || content !== null"
    role="alert"
    data-slot="field-error"
    :class="cn('text-sm font-normal text-destructive', props.class)"
  >
    <slot v-if="$slots.default" />

    <template v-else-if="typeof content === 'string'">
      {{ content }}
    </template>

    <ul
      v-else-if="Array.isArray(content)"
      class="ms-4 flex list-disc flex-col gap-1"
    >
      <li
        v-for="(error, index) in content"
        :key="index"
      >
        {{ error }}
      </li>
    </ul>
  </div>
</template>
