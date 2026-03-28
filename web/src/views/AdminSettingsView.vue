<script setup lang="ts">
import { ShieldCheck } from 'lucide-vue-next'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const i18n = useI18n()

function getMessageList(key: 'systemConfig.sections.access.points' | 'systemConfig.sections.roadmap.items') {
  const value = i18n.tm(key)
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string') : []
}

const accessPoints = computed(() => getMessageList('systemConfig.sections.access.points'))
const roadmapItems = computed(() => getMessageList('systemConfig.sections.roadmap.items'))
</script>

<template>
  <div class="flex flex-1 flex-col gap-4 p-4 lg:gap-6 lg:p-6">
    <div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
      <div class="space-y-2">
        <div class="flex items-center gap-2">
          <h1 class="text-2xl font-semibold">{{ i18n.t('systemConfig.title') }}</h1>
          <Badge variant="outline">{{ i18n.t('systemConfig.badge') }}</Badge>
        </div>
        <p class="text-muted-foreground text-sm">{{ i18n.t('systemConfig.description') }}</p>
      </div>
      <div class="flex h-11 w-11 items-center justify-center rounded-xl border bg-muted/40">
        <ShieldCheck class="h-5 w-5" />
      </div>
    </div>

    <div class="grid gap-4 xl:grid-cols-3">
      <Card>
        <CardHeader>
          <CardTitle>{{ i18n.t('systemConfig.sections.access.title') }}</CardTitle>
          <CardDescription>{{ i18n.t('systemConfig.sections.access.description') }}</CardDescription>
        </CardHeader>
        <CardContent>
          <ul class="text-muted-foreground space-y-2 text-sm">
            <li
              v-for="point in accessPoints"
              :key="point"
              class="rounded-lg border bg-muted/30 px-3 py-2"
            >
              {{ point }}
            </li>
          </ul>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{{ i18n.t('systemConfig.sections.observability.title') }}</CardTitle>
          <CardDescription>{{ i18n.t('systemConfig.sections.observability.description') }}</CardDescription>
        </CardHeader>
        <CardContent class="flex h-full items-end">
          <Button as-child>
            <RouterLink :to="{ name: 'system-logs' }">
              {{ i18n.t('systemConfig.sections.observability.cta') }}
            </RouterLink>
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{{ i18n.t('systemConfig.sections.roadmap.title') }}</CardTitle>
          <CardDescription>{{ i18n.t('systemConfig.sections.roadmap.description') }}</CardDescription>
        </CardHeader>
        <CardContent>
          <ul class="text-muted-foreground space-y-2 text-sm">
            <li
              v-for="item in roadmapItems"
              :key="item"
              class="rounded-lg border border-dashed px-3 py-2"
            >
              {{ item }}
            </li>
          </ul>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
