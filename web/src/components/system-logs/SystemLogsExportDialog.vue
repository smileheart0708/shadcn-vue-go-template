<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import type { SystemLogEntry, SystemLogHistoryLimit, SystemLogLevel } from '@/lib/api/system-logs'
import {
  downloadSystemLogs,
  selectSystemLogEntries,
  SYSTEM_LOG_EXPORT_FORMAT_VALUES,
  type SystemLogExportFormat,
} from '@/lib/system-logs/export'

interface Props {
  entries: SystemLogEntry[]
  historyLimit: SystemLogHistoryLimit
  levels: readonly SystemLogLevel[]
  open: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const { t } = useI18n()

const exportFormat = defineModel<SystemLogExportFormat>('format', { default: 'csv' })

const openModel = computed({
  get: () => props.open,
  set: (value: boolean) => {
    emit('update:open', value)
  },
})

const exportableCount = computed(
  () =>
    selectSystemLogEntries({
      entries: props.entries,
      historyLimit: props.historyLimit,
      levels: props.levels,
      format: exportFormat.value,
    }).length,
)

function handleExport() {
  const count = exportableCount.value
  if (count === 0) {
    toast.error(t('systemLogs.feedback.exportEmpty'))
    return
  }

  const exportedCount = downloadSystemLogs({
    entries: props.entries,
    historyLimit: props.historyLimit,
    levels: props.levels,
    format: exportFormat.value,
  })

  toast.success(t('systemLogs.feedback.exportSuccess', { count: exportedCount }))
  openModel.value = false
}
</script>

<template>
  <Dialog v-model:open="openModel">
    <DialogContent class="sm:max-inline-md">
      <DialogHeader>
        <DialogTitle>{{ t('systemLogs.export.title') }}</DialogTitle>
        <DialogDescription>
          {{ t('systemLogs.export.description', { count: exportableCount }) }}
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4">
        <div class="space-y-2">
          <Label>{{ t('systemLogs.export.fields.format') }}</Label>
          <Select v-model="exportFormat">
            <SelectTrigger>
              <SelectValue :placeholder="t('systemLogs.export.fields.format')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem
                v-for="format in SYSTEM_LOG_EXPORT_FORMAT_VALUES"
                :key="format"
                :value="format"
              >
                {{ t(`systemLogs.export.formats.${format}`) }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        <p class="text-sm text-muted-foreground">
          {{ t('systemLogs.export.preview', { count: exportableCount }) }}
        </p>
      </div>

      <DialogFooter>
        <Button
          variant="outline"
          @click="openModel = false"
        >
          {{ t('common.action.cancel') }}
        </Button>
        <Button @click="handleExport">
          {{ t('systemLogs.actions.export') }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
