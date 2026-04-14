<script setup lang="ts">
import type { ListboxItemEmits, ListboxItemProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { reactiveOmit } from '@vueuse/core'
import { ListboxItem, useForwardPropsEmits, useId } from 'reka-ui'
import { computed, onMounted, onUnmounted } from 'vue'
import { cn } from '@/lib/utils'
import { useCommand, useCommandGroup } from '.'

const props = defineProps<ListboxItemProps & { class?: HTMLAttributes['class'] }>()
const emits = defineEmits<ListboxItemEmits>()

const delegatedProps = reactiveOmit(props, 'class')

const forwarded = useForwardPropsEmits(delegatedProps, emits)

const id = useId()
const { filterState, allItems, allGroups } = useCommand()
const groupContext = useCommandGroup()

function getItemTextValue(value: ListboxItemProps['value']) {
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean' || typeof value === 'bigint') {
    return String(value)
  }

  return ''
}

const isRender = computed(() => {
  if (filterState.search === '') {
    return true
  }

  const filteredCurrentItem = filterState.filtered.items.get(id)
  // If the filtered items is undefined means not in the all times map yet
  // Do the first render to add into the map
  if (filteredCurrentItem === undefined) {
    return true
  }

  // Check with filter
  return filteredCurrentItem > 0
})

onMounted(() => {
  const currentElement = document.getElementById(id)
  if (!(currentElement instanceof HTMLElement)) {
    return
  }

  // textValue to perform filter
  allItems.value.set(id, currentElement.textContent || getItemTextValue(props.value))

  const groupId = groupContext.id
  if (groupId !== undefined) {
    if (!allGroups.value.has(groupId)) {
      allGroups.value.set(groupId, new Set([id]))
    } else {
      const groupItems = allGroups.value.get(groupId)
      if (groupItems !== undefined) {
        groupItems.add(id)
      }
    }
  }
})
onUnmounted(() => {
  allItems.value.delete(id)
})
</script>

<template>
  <ListboxItem
    v-if="isRender"
    v-bind="forwarded"
    :id="id"
    data-slot="command-item"
    :class="
      cn(
        'relative flex cursor-default items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-hidden select-none data-disabled:pointer-events-none data-disabled:opacity-50 data-highlighted:bg-accent data-highlighted:text-accent-foreground [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*=\'size-\'])]:size-4 [&_svg:not([class*=\'text-\'])]:text-muted-foreground',
        props.class,
      )
    "
    @select="
      () => {
        filterState.search = ''
      }
    "
  >
    <slot />
  </ListboxItem>
</template>
