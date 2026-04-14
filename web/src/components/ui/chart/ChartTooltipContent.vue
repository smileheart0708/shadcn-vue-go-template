<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import type { ChartConfig } from '.'
import { computed } from 'vue'
import { cn } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    hideLabel?: boolean
    hideIndicator?: boolean
    indicator?: 'line' | 'dot' | 'dashed'
    nameKey?: string
    labelKey?: string
    labelFormatter?: (d: number | Date) => string
    payload?: Record<string, unknown>
    config?: ChartConfig
    class?: HTMLAttributes['class']
    color?: string
    x?: number | Date
  }>(),
  { payload: () => ({}), config: () => ({}), indicator: 'dot' },
)

// TODO: currently we use `createElement` and `render` to render the
// const chartContext = useChart(null)

type TooltipPayloadItem = {
  key: string
  value: unknown
  itemConfig: NonNullable<ChartConfig[string]>
  indicatorColor?: string
}

const payload = computed<TooltipPayloadItem[]>(() => {
  const items: TooltipPayloadItem[] = []

  for (const [key, value] of Object.entries(props.payload)) {
    const itemConfig = props.config[key]

    if (itemConfig === undefined) {
      continue
    }

    items.push({
      key,
      value,
      itemConfig,
      indicatorColor: itemConfig.color ?? readPayloadColor(props.payload),
    })
  }

  return items
})

const nestLabel = computed(() => Object.keys(props.payload).length === 1 && props.indicator !== 'dot')
const tooltipLabel = computed(() => {
  if (props.hideLabel === true) {
    return null
  }

  if (props.labelFormatter !== undefined && props.x !== undefined) {
    return props.labelFormatter(props.x)
  }

  if (props.labelKey !== undefined) {
    const labelConfig = props.config[props.labelKey]
    const label = labelConfig?.label

    if (label !== undefined) {
      return label
    }

    return props.payload[props.labelKey] ?? null
  }

  return props.x ?? null
})

function readPayloadColor(payload: Record<string, unknown>): string | undefined {
  const fill = payload.fill
  return typeof fill === 'string' ? fill : undefined
}

function formatTooltipValue(value: unknown): string {
  if (value === null || value === undefined) {
    return ''
  }

  if (typeof value === 'number' || typeof value === 'bigint' || typeof value === 'string') {
    return value.toLocaleString()
  }

  if (value instanceof Date) {
    return value.toLocaleString()
  }

  if (typeof value === 'boolean') {
    return value ? 'true' : 'false'
  }

  if (typeof value === 'object') {
    const json = JSON.stringify(value)
    return json === undefined ? '' : json
  }

  return ''
}
</script>

<template>
  <div :class="cn('grid min-w-32 items-start gap-1.5 rounded-lg border border-border/50 bg-background px-2.5 py-1.5 text-xs shadow-xl', props.class)">
    <slot>
      <div
        v-if="!nestLabel && tooltipLabel !== null"
        class="font-medium"
      >
        {{ tooltipLabel }}
      </div>
      <div class="grid gap-1.5">
        <div
          v-for="{ value, itemConfig, indicatorColor, key } in payload"
          :key="key"
          :class="cn('flex w-full flex-wrap items-stretch gap-2 [&>svg]:size-2.5 [&>svg]:text-muted-foreground', indicator === 'dot' ? 'items-center' : undefined)"
        >
          <component
            :is="itemConfig.icon"
            v-if="itemConfig.icon !== undefined"
          />
          <template v-else-if="!hideIndicator">
            <div
              :class="
                cn('shrink-0 rounded-[2px] border-border bg-(--color-bg)', {
                  'size-2.5': indicator === 'dot',
                  'w-1': indicator === 'line',
                  'w-0 border-[1.5px] border-dashed bg-transparent': indicator === 'dashed',
                  'my-0.5': nestLabel && indicator === 'dashed',
                })
              "
              :style="{ '--color-bg': indicatorColor, '--color-border': indicatorColor }"
            />
          </template>

          <div :class="cn('flex flex-1 justify-between leading-none', nestLabel ? 'items-end' : 'items-center')">
            <div class="grid gap-1.5">
              <div
                v-if="nestLabel"
                class="font-medium"
              >
                {{ tooltipLabel }}
              </div>
              <span class="text-muted-foreground">
                {{ itemConfig.label ?? value }}
              </span>
            </div>
            <span
              v-if="value !== undefined && value !== null"
              class="font-mono font-medium text-foreground tabular-nums"
            >
              {{ formatTooltipValue(value) }}
            </span>
          </div>
        </div>
      </div>
    </slot>
  </div>
</template>
