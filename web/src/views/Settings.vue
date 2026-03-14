<script setup lang="ts">
import imageCompression from 'browser-image-compression'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
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
import { MAX_AVATAR_FILE_SIZE_BYTES, getAvatarFallbackText } from '@/lib/avatar'
import { localeNames, type AppLocale } from '@/plugins/i18n/locales'

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

const privacy = ref({
  publicProfile: false,
  allowSearch: true,
  showActivity: true,
})

const saved = ref(false)
const deleteDialogOpen = ref(false)
const deleteCountdown = ref(0)
const deleteAccountConfirmed = ref(false)

const editDialogOpen = ref(false)
const editForm = ref<{ name: string; avatar: string | null }>({
  name: authStore.user?.name ?? '',
  avatar: authStore.user?.avatar ?? null,
})
const profileError = ref('')
const avatarInput = ref<HTMLInputElement | null>(null)

function saveSettings() {
  saved.value = true
  setTimeout(() => {
    saved.value = false
  }, 2000)
}

function startDeleteCountdown() {
  deleteCountdown.value = 3
  const timer = setInterval(() => {
    deleteCountdown.value--
    if (deleteCountdown.value <= 0) {
      clearInterval(timer)
    }
  }, 1000)
}

function confirmDelete() {
  console.log('Account deleted')
  deleteDialogOpen.value = false
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

const avatarFallbackText = computed(() => getAvatarFallbackText(authStore.user?.name))
const avatarImageSrc = computed(() => authStore.user?.avatar ?? null)
const editAvatarFallbackText = computed(() => getAvatarFallbackText(editForm.value.name || authStore.user?.name))
const editAvatarImageSrc = computed(() => editForm.value.avatar ?? null)

function isSupportedAvatarFileType(fileType: string): boolean {
  return fileType === 'image/jpeg' || fileType === 'image/png' || fileType === 'image/webp'
}

function resetEditForm() {
  editForm.value = {
    name: authStore.user?.name ?? '',
    avatar: authStore.user?.avatar ?? null,
  }
  profileError.value = ''
  if (avatarInput.value) {
    avatarInput.value.value = ''
  }
}

watch(editDialogOpen, (isOpen) => {
  if (isOpen) {
    resetEditForm()
  }
})

function openAvatarPicker() {
  avatarInput.value?.click()
}

async function handleAvatarChange(event: Event) {
  if (!(event.target instanceof HTMLInputElement)) {
    return
  }

  const input = event.target
  const file = input.files?.[0]

  if (!file) {
    return
  }

  if (!isSupportedAvatarFileType(file.type)) {
    profileError.value = t('settings.account.avatarUnsupportedType')
    input.value = ''
    return
  }

  if (file.size > MAX_AVATAR_FILE_SIZE_BYTES) {
    profileError.value = t('settings.account.avatarFileTooLarge')
    input.value = ''
    return
  }

  try {
    const compressedFile = await imageCompression(file, {
      maxSizeMB: 0.4,
      maxWidthOrHeight: 512,
      useWebWorker: true,
      fileType: file.type,
      initialQuality: 0.8,
    })

    editForm.value.avatar = await imageCompression.getDataUrlFromFile(compressedFile)
    profileError.value = ''
  } catch {
    profileError.value = t('settings.account.avatarProcessFailed')
  } finally {
    input.value = ''
  }
}

function saveProfile() {
  const nextName = editForm.value.name.trim()

  if (!nextName) {
    profileError.value = t('settings.account.nameRequired')
    return
  }

  authStore.updateProfile({
    name: nextName,
    avatar: editForm.value.avatar,
  })

  editDialogOpen.value = false
  saveSettings()
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
                    :alt="authStore.user?.name ?? ''"
                  />
                  <AvatarFallback class="rounded-full">{{ avatarFallbackText }}</AvatarFallback>
                </Avatar>
                <div>
                  <p class="font-medium">{{ authStore.user?.name }}</p>
                  <p class="text-sm text-muted-foreground">{{ authStore.user?.email }}</p>
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
                <DialogContent class="sm:max-w-100">
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
                          :alt="editForm.name || authStore.user?.name || ''"
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
                      <Label for="edit-name">{{ t('settings.account.name') }}</Label>
                      <Input
                        id="edit-name"
                        v-model="editForm.name"
                        :placeholder="t('settings.account.namePlaceholder')"
                      />
                    </div>
                  </div>
                  <DialogFooter>
                    <DialogClose as-child>
                      <Button variant="outline">
                        {{ t('common.action.cancel') }}
                      </Button>
                    </DialogClose>
                    <Button @click="saveProfile">
                      {{ t('settings.save') }}
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
                type="password"
                :placeholder="t('settings.account.currentPasswordPlaceholder')"
              />
            </div>
            <div class="grid gap-4 md:grid-cols-2">
              <div class="space-y-2">
                <Label for="new-password">{{ t('settings.account.newPassword') }}</Label>
                <Input
                  id="new-password"
                  type="password"
                  :placeholder="t('settings.account.newPasswordPlaceholder')"
                />
              </div>
              <div class="space-y-2">
                <Label for="confirm-password">{{ t('settings.account.confirmPassword') }}</Label>
                <Input
                  id="confirm-password"
                  type="password"
                  :placeholder="t('settings.account.confirmPasswordPlaceholder')"
                />
              </div>
            </div>
          </CardContent>
          <CardFooter class="justify-end">
            <Button variant="outline">{{ t('settings.account.updatePassword') }}</Button>
          </CardFooter>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{{ t('settings.account.privacy') }}</CardTitle>
            <CardDescription>{{ t('settings.account.privacyDesc') }}</CardDescription>
          </CardHeader>
          <CardContent class="space-y-6">
            <div class="flex items-center justify-between">
              <div class="space-y-0.5">
                <Label>{{ t('settings.account.publicProfile') }}</Label>
                <p class="text-muted-foreground text-sm">{{ t('settings.account.publicProfileDesc') }}</p>
              </div>
              <Switch v-model="privacy.publicProfile" />
            </div>

            <div class="flex items-center justify-between">
              <div class="space-y-0.5">
                <Label>{{ t('settings.account.allowSearch') }}</Label>
                <p class="text-muted-foreground text-sm">{{ t('settings.account.allowSearchDesc') }}</p>
              </div>
              <Switch v-model="privacy.allowSearch" />
            </div>

            <div class="flex items-center justify-between">
              <div class="space-y-0.5">
                <Label>{{ t('settings.account.showActivity') }}</Label>
                <p class="text-muted-foreground text-sm">{{ t('settings.account.showActivityDesc') }}</p>
              </div>
              <Switch v-model="privacy.showActivity" />
            </div>
          </CardContent>
          <CardFooter class="justify-end">
            <Button @click="saveSettings">
              {{ saved ? t('settings.saved') : t('settings.save') }}
            </Button>
          </CardFooter>
        </Card>

        <Card class="border-destructive">
          <CardHeader>
            <CardTitle class="text-destructive">{{ t('settings.account.dangerZone') }}</CardTitle>
            <CardDescription>{{ t('settings.account.dangerZoneDesc') }}</CardDescription>
          </CardHeader>
          <CardContent>
            <div class="flex items-center gap-2">
              <Checkbox
                id="delete-account"
                v-model="deleteAccountConfirmed"
              />
              <Label for="delete-account">{{ t('settings.account.dangerZoneConfirm') }}</Label>
            </div>
          </CardContent>
          <CardFooter>
            <AlertDialog v-model:open="deleteDialogOpen">
              <AlertDialogTrigger as-child>
                <Button
                  variant="destructive"
                  :disabled="!deleteAccountConfirmed"
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
                      :disabled="deleteCountdown > 0"
                      @click="deleteCountdown > 0 ? undefined : confirmDelete"
                      @vue:mounted="startDeleteCountdown"
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
