<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { ShieldBan, ShieldCheck, UserPlus } from 'lucide-vue-next'
import AdminUserDialog from '@/components/admin-users/AdminUserDialog.vue'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { Table, TableBody, TableCell, TableEmpty, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { banAdminUser, createAdminUser, listAdminUsers, type AdminUser, updateAdminUser, unbanAdminUser } from '@/lib/api/admin-users'
import { getAPIErrorMessage } from '@/lib/api/error-messages'
import { getUserRoleLabelKey, USER_ROLE } from '@/lib/auth/roles'
import { useAuthStore } from '@/stores/auth'

const { t, locale } = useI18n()
const authStore = useAuthStore()

const users = ref<AdminUser[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const searchQuery = ref('')
const roleFilter = ref<'ALL' | '0' | '1' | '2'>('ALL')
const statusFilter = ref<'ALL' | 'active' | 'banned'>('ALL')
const loading = ref(true)
const initialLoadResolved = ref(false)
const refreshing = ref(false)
const loadFailed = ref(false)

const dialogOpen = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const dialogPending = ref(false)
const selectedUser = ref<AdminUser | null>(null)

const confirmOpen = ref(false)
const confirmTarget = ref<AdminUser | null>(null)
const confirmPending = ref(false)

const isSuperAdmin = computed(() => authStore.user?.role === USER_ROLE.superAdmin)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const canGoPrevious = computed(() => page.value > 1)
const canGoNext = computed(() => page.value < totalPages.value)
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
  const useBackgroundRefresh = options.background ?? initialLoadResolved.value

  if (useBackgroundRefresh) {
    refreshing.value = true
  } else {
    loading.value = true
  }

  try {
    const response = await listAdminUsers({
      q: searchQuery.value,
      role: roleFilter.value === 'ALL' ? null : Number(roleFilter.value),
      status: statusFilter.value === 'ALL' ? null : statusFilter.value,
      page: page.value,
      pageSize: pageSize.value,
      sort: 'created_at_desc',
    })

    users.value = response.items
    total.value = response.total
    loadFailed.value = false
  } catch (error) {
    if (!useBackgroundRefresh) {
      loadFailed.value = true
    }
    toast.error(getAPIErrorMessage(t, error, 'adminUsers.feedback.loadFailed'))
  } finally {
    initialLoadResolved.value = true
    loading.value = false
    refreshing.value = false
  }
}

function submitFilters() {
  page.value = 1
  void loadUsers()
}

function openCreateDialog() {
  dialogMode.value = 'create'
  selectedUser.value = null
  dialogOpen.value = true
}

function openEditDialog(user: AdminUser) {
  dialogMode.value = 'edit'
  selectedUser.value = user
  dialogOpen.value = true
}

function requestToggleBan(user: AdminUser) {
  confirmTarget.value = user
  confirmOpen.value = true
}

async function handleDialogSubmit(payload: { username: string; email: string | null; password?: string; role?: number | null }) {
  if (payload.username.length === 0) {
    toast.error(t('settings.account.usernameRequired'))
    return
  }
  if (dialogMode.value === 'create' && (payload.password === undefined || payload.password.length < 8)) {
    toast.error(t('apiError.passwordTooShort'))
    return
  }

  dialogPending.value = true

  const actionPromise =
    dialogMode.value === 'create'
      ? createAdminUser({
          username: payload.username,
          email: payload.email,
          password: payload.password ?? '',
          role: payload.role,
        })
      : updateAdminUser(selectedUser.value?.id ?? 0, {
          username: payload.username,
          email: payload.email,
          role: payload.role,
        })

  toast.promise(actionPromise, {
    loading: dialogMode.value === 'create' ? t('adminUsers.feedback.creating') : t('adminUsers.feedback.updating'),
    success: () => (dialogMode.value === 'create' ? t('adminUsers.feedback.createSuccess') : t('adminUsers.feedback.updateSuccess')),
    error: (error: unknown) => getAPIErrorMessage(t, error, dialogMode.value === 'create' ? 'adminUsers.feedback.createFailed' : 'adminUsers.feedback.updateFailed'),
  })

  try {
    await actionPromise
    dialogOpen.value = false
    await loadUsers({ background: true })
  } catch {
    return
  } finally {
    dialogPending.value = false
  }
}

async function confirmToggleBan() {
  if (!confirmTarget.value) {
    return
  }

  confirmPending.value = true
  const actionPromise = confirmTarget.value.status === 'banned' ? unbanAdminUser(confirmTarget.value.id) : banAdminUser(confirmTarget.value.id)

  toast.promise(actionPromise, {
    loading: confirmTarget.value.status === 'banned' ? t('adminUsers.feedback.unbanning') : t('adminUsers.feedback.banning'),
    success: () => (confirmTarget.value?.status === 'banned' ? t('adminUsers.feedback.unbanSuccess') : t('adminUsers.feedback.banSuccess')),
    error: (error: unknown) => getAPIErrorMessage(t, error, confirmTarget.value?.status === 'banned' ? 'adminUsers.feedback.unbanFailed' : 'adminUsers.feedback.banFailed'),
  })

  try {
    await actionPromise
    confirmOpen.value = false
    confirmTarget.value = null
    await loadUsers({ background: true })
  } catch {
    return
  } finally {
    confirmPending.value = false
  }
}

function changePage(nextPage: number) {
  if (nextPage < 1 || nextPage > totalPages.value || nextPage === page.value) {
    return
  }

  page.value = nextPage
  void loadUsers({ background: true })
}

function getStatusBadgeVariant(status: AdminUser['status']) {
  return status === 'banned' ? 'destructive' : 'success'
}

function getRoleBadgeVariant(role: number) {
  switch (role) {
    case USER_ROLE.superAdmin:
      return 'warning'
    case USER_ROLE.admin:
      return 'secondary'
    default:
      return 'outline'
  }
}

function canManageUser(user: AdminUser) {
  if (user.id === authStore.user?.id) {
    return false
  }

  if (isSuperAdmin.value) {
    return user.role !== USER_ROLE.superAdmin
  }

  return user.role === USER_ROLE.user
}

function formatDateTime(value: string) {
  return dateTimeFormatter.value.format(new Date(value))
}
</script>

<template>
  <div class="flex flex-1 flex-col gap-6 p-4 lg:p-6">
    <div class="flex items-start justify-between gap-4">
      <div class="space-y-1">
        <h2 class="text-2xl font-semibold">{{ t('adminUsers.title') }}</h2>
        <p class="text-sm text-muted-foreground">{{ t('adminUsers.description') }}</p>
      </div>
      <Button @click="openCreateDialog">
        <UserPlus class="me-2 size-4" />
        {{ t('adminUsers.actions.createUser') }}
      </Button>
    </div>

    <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex flex-col gap-3 sm:flex-row">
        <Input
          v-model="searchQuery"
          :placeholder="t('adminUsers.filters.searchPlaceholder')"
          class="w-full sm:w-80 lg:w-94"
          @keydown.enter="submitFilters"
        />
        <div class="flex flex-wrap gap-3">
          <Select
            v-model="roleFilter"
            @update:model-value="submitFilters"
          >
            <SelectTrigger class="w-full sm:w-45">
              <SelectValue :placeholder="t('adminUsers.filters.rolePlaceholder')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="ALL">{{ t('adminUsers.filters.roleAll') }}</SelectItem>
              <SelectItem value="0">{{ t('common.userRole.0') }}</SelectItem>
              <SelectItem
                v-if="isSuperAdmin"
                value="1"
                >{{ t('common.userRole.1') }}</SelectItem
              >
              <SelectItem
                v-if="isSuperAdmin"
                value="2"
                >{{ t('common.userRole.2') }}</SelectItem
              >
            </SelectContent>
          </Select>
          <Select
            v-model="statusFilter"
            @update:model-value="submitFilters"
          >
            <SelectTrigger class="w-full sm:w-45">
              <SelectValue :placeholder="t('adminUsers.filters.statusPlaceholder')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="ALL">{{ t('adminUsers.filters.statusAll') }}</SelectItem>
              <SelectItem value="active">{{ t('adminUsers.status.active') }}</SelectItem>
              <SelectItem value="banned">{{ t('adminUsers.status.banned') }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>
    </div>

    <div
      v-if="loading"
      class="overflow-hidden rounded-lg border"
    >
      <Table>
        <TableHeader class="bg-muted">
          <TableRow class="hover:bg-transparent">
            <TableHead class="text-sm font-semibold">{{ t('adminUsers.table.username') }}</TableHead>
            <TableHead class="text-sm font-semibold">{{ t('adminUsers.table.email') }}</TableHead>
            <TableHead class="text-sm font-semibold">{{ t('adminUsers.table.role') }}</TableHead>
            <TableHead class="text-sm font-semibold">{{ t('adminUsers.table.status') }}</TableHead>
            <TableHead class="text-sm font-semibold">{{ t('adminUsers.table.createdAt') }}</TableHead>
            <TableHead class="text-end text-sm font-semibold">{{ t('adminUsers.table.actions') }}</TableHead>
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
      class="overflow-hidden rounded-lg border"
    >
      <div class="flex h-96 items-center justify-center">
        <Empty>
          <EmptyHeader>
            <EmptyTitle>{{ t('adminUsers.feedback.loadFailedTitle') }}</EmptyTitle>
            <EmptyDescription>{{ t('adminUsers.feedback.loadFailed') }}</EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <Button
              :disabled="refreshing"
              @click="loadUsers()"
            >
              <Spinner
                v-if="refreshing"
                class="me-2"
              />
              {{ t('adminUsers.actions.retry') }}
            </Button>
          </EmptyContent>
        </Empty>
      </div>
    </div>

    <div
      v-else
      class="space-y-4"
    >
      <div class="overflow-hidden rounded-lg border">
        <Table>
          <TableHeader class="bg-muted">
            <TableRow class="hover:bg-transparent">
              <TableHead class="text-sm font-semibold">{{ t('adminUsers.table.username') }}</TableHead>
              <TableHead class="text-sm font-semibold">{{ t('adminUsers.table.email') }}</TableHead>
              <TableHead class="text-sm font-semibold">{{ t('adminUsers.table.role') }}</TableHead>
              <TableHead class="text-sm font-semibold">{{ t('adminUsers.table.status') }}</TableHead>
              <TableHead class="text-sm font-semibold">{{ t('adminUsers.table.createdAt') }}</TableHead>
              <TableHead class="text-end text-sm font-semibold">{{ t('adminUsers.table.actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <template v-if="users.length > 0">
              <TableRow
                v-for="user in users"
                :key="user.id"
              >
                <TableCell>
                  <div class="flex min-w-0 flex-col">
                    <span class="font-medium">{{ user.username }}</span>
                    <span
                      v-if="user.mustChangePassword"
                      class="text-xs text-muted-foreground"
                    >
                      {{ t('adminUsers.table.mustChangePassword') }}
                    </span>
                  </div>
                </TableCell>
                <TableCell class="text-muted-foreground">
                  {{ user.email ?? t('adminUsers.table.noEmail') }}
                </TableCell>
                <TableCell>
                  <Badge :variant="getRoleBadgeVariant(user.role)">
                    {{ t(getUserRoleLabelKey(user.role)) }}
                  </Badge>
                </TableCell>
                <TableCell>
                  <Badge :variant="getStatusBadgeVariant(user.status)">
                    {{ t(`adminUsers.status.${user.status}`) }}
                  </Badge>
                </TableCell>
                <TableCell class="whitespace-nowrap text-muted-foreground">
                  {{ formatDateTime(user.createdAt) }}
                </TableCell>
                <TableCell>
                  <div class="flex justify-end gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      :disabled="!canManageUser(user)"
                      @click="openEditDialog(user)"
                    >
                      {{ t('common.action.edit') }}
                    </Button>
                    <Button
                      size="sm"
                      :variant="user.status === 'banned' ? 'outline' : 'destructive'"
                      :disabled="!canManageUser(user)"
                      @click="requestToggleBan(user)"
                    >
                      <component
                        :is="user.status === 'banned' ? ShieldCheck : ShieldBan"
                        class="me-2 size-4"
                      />
                      {{ user.status === 'banned' ? t('adminUsers.actions.unban') : t('adminUsers.actions.ban') }}
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

      <div class="flex flex-col gap-4 px-1 lg:flex-row lg:items-center lg:justify-between">
        <div
          class="flex items-center gap-2 text-sm text-muted-foreground"
          aria-live="polite"
        >
          <Spinner
            v-if="refreshing"
            class="size-4"
          />
          {{ t('adminUsers.table.pageSummary', { page, totalPages, total }) }}
        </div>

        <div class="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            :disabled="!canGoPrevious || refreshing"
            @click="changePage(page - 1)"
          >
            {{ t('adminUsers.actions.previousPage') }}
          </Button>
          <Button
            variant="outline"
            size="sm"
            :disabled="!canGoNext || refreshing"
            @click="changePage(page + 1)"
          >
            {{ t('adminUsers.actions.nextPage') }}
          </Button>
        </div>
      </div>
    </div>

    <AdminUserDialog
      v-model:open="dialogOpen"
      :mode="dialogMode"
      :user="selectedUser"
      :pending="dialogPending"
      :can-assign-admin-role="isSuperAdmin"
      @submit="handleDialogSubmit"
    />

    <AlertDialog v-model:open="confirmOpen">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {{ confirmTarget?.status === 'banned' ? t('adminUsers.confirm.unbanTitle') : t('adminUsers.confirm.banTitle') }}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {{
              confirmTarget?.status === 'banned'
                ? t('adminUsers.confirm.unbanDescription', { username: confirmTarget?.username ?? '' })
                : t('adminUsers.confirm.banDescription', { username: confirmTarget?.username ?? '' })
            }}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel :disabled="confirmPending">{{ t('common.action.cancel') }}</AlertDialogCancel>
          <AlertDialogAction
            :disabled="confirmPending"
            @click.prevent="confirmToggleBan"
          >
            <Spinner
              v-if="confirmPending"
              class="me-2"
            />
            {{ confirmTarget?.status === 'banned' ? t('adminUsers.actions.unban') : t('adminUsers.actions.ban') }}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>
