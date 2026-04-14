<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  ArrowDown,
  ArrowRight,
  ArrowUp,
  ArrowUpDown,
  CheckCircle2,
  Circle,
  CircleHelp,
  CirclePlus,
  CircleSlash2,
  Clock3,
  MoreHorizontal,
  SlidersHorizontal,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
} from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { DropdownMenu, DropdownMenuCheckboxItem, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

type TaskLabel = 'Documentation' | 'Bug' | 'Feature'
type TaskStatus = 'Backlog' | 'Todo' | 'In Progress' | 'Done' | 'Canceled'
type TaskPriority = 'Low' | 'Medium' | 'High'

interface TaskRow {
  id: string
  label: TaskLabel
  title: string
  status: TaskStatus
  priority: TaskPriority
}

type TaskColumnId = 'task' | 'title' | 'status' | 'priority'

const tasks: TaskRow[] = [
  { id: 'TASK-8782', label: 'Documentation', title: "You can't compress the program without quantifying the open-source SSD pixel!", status: 'In Progress', priority: 'Medium' },
  { id: 'TASK-7878', label: 'Documentation', title: 'Try to calculate the EXE feed, maybe it will index the multi-byte pixel!', status: 'Backlog', priority: 'Medium' },
  { id: 'TASK-7839', label: 'Bug', title: 'We need to bypass the neural TCP card!', status: 'Todo', priority: 'High' },
  { id: 'TASK-5562', label: 'Feature', title: 'The SAS interface is down, bypass the open-source pixel so we can back up the PNG bandwidth!', status: 'Backlog', priority: 'Medium' },
  { id: 'TASK-8686', label: 'Feature', title: "I'll parse the wireless SSL protocol, that should driver the API panel!", status: 'Canceled', priority: 'Medium' },
  { id: 'TASK-1280', label: 'Bug', title: 'Use the digital TLS panel, then you can transmit the haptic system!', status: 'Done', priority: 'High' },
  { id: 'TASK-7262', label: 'Feature', title: 'The UTF8 application is down, parse the neural bandwidth so we can back up the PNG firewall!', status: 'Done', priority: 'High' },
  { id: 'TASK-1138', label: 'Feature', title: "Generating the driver won't do anything, we need to quantify the 1080p SMTP bandwidth!", status: 'In Progress', priority: 'Medium' },
  { id: 'TASK-7184', label: 'Feature', title: 'We need to program the back-end THX pixel!', status: 'Todo', priority: 'Low' },
  { id: 'TASK-5160', label: 'Documentation', title: "Calculating the bus won't do anything, we need to navigate the back-end JSON protocol!", status: 'In Progress', priority: 'High' },
]

const filterText = ref('')
const rowsPerPage = ref('10')
const currentPage = ref(1)
const selectedTaskIds = ref<string[]>([])
const visibleColumns = ref<TaskColumnId[]>(['task', 'title', 'status', 'priority'])

const columnOptions = [
  { id: 'task', label: 'Task' },
  { id: 'title', label: 'Title' },
  { id: 'status', label: 'Status' },
  { id: 'priority', label: 'Priority' },
] as const satisfies readonly { id: TaskColumnId; label: string }[]

const statusMeta = {
  Backlog: { icon: CircleHelp, iconClass: 'text-muted-foreground', labelClass: 'text-foreground' },
  Todo: { icon: Circle, iconClass: 'text-muted-foreground', labelClass: 'text-foreground' },
  'In Progress': { icon: Clock3, iconClass: 'text-muted-foreground', labelClass: 'text-foreground' },
  Done: { icon: CheckCircle2, iconClass: 'text-muted-foreground', labelClass: 'text-foreground' },
  Canceled: { icon: CircleSlash2, iconClass: 'text-muted-foreground', labelClass: 'text-foreground' },
} as const satisfies Record<TaskStatus, { icon: object; iconClass: string; labelClass: string }>

const priorityMeta = {
  Low: { icon: ArrowDown, class: 'text-muted-foreground' },
  Medium: { icon: ArrowRight, class: 'text-muted-foreground' },
  High: { icon: ArrowUp, class: 'text-muted-foreground' },
} as const satisfies Record<TaskPriority, { icon: object; class: string }>

const filteredTasks = computed(() => {
  const query = filterText.value.trim().toLowerCase()

  if (!query) {
    return tasks
  }

  return tasks.filter((task) => [task.id, task.label, task.title, task.status, task.priority].some((value) => value.toLowerCase().includes(query)))
})

const pageSize = computed(() => Number(rowsPerPage.value))
const pageCount = computed(() => Math.max(1, Math.ceil(filteredTasks.value.length / pageSize.value)))
const pagedTasks = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return filteredTasks.value.slice(start, start + pageSize.value)
})
const isColumnVisible = computed(() => {
  const visible = new Set(visibleColumns.value)
  return (columnId: TaskColumnId) => visible.has(columnId)
})
const selectedCount = computed(() => filteredTasks.value.filter((task) => selectedTaskIds.value.includes(task.id)).length)
const allPageSelected = computed(() => pagedTasks.value.length > 0 && pagedTasks.value.every((task) => selectedTaskIds.value.includes(task.id)))
const somePageSelected = computed(() => !allPageSelected.value && pagedTasks.value.some((task) => selectedTaskIds.value.includes(task.id)))

