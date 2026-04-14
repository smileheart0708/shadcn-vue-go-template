<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import type { ChartConfig } from '.'
import { computed, onMounted, ref } from 'vue'
import { cn } from '@/lib/utils'
import { useChart } from '.'

const props = withDefaults(
  defineProps<{
    hideIcon?: boolean
    nameKey?: string
    verticalAlign?: 'bottom' | 'top'
    // payload?: any[]
    class?: HTMLAttributes['class']
  }>(),
  { verticalAlign: 'bottom' },
)

const { id, config } = useChart()

interface ChartLegendItem {
  key: string
  itemConfig: NonNullable<ChartConfig[string]>
}

const payload = computed<ChartLegendItem[]>(() => {
  return Object.entries(config.value).flatMap(([key, itemConfig]) => {
    if (itemConfig === undefined) {
      return []
    }

    return [
      {
        key: props.nameKey ?? key,
        itemConfig,
      },
    ]
  })
})

const containerSelector = ref('')
onMounted(() => {
  containerSelector.value = `[data-chart="chart-${id}"]>[data-vis-xy-container]`
})
</script>

<template>
  <div
    v-if="containerSelector !== ''"
    :class="cn('flex items-center justify-center gap-4', verticalAlign === 'top' ? 'pbe-3' : 'pbs-3', props.class)"
  >
    <div
      v-for="{ key, itemConfig } in payload"
      :key="key"
      :class="cn('flex items-center gap-1.5 [&>svg]:size-3 [&>svg]:text-muted-foreground')"
    >
      <component
        :is="itemConfig.icon"
        v-if="itemConfig.icon"
      />
      <div
        v-else
        class="size-2 shrink-0 rounded-[2px]"
        :style="{ backgroundColor: itemConfig.color }"
      />

      {{ itemConfig.label }}
    </div>
  </div>
</template>
