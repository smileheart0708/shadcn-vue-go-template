<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'

defineProps<{
  bufferedCount: number
  canReadAuditLogs: boolean
}>()

const activeTab = defineModel<'console' | 'audit'>('activeTab', { required: true })

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

    <Tabs
      v-model="activeTab"
      class="inline-full lg:inline-auto"
    >
      <TabsList class="inline-full lg:inline-auto">
        <TabsTrigger value="console">{{ t('systemLogs.tabs.console') }}</TabsTrigger>
        <TabsTrigger
          v-if="canReadAuditLogs"
          value="audit"
        >
          {{ t('systemLogs.tabs.audit') }}
        </TabsTrigger>
      </TabsList>
    </Tabs>
  </section>
</template>
