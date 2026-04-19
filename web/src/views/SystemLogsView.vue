<script setup lang="ts">
import { useDocumentVisibility } from '@vueuse/core'
import { computed, nextTick, onWatcherCleanup, ref, useTemplateRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { Download, RefreshCw } from 'lucide-vue-next'
import SystemLogsExportDialog from '@/components/system-logs/SystemLogsExportDialog.vue'
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
import { listAuditLogs, openSystemLogsStream, SYSTEM_LOG_LEVEL_VALUES, type AuditLogEntry, type SystemLogEntry, type SystemLogLevelFilter } from '@/lib/api/system-logs'
import { CAPABILITY } from '@/lib/auth/roles'
import { formatSystemLogTimestamp } from '@/lib/system-logs/export'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth'

const MAX_LOCAL_ENTRIES = 1000
const INITIAL_TAIL = 200
const SCROLL_BOTTOM_THRESHOLD = 24
const RECONNECT_BASE_DELAY_MS = 1_000
const RECONNECT_MAX_DELAY_MS = 30_000

const { t, locale } = useI18n()
const authStore = useAuthStore()
const documentVisibility = useDocumentVisibility()

const activeTab = ref<'console' | 'audit'>('console')

const entries = ref<SystemLogEntry[]>([])
const loading = ref(true)
const connecting = ref(false)
const connected = ref(false)
const autoScroll = ref(true)
const searchQuery = ref('')
const levelFilter = ref<SystemLogLevelFilter>('ALL')
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

const canReadAuditLogs = computed(() => authStore.can(CAPABILITY.managementAuditLogsRead))
const visibleEntries = computed(() => {
  const normalizedQuery = searchQuery.value.trim().toLowerCase()

  return entries.value.filter((entry) => {
    const matchesLevel = levelFilter.value === 'ALL' || entry.level === levelFilter.value
    if (!matchesLevel) {
      return false
    }

    if (normalizedQuery === '') {
      return true
    }

    return [entry.text, entry.message, entry.source].some((value) => value.toLowerCase().includes(normalizedQuery))
  })
})
const connectionLabel = computed(() => {
  if (connecting.value) {
    return t('systemLogs.connection.connecting')
  }
  if (connected.value) {
    return t('systemLogs.connection.connected')
  }
  return t('systemLogs.connection.disconnected')
})
const connectionVariant = computed(() => {
  if (connecting.value) {
    return 'warning'
  }
  if (connected.value) {
    return 'outline'
  }
  return 'secondary'
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
        tail: INITIAL_TAIL,
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
  // Reconnects replay a recent tail to avoid gaps, so dedupe by server-issued id.
  if (entries.value.some((existingEntry) => existingEntry.id === entry.id)) {
    return
  }

  entries.value = [...entries.value, entry].slice(-MAX_LOCAL_ENTRIES)

  if (autoScroll.value) {
    void scrollToBottom()
  }
}

function clearEntries() {
  entries.value = []
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
  <div class="flex flex-1 flex-col gap-6 p-4 lg:p-6">
    <section class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
      <div>
        <h1 class="text-2xl font-semibold">{{ t('systemLogs.title') }}</h1>
        <p class="text-sm text-muted-foreground">{{ t('systemLogs.description') }}</p>
      </div>

      <Tabs
        v-model="activeTab"
        class="w-full lg:w-auto"
      >
        <TabsList class="w-full lg:w-auto">
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

    <Tabs v-model="activeTab">
      <TabsContent
        value="console"
        class="space-y-6"
      >
        <div class="flex flex-wrap items-center gap-2">
          <Badge :variant="connectionVariant">
            {{ connectionLabel }}
          </Badge>
          <Badge variant="outline">
            {{ t('systemLogs.summary.buffered', { count: entries.length }) }}
          </Badge>
          <Badge
            v-if="streamError"
            variant="destructive"
            class="max-w-full truncate"
          >
            {{ streamError }}
          </Badge>
        </div>

        <Card>
          <CardHeader class="flex flex-row items-center justify-between">
            <CardTitle class="text-lg">{{ t('systemLogs.console.title') }}</CardTitle>

            <div class="flex flex-wrap items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                :disabled="entries.length === 0"
                @click="exportDialogOpen = true"
              >
                <Download class="size-4" />
                {{ t('systemLogs.actions.export') }}
              </Button>
              <Button
                variant="outline"
                size="sm"
                @click="clearEntries"
              >
                {{ t('systemLogs.actions.clear') }}
              </Button>
              <Button
                variant="outline"
                size="sm"
                @click="autoScroll ? (autoScroll = false) : resumeAutoScroll()"
              >
                {{ autoScroll ? t('systemLogs.actions.pauseFollow') : t('systemLogs.actions.resumeFollow') }}
              </Button>
              <Button
                variant="outline"
                size="sm"
                :disabled="connecting"
                @click="reconnect"
              >
                <RefreshCw :class="cn('size-4', connecting && 'animate-spin')" />
                {{ t('systemLogs.actions.reconnect') }}
              </Button>
            </div>
          </CardHeader>

          <CardContent class="space-y-4">
            <div class="flex flex-col gap-3 md:flex-row md:items-center">
              <Input
                v-model="searchQuery"
                :placeholder="t('systemLogs.filters.searchPlaceholder')"
                class="flex-1"
              />

              <Select v-model="levelFilter">
                <SelectTrigger class="w-full md:w-40">
                  <SelectValue :placeholder="t('systemLogs.filters.levelPlaceholder')" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="ALL">{{ t('systemLogs.filters.level.all') }}</SelectItem>
                  <SelectItem
                    v-for="level in SYSTEM_LOG_LEVEL_VALUES"
                    :key="level"
                    :value="level"
                  >
                    {{ t(`systemLogs.filters.level.${level}`) }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div class="rounded-xl border bg-sidebar/40">
              <div
                ref="viewport"
                class="h-128 overflow-auto rounded-xl"
                @scroll="handleViewportScroll"
              >
                <div
                  v-if="loading"
                  class="space-y-2 p-3"
                >
                  <Skeleton
                    v-for="index in 8"
                    :key="index"
                    class="h-11 rounded-lg"
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
                  class="min-w-max space-y-1 p-3 font-mono text-xs/5 whitespace-nowrap"
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
        class="space-y-4"
      >
        <Card>
          <CardHeader class="space-y-1">
            <CardTitle class="text-lg">{{ t('systemLogs.audit.title') }}</CardTitle>
            <p class="text-sm text-muted-foreground">{{ t('systemLogs.audit.description') }}</p>
          </CardHeader>

          <CardContent class="space-y-4">
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
                      <Skeleton class="h-5 rounded-md" />
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
              class="space-y-4"
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
                        <TableCell>{{ formatAuditActor(entry.actorUserID) }}</TableCell>
                        <TableCell>{{ formatAuditActor(entry.subjectUserID) }}</TableCell>
                        <TableCell class="max-w-96 truncate">{{ formatAuditReason(entry.reason) }}</TableCell>
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
                    class="size-4"
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
      :entries="entries"
    />
  </div>
</template>
