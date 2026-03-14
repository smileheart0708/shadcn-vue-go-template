<script setup lang="ts">
import imageCompression from 'browser-image-compression'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { useLocaleStore } from '@/stores/locale'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Checkbox } from '@/components/ui/checkbox'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { getAPIErrorMessage } from '@/lib/api/error-messages'
import { canDeleteOwnAccount, getUserRoleBadgeVariant, getUserRoleLabelKey } from '@/lib/auth/roles'
import { MAX_AVATAR_FILE_SIZE_BYTES, SUPPORTED_AVATAR_FILE_TYPES, getAvatarFallbackText } from '@/lib/avatar'
import { localeNames, type AppLocale } from '@/plugins/i18n/locales'

const router = useRouter()
const { t } = useI18n()
const authStore = useAuthStore()
const themeStore = useThemeStore()
const localeStore = useLocaleStore()

const notifications = ref({
  emailNotifications: true,
  pushNotifications: true,
  weeklyDigest: false,
  securityAlerts: true,
})

const saved = ref(false)
const deleteDialogOpen = ref(false)
const deleteCountdown = ref(0)
const deleteAccountConfirmed = ref(false)

const editDialogOpen = ref(false)
const editForm = ref({
  username: '',
  email: '',
})
const pendingAvatarFile = ref<File | null>(null)
const pendingAvatarPreviewURL = ref<string | null>(null)
const profileError = ref('')
const passwordError = ref('')
const passwordForm = ref({
  currentPassword: '',
  newPassword: '',
  confirmPassword: '',
})
const avatarInput = ref<HTMLInputElement | null>(null)
const isSavingProfile = ref(false)
const isUpdatingPassword = ref(false)
const isDeletingAccount = ref(false)

let deleteCountdownTimer: ReturnType<typeof setInterval> | null = null

function saveSettings() {
  saved.value = true
  window.setTimeout(() => {
    saved.value = false
  }, 2000)
}

function isThemePreference(value: unknown): value is 'light' | 'dark' | 'system' {
  return value === 'light' || value === 'dark' || value === 'system'
}

function isAppLocale(value: unknown): value is AppLocale {
  return value === 'en-US' || value === 'zh-CN'
}

function handleThemeChange(value: unknown) {
  if (isThemePreference(value)) {
    themeStore.setTheme(value)
  }
}

function handleLocaleChange(value: unknown) {
  if (isAppLocale(value)) {
    localeStore.setLocale(value)
  }
}

const localeOptions = Object.entries(localeNames).map(([value, label]) => ({
  value,
  label,
}))

const avatarFallbackText = computed(() => getAvatarFallbackText(authStore.user?.username))
const avatarImageSrc = computed(() => authStore.user?.avatarUrl ?? null)
const editAvatarFallbackText = computed(() => getAvatarFallbackText(editForm.value.username || authStore.user?.username))
const editAvatarImageSrc = computed(() => pendingAvatarPreviewURL.value ?? authStore.user?.avatarUrl ?? null)
const roleLabel = computed(() => t(getUserRoleLabelKey(authStore.user?.role ?? 0)))
const roleBadgeVariant = computed(() => getUserRoleBadgeVariant(authStore.user?.role ?? 0))
const canDeleteAccount = computed(() => canDeleteOwnAccount(authStore.user?.role ?? 0))

function resetEditForm() {
  editForm.value = {
    username: authStore.user?.username ?? '',
    email: authStore.user?.email ?? '',
  }
  profileError.value = ''
  clearPendingAvatarPreview()
  pendingAvatarFile.value = null
  if (avatarInput.value) {
    avatarInput.value.value = ''
  }
}

function clearPendingAvatarPreview() {
  if (pendingAvatarPreviewURL.value) {
    URL.revokeObjectURL(pendingAvatarPreviewURL.value)
    pendingAvatarPreviewURL.value = null
  }
}

watch(editDialogOpen, (isOpen) => {
  if (isOpen) {
    resetEditForm()
  }
})

