<script setup lang="ts">
import type { VariantProps } from 'class-variance-authority'
import type { ToggleGroupItemProps } from 'reka-ui'
import type { ComputedRef, HTMLAttributes } from 'vue'
import { reactiveOmit } from '@vueuse/core'
import { ToggleGroupItem, useForwardProps } from 'reka-ui'
import { computed, inject } from 'vue'
import { cn } from '@/lib/utils'
import { toggleVariants } from '@/components/ui/toggle'

type ToggleGroupVariants = VariantProps<typeof toggleVariants> & {
  spacing?: number
}
interface ToggleGroupContext {
  variant: Readonly<ComputedRef<ToggleGroupVariants['variant'] | undefined>>
  size: Readonly<ComputedRef<ToggleGroupVariants['size'] | undefined>>
  spacing: Readonly<ComputedRef<number>>
}

const props = defineProps<
  ToggleGroupItemProps & {
    class?: HTMLAttributes['class']
    variant?: ToggleGroupVariants['variant']
    size?: ToggleGroupVariants['size']
  }
>()

const context = inject<ToggleGroupContext>('toggleGroup')
const resolvedVariant = computed(() => context?.variant.value ?? props.variant)
const resolvedSize = computed(() => context?.size.value ?? props.size)
const resolvedSpacing = computed(() => context?.spacing.value)

const delegatedProps = reactiveOmit(props, 'class', 'size', 'variant')
const forwardedProps = useForwardProps(delegatedProps)
</script>

<template>
  <ToggleGroupItem
    v-slot="slotProps"
    data-slot="toggle-group-item"
    :data-variant="resolvedVariant"
    :data-size="resolvedSize"
    :data-spacing="resolvedSpacing"
    v-bind="forwardedProps"
    :class="
      cn(
        toggleVariants({
          variant: resolvedVariant,
          size: resolvedSize,
        }),
        'w-auto min-w-0 shrink-0 px-3 focus:z-10 focus-visible:z-10',
        'data-[spacing=0]:rounded-none data-[spacing=0]:shadow-none data-[spacing=0]:first:rounded-s-md data-[spacing=0]:last:rounded-e-md data-[spacing=0]:data-[variant=outline]:border-s-0 data-[spacing=0]:data-[variant=outline]:first:border-s',
        props.class,
      )
    "
  >
    <slot v-bind="slotProps" />
  </ToggleGroupItem>
</template>
