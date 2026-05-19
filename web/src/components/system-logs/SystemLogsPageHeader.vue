<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Badge } from '@/components/ui/badge'

defineProps<{
  bufferedCount: number
}>()

const { t } = useI18n()
</script>

<template>
  <section class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
    <div class="flex flex-wrap items-center gap-3">
      <h1 class="text-2xl font-semibold">{{ t('systemLogs.title') }}</h1>
      <Badge variant="outline">
        {{ t('systemLogs.summary.buffered', { count: bufferedCount }) }}
      </Badge>
    </div>
  </section>
</template>