watch([filterText, rowsPerPage], () => {
  currentPage.value = 1
})

watch(pageCount, (nextPageCount) => {
  if (currentPage.value > nextPageCount) {
    currentPage.value = nextPageCount
  }
})

function toggleAllOnPage(nextValue: boolean) {
  const pageIds = pagedTasks.value.map((task) => task.id)

  if (nextValue) {
    selectedTaskIds.value = Array.from(new Set([...selectedTaskIds.value, ...pageIds]))
    return
  }

  selectedTaskIds.value = selectedTaskIds.value.filter((id) => !pageIds.includes(id))
}

function toggleTaskSelection(taskId: string, nextValue: boolean) {
  if (nextValue) {
    selectedTaskIds.value = Array.from(new Set([...selectedTaskIds.value, taskId]))
    return
  }

  selectedTaskIds.value = selectedTaskIds.value.filter((id) => id !== taskId)
}

function toggleColumnVisibility(columnId: TaskColumnId, nextValue: boolean) {
  if (nextValue) {
    visibleColumns.value = columnOptions.filter((column) => column.id === columnId || visibleColumns.value.includes(column.id)).map((column) => column.id)
    return
  }

  if (visibleColumns.value.length === 1) {
    return
  }

  visibleColumns.value = visibleColumns.value.filter((id) => id !== columnId)
}
</script>