watch(deleteDialogOpen, (isOpen) => {
  if (deleteCountdownTimer) {
    clearInterval(deleteCountdownTimer)
    deleteCountdownTimer = null
  }

  if (!isOpen) {
    deleteCountdown.value = 0
    return
  }

  deleteCountdown.value = 3
  deleteCountdownTimer = setInterval(() => {
    deleteCountdown.value -= 1
    if (deleteCountdown.value <= 0 && deleteCountdownTimer) {
      clearInterval(deleteCountdownTimer)
      deleteCountdownTimer = null
    }
  }, 1000)
})

onBeforeUnmount(() => {
  clearPendingAvatarPreview()
  if (deleteCountdownTimer) {
    clearInterval(deleteCountdownTimer)
  }
})

function openAvatarPicker() {
  avatarInput.value?.click()
}

function isSupportedAvatarFileType(fileType: string): fileType is (typeof SUPPORTED_AVATAR_FILE_TYPES)[number] {
  return fileType === 'image/jpeg' || fileType === 'image/png' || fileType === 'image/webp'
}

async function handleAvatarChange(event: Event) {
  if (!(event.target instanceof HTMLInputElement)) {
    return
  }

  const file = event.target.files?.[0]
  if (!file) {
    return
  }

  if (!isSupportedAvatarFileType(file.type)) {
    profileError.value = t('settings.account.avatarUnsupportedType')
    event.target.value = ''
    return
  }

  if (file.size > MAX_AVATAR_FILE_SIZE_BYTES) {
    profileError.value = t('settings.account.avatarFileTooLarge')
    event.target.value = ''
    return
  }

  try {
    const compressedAvatar = await imageCompression(file, {
      maxSizeMB: 0.4,
      maxWidthOrHeight: 512,
      useWebWorker: true,
      fileType: file.type,
      initialQuality: 0.8,
    })

    pendingAvatarFile.value = compressedAvatar instanceof File ? compressedAvatar : new File([compressedAvatar], file.name, { type: file.type })
    clearPendingAvatarPreview()
    pendingAvatarPreviewURL.value = URL.createObjectURL(pendingAvatarFile.value)
    profileError.value = ''
  } catch {
    profileError.value = t('settings.account.avatarProcessFailed')
  } finally {
    event.target.value = ''
  }
}

async function saveProfile() {
  const username = editForm.value.username.trim()
  if (!username) {
    profileError.value = t('settings.account.usernameRequired')
    return
  }

  isSavingProfile.value = true
  profileError.value = ''

  try {
    await authStore.saveProfile({
      username,
      email: editForm.value.email.trim() || null,
    })

    if (pendingAvatarFile.value) {
      await authStore.uploadAvatar(pendingAvatarFile.value)
      pendingAvatarFile.value = null
      clearPendingAvatarPreview()
    }

    toast.success(t('settings.account.profileUpdated'))
    editDialogOpen.value = false
  } catch (error) {
    const message = getAPIErrorMessage(t, error, 'apiError.profileUpdateFailed')
    profileError.value = message
    toast.error(message)
  } finally {
    isSavingProfile.value = false
  }
}

async function updatePassword() {
  passwordError.value = ''

  if (passwordForm.value.newPassword !== passwordForm.value.confirmPassword) {
    passwordError.value = t('settings.account.passwordMismatch')
    return
  }

  isUpdatingPassword.value = true

  try {
    await authStore.changePassword({
      currentPassword: passwordForm.value.currentPassword,
      newPassword: passwordForm.value.newPassword,
    })

    passwordForm.value = {
      currentPassword: '',
      newPassword: '',
      confirmPassword: '',
    }

    toast.success(t('settings.account.passwordUpdated'))
  } catch (error) {
    const message = getAPIErrorMessage(t, error, 'apiError.passwordUpdateFailed')
    passwordError.value = message
    toast.error(message)
  } finally {
    isUpdatingPassword.value = false
  }
}

