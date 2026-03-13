<script setup lang="ts">
import type {
  ColumnDef,
  ColumnFiltersState,
  SortingState,
  VisibilityState,
} from '@tanstack/vue-table'
import { RestrictToVerticalAxis } from '@dnd-kit/abstract/modifiers'
import { h, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  IconChevronDown,
  IconChevronLeft,
  IconChevronRight,
  IconChevronsLeft,
  IconChevronsRight,
  IconCircleCheckFilled,
  IconDotsVertical,
  IconLayoutColumns,
  IconLoader,
  IconPlus,
} from '@tabler/icons-vue'
import {
  FlexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useVueTable,
} from '@tanstack/vue-table'
import { DragDropProvider } from 'dnd-kit-vue'
import type { TableData } from '@/components/data-table-schema'
import { Badge } from '@/components/ui/badge'

import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import DragHandle from './DragHandle.vue'
import DraggableRow from './DraggableRow.vue'

const { t } = useI18n()

const props = defineProps<{ data: TableData[] }>()

type CheckedState = boolean | 'indeterminate'

const sorting = ref<SortingState>([])
const columnFilters = ref<ColumnFiltersState>([])
const columnVisibility = ref<VisibilityState>({})
const rowSelection = ref({})

const columns: ColumnDef<TableData>[] = [
  { id: 'drag', header: () => null, cell: () => h(DragHandle) },
  {
    id: 'select',
    header: ({ table }) =>
      h(Checkbox, {
        modelValue:
          table.getIsAllPageRowsSelected() ||
          (table.getIsSomePageRowsSelected() && 'indeterminate'),
        'onUpdate:modelValue': (value: CheckedState) => {
          table.toggleAllPageRowsSelected(!!value)
        },
        'aria-label': 'Select all',
      }),
    cell: ({ row }) =>
      h(Checkbox, {
        modelValue: row.getIsSelected(),
        'onUpdate:modelValue': (value: CheckedState) => {
          row.toggleSelected(!!value)
        },
        'aria-label': 'Select row',
      }),
    enableSorting: false,
    enableHiding: false,
  },
  {
    accessorKey: 'header',
    header: () => h('div', {}, t('table.column.header')),
    cell: ({ row }) => h('div', String(row.getValue('header'))),
    enableHiding: false,
  },
  {
    accessorKey: 'type',
    header: () => h('div', {}, t('table.column.sectionType')),
    cell: ({ row }) => h(Badge, { variant: 'outline' }, () => String(row.getValue('type'))),
  },
  {
    accessorKey: 'status',
    header: () => h('div', {}, t('table.column.status')),
    cell: ({ row }) => {
      const { status } = row.original
      return h('div', { class: 'flex items-center gap-2' }, [
        status === 'Done'
          ? h(IconCircleCheckFilled, { class: 'h-4 w-4 text-emerald-500' })
          : h(IconLoader, { class: 'h-4 w-4 animate-spin text-muted-foreground' }),
        h('span', {}, status),
      ])
    },
  },
  {
    accessorKey: 'target',
    header: () => h('div', { class: 'flex items-center gap-1' }, [t('table.column.target')]),
    cell: ({ row }) =>
      h(Button, { variant: 'ghost', size: 'sm', class: 'h-auto p-1 text-xs font-mono' }, () => [
        h('span', { class: 'ml-1 font-semibold' }, String(row.getValue('target'))),
      ]),
  },
  {
    accessorKey: 'limit',
    header: () => h('div', { class: 'flex items-center gap-1' }, [t('table.column.limit')]),
    cell: ({ row }) =>
      h(Button, { variant: 'ghost', size: 'sm', class: 'h-auto p-1 text-xs font-mono' }, () => [
        h('span', { class: 'ml-1 font-semibold' }, String(row.getValue('limit'))),
      ]),
  },
  {
    accessorKey: 'reviewer',
    header: () => h('div', {}, t('table.column.reviewer')),
    cell: ({ row }) => {
      const { reviewer } = row.original
      const isAssigned = reviewer !== t('table.select.assignReviewer')

      if (isAssigned) {
        return h('span', {}, reviewer)
      }

      return h(
        Select,
        {},
        {
          default: () => [
            h(
              SelectTrigger,
              { class: 'w-full' },
              { default: () => h(SelectValue, { placeholder: t('table.select.assignReviewer') }) },
            ),
            h(
              SelectContent,
              {},
              {
                default: () => [
                  h(SelectItem, { value: 'eddie' }, () => 'Eddie Lake'),
                  h(SelectItem, { value: 'jamik' }, () => 'Jamik Tashpulatov'),
                ],
              },
            ),
          ],
        },
      )
    },
  },
  {
    id: 'actions',
    cell: () =>
      h(
        DropdownMenu,
        {},
        {
          default: () => [
            h(
              DropdownMenuTrigger,
              { asChild: true },
              {
                default: () =>
                  h(
                    Button,
                    { variant: 'ghost', class: 'h-8 w-8 p-0' },
                    {
                      default: () => [
                        h('span', { class: 'sr-only' }, t('common.action.menu')),
                        h(IconDotsVertical, { class: 'h-4 w-4' }),
                      ],
                    },
                  ),
              },
            ),
            h(
              DropdownMenuContent,
              { align: 'end' },
              {
                default: () => [
                  h(DropdownMenuItem, {}, () => t('table.action.edit')),
                  h(DropdownMenuItem, {}, () => t('table.action.makeCopy')),
                  h(DropdownMenuItem, {}, () => t('table.action.favorite')),
                  h(DropdownMenuSeparator, {}),
                  h(DropdownMenuItem, {}, () => t('table.action.delete')),
                ],
              },
            ),
          ],
        },
      ),
  },
]

