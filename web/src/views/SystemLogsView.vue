<script setup lang="ts">
import { useDocumentVisibility } from '@vueuse/core'
import { computed, nextTick, onWatcherCleanup, ref, watch } from 'vue'
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
import { APIError } from '@/lib/api/client'
import { getAPIErrorMessage } from '@/lib/api/error-messages'
import { openSystemLogsStream, SYSTEM_LOG_LEVEL_VALUES, type SystemLogEntry, type SystemLogLevelFilter } from '@/lib/api/system-logs'
import { formatSystemLogTimestamp } from '@/lib/system-logs/export'
import { cn } from '@/lib/utils'

const MAX_LOCAL_ENTRIES = 1000
const INITIAL_TAIL = 200
const SCROLL_BOTTOM_THRESHOLD = 24
const RECONNECT_BASE_DELAY_MS = 1_000
const RECONNECT_MAX_DELAY_MS = 30_000

const { t } = useI18n()
const documentVisibility = useDocumentVisibility()

const entries = ref<SystemLogEntry[]>([])
const loading = ref(true)
const connecting = ref(false)
const connected = ref(false)
const autoScroll = ref(true)
const searchQuery = ref('')
const levelFilter = ref<SystemLogLevelFilter>('ALL')
const streamError = ref('')
const exportDialogOpen = ref(false)
const viewport = ref<HTMLDivElement | null>(null)
const reconnectKey = ref(0)
const pageVisible = computed(() => documentVisibility.value === undefined || documentVisibility.value === 'visible')

let abortController: AbortController | null = null

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

watch(
  () => [pageVisible.value, reconnectKey.value] as const,
  ([visible]) => {
    if (!visible) {
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
          if (abortController !== controller || controller.signal.aborted) {
            return
          }

          retryAttempt = 0
          connecting.value = false
          connected.value = true
          loading.value = false
          streamError.value = ''
        },
        onEntry(entry) {
          if (abortController !== controller || controller.signal.aborted) {
            return
          }

          appendEntry(entry)
        },
      })
    } catch (error) {
      if (controller.signal.aborted || abortController !== controller || sessionSignal.aborted) {
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

    if (sessionSignal.aborted || controller.signal.aborted) {
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

function waitForReconnect(delayMs: number, signal: AbortSignal): Promise<boolean> {
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
        <p class="text-muted-foreground text-sm">{{ t('systemLogs.description') }}</p>
      </div>

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
    </section>

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
              <SelectItem value="ALL">
                {{ t('systemLogs.filters.level.all') }}
              </SelectItem>
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
              class="min-w-max space-y-1 p-3 font-mono text-xs leading-5 whitespace-nowrap"
            >
              <div
                v-for="entry in visibleEntries"
                :key="entry.id"
                class="hover:bg-muted/60 grid grid-cols-[auto_auto_auto_1fr] items-start gap-2 rounded-lg border border-transparent px-3 py-2 transition-colors"
              >
                <span class="text-muted-foreground tabular-nums">
                  {{ formatSystemLogTimestamp(entry.timestamp) }}
                </span>
                <Badge :variant="getLevelBadgeVariant(entry.level)">
                  {{ entry.level }}
                </Badge>
                <span class="text-muted-foreground">{{ entry.source }}</span>
                <span>{{ entry.text }}</span>
              </div>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>

    <SystemLogsExportDialog
      v-model:open="exportDialogOpen"
      :entries="entries"
    />
  </div>
</template>
