<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { Primitive } from 'reka-ui'
import { computed } from 'vue'
import { THEMES, useChart } from '.'

defineProps<{ id?: HTMLAttributes['id'] }>()

const { config } = useChart()
const themeEntries = [
  ['light', THEMES.light],
  ['dark', THEMES.dark],
] as const

const colorConfig = computed(() => {
  return Object.entries(config.value).filter(([, itemConfig]) => itemConfig.theme !== undefined || itemConfig.color !== undefined)
})
</script>

<template>
  <Primitive
    v-if="colorConfig.length > 0"
    as="style"
  >
    {{
      themeEntries
        .map(
          ([theme, prefix]) => `
${prefix} [data-chart=${id}] {
${colorConfig
  .map(([key, itemConfig]) => {
    const color = itemConfig.theme?.[theme] ?? itemConfig.color
    return color !== undefined ? `  --color-${key}: ${color};` : null
  })
  .join('\n')}
}
`,
        )
        .join('\n')
    }}
  </Primitive>
</template>