const table = useVueTable({
  get data() {
    return props.data
  },
  columns,
  getCoreRowModel: getCoreRowModel(),
  getPaginationRowModel: getPaginationRowModel(),
  getSortedRowModel: getSortedRowModel(),
  getFilteredRowModel: getFilteredRowModel(),
  onSortingChange: (updaterOrValue) => {
    sorting.value =
      typeof updaterOrValue === 'function' ? updaterOrValue(sorting.value) : updaterOrValue
  },
  onColumnFiltersChange: (updaterOrValue) => {
    columnFilters.value =
      typeof updaterOrValue === 'function' ? updaterOrValue(columnFilters.value) : updaterOrValue
  },
  onColumnVisibilityChange: (updaterOrValue) => {
    columnVisibility.value =
      typeof updaterOrValue === 'function' ? updaterOrValue(columnVisibility.value) : updaterOrValue
  },
  onRowSelectionChange: (updaterOrValue) => {
    rowSelection.value =
      typeof updaterOrValue === 'function' ? updaterOrValue(rowSelection.value) : updaterOrValue
  },
  state: {
    get sorting() {
      return sorting.value
    },
    get columnFilters() {
      return columnFilters.value
    },
    get columnVisibility() {
      return columnVisibility.value
    },
    get rowSelection() {
      return rowSelection.value
    },
  },
})
</script>

