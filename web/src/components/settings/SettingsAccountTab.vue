<script setup lang="ts">
import type { SUPPORTED_AVATAR_FILE_TYPES } from '@/lib/avatar'
import imageCompression from 'browser-image-compression'
import { computed, onBeforeUnmount, ref, useTemplateRef, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from '@/components/ui/alert-dialog'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { getAPIErrorMessage } from '@/lib/api/error-messages'
import { canDeleteOwnAccount, getUserRoleBadgeVariant, getUserRoleLabelKey } from '@/lib/auth/roles'
import { MAX_AVATAR_FILE_SIZE_BYTES, getAvatarFallbackText } from '@/lib/avatar'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const { t } = useI18n()
const authStore = useAuthStore()

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
const isSavingProfile = ref(false)
const isUpdatingPassword = ref(false)
const isDeletingAccount = ref(false)
const avatarInput = useTemplateRef<HTMLInputElement>('avatarInput')

let deleteCountdownTimer: ReturnType<typeof setInterval> | null = null

const avatarFallbackText = computed(() => getAvatarFallbackText(authStore.user?.username))
const avatarImageSrc = computed(() => authStore.user?.avatarUrl ?? null)
const editAvatarFallbackText = computed(() => {
  const username = editForm.value.username.trim()
  return getAvatarFallbackText(username.length > 0 ? username : authStore.user?.username)
})
const editAvatarImageSrc = computed(() => pendingAvatarPreviewURL.value ?? authStore.user?.avatarUrl ?? null)
const roleLabel = computed(() => t(getUserRoleLabelKey(authStore.user?.role ?? 0)))
const roleBadgeVariant = computed(() => getUserRoleBadgeVariant(authStore.user?.role ?? 0))
const canDeleteAccount = computed(() => canDeleteOwnAccount(authStore.user?.role ?? 0))
const mustChangePassword = computed(() => authStore.user?.mustChangePassword === true)

watch(editDialogOpen, (isOpen) => {
  if (isOpen) {
    resetEditForm()
  }
})

watch(deleteDialogOpen, (isOpen) => {
  if (deleteCountdownTimer !== null) {
    clearInterval(deleteCountdownTimer)
    deleteCountdownTimer = null
  }

  if (!isOpen) {
    deleteCountdown.value = 0
    return
  }

  deleteCountdown.value = 3
  deleteCountdownTimer = window.setInterval(() => {
    deleteCountdown.value -= 1

    if (deleteCountdown.value <= 0 && deleteCountdownTimer !== null) {
      clearInterval(deleteCountdownTimer)
      deleteCountdownTimer = null
    }
  }, 1000)
})

onBeforeUnmount(() => {
  clearPendingAvatarPreview()

  if (deleteCountdownTimer !== null) {
    clearInterval(deleteCountdownTimer)
  }
})

function resetEditForm() {
  editForm.value = {
    username: authStore.user?.username ?? '',
    email: authStore.user?.email ?? '',
  }
  profileError.value = ''
  clearPendingAvatarPreview()
  pendingAvatarFile.value = null

  const input = avatarInput.value
  if (input !== null) {
    input.value = ''
  }
}

function clearPendingAvatarPreview() {
  const previewUrl = pendingAvatarPreviewURL.value
  if (previewUrl === null) {
    return
  }

  URL.revokeObjectURL(previewUrl)
  pendingAvatarPreviewURL.value = null
}

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
  if (file === undefined) {
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
  if (username.length === 0) {
    profileError.value = t('settings.account.usernameRequired')
    return
  }

  isSavingProfile.value = true
  profileError.value = ''

  try {
    const email = editForm.value.email.trim()

    await authStore.saveProfile({
      username,
      email: email.length > 0 ? email : null,
    })

    if (pendingAvatarFile.value !== null) {
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
    await router.push({ name: 'login' })
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
  <div class="space-y-4">
    <Card
      v-if="mustChangePassword"
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
            <Avatar class="size-12 rounded-full">
              <AvatarImage
                v-if="avatarImageSrc !== null"
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
                  <Avatar class="size-20 rounded-full">
                    <AvatarImage
                      v-if="editAvatarImageSrc !== null"
                      :src="editAvatarImageSrc"
                      :alt="editForm.username"
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
                  <p class="text-xs text-muted-foreground">{{ t('settings.account.avatarHint') }}</p>
                  <p
                    v-if="profileError !== ''"
                    class="text-xs text-destructive"
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
          v-if="passwordError !== ''"
          class="text-sm text-destructive"
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
              :disabled="deleteAccountConfirmed === false || canDeleteAccount === false"
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
  </div>
</template>
