<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import { Spinner } from '@/components/ui/spinner'
import type { AuditLogEntry } from '@/lib/api/system-logs'
import AuditLogsTable from './AuditLogsTable.vue'

defineProps<{
  entries: readonly AuditLogEntry[]
  loading: boolean
  refreshing: boolean
  loadFailed: boolean
  page: number
  total: number
  totalPages: number
}>()

const emit = defineEmits<{
  retry: []
  'previous-page': []
  'next-page': []
}>()

const { t } = useI18n()
</script>

<template>
  <Card>
    <CardHeader class="flex flex-col gap-2">
      <CardTitle class="text-lg">{{ t('systemLogs.audit.title') }}</CardTitle>
      <p class="text-sm text-muted-foreground">{{ t('systemLogs.audit.description') }}</p>
    </CardHeader>

    <CardContent class="flex flex-col gap-4">
      <AuditLogsTable
        v-if="loading || !loadFailed"
        :entries="entries"
        :loading="loading"
      />

      <div
        v-else
        class="rounded-xl border bg-card p-4"
      >
        <Empty>
          <EmptyHeader>
            <EmptyTitle>{{ t('systemLogs.audit.feedback.loadFailedTitle') }}</EmptyTitle>
            <EmptyDescription>{{ t('systemLogs.audit.feedback.loadFailed') }}</EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <Button @click="emit('retry')">{{ t('common.action.retry') }}</Button>
          </EmptyContent>
        </Empty>
      </div>

      <div
        v-if="!loading && !loadFailed"
        class="flex items-center justify-between"
      >
        <div class="flex items-center gap-2 text-sm text-muted-foreground">
          <Spinner
            v-if="refreshing"
            class="block-4 inline-4"
          />
          {{ t('systemLogs.audit.pageSummary', { page, totalPages, total }) }}
        </div>

        <div class="flex gap-2">
          <Button
            size="sm"
            variant="outline"
            :disabled="page <= 1 || refreshing"
            @click="emit('previous-page')"
          >
            {{ t('common.action.back') }}
          </Button>
          <Button
            size="sm"
            variant="outline"
            :disabled="page >= totalPages || refreshing"
            @click="emit('next-page')"
          >
            {{ t('common.action.next') }}
          </Button>
        </div>
      </div>
    </CardContent>
  </Card>
</template>
