<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableEmpty, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import type { AuditLogEntry } from '@/lib/api/system-logs'
import { createAuditDateFormatter, formatAuditActor, formatAuditDate, formatAuditReason, getAuditOutcomeVariant } from '@/utils/system-logs/audit'

defineProps<{
  entries: readonly AuditLogEntry[]
  loading: boolean
}>()

const { t, locale } = useI18n()

const noneLabel = computed(() => t('common.text.none'))
const auditDateFormatter = computed(() => createAuditDateFormatter(locale.value))
</script>

<template>
  <div class="overflow-hidden rounded-lg border">
    <Table>
      <TableHeader class="bg-muted">
        <TableRow class="hover:bg-transparent">
          <TableHead>{{ t('systemLogs.audit.table.occurredAt') }}</TableHead>
          <TableHead>{{ t('systemLogs.audit.table.eventType') }}</TableHead>
          <TableHead>{{ t('systemLogs.audit.table.outcome') }}</TableHead>
          <TableHead>{{ t('systemLogs.audit.table.actor') }}</TableHead>
          <TableHead>{{ t('systemLogs.audit.table.subject') }}</TableHead>
          <TableHead>{{ t('systemLogs.audit.table.reason') }}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        <template v-if="loading">
          <TableRow
            v-for="index in 8"
            :key="index"
          >
            <TableCell
              v-for="cell in 6"
              :key="cell"
            >
              <Skeleton class="rounded-md block-5" />
            </TableCell>
          </TableRow>
        </template>

        <template v-else-if="entries.length > 0">
          <TableRow
            v-for="entry in entries"
            :key="entry.id"
          >
            <TableCell>{{ formatAuditDate(entry.occurredAt, auditDateFormatter) }}</TableCell>
            <TableCell class="font-medium">{{ entry.eventType }}</TableCell>
            <TableCell>
              <Badge :variant="getAuditOutcomeVariant(entry.outcome)">
                {{ t(`systemLogs.audit.outcome.${entry.outcome}`) }}
              </Badge>
            </TableCell>
            <TableCell>{{ formatAuditActor(entry.actorUserId, noneLabel) }}</TableCell>
            <TableCell>{{ formatAuditActor(entry.subjectUserId, noneLabel) }}</TableCell>
            <TableCell class="truncate max-inline-96">{{ formatAuditReason(entry.reason, noneLabel) }}</TableCell>
          </TableRow>
        </template>

        <TableEmpty
          v-else
          :colspan="6"
        >
          {{ t('systemLogs.audit.table.empty') }}
        </TableEmpty>
      </TableBody>
    </Table>
  </div>
</template>
