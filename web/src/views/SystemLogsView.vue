<script setup lang="ts">
import { useDocumentVisibility } from '@vueuse/core'
import { storeToRefs } from 'pinia'
import { computed, nextTick, onWatcherCleanup, ref, useTemplateRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { ArrowDownToLine, Download, Pause, RefreshCw, Trash2 } from 'lucide-vue-next'
import SystemLogsExportDialog from '@/components/system-logs/SystemLogsExportDialog.vue'
import SystemLogLevelMultiSelect from '@/components/system-logs/SystemLogLevelMultiSelect.vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { Table, TableBody, TableCell, TableEmpty, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { APIError } from '@/lib/api/client'
import { getAPIErrorMessage } from '@/lib/api/error-messages'
import {
  isSystemLogHistoryLimit,
  isSystemLogLevel,
  listAuditLogs,
  openSystemLogsStream,
  SYSTEM_LOG_HISTORY_LIMIT_VALUES,
  type AuditLogEntry,
  type SystemLogEntry,
  type SystemLogHistoryLimit,
} from '@/lib/api/system-logs'
import { CAPABILITY } from '@/lib/auth/roles'
import { formatSystemLogTimestamp } from '@/lib/system-logs/export'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth'
import { useSystemLogsPreferencesStore } from '@/stores/system-logs-preferences'

const LOCAL_STREAM_MAX_BYTES = 1 * 1024 * 1024
const STREAM_ENTRY_BASE_BYTES = 64
const SCROLL_BOTTOM_THRESHOLD = 24
const RECONNECT_BASE_DELAY_MS = 1_000
const RECONNECT_MAX_DELAY_MS = 30_000

const { t, locale } = useI18n()
const authStore = useAuthStore()
const systemLogsPreferences = useSystemLogsPreferencesStore()
const { levels, historyLimit, exportFormat } = storeToRefs(systemLogsPreferences)
const documentVisibility = useDocumentVisibility()

const activeTab = ref<'console' | 'audit'>('console')

const entries = ref<SystemLogEntry[]>([])
const loading = ref(true)
const connecting = ref(false)
const connected = ref(false)
const autoScroll = ref(true)
const searchQuery = ref('')
const streamError = ref('')
const exportDialogOpen = ref(false)
const viewport = useTemplateRef<HTMLDivElement>('viewport')
const reconnectKey = ref(0)
const pageVisible = computed(() => documentVisibility.value === 'visible')

const auditEntries = ref<AuditLogEntry[]>([])
const auditLoading = ref(true)
const auditRefreshing = ref(false)
const auditLoadFailed = ref(false)
const auditPage = ref(1)
const auditPageSize = ref(20)
const auditTotal = ref(0)

let abortController: AbortController | null = null
const entryIds = new Set<number>()
let localBufferedBytes = 0

const canReadAuditLogs = computed(() => authStore.can(CAPABILITY.managementAuditLogsRead))
const visibleEntries = computed(() => {
  const normalizedQuery = searchQuery.value.trim().toLowerCase()
  const selectedLevels = new Set(levels.value)

  const filteredEntries = entries.value.filter((entry) => {
    if (!isSystemLogLevel(entry.level) || !selectedLevels.has(entry.level)) {
      return false
    }

    if (normalizedQuery === '') {
      return true
    }

    return [entry.text, entry.message, entry.source].some((value) => value.toLowerCase().includes(normalizedQuery))
  })

  return applyHistoryLimit(filteredEntries, historyLimit.value)
})
const connectionStatusLabel = computed(() => {
  if (connecting.value) {
    return 'Connecting'
  }
  if (connected.value) {
    return 'Connected'
  }
  return 'Disconnected'
})
const connectionIndicatorClass = computed(() => {
  if (connecting.value) {
    return 'bg-amber-500'
  }
  if (connected.value) {
    return 'bg-emerald-500'
  }
  return 'bg-red-500'
})
const auditTotalPages = computed(() => Math.max(1, Math.ceil(auditTotal.value / auditPageSize.value)))
const auditDateFormatter = computed(
  () =>
    new Intl.DateTimeFormat(locale.value, {
      dateStyle: 'medium',
      timeStyle: 'short',
    }),
)

watch(
  () => [pageVisible.value, reconnectKey.value, activeTab.value] as const,
  ([visible, , tab]) => {
    if (!visible || tab !== 'console') {
      disconnect()
      loading.value = entries.value.length === 0
      return
    }

    const sessionController = new AbortController()
    onWatcherCleanup(() => {
      sessionController.abort()
      disconnect()
    })
    void maintainStreamConnection(sessionController.signal)
  },
  { immediate: true },
)

watch(historyLimit, (nextLimit, previousLimit) => {
  if (historyLimitRank(nextLimit) > historyLimitRank(previousLimit)) {
    reconnect()
  }
})

watch(
  () => activeTab.value,
  (tab) => {
    if (tab === 'audit' && canReadAuditLogs.value && auditEntries.value.length === 0 && !auditLoadFailed.value) {
      void loadAuditLogs()
    }
  },
  { immediate: true },
)

async function maintainStreamConnection(sessionSignal: AbortSignal) {
  let retryAttempt = 0

  while (!sessionSignal.aborted) {
    const controller = new AbortController()
    abortController = controller
    connecting.value = true
    connected.value = false
    loading.value = entries.value.length === 0

    try {
      await openSystemLogsStream({
        tail: historyLimit.value,
        signal: controller.signal,
        onOpen() {
          if (!isCurrentStream(controller)) {
            return
          }

          retryAttempt = 0
          connecting.value = false
          connected.value = true
          loading.value = false
          streamError.value = ''
        },
        onEntry(entry) {
          if (!isCurrentStream(controller)) {
            return
          }

          appendEntry(entry)
        },
      })
    } catch (error) {
      if (shouldStopStream(controller, sessionSignal)) {
        return
      }

      connecting.value = false
      connected.value = false
      loading.value = false

      const message = getAPIErrorMessage(t, error, 'apiError.systemLogStreamFailed')
      streamError.value = message

      if (!shouldRetryStreamError(error)) {
        toast.error(message)
        return
      }

      retryAttempt += 1
      if (!(await waitForReconnect(getReconnectDelayMs(retryAttempt), sessionSignal))) {
        return
      }
      continue
    } finally {
      if (abortController === controller) {
        abortController = null
      }
    }

    if (shouldStopStream(controller, sessionSignal)) {
      return
    }

    connecting.value = false
    connected.value = false
    loading.value = false

    retryAttempt += 1
    if (!(await waitForReconnect(getReconnectDelayMs(retryAttempt), sessionSignal))) {
      return
    }
  }
}

async function loadAuditLogs(options: { background?: boolean } = {}) {
  if (!canReadAuditLogs.value) {
    return
  }

  if (options.background === true) {
    auditRefreshing.value = true
  } else {
    auditLoading.value = true
  }

  try {
    const response = await listAuditLogs(auditPage.value, auditPageSize.value)
    auditEntries.value = response.items
    auditPage.value = response.page
    auditPageSize.value = response.pageSize
    auditTotal.value = response.total
    auditLoadFailed.value = false
  } catch (error) {
    auditLoadFailed.value = true
    toast.error(getAPIErrorMessage(t, error, 'systemLogs.audit.feedback.loadFailed'))
  } finally {
    auditLoading.value = false
    auditRefreshing.value = false
  }
}

function disconnect() {
  const controller = abortController
  abortController = null
  controller?.abort()
  connecting.value = false
  connected.value = false
}

function appendEntry(entry: SystemLogEntry) {
  if (entryIds.has(entry.id)) {
    return
  }

  const nextEntries = [...entries.value]
  nextEntries.splice(findEntryInsertIndex(nextEntries, entry.id), 0, entry)
  entryIds.add(entry.id)
  localBufferedBytes += getSystemLogEntrySize(entry)
  pruneLocalEntries(nextEntries)
  entries.value = nextEntries

  if (autoScroll.value) {
    void scrollToBottom()
  }
}

function clearEntries() {
  entries.value = []
  entryIds.clear()
  localBufferedBytes = 0
}

function reconnect() {
  streamError.value = ''
  reconnectKey.value += 1
}

function resumeAutoScroll() {
  autoScroll.value = true
  void scrollToBottom()
}

async function scrollToBottom() {
  await nextTick()
  if (!viewport.value) {
    return
  }
  viewport.value.scrollTop = viewport.value.scrollHeight
}

function handleViewportScroll(event: Event) {
  const target = event.target
  if (!(target instanceof HTMLDivElement)) {
    return
  }

  const distanceFromBottom = target.scrollHeight - target.scrollTop - target.clientHeight
  autoScroll.value = distanceFromBottom <= SCROLL_BOTTOM_THRESHOLD
}

function getLevelBadgeVariant(level: string): 'destructive' | 'warning' | 'outline' | 'secondary' {
  switch (level) {
    case 'ERROR':
      return 'destructive'
    case 'WARN':
      return 'warning'
    case 'INFO':
      return 'outline'
    default:
      return 'secondary'
  }
}

function applyHistoryLimit(sourceEntries: SystemLogEntry[], limit: SystemLogHistoryLimit) {
  if (limit === 'ALL') {
    return sourceEntries
  }

  return sourceEntries.slice(-limit)
}

function findEntryInsertIndex(sourceEntries: SystemLogEntry[], entryId: number) {
  let low = 0
  let high = sourceEntries.length
  while (low < high) {
    const middle = Math.floor((low + high) / 2)
    const middleEntry = sourceEntries[middle]
    if (middleEntry.id < entryId) {
      low = middle + 1
    } else {
      high = middle
    }
  }
  return low
}

function pruneLocalEntries(sourceEntries: SystemLogEntry[]) {
  while (localBufferedBytes > LOCAL_STREAM_MAX_BYTES && sourceEntries.length > 0) {
    const removedEntry = sourceEntries.shift()
    if (!removedEntry) {
      break
    }
    entryIds.delete(removedEntry.id)
    localBufferedBytes -= getSystemLogEntrySize(removedEntry)
  }
  if (localBufferedBytes < 0) {
    localBufferedBytes = 0
  }
}

function getSystemLogEntrySize(entry: SystemLogEntry) {
  return STREAM_ENTRY_BASE_BYTES + entry.level.length + entry.message.length + entry.text.length + entry.source.length
}

function handleHistoryLimitChange(value: unknown) {
  const nextLimit = normalizeHistoryLimitSelectValue(value)
  if (nextLimit === null) {
    return
  }
  systemLogsPreferences.setHistoryLimit(nextLimit)
}

function normalizeHistoryLimitSelectValue(value: unknown): SystemLogHistoryLimit | null {
  if (isSystemLogHistoryLimit(value)) {
    return value
  }
  if (typeof value !== 'string') {
    return null
  }

  const normalizedValue = value.trim().toUpperCase()
  if (normalizedValue === 'ALL') {
    return 'ALL'
  }

  const numericValue = Number(normalizedValue)
  return isSystemLogHistoryLimit(numericValue) ? numericValue : null
}

function historyLimitRank(limit: SystemLogHistoryLimit) {
  return limit === 'ALL' ? Number.POSITIVE_INFINITY : limit
}

function getAuditOutcomeVariant(outcome: 'success' | 'failure') {
  return outcome === 'success' ? 'outline' : 'destructive'
}

function formatAuditDate(value: string) {
  return auditDateFormatter.value.format(new Date(value))
}

function formatAuditActor(id: number | null) {
  return id === null ? t('common.text.none') : `#${String(id)}`
}

function formatAuditReason(reason: string | null) {
  return reason === null || reason.trim() === '' ? t('common.text.none') : reason
}

function goToPreviousAuditPage() {
  if (auditPage.value <= 1 || auditRefreshing.value) {
    return
  }

  auditPage.value -= 1
  void loadAuditLogs({ background: true })
}

function goToNextAuditPage() {
  if (auditPage.value >= auditTotalPages.value || auditRefreshing.value) {
    return
  }

  auditPage.value += 1
  void loadAuditLogs({ background: true })
}

function shouldRetryStreamError(error: unknown) {
  if (error instanceof APIError) {
    return error.status === 408 || error.status === 429 || error.status >= 500
  }

  return error instanceof TypeError
}

function getReconnectDelayMs(attempt: number) {
  const exponent = Math.max(0, attempt - 1)
  return Math.min(RECONNECT_MAX_DELAY_MS, RECONNECT_BASE_DELAY_MS * 2 ** exponent)
}

function isCurrentStream(controller: AbortController) {
  return abortController === controller && !controller.signal.aborted
}

function shouldStopStream(controller: AbortController, sessionSignal: AbortSignal) {
  return sessionSignal.aborted || !isCurrentStream(controller)
}

async function waitForReconnect(delayMs: number, signal: AbortSignal): Promise<boolean> {
  return new Promise((resolve) => {
    if (signal.aborted) {
      resolve(false)
      return
    }

    const timeoutId = window.setTimeout(() => {
      signal.removeEventListener('abort', handleAbort)
      resolve(true)
    }, delayMs)

    function handleAbort() {
      window.clearTimeout(timeoutId)
      resolve(false)
    }

    signal.addEventListener('abort', handleAbort, { once: true })
  })
}
</script>

<template>
  <div class="flex flex-1 flex-col gap-4 p-4 lg:gap-6 lg:overflow-hidden lg:p-6 lg:min-block-0">
    <section class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex flex-wrap items-center gap-3">
        <h1 class="text-2xl font-semibold">{{ t('systemLogs.title') }}</h1>
        <Badge variant="outline">
          {{ t('systemLogs.summary.buffered', { count: entries.length }) }}
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

    <Tabs
      v-model="activeTab"
      class="lg:flex-1 lg:min-block-0"
    >
      <TabsContent
        value="console"
        class="flex flex-col gap-4 lg:flex-1 lg:min-block-0"
      >
        <Card class="lg:flex-1 lg:overflow-hidden lg:min-block-0">
          <CardHeader class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div class="flex flex-wrap items-center gap-2 min-inline-0">
              <CardTitle class="text-lg">{{ t('systemLogs.console.title') }}</CardTitle>
              <span
                class="relative inline-flex items-center justify-center block-3 inline-3"
                :aria-label="connectionStatusLabel"
                role="status"
              >
                <span
                  v-if="connected"
                  class="absolute inline-flex animate-ping rounded-full bg-emerald-500 opacity-60 block-full inline-full"
                />
                <span :class="cn('relative inline-flex rounded-full shadow-sm ring-2 ring-background block-3 inline-3', connectionIndicatorClass)" />
              </span>
              <Badge
                v-if="streamError"
                variant="destructive"
                class="truncate max-inline-full"
              >
                {{ streamError }}
              </Badge>
            </div>

            <div class="flex flex-wrap items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                :disabled="entries.length === 0"
                @click="exportDialogOpen = true"
              >
                <Download class="block-4 inline-4" />
                {{ t('systemLogs.actions.export') }}
              </Button>
              <Button
                variant="outline"
                size="sm"
                @click="clearEntries"
              >
                <Trash2 class="block-4 inline-4" />
                {{ t('systemLogs.actions.clear') }}
              </Button>
              <Button
                variant="outline"
                size="sm"
                @click="autoScroll ? (autoScroll = false) : resumeAutoScroll()"
              >
                <Pause
                  v-if="autoScroll"
                  class="block-4 inline-4"
                />
                <ArrowDownToLine
                  v-else
                  class="block-4 inline-4"
                />
                {{ autoScroll ? t('systemLogs.actions.pauseFollow') : t('systemLogs.actions.resumeFollow') }}
              </Button>
              <Button
                variant="outline"
                size="sm"
                :disabled="connecting"
                @click="reconnect"
              >
                <RefreshCw :class="cn('block-4 inline-4', connecting && 'animate-spin')" />
                {{ t('systemLogs.actions.reconnect') }}
              </Button>
            </div>
          </CardHeader>

          <CardContent class="flex flex-col gap-4 lg:flex-1 lg:min-block-0">
            <div class="flex flex-col gap-2 md:flex-row md:items-center">
              <Input
                v-model="searchQuery"
                :placeholder="t('systemLogs.filters.searchPlaceholder')"
                class="md:flex-1"
              />

              <SystemLogLevelMultiSelect
                :model-value="levels"
                @update:model-value="systemLogsPreferences.setLevels"
              />

              <Select
                :model-value="historyLimit"
                @update:model-value="handleHistoryLimitChange"
              >
                <SelectTrigger class="inline-full md:inline-auto">
                  <SelectValue :placeholder="t('systemLogs.filters.historyPlaceholder')" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem
                    v-for="limit in SYSTEM_LOG_HISTORY_LIMIT_VALUES"
                    :key="String(limit)"
                    :value="limit"
                  >
                    {{ t(`systemLogs.filters.history.${String(limit)}`) }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div class="flex flex-1 flex-col rounded-xl border bg-sidebar/40 min-block-0">
              <div
                ref="viewport"
                class="overflow-auto rounded-xl block-128 lg:flex-1 lg:block-auto lg:min-block-0"
                @scroll="handleViewportScroll"
              >
                <div
                  v-if="loading"
                  class="space-y-2 p-3"
                >
                  <Skeleton
                    v-for="index in 8"
                    :key="index"
                    class="rounded-lg block-11"
                  />
                </div>

                <div
                  v-else-if="visibleEntries.length === 0"
                  class="p-10"
                >
                  <Empty>
                    <EmptyHeader>
                      <EmptyTitle>{{ t('systemLogs.empty.title') }}</EmptyTitle>
                      <EmptyDescription>{{ t('systemLogs.empty.description') }}</EmptyDescription>
                    </EmptyHeader>
                    <EmptyContent />
                  </Empty>
                </div>

                <div
                  v-else
                  class="space-y-1 p-3 font-mono text-xs/5 whitespace-nowrap min-inline-max"
                >
                  <div
                    v-for="entry in visibleEntries"
                    :key="entry.id"
                    class="grid grid-cols-[auto_auto_auto_1fr] items-start gap-2 rounded-lg border border-transparent px-3 py-2 transition-colors hover:bg-muted/60"
                  >
                    <span class="text-muted-foreground tabular-nums">{{ formatSystemLogTimestamp(entry.timestamp) }}</span>
                    <Badge :variant="getLevelBadgeVariant(entry.level)">{{ entry.level }}</Badge>
                    <span class="text-muted-foreground">{{ entry.source }}</span>
                    <span>{{ entry.text }}</span>
                  </div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent
        v-if="canReadAuditLogs"
        value="audit"
        class="flex flex-col gap-4"
      >
        <Card>
          <CardHeader class="flex flex-col gap-2">
            <CardTitle class="text-lg">{{ t('systemLogs.audit.title') }}</CardTitle>
            <p class="text-sm text-muted-foreground">{{ t('systemLogs.audit.description') }}</p>
          </CardHeader>

          <CardContent class="flex flex-col gap-4">
            <div
              v-if="auditLoading"
              class="overflow-hidden rounded-lg border"
            >
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
                </TableBody>
              </Table>
            </div>

            <div
              v-else-if="auditLoadFailed"
              class="rounded-xl border bg-card p-4"
            >
              <Empty>
                <EmptyHeader>
                  <EmptyTitle>{{ t('systemLogs.audit.feedback.loadFailedTitle') }}</EmptyTitle>
                  <EmptyDescription>{{ t('systemLogs.audit.feedback.loadFailed') }}</EmptyDescription>
                </EmptyHeader>
                <EmptyContent>
                  <Button @click="loadAuditLogs">{{ t('common.action.retry') }}</Button>
                </EmptyContent>
              </Empty>
            </div>

            <div
              v-else
              class="flex flex-col gap-4"
            >
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
                    <template v-if="auditEntries.length > 0">
                      <TableRow
                        v-for="entry in auditEntries"
                        :key="entry.id"
                      >
                        <TableCell>{{ formatAuditDate(entry.occurredAt) }}</TableCell>
                        <TableCell class="font-medium">{{ entry.eventType }}</TableCell>
                        <TableCell>
                          <Badge :variant="getAuditOutcomeVariant(entry.outcome)">
                            {{ t(`systemLogs.audit.outcome.${entry.outcome}`) }}
                          </Badge>
                        </TableCell>
                        <TableCell>{{ formatAuditActor(entry.actorUserId) }}</TableCell>
                        <TableCell>{{ formatAuditActor(entry.subjectUserId) }}</TableCell>
                        <TableCell class="truncate max-inline-96">{{ formatAuditReason(entry.reason) }}</TableCell>
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

              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2 text-sm text-muted-foreground">
                  <Spinner
                    v-if="auditRefreshing"
                    class="block-4 inline-4"
                  />
                  {{ t('systemLogs.audit.pageSummary', { page: auditPage, totalPages: auditTotalPages, total: auditTotal }) }}
                </div>

                <div class="flex gap-2">
                  <Button
                    size="sm"
                    variant="outline"
                    :disabled="auditPage <= 1 || auditRefreshing"
                    @click="goToPreviousAuditPage"
                  >
                    {{ t('common.action.back') }}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    :disabled="auditPage >= auditTotalPages || auditRefreshing"
                    @click="goToNextAuditPage"
                  >
                    {{ t('common.action.next') }}
                  </Button>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>

    <SystemLogsExportDialog
      v-model:open="exportDialogOpen"
      v-model:format="exportFormat"
      :entries="entries"
      :history-limit="historyLimit"
      :levels="levels"
    />
  </div>
</template>