async function confirmDelete() {
  if (!canDeleteAccount.value) {
    toast.error(t('settings.account.superAdminDeleteForbidden'))
    return
  }

  isDeletingAccount.value = true

  try {
    await authStore.deleteAccount()
    toast.success(t('settings.account.deleteAccountSuccess'))
    await router.push({ name: 'login' })
  } catch (error) {
    toast.error(getAPIErrorMessage(t, error, 'apiError.accountDeleteFailed'))
  } finally {
    isDeletingAccount.value = false
    deleteDialogOpen.value = false
  }
}
</script>

<template>
  <div class="flex flex-1 flex-col gap-4 p-4 lg:gap-6 lg:p-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold">{{ t('settings.title') }}</h1>
        <p class="text-muted-foreground text-sm">{{ t('settings.description') }}</p>
      </div>
    </div>

    <Tabs
      default-value="account"
      class="space-y-4"
    >
      <TabsList>
        <TabsTrigger value="account"> {{ t('settings.tabs.account') }} </TabsTrigger>
        <TabsTrigger value="appearance"> {{ t('settings.tabs.appearance') }} </TabsTrigger>
        <TabsTrigger value="notifications"> {{ t('settings.tabs.notifications') }} </TabsTrigger>
      </TabsList>

      <TabsContent
        value="account"
        class="space-y-4"
      >
        <Card
          v-if="authStore.user?.mustChangePassword"
          class="border-amber-500/40 bg-amber-500/5"
        >
          <CardHeader>
            <CardTitle>{{ t('settings.account.mustChangePasswordTitle') }}</CardTitle>
            <CardDescription>{{ t('settings.account.mustChangePasswordDesc') }}</CardDescription>
          </CardHeader>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{{ t('settings.account.profile') }}</CardTitle>
            <CardDescription>{{ t('settings.account.profileDesc') }}</CardDescription>
          </CardHeader>
          <CardContent class="space-y-4">
            <div class="flex items-center justify-between rounded-lg border p-4">
              <div class="flex items-center gap-4">
                <Avatar class="h-12 w-12 rounded-full">
                  <AvatarImage
                    v-if="avatarImageSrc"
                    :src="avatarImageSrc"
                    :alt="authStore.user?.username ?? ''"
                  />
                  <AvatarFallback class="rounded-full">{{ avatarFallbackText }}</AvatarFallback>
                </Avatar>
                <div class="space-y-1">
                  <div class="flex items-center gap-2">
                    <p class="font-medium">{{ authStore.user?.username }}</p>
                    <Badge :variant="roleBadgeVariant">{{ roleLabel }}</Badge>
                  </div>
                  <p class="text-sm text-muted-foreground">{{ authStore.user?.email ?? t('settings.account.emailNotSet') }}</p>
                </div>
              </div>
              <Dialog v-model:open="editDialogOpen">
                <DialogTrigger as-child>
                  <Button
                    variant="outline"
                    size="sm"
                  >
                    {{ t('settings.account.edit') }}
                  </Button>
                </DialogTrigger>
                <DialogContent class="sm:max-w-110">
                  <DialogHeader>
                    <DialogTitle>{{ t('settings.account.editProfile') }}</DialogTitle>
                    <DialogDescription>
                      {{ t('settings.account.editProfileDesc') }}
                    </DialogDescription>
                  </DialogHeader>
                  <div class="grid gap-4 py-4">
                    <div class="flex flex-col items-center gap-2">
                      <Avatar class="h-20 w-20 rounded-full">
                        <AvatarImage
                          v-if="editAvatarImageSrc"
                          :src="editAvatarImageSrc"
                          :alt="editForm.username || authStore.user?.username || ''"
                        />
                        <AvatarFallback class="rounded-full">{{ editAvatarFallbackText }}</AvatarFallback>
                      </Avatar>
                      <input
                        ref="avatarInput"
                        type="file"
                        accept="image/jpeg,image/png,image/webp"
                        class="hidden"
                        @change="handleAvatarChange"
                      />
                      <Button
                        variant="outline"
                        size="sm"
                        type="button"
                        @click="openAvatarPicker"
                      >
                        {{ t('settings.account.changeAvatar') }}
                      </Button>
                      <p class="text-muted-foreground text-xs">{{ t('settings.account.avatarHint') }}</p>
                      <p
                        v-if="profileError"
                        class="text-destructive text-xs"
                      >
                        {{ profileError }}
                      </p>
                    </div>
                    <div class="space-y-2">
                      <Label for="edit-username">{{ t('settings.account.username') }}</Label>
                      <Input
                        id="edit-username"
                        v-model="editForm.username"
                        :placeholder="t('settings.account.usernamePlaceholder')"
                      />
                    </div>
                    <div class="space-y-2">
                      <Label for="edit-email">{{ t('settings.account.email') }}</Label>
                      <Input
                        id="edit-email"
                        v-model="editForm.email"
                        type="email"
                        :placeholder="t('settings.account.emailPlaceholder')"
                      />
                    </div>
                  </div>
                  <DialogFooter>
                    <DialogClose as-child>
                      <Button variant="outline">
                        {{ t('common.action.cancel') }}
                      </Button>
                    </DialogClose>
                    <Button
                      :disabled="isSavingProfile"
                      @click="saveProfile"
                    >
                      {{ isSavingProfile ? t('settings.account.savingProfile') : t('settings.account.saveProfile') }}
                    </Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{{ t('settings.account.password') }}</CardTitle>
            <CardDescription>{{ t('settings.account.passwordDesc') }}</CardDescription>
          </CardHeader>
          <CardContent class="space-y-4">
            <div class="space-y-2">
              <Label for="current-password">{{ t('settings.account.currentPassword') }}</Label>
              <Input
                id="current-password"
                v-model="passwordForm.currentPassword"
                type="password"
                :placeholder="t('settings.account.currentPasswordPlaceholder')"
              />
            </div>
            <div class="grid gap-4 md:grid-cols-2">
              <div class="space-y-2">
                <Label for="new-password">{{ t('settings.account.newPassword') }}</Label>
                <Input
                  id="new-password"
                  v-model="passwordForm.newPassword"
                  type="password"
                  :placeholder="t('settings.account.newPasswordPlaceholder')"
                />
              </div>
              <div class="space-y-2">
                <Label for="confirm-password">{{ t('settings.account.confirmPassword') }}</Label>
                <Input
                  id="confirm-password"
                  v-model="passwordForm.confirmPassword"
                  type="password"
                  :placeholder="t('settings.account.confirmPasswordPlaceholder')"
                />
              </div>
            </div>
            <p
              v-if="passwordError"
              class="text-destructive text-sm"
            >
              {{ passwordError }}
            </p>
          </CardContent>
          <CardFooter class="justify-end">
            <Button
              variant="outline"
              :disabled="isUpdatingPassword"
              @click="updatePassword"
            >
              {{ isUpdatingPassword ? t('settings.account.updatingPassword') : t('settings.account.updatePassword') }}
            </Button>
          </CardFooter>
        </Card>

        <Card class="border-destructive">
          <CardHeader>
            <CardTitle class="text-destructive">{{ t('settings.account.dangerZone') }}</CardTitle>
            <CardDescription>{{ t('settings.account.dangerZoneDesc') }}</CardDescription>
          </CardHeader>
          <CardContent class="space-y-3">
            <div class="flex items-center gap-2">
              <Checkbox
                id="delete-account"
                v-model="deleteAccountConfirmed"
                :disabled="!canDeleteAccount"
              />
              <Label for="delete-account">{{ t('settings.account.dangerZoneConfirm') }}</Label>
            </div>
            <p
              v-if="!canDeleteAccount"
              class="text-sm text-muted-foreground"
            >
              {{ t('settings.account.superAdminDeleteForbidden') }}
            </p>
          </CardContent>
          <CardFooter>
            <AlertDialog v-model:open="deleteDialogOpen">
              <AlertDialogTrigger as-child>
                <Button
                  variant="destructive"
                  :disabled="!deleteAccountConfirmed || !canDeleteAccount"
                >
                  {{ t('settings.account.deleteAccount') }}
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>{{ t('settings.account.deleteAccount') }}</AlertDialogTitle>
                  <AlertDialogDescription>
                    {{ t('settings.account.deleteAccountConfirm') }}
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>{{ t('common.action.cancel') }}</AlertDialogCancel>
                  <AlertDialogAction as-child>
                    <Button
                      variant="destructive"
                      :disabled="deleteCountdown > 0 || isDeletingAccount"
                      @click="deleteCountdown > 0 ? undefined : confirmDelete"
                    >
                      {{ deleteCountdown > 0 ? `${deleteCountdown}s` : t('settings.account.deleteAccount') }}
                    </Button>
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </CardFooter>
        </Card>
      </TabsContent>

      <TabsContent
        value="appearance"
        class="space-y-4"
      >
        <Card>
          <CardHeader>
            <CardTitle>{{ t('settings.appearance.theme') }}</CardTitle>
            <CardDescription>{{ t('settings.appearance.themeDesc') }}</CardDescription>
          </CardHeader>
          <CardContent class="space-y-6">
            <div class="space-y-2">
              <Label>{{ t('settings.appearance.colorTheme') }}</Label>
              <Tabs
                :model-value="themeStore.theme"
                @update:model-value="handleThemeChange"
              >
                <TabsList>
                  <TabsTrigger value="light">
                    {{ t('settings.appearance.light') }}
                  </TabsTrigger>
                  <TabsTrigger value="dark">
                    {{ t('settings.appearance.dark') }}
                  </TabsTrigger>
                  <TabsTrigger value="system">
                    {{ t('settings.appearance.system') }}
                  </TabsTrigger>
                </TabsList>
              </Tabs>
            </div>

            <div class="space-y-2">
              <Label>{{ t('settings.appearance.language') }}</Label>
              <Select
                :model-value="localeStore.locale"
                @update:model-value="handleLocaleChange"
              >
                <SelectTrigger>
                  <SelectValue :placeholder="t('settings.appearance.selectLanguage')" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem
                    v-for="locale in localeOptions"
                    :key="locale.value"
                    :value="locale.value"
                  >
                    {{ locale.label }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent
        value="notifications"
        class="space-y-4"
      >
        <Card>
          <CardHeader>
            <CardTitle>{{ t('settings.notifications.title') }}</CardTitle>
            <CardDescription>{{ t('settings.notifications.desc') }}</CardDescription>
          </CardHeader>
          <CardContent class="space-y-6">
            <div class="flex items-center justify-between">
              <div class="space-y-0.5">
                <Label>{{ t('settings.notifications.email') }}</Label>
                <p class="text-muted-foreground text-sm">{{ t('settings.notifications.emailDesc') }}</p>
              </div>
              <Switch v-model="notifications.emailNotifications" />
            </div>

            <div class="flex items-center justify-between">
              <div class="space-y-0.5">
                <Label>{{ t('settings.notifications.push') }}</Label>
                <p class="text-muted-foreground text-sm">{{ t('settings.notifications.pushDesc') }}</p>
              </div>
              <Switch v-model="notifications.pushNotifications" />
            </div>

            <div class="flex items-center justify-between">
              <div class="space-y-0.5">
                <Label>{{ t('settings.notifications.digest') }}</Label>
                <p class="text-muted-foreground text-sm">{{ t('settings.notifications.digestDesc') }}</p>
              </div>
              <Switch v-model="notifications.weeklyDigest" />
            </div>

            <div class="flex items-center justify-between">
              <div class="space-y-0.5">
                <Label>{{ t('settings.notifications.security') }}</Label>
                <p class="text-muted-foreground text-sm">{{ t('settings.notifications.securityDesc') }}</p>
              </div>
              <Switch v-model="notifications.securityAlerts" />
            </div>
          </CardContent>
          <CardFooter class="justify-end">
            <Button @click="saveSettings">
              {{ saved ? t('settings.saved') : t('settings.save') }}
            </Button>
          </CardFooter>
        </Card>
      </TabsContent>
    </Tabs>
  </div>
</template>
