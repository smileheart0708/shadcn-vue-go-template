<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { RefreshCw } from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { getAPIErrorMessage } from '@/lib/api/error-messages'
import { openSystemLogsStream, SYSTEM_LOG_LEVEL_VALUES, type SystemLogEntry, type SystemLogLevelFilter } from '@/lib/api/system-logs'
import { cn } from '@/lib/utils'

const MAX_LOCAL_ENTRIES = 1000
const INITIAL_TAIL = 200
const SCROLL_BOTTOM_THRESHOLD = 24

const { t } = useI18n()

const entries = ref<SystemLogEntry[]>([])
const loading = ref(true)
const connecting = ref(false)
const connected = ref(false)
const autoScroll = ref(true)
const searchQuery = ref('')
const levelFilter = ref<SystemLogLevelFilter>('ALL')
const streamError = ref('')
const viewport = ref<HTMLDivElement | null>(null)

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

onMounted(() => {
  void connect()
})

onBeforeUnmount(() => {
  disconnect()
})

async function connect() {
  disconnect()
  connecting.value = true
  connected.value = false
  loading.value = entries.value.length === 0
  streamError.value = ''
  abortController = new AbortController()

  try {
    await openSystemLogsStream({
      tail: INITIAL_TAIL,
      signal: abortController.signal,
      onEntry(entry) {
        appendEntry(entry)
      },
    })
  } catch (error) {
    if (abortController?.signal.aborted) {
      return
    }

    const message = getAPIErrorMessage(t, error, 'apiError.systemLogStreamFailed')
    streamError.value = message
    toast.error(message)
  } finally {
    connecting.value = false
    connected.value = false
    loading.value = false
  }
}

function disconnect() {
  abortController?.abort()
  abortController = null
  connecting.value = false
  connected.value = false
}

function appendEntry(entry: SystemLogEntry) {
  connected.value = true
  connecting.value = false
  loading.value = false
  entries.value = [...entries.value, entry].slice(-MAX_LOCAL_ENTRIES)

  if (autoScroll.value) {
    void scrollToBottom()
  }
}

function clearEntries() {
  entries.value = []
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

function formatUnixTimestamp(timestamp: number) {
  return new Date(timestamp * 1000).toLocaleString()
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
            @click="connect"
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
            class="h-[32rem] overflow-auto rounded-xl"
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
                  {{ formatUnixTimestamp(entry.timestamp) }}
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
  </div>
</template>
