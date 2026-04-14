<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { computed } from 'vue'
import { DialogClose } from 'reka-ui'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

const props = defineProps<{
  class?: HTMLAttributes['class']
  showCloseButton?: boolean
}>()

const showCloseButton = computed(() => props.showCloseButton)
</script>

<template>
  <div
    data-slot="dialog-footer"
    :class="cn('flex flex-col-reverse gap-2 sm:flex-row sm:justify-end', props.class)"
  >
    <slot />
    <DialogClose
      v-if="showCloseButton"
      as-child
    >
      <Button variant="outline"> Close </Button>
    </DialogClose>
  </div>
</template>
