<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { ShieldBan, ShieldCheck, UserPlus } from 'lucide-vue-next'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { Table, TableBody, TableCell, TableEmpty, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { createAdminUser, disableAdminUser, enableAdminUser, listAdminUsers, type ManagedUser, updateAdminUser } from '@/lib/api/admin-users'
import { getAPIErrorMessage } from '@/lib/api/error-messages'
import { CAPABILITY, getUserRoleBadgeVariant, getUserRoleLabelKey } from '@/lib/auth/roles'
import { useAuthStore } from '@/stores/auth'

const { t, locale } = useI18n()
const authStore = useAuthStore()

const users = ref<ManagedUser[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const searchQuery = ref('')
const statusFilter = ref<'ALL' | 'active' | 'disabled'>('ALL')
const loading = ref(true)
const refreshing = ref(false)
const loadFailed = ref(false)

const dialogOpen = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const dialogPending = ref(false)
const selectedUser = ref<ManagedUser | null>(null)
const form = ref({
  username: '',
  email: '',
  password: '',
})

const confirmOpen = ref(false)
const confirmTarget = ref<ManagedUser | null>(null)
const confirmPending = ref(false)

const canCreateUsers = computed(() => authStore.can(CAPABILITY.managementUsersCreate))
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const dateTimeFormatter = computed(
  () =>
    new Intl.DateTimeFormat(locale.value, {
      dateStyle: 'medium',
      timeStyle: 'short',
    }),
)

onMounted(() => {
  void loadUsers()
})

async function loadUsers(options: { background?: boolean } = {}) {
  if (options.background === true) {
    refreshing.value = true
  } else {
    loading.value = true
  }

  try {
    const response = await listAdminUsers({
      q: searchQuery.value,
      status: statusFilter.value === 'ALL' ? null : statusFilter.value,
      page: page.value,
      pageSize: pageSize.value,
    })

    users.value = response.items
    total.value = response.total
    loadFailed.value = false
  } catch (error) {
    loadFailed.value = true
    toast.error(getAPIErrorMessage(t, error, 'adminUsers.feedback.loadFailed'))
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

function submitFilters() {
  page.value = 1
  void loadUsers()
}

function goToPreviousPage() {
  if (page.value <= 1 || refreshing.value) {
    return
  }

  page.value -= 1
  void loadUsers({ background: true })
}

function goToNextPage() {
  if (page.value >= totalPages.value || refreshing.value) {
    return
  }

  page.value += 1
  void loadUsers({ background: true })
}

function openCreateDialog() {
  dialogMode.value = 'create'
  selectedUser.value = null
  form.value = {
    username: '',
    email: '',
    password: '',
  }
  dialogOpen.value = true
}

function openEditDialog(user: ManagedUser) {
  dialogMode.value = 'edit'
  selectedUser.value = user
  form.value = {
    username: user.username,
    email: user.email ?? '',
    password: '',
  }
  dialogOpen.value = true
}

async function submitDialog() {
  if (dialogPending.value) {
    return
  }

  dialogPending.value = true
  const payload = {
    username: form.value.username.trim(),
    email: form.value.email.trim() === '' ? null : form.value.email.trim(),
  }
  const creating = dialogMode.value === 'create'
  const requestPromise = creating
    ? createAdminUser({
        ...payload,
        password: form.value.password,
      })
    : updateAdminUser(selectedUser.value?.id ?? 0, payload)

  toast.promise(requestPromise, {
    loading: creating ? t('adminUsers.feedback.creating') : t('adminUsers.feedback.updating'),
    success: () => (creating ? t('adminUsers.feedback.createSuccess') : t('adminUsers.feedback.updateSuccess')),
    error: (error: unknown) => getAPIErrorMessage(t, error, creating ? 'adminUsers.feedback.createFailed' : 'adminUsers.feedback.updateFailed'),
  })

  try {
    await requestPromise
    dialogOpen.value = false
    await loadUsers({ background: true })
  } finally {
    dialogPending.value = false
  }
}

function requestToggleStatus(user: ManagedUser) {
  confirmTarget.value = user
  confirmOpen.value = true
}

async function confirmToggleStatus() {
  const target = confirmTarget.value
  if (!target || confirmPending.value) {
    return
  }

  confirmPending.value = true
  const disabling = target.status === 'active'
  const requestPromise = disabling ? disableAdminUser(target.id) : enableAdminUser(target.id)

  toast.promise(requestPromise, {
    loading: disabling ? t('adminUsers.feedback.disabling') : t('adminUsers.feedback.enabling'),
    success: () => (disabling ? t('adminUsers.feedback.disableSuccess') : t('adminUsers.feedback.enableSuccess')),
    error: (error: unknown) => getAPIErrorMessage(t, error, disabling ? 'adminUsers.feedback.disableFailed' : 'adminUsers.feedback.enableFailed'),
  })

  try {
    await requestPromise
    confirmOpen.value = false
    confirmTarget.value = null
    await loadUsers({ background: true })
  } finally {
    confirmPending.value = false
  }
}

function hasAction(user: ManagedUser, action: 'update' | 'disable' | 'enable') {
  return user.actions.includes(action)
}

function formatDateTime(value: string) {
  return dateTimeFormatter.value.format(new Date(value))
}

function getStatusLabel(status: 'active' | 'disabled') {
  return status === 'active' ? t('adminUsers.status.active') : t('adminUsers.status.disabled')
}
</script>

<template>
  <div class="flex flex-1 flex-col gap-6 p-4 lg:p-6">
    <section class="flex items-start justify-between gap-4">
      <div class="space-y-1">
        <h2 class="text-2xl font-semibold">{{ t('adminUsers.title') }}</h2>
        <p class="text-sm text-muted-foreground">{{ t('adminUsers.description') }}</p>
      </div>

      <Button
        :disabled="!canCreateUsers"
        @click="openCreateDialog"
      >
        <UserPlus class="me-2 size-4" />
        {{ t('adminUsers.actions.createUser') }}
      </Button>
    </section>

    <section class="flex flex-col gap-3 sm:flex-row">
      <Input
        v-model="searchQuery"
        :placeholder="t('adminUsers.filters.searchPlaceholder')"
        class="w-full sm:w-80"
        @keydown.enter="submitFilters"
      />

      <Select
        v-model="statusFilter"
        @update:model-value="submitFilters"
      >
        <SelectTrigger class="w-full sm:w-44">
          <SelectValue :placeholder="t('adminUsers.filters.statusPlaceholder')" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="ALL">{{ t('adminUsers.filters.statusAll') }}</SelectItem>
          <SelectItem value="active">{{ t('adminUsers.status.active') }}</SelectItem>
          <SelectItem value="disabled">{{ t('adminUsers.status.disabled') }}</SelectItem>
        </SelectContent>
      </Select>
    </section>

    <div
      v-if="loading"
      class="overflow-hidden rounded-lg border"
    >
      <Table>
        <TableHeader class="bg-muted">
          <TableRow class="hover:bg-transparent">
            <TableHead>{{ t('adminUsers.table.username') }}</TableHead>
            <TableHead>{{ t('adminUsers.table.email') }}</TableHead>
            <TableHead>{{ t('adminUsers.table.role') }}</TableHead>
            <TableHead>{{ t('adminUsers.table.status') }}</TableHead>
            <TableHead>{{ t('adminUsers.table.createdAt') }}</TableHead>
            <TableHead class="text-end">{{ t('adminUsers.table.actions') }}</TableHead>
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
      v-else-if="loadFailed"
      class="rounded-xl border bg-card p-4"
    >
      <Empty>
        <EmptyHeader>
          <EmptyTitle>{{ t('adminUsers.feedback.loadFailedTitle') }}</EmptyTitle>
          <EmptyDescription>{{ t('adminUsers.feedback.loadFailed') }}</EmptyDescription>
        </EmptyHeader>
        <EmptyContent>
          <Button @click="loadUsers">{{ t('adminUsers.actions.retry') }}</Button>
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
              <TableHead>{{ t('adminUsers.table.username') }}</TableHead>
              <TableHead>{{ t('adminUsers.table.email') }}</TableHead>
              <TableHead>{{ t('adminUsers.table.role') }}</TableHead>
              <TableHead>{{ t('adminUsers.table.status') }}</TableHead>
              <TableHead>{{ t('adminUsers.table.createdAt') }}</TableHead>
              <TableHead class="text-end">{{ t('adminUsers.table.actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <template v-if="users.length > 0">
              <TableRow
                v-for="user in users"
                :key="user.id"
              >
                <TableCell class="font-medium">{{ user.username }}</TableCell>
                <TableCell>{{ user.email ?? t('adminUsers.table.noEmail') }}</TableCell>
                <TableCell>
                  <Badge :variant="getUserRoleBadgeVariant(user.role)">
                    {{ t(getUserRoleLabelKey(user.role)) }}
                  </Badge>
                </TableCell>
                <TableCell>
                  <Badge :variant="user.status === 'active' ? 'outline' : 'secondary'">
                    {{ getStatusLabel(user.status) }}
                  </Badge>
                </TableCell>
                <TableCell>{{ formatDateTime(user.createdAt) }}</TableCell>
                <TableCell>
                  <div
                    v-if="user.role === 'owner'"
                    class="text-end text-sm text-muted-foreground"
                  >
                    {{ t('adminUsers.table.ownerReadonly') }}
                  </div>
                  <div
                    v-else
                    class="flex justify-end gap-2"
                  >
                    <Button
                      variant="outline"
                      size="sm"
                      :disabled="!hasAction(user, 'update')"
                      @click="openEditDialog(user)"
                    >
                      {{ t('common.action.edit') }}
                    </Button>
                    <Button
                      size="sm"
                      :variant="user.status === 'active' ? 'destructive' : 'outline'"
                      :disabled="!(hasAction(user, 'disable') || hasAction(user, 'enable'))"
                      @click="requestToggleStatus(user)"
                    >
                      <component
                        :is="user.status === 'active' ? ShieldBan : ShieldCheck"
                        class="me-2 size-4"
                      />
                      {{ user.status === 'active' ? t('adminUsers.actions.disable') : t('adminUsers.actions.enable') }}
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            </template>

            <TableEmpty
              v-else
              :colspan="6"
            >
              {{ t('adminUsers.table.empty') }}
            </TableEmpty>
          </TableBody>
        </Table>
      </div>

      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2 text-sm text-muted-foreground">
          <Spinner
            v-if="refreshing"
            class="size-4"
          />
          {{ t('adminUsers.table.pageSummary', { page, totalPages, total }) }}
        </div>

        <div class="flex gap-2">
          <Button
            size="sm"
            variant="outline"
            :disabled="page <= 1 || refreshing"
            @click="goToPreviousPage"
          >
            {{ t('adminUsers.actions.previousPage') }}
          </Button>
          <Button
            size="sm"
            variant="outline"
            :disabled="page >= totalPages || refreshing"
            @click="goToNextPage"
          >
            {{ t('adminUsers.actions.nextPage') }}
          </Button>
        </div>
      </div>
    </div>

    <Dialog v-model:open="dialogOpen">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{{ dialogMode === 'create' ? t('adminUsers.dialog.createTitle') : t('adminUsers.dialog.editTitle') }}</DialogTitle>
          <DialogDescription>{{ dialogMode === 'create' ? t('adminUsers.dialog.createDescription') : t('adminUsers.dialog.editDescription') }}</DialogDescription>
        </DialogHeader>

        <div class="space-y-4 py-2">
          <Input
            v-model="form.username"
            :placeholder="t('adminUsers.dialog.usernamePlaceholder')"
          />
          <Input
            v-model="form.email"
            :placeholder="t('adminUsers.dialog.emailPlaceholder')"
          />
          <Input
            v-if="dialogMode === 'create'"
            v-model="form.password"
            type="password"
            :placeholder="t('adminUsers.dialog.passwordPlaceholder')"
          />
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            :disabled="dialogPending"
            @click="dialogOpen = false"
          >
            {{ t('common.action.cancel') }}
          </Button>
          <Button
            :disabled="dialogPending"
            @click="submitDialog"
          >
            <Spinner
              v-if="dialogPending"
              class="me-2"
            />
            {{ dialogMode === 'create' ? t('adminUsers.dialog.createSubmit') : t('adminUsers.dialog.editSubmit') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <AlertDialog v-model:open="confirmOpen">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {{ confirmTarget?.status === 'active' ? t('adminUsers.confirm.disableTitle') : t('adminUsers.confirm.enableTitle') }}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {{
              confirmTarget?.status === 'active'
                ? t('adminUsers.confirm.disableDescription', { username: confirmTarget?.username ?? '' })
                : t('adminUsers.confirm.enableDescription', { username: confirmTarget?.username ?? '' })
            }}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel :disabled="confirmPending">{{ t('common.action.cancel') }}</AlertDialogCancel>
          <AlertDialogAction
            :disabled="confirmPending"
            @click.prevent="confirmToggleStatus"
          >
            <Spinner
              v-if="confirmPending"
              class="me-2"
            />
            {{ confirmTarget?.status === 'active' ? t('adminUsers.actions.disable') : t('adminUsers.actions.enable') }}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