<template>
  <Tabs
    default-value="outline"
    class="w-full flex-col justify-start gap-6"
  >
    <div class="flex items-center justify-between px-4 lg:px-6">
      <Label
        for="view-selector"
        class="sr-only"
      >
        {{ t('table.select.view') }}
      </Label>
      <Select default-value="outline">
        <SelectTrigger
          id="view-selector"
          class="flex w-fit @4xl/main:hidden"
          size="sm"
        >
          <SelectValue :placeholder="t('table.select.view')" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="outline"> {{ t('table.tab.outline') }} </SelectItem>
          <SelectItem value="past-performance"> {{ t('table.tab.pastPerformance') }} </SelectItem>
          <SelectItem value="key-personnel"> {{ t('table.tab.keyPersonnel') }} </SelectItem>
          <SelectItem value="focus-documents"> {{ t('table.tab.focusDocuments') }} </SelectItem>
        </SelectContent>
      </Select>
      <TabsList
        class="**:data-[slot=badge]:bg-muted-foreground/30 hidden **:data-[slot=badge]:size-5 **:data-[slot=badge]:rounded-full **:data-[slot=badge]:px-1 @4xl/main:flex"
      >
        <TabsTrigger value="outline"> {{ t('table.tab.outline') }} </TabsTrigger>
        <TabsTrigger value="past-performance">
          {{ t('table.tab.pastPerformance') }} <Badge variant="secondary"> 3 </Badge>
        </TabsTrigger>
        <TabsTrigger value="key-personnel">
          {{ t('table.tab.keyPersonnel') }} <Badge variant="secondary"> 2 </Badge>
        </TabsTrigger>
        <TabsTrigger value="focus-documents"> {{ t('table.tab.focusDocuments') }} </TabsTrigger>
      </TabsList>
      <div class="flex items-center gap-2">
        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <Button
              variant="outline"
              size="sm"
            >
              <IconLayoutColumns />
              <span class="hidden lg:inline">{{ t('table.action.customizeColumns') }}</span>
              <span class="lg:hidden">{{ t('table.action.columns') }}</span>
              <IconChevronDown />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            align="end"
            class="w-56"
          >
            <template
              v-for="column in table
                .getAllColumns()
                .filter(
                  (column) => typeof column.accessorFn !== 'undefined' && column.getCanHide(),
                )"
              :key="column.id"
            >
              <DropdownMenuCheckboxItem
                class="capitalize"
                :model-value="column.getIsVisible()"
                @update:model-value="
                  (value) => {
                    column.toggleVisibility(!!value)
                  }
                "
              >
                {{ column.id }}
              </DropdownMenuCheckboxItem>
            </template>
          </DropdownMenuContent>
        </DropdownMenu>
        <Button
          variant="outline"
          size="sm"
        >
          <IconPlus />
          <span class="hidden lg:inline">{{ t('table.action.addSection') }}</span>
        </Button>
      </div>
    </div>
    <TabsContent
      value="outline"
      class="relative flex flex-col gap-4 overflow-auto px-4 lg:px-6"
    >
      <div class="overflow-hidden rounded-lg border">
        <DragDropProvider :modifiers="[RestrictToVerticalAxis]">
          <Table>
            <TableHeader class="bg-muted sticky top-0 z-10">
              <TableRow
                v-for="headerGroup in table.getHeaderGroups()"
                :key="headerGroup.id"
              >
                <TableHead
                  v-for="header in headerGroup.headers"
                  :key="header.id"
                  :col-span="header.colSpan"
                >
                  <FlexRender
                    v-if="!header.isPlaceholder"
                    :render="header.column.columnDef.header"
                    :props="header.getContext()"
                  />
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody class="**:data-[slot=table-cell]:first:w-8">
              <template v-if="table.getRowModel().rows.length">
                <DraggableRow
                  v-for="row in table.getRowModel().rows"
                  :key="row.id"
                  :row="row"
                  :index="row.index"
                />
              </template>
              <TableRow v-else>
                <TableCell
                  :col-span="columns.length"
                  class="h-24 text-center"
                >
                  {{ t('table.empty') }}
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </DragDropProvider>
        <!-- <DndContext
            collisionDetection={closestCenter}
            modifiers={[restrictToVerticalAxis]}
            onDragEnd={handleDragEnd}
            sensors={sensors}
            id={sortableId}
          > -->
        <!-- </DndContext> -->
      </div>
      <div class="flex items-center justify-between px-4">
        <div class="text-muted-foreground hidden flex-1 text-sm lg:flex">
          {{ t('table.pagination.rowSelected', { selected: table.getFilteredSelectedRowModel().rows.length, total: table.getFilteredRowModel().rows.length }) }}
        </div>
        <div class="flex w-full items-center gap-8 lg:w-fit">
          <div class="hidden items-center gap-2 lg:flex">
            <Label
              for="rows-per-page"
              class="text-sm font-medium"
            >
              {{ t('table.pagination.rowsPerPage') }}
            </Label>
            <Select
              :model-value="table.getState().pagination.pageSize"
              @update:model-value="
                (value) => {
                  table.setPageSize(Number(value))
                }
              "
            >
              <SelectTrigger
                id="rows-per-page"
                size="sm"
                class="w-20"
              >
                <SelectValue :placeholder="`${table.getState().pagination.pageSize}`" />
              </SelectTrigger>
              <SelectContent side="top">
                <SelectItem
                  v-for="pageSize in [10, 20, 30, 40, 50]"
                  :key="pageSize"
                  :value="`${pageSize}`"
                >
                  {{ pageSize }}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="flex w-fit items-center justify-center text-sm font-medium">
            {{ t('table.pagination.pageOf', { page: table.getState().pagination.pageIndex + 1, total: table.getPageCount() }) }}
          </div>
          <div class="ml-auto flex items-center gap-2 lg:ml-0">
            <Button
              variant="outline"
              class="hidden h-8 w-8 p-0 lg:flex"
              :disabled="!table.getCanPreviousPage()"
              @click="table.setPageIndex(0)"
            >
              <span class="sr-only">{{ t('table.pagination.goToFirstPage') }}</span>
              <IconChevronsLeft />
            </Button>
            <Button
              variant="outline"
              class="size-8"
              size="icon"
              :disabled="!table.getCanPreviousPage()"
              @click="table.previousPage()"
            >
              <span class="sr-only">{{ t('table.pagination.goToPreviousPage') }}</span>
              <IconChevronLeft />
            </Button>
            <Button
              variant="outline"
              class="size-8"
              size="icon"
              :disabled="!table.getCanNextPage()"
              @click="table.nextPage()"
            >
              <span class="sr-only">{{ t('table.pagination.goToNextPage') }}</span>
              <IconChevronRight />
            </Button>
            <Button
              variant="outline"
              class="hidden size-8 lg:flex"
              size="icon"
              :disabled="!table.getCanNextPage()"
              @click="table.setPageIndex(table.getPageCount() - 1)"
            >
              <span class="sr-only">{{ t('table.pagination.goToLastPage') }}</span>
              <IconChevronsRight />
            </Button>
          </div>
        </div>
      </div>
    </TabsContent>
    <TabsContent
      value="past-performance"
      class="flex flex-col px-4 lg:px-6"
    >
      <div class="aspect-video w-full flex-1 rounded-lg border border-dashed" />
    </TabsContent>
    <TabsContent
      value="key-personnel"
      class="flex flex-col px-4 lg:px-6"
    >
      <div class="aspect-video w-full flex-1 rounded-lg border border-dashed" />
    </TabsContent>
    <TabsContent
      value="focus-documents"
      class="flex flex-col px-4 lg:px-6"
    >
      <div class="aspect-video w-full flex-1 rounded-lg border border-dashed" />
    </TabsContent>
  </Tabs>
</template>
