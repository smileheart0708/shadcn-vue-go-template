<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import type { ButtonVariants } from '@/components/ui/button'
import { buttonVariants } from '@/components/ui/button'
import { cn } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    href?: string
    size?: ButtonVariants['size']
    isActive?: boolean
    class?: HTMLAttributes['class']
  }>(),
  {
    size: 'icon',
  },
)
</script>

<template>
  <a
    :href="href"
    data-slot="pagination-link"
    :data-active="isActive === true ? 'true' : undefined"
    :aria-current="isActive === true ? 'page' : undefined"
    :class="
      cn(
        buttonVariants({
          variant: isActive === true ? 'outline' : 'ghost',
          size,
        }),
        props.class,
      )
    "
  >
    <slot />
  </a>
</template>