<template>
  <div class="flex flex-1 flex-col gap-6 p-4 lg:p-6">
    <div class="flex items-start justify-between gap-4">
      <div class="space-y-1">
        <h2 class="text-2xl font-semibold">Welcome back!</h2>
        <p class="text-sm text-muted-foreground">Here's a list of your tasks for this month.</p>
      </div>
    </div>

    <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex flex-col gap-3 sm:flex-row">
        <Input
          v-model="filterText"
          placeholder="Filter tasks..."
          class="h-9 w-full sm:w-80 lg:w-94"
        />
        <div class="flex flex-wrap gap-3">
          <Button
            variant="outline"
            size="sm"
            class="h-9 justify-start gap-2.5"
          >
            <CirclePlus class="size-4" />
            Status
          </Button>
          <Button
            variant="outline"
            size="sm"
            class="h-9 justify-start gap-2.5"
          >
            <CirclePlus class="size-4" />
            Priority
          </Button>
        </div>
      </div>

      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <Button
            variant="outline"
            size="sm"
            class="h-9 w-full gap-2.5 sm:w-auto"
          >
            <SlidersHorizontal class="size-4" />
            View
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          align="end"
          class="w-48"
        >
          <DropdownMenuCheckboxItem
            v-for="column in columnOptions"
            :key="column.id"
            :model-value="isColumnVisible(column.id)"
            :disabled="visibleColumns.length === 1 && isColumnVisible(column.id)"
            @update:model-value="toggleColumnVisibility(column.id, !!$event)"
          >
            {{ column.label }}
          </DropdownMenuCheckboxItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>

    <div class="overflow-hidden rounded-lg border">
      <Table>
        <TableHeader class="bg-muted">
          <TableRow class="hover:bg-transparent">
            <TableHead class="w-12">
              <Checkbox
                aria-label="Select all tasks on the current page"
                :model-value="allPageSelected || (somePageSelected && 'indeterminate')"
                @update:model-value="toggleAllOnPage(!!$event)"
              />
            </TableHead>
            <TableHead
              v-if="isColumnVisible('task')"
              class="w-17.5 min-w-17.5 text-sm font-semibold"
            >
              Task
            </TableHead>
            <TableHead
              v-if="isColumnVisible('title')"
              class="min-w-105 text-sm font-semibold"
            >
              <button
                type="button"
                class="inline-flex items-center gap-2 font-semibold"
              >
                Title
                <ArrowUpDown class="size-4 text-muted-foreground" />
              </button>
            </TableHead>
            <TableHead
              v-if="isColumnVisible('status')"
              class="w-22 min-w-22 text-sm font-semibold"
            >
              <button
                type="button"
                class="inline-flex items-center gap-2 font-semibold"
              >
                Status
                <ArrowUpDown class="size-4 text-muted-foreground" />
              </button>
            </TableHead>
            <TableHead
              v-if="isColumnVisible('priority')"
              class="w-20 min-w-20 text-sm font-semibold"
            >
              <button
                type="button"
                class="inline-flex items-center gap-2 font-semibold"
              >
                Priority
                <ArrowUpDown class="size-4 text-muted-foreground" />
              </button>
            </TableHead>
            <TableHead class="w-14" />
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow
            v-for="task in pagedTasks"
            :key="task.id"
          >
            <TableCell class="align-middle">
              <Checkbox
                aria-label="Select task"
                :model-value="selectedTaskIds.includes(task.id)"
                @update:model-value="toggleTaskSelection(task.id, !!$event)"
              />
            </TableCell>
            <TableCell
              v-if="isColumnVisible('task')"
              class="font-medium"
            >
              {{ task.id }}
            </TableCell>
            <TableCell v-if="isColumnVisible('title')">
              <div class="flex min-w-0 items-center gap-3">
                <Badge
                  variant="outline"
                  class="rounded-full px-3 py-1 text-xs font-medium"
                >
                  {{ task.label }}
                </Badge>
                <span class="block min-w-0 truncate text-sm font-medium">
                  {{ task.title }}
                </span>
              </div>
            </TableCell>
            <TableCell v-if="isColumnVisible('status')">
              <div class="flex items-center gap-2 text-sm">
                <component
                  :is="statusMeta[task.status].icon"
                  :class="['size-4 shrink-0', statusMeta[task.status].iconClass]"
                />
                <span :class="statusMeta[task.status].labelClass">{{ task.status }}</span>
              </div>
            </TableCell>
            <TableCell v-if="isColumnVisible('priority')">
              <div class="flex items-center gap-2 text-sm">
                <component
                  :is="priorityMeta[task.priority].icon"
                  :class="['size-4 shrink-0', priorityMeta[task.priority].class]"
                />
                <span>{{ task.priority }}</span>
              </div>
            </TableCell>
            <TableCell>
              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    class="rounded-full"
                  >
                    <span class="sr-only">Open menu</span>
                    <MoreHorizontal class="size-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem>View task</DropdownMenuItem>
                  <DropdownMenuItem>Copy id</DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem>Archive</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>

    <div class="flex flex-col gap-4 px-1 lg:flex-row lg:items-center lg:justify-between">
      <div class="text-sm text-muted-foreground">{{ selectedCount }} of {{ filteredTasks.length }} row(s) selected.</div>

      <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-end sm:gap-6">
        <div class="flex items-center gap-3">
          <span class="text-sm font-medium">Rows per page</span>
          <Select v-model="rowsPerPage">
            <SelectTrigger class="w-20">
              <SelectValue placeholder="10" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="10">10</SelectItem>
              <SelectItem value="20">20</SelectItem>
              <SelectItem value="30">30</SelectItem>
              <SelectItem value="40">40</SelectItem>
              <SelectItem value="50">50</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div class="text-sm font-medium">Page {{ currentPage }} of {{ pageCount }}</div>

        <div class="flex items-center gap-2">
          <Button
            variant="outline"
            size="icon"
            :disabled="currentPage === 1"
            @click="currentPage = 1"
          >
            <span class="sr-only">Go to first page</span>
            <ChevronsLeft class="size-4" />
          </Button>
          <Button
            variant="outline"
            size="icon"
            :disabled="currentPage === 1"
            @click="currentPage = Math.max(1, currentPage - 1)"
          >
            <span class="sr-only">Go to previous page</span>
            <ChevronLeft class="size-4" />
          </Button>
          <Button
            variant="outline"
            size="icon"
            :disabled="currentPage === pageCount"
            @click="currentPage = Math.min(pageCount, currentPage + 1)"
          >
            <span class="sr-only">Go to next page</span>
            <ChevronRight class="size-4" />
          </Button>
          <Button
            variant="outline"
            size="icon"
            :disabled="currentPage === pageCount"
            @click="currentPage = pageCount"
          >
            <span class="sr-only">Go to last page</span>
            <ChevronsRight class="size-4" />
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>
