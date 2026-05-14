<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed, ref } from 'vue'
import { Tabs, TabsContent } from '@/components/ui/tabs'
import AuditLogsPanel from '@/components/system-logs/AuditLogsPanel.vue'
import SystemLogConsolePanel from '@/components/system-logs/SystemLogConsolePanel.vue'
import SystemLogsPageHeader from '@/components/system-logs/SystemLogsPageHeader.vue'
import { useAuditLogs } from '@/composables/system-logs/useAuditLogs'
import { useSystemLogStream } from '@/composables/system-logs/useSystemLogStream'
import { CAPABILITY } from '@/lib/auth/roles'
import { useAuthStore } from '@/stores/auth'
import { useSystemLogsPreferencesStore } from '@/stores/system-logs-preferences'

const authStore = useAuthStore()
const systemLogsPreferences = useSystemLogsPreferencesStore()
const { levels, historyLimit, exportFormat } = storeToRefs(systemLogsPreferences)

const activeTab = ref<'console' | 'audit'>('console')
const canReadAuditLogs = computed(() => authStore.can(CAPABILITY.managementAuditLogsRead))

const {
  entries: systemLogEntries,
  loading: systemLogLoading,
  connecting,
  connected,
  streamError,
  clearEntries,
  reconnect,
} = useSystemLogStream({
  activeTab,
  historyLimit,
})

const {
  entries: auditEntries,
  loading: auditLoading,
  refreshing: auditRefreshing,
  loadFailed: auditLoadFailed,
  page: auditPage,
  total: auditTotal,
  totalPages: auditTotalPages,
  loadAuditLogs,
  goToPreviousPage,
  goToNextPage,
} = useAuditLogs({
  activeTab,
  canReadAuditLogs,
})
</script>

<template>
  <div class="flex flex-1 flex-col gap-4 p-4 lg:gap-6 lg:overflow-hidden lg:p-6 lg:min-block-0">
    <SystemLogsPageHeader
      v-model:active-tab="activeTab"
      :buffered-count="systemLogEntries.length"
      :can-read-audit-logs="canReadAuditLogs"
    />

    <Tabs
      v-model="activeTab"
      class="lg:flex-1 lg:min-block-0"
    >
      <TabsContent
        value="console"
        class="flex flex-col gap-4 lg:flex-1 lg:min-block-0"
      >
        <SystemLogConsolePanel
          :entries="systemLogEntries"
          :loading="systemLogLoading"
          :connecting="connecting"
          :connected="connected"
          :stream-error="streamError"
          :levels="levels"
          :history-limit="historyLimit"
          :export-format="exportFormat"
          @clear="clearEntries"
          @reconnect="reconnect"
          @update:levels="systemLogsPreferences.setLevels"
          @update:history-limit="systemLogsPreferences.setHistoryLimit"
          @update:export-format="systemLogsPreferences.setExportFormat"
        />
      </TabsContent>

      <TabsContent
        v-if="canReadAuditLogs"
        value="audit"
        class="flex flex-col gap-4"
      >
        <AuditLogsPanel
          :entries="auditEntries"
          :loading="auditLoading"
          :refreshing="auditRefreshing"
          :load-failed="auditLoadFailed"
          :page="auditPage"
          :total="auditTotal"
          :total-pages="auditTotalPages"
          @retry="loadAuditLogs"
          @previous-page="goToPreviousPage"
          @next-page="goToNextPage"
        />
      </TabsContent>
    </Tabs>
  </div>
</template>
