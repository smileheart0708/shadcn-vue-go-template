<script setup lang="ts">
import { IconChartAreaLine, IconClockPause, IconTrendingDown, IconTrendingUp } from '@tabler/icons-vue'

import { Badge } from '@/components/ui/badge'
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty'
import LineChart from '@/components/dashboard/LineChart.vue'

const metrics = [
  {
    title: 'Total Revenue',
    value: '$1,250.00',
    trend: '+12.5%',
    summary: 'Trending up this month',
    detail: 'Visitors for the last 6 months',
    icon: IconTrendingUp,
  },
  {
    title: 'New Customers',
    value: '1,234',
    trend: '-20%',
    summary: 'Down 20% this period',
    detail: 'Acquisition needs attention',
    icon: IconTrendingDown,
  },
  {
    title: 'Active Accounts',
    value: '45,678',
    trend: '+12.5%',
    summary: 'Strong user retention',
    detail: 'Engagement exceed targets',
    icon: IconTrendingUp,
  },
  {
    title: 'Growth Rate',
    value: '4.5%',
    trend: '+4.5%',
    summary: 'Steady performance increase',
    detail: 'Meets growth projections',
    icon: IconTrendingUp,
  },
] as const
</script>

<template>
  <div class="flex flex-col gap-4 lg:min-h-0 lg:flex-1 lg:gap-6">
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <Card
        v-for="metric in metrics"
        :key="metric.title"
        class="@container/card gap-4 py-5"
      >
        <CardHeader class="pb-0">
          <CardDescription>{{ metric.title }}</CardDescription>
          <CardTitle class="text-2xl font-semibold tabular-nums @[250px]/card:text-3xl">
            {{ metric.value }}
          </CardTitle>
          <CardAction>
            <Badge variant="outline">
              <component
                :is="metric.icon"
                class="size-4"
              />
              {{ metric.trend }}
            </Badge>
          </CardAction>
        </CardHeader>
        <CardFooter class="flex-col items-start gap-1.5 pt-0 text-sm">
          <div class="line-clamp-1 flex gap-2 font-medium">
            {{ metric.summary }}
            <component
              :is="metric.icon"
              class="size-4"
            />
          </div>
          <div class="text-muted-foreground">
            {{ metric.detail }}
          </div>
        </CardFooter>
      </Card>
    </div>

    <div class="flex min-h-0 flex-1 flex-col gap-4 lg:grid lg:grid-cols-2">
      <LineChart />

      <Card class="min-h-0 min-w-0 gap-0 py-5 overflow-hidden">
        <CardHeader class="shrink-0 pb-2">
          <CardDescription>Reserved Panel</CardDescription>
          <CardTitle class="text-xl font-semibold">Next Widget Slot</CardTitle>
          <CardAction>
            <Badge variant="outline">
              <IconClockPause class="size-4" />
              Placeholder
            </Badge>
          </CardAction>
        </CardHeader>
        <CardContent class="flex min-h-0 min-w-0 flex-1">
          <Empty class="bg-muted/30 h-full min-w-0 border-border/70">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <IconChartAreaLine class="size-5" />
              </EmptyMedia>
              <EmptyTitle>Reserved for the next dashboard module</EmptyTitle>
              <EmptyDescription>
                Keep this panel available for alerts, conversion funnels, or a quick summary table without changing the current desktop fold.
              </EmptyDescription>
            </EmptyHeader>
            <EmptyContent class="max-w-none min-w-0">
              <div class="grid w-full gap-3">
                <div class="bg-background/80 flex min-w-0 items-center justify-between gap-3 rounded-lg border px-4 py-3">
                  <span class="text-muted-foreground">Suggested width</span>
                  <span class="truncate text-right font-medium">Right-side companion card</span>
                </div>
                <div class="bg-background/80 flex min-w-0 items-center justify-between gap-3 rounded-lg border px-4 py-3">
                  <span class="text-muted-foreground">Suggested content</span>
                  <span class="truncate text-right font-medium">Tasks, notices, or small table</span>
                </div>
                <div class="bg-background/80 flex min-w-0 items-center justify-between gap-3 rounded-lg border px-4 py-3">
                  <span class="text-muted-foreground">Viewport rule</span>
                  <span class="truncate text-right font-medium">No vertical scroll on desktop</span>
                </div>
              </div>
            </EmptyContent>
          </Empty>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
