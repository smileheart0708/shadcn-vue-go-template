<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { ShieldCheck } from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { Switch } from '@/components/ui/switch'
import { getAPIErrorMessage } from '@/lib/api/error-messages'
import { getAdminSystemSettings, updateSystemSettings, type SystemSettings } from '@/lib/api/system-settings'
import { useAuthStore } from '@/stores/auth'

const { t, locale } = useI18n()
const authStore = useAuthStore()

const loading = ref(true)
const saving = ref(false)
const loadFailed = ref(false)
const settings = ref<SystemSettings | null>(null)
const form = ref<{
  authMode: 'single_user' | 'multi_user'
  registrationMode: 'disabled' | 'public'
  adminUserCreateEnabled: boolean
  selfServiceAccountDeletionEnabled: boolean
}>({
  authMode: 'single_user',
  registrationMode: 'disabled',
  adminUserCreateEnabled: true,
  selfServiceAccountDeletionEnabled: true,
})

const isDirty = computed(() => {
  const currentSettings = settings.value
  if (!currentSettings) {
    return false
  }

  return (
    form.value.authMode !== currentSettings.authMode ||
    form.value.registrationMode !== currentSettings.registrationMode ||
    form.value.adminUserCreateEnabled !== currentSettings.adminUserCreateEnabled ||
    form.value.selfServiceAccountDeletionEnabled !== currentSettings.selfServiceAccountDeletionEnabled
  )
})

const updatedAtLabel = computed(() => {
  if (settings.value === null) {
    return null
  }

  return new Intl.DateTimeFormat(locale.value, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(settings.value.updatedAt))
})

const publicRegistrationEffective = computed(() => form.value.authMode === 'multi_user' && form.value.registrationMode === 'public')

onMounted(() => {
  void loadSettings()
})

async function loadSettings() {
  loading.value = true

  try {
    const nextSettings = await getAdminSystemSettings()
    settings.value = nextSettings
    form.value = {
      authMode: nextSettings.authMode,
      registrationMode: nextSettings.registrationMode,
      adminUserCreateEnabled: nextSettings.adminUserCreateEnabled,
      selfServiceAccountDeletionEnabled: nextSettings.selfServiceAccountDeletionEnabled,
    }
    loadFailed.value = false
  } catch (error) {
    loadFailed.value = true
    toast.error(getAPIErrorMessage(t, error, 'systemConfig.feedback.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  if (!isDirty.value || saving.value) {
    return
  }

  saving.value = true
  const savePromise = updateSystemSettings({
    authMode: form.value.authMode,
    registrationMode: form.value.registrationMode,
    adminUserCreateEnabled: form.value.adminUserCreateEnabled,
    selfServiceAccountDeletionEnabled: form.value.selfServiceAccountDeletionEnabled,
  })

  toast.promise(savePromise, {
    loading: t('systemConfig.feedback.saving'),
    success: () => t('systemConfig.feedback.saved'),
    error: (error: unknown) => getAPIErrorMessage(t, error, 'systemConfig.feedback.saveFailed'),
  })

  try {
    settings.value = await savePromise
    form.value = {
      authMode: settings.value.authMode,
      registrationMode: settings.value.registrationMode,
      adminUserCreateEnabled: settings.value.adminUserCreateEnabled,
      selfServiceAccountDeletionEnabled: settings.value.selfServiceAccountDeletionEnabled,
    }
    await authStore.refreshPublicState()
    // 系统设置会直接影响当前 viewer 的权限集合，保存后刷新一次，避免侧边栏和路由守卫继续使用旧权限。
    try {
      await authStore.refreshViewer({ backgroundRequest: true })
    } catch (error: unknown) {
      console.warn('Failed to refresh viewer after saving system settings', error)
    }
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="flex flex-1 flex-col gap-6 p-4 lg:p-6">
    <section class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
      <div class="space-y-2">
        <div class="flex items-center gap-2">
          <h1 class="text-2xl font-semibold">{{ t('systemConfig.title') }}</h1>
          <Badge variant="outline">{{ t('systemConfig.badge') }}</Badge>
        </div>
        <p class="text-sm text-muted-foreground">{{ t('systemConfig.description') }}</p>
      </div>
      <div class="flex size-11 items-center justify-center rounded-xl border bg-muted/40">
        <ShieldCheck class="size-5" />
      </div>
    </section>

    <div
      v-if="loading"
      class="grid gap-4 xl:grid-cols-[minmax(0,1.3fr)_minmax(280px,0.7fr)]"
    >
      <Card>
        <CardHeader>
          <Skeleton class="h-6 w-40 rounded-md" />
          <Skeleton class="h-4 w-full rounded-md" />
        </CardHeader>
        <CardContent class="space-y-3">
          <Skeleton class="h-18 rounded-xl" />
          <Skeleton class="h-18 rounded-xl" />
          <Skeleton class="h-18 rounded-xl" />
          <Skeleton class="h-18 rounded-xl" />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <Skeleton class="h-6 w-32 rounded-md" />
          <Skeleton class="h-4 w-full rounded-md" />
        </CardHeader>
        <CardContent class="space-y-3">
          <Skeleton class="h-10 w-40 rounded-md" />
        </CardContent>
      </Card>
    </div>

    <div
      v-else-if="loadFailed"
      class="rounded-xl border bg-card p-4"
    >
      <Empty>
        <EmptyHeader>
          <EmptyTitle>{{ t('systemConfig.feedback.loadFailedTitle') }}</EmptyTitle>
          <EmptyDescription>{{ t('systemConfig.feedback.loadFailed') }}</EmptyDescription>
        </EmptyHeader>
        <EmptyContent>
          <Button @click="loadSettings">{{ t('systemConfig.actions.retry') }}</Button>
        </EmptyContent>
      </Empty>
    </div>

    <div
      v-else
      class="grid gap-4 xl:grid-cols-[minmax(0,1.3fr)_minmax(280px,0.7fr)]"
    >
      <Card>
        <CardHeader>
          <CardTitle>{{ t('systemConfig.cards.auth.title') }}</CardTitle>
          <CardDescription>{{ t('systemConfig.cards.auth.description') }}</CardDescription>
        </CardHeader>
        <CardContent class="space-y-6">
          <div class="grid gap-3 md:grid-cols-2">
            <div class="space-y-2">
              <div class="text-sm font-medium">{{ t('systemConfig.fields.authMode.title') }}</div>
              <p class="text-sm text-muted-foreground">{{ t('systemConfig.fields.authMode.description') }}</p>
              <Select v-model="form.authMode">
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="single_user">{{ t('systemConfig.options.authMode.singleUser') }}</SelectItem>
                  <SelectItem value="multi_user">{{ t('systemConfig.options.authMode.multiUser') }}</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div class="space-y-2">
              <div class="text-sm font-medium">{{ t('systemConfig.fields.registrationMode.title') }}</div>
              <p class="text-sm text-muted-foreground">{{ t('systemConfig.fields.registrationMode.description') }}</p>
              <Select v-model="form.registrationMode">
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="disabled">{{ t('systemConfig.options.registrationMode.disabled') }}</SelectItem>
                  <SelectItem value="public">{{ t('systemConfig.options.registrationMode.public') }}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div class="grid gap-4 rounded-xl border p-4 md:grid-cols-2">
            <label class="flex items-center justify-between gap-4">
              <div>
                <div class="font-medium">{{ t('systemConfig.fields.adminUserCreateEnabled.title') }}</div>
                <p class="text-sm text-muted-foreground">{{ t('systemConfig.fields.adminUserCreateEnabled.description') }}</p>
              </div>
              <Switch v-model="form.adminUserCreateEnabled" />
            </label>

            <label class="flex items-center justify-between gap-4">
              <div>
                <div class="font-medium">{{ t('systemConfig.fields.selfServiceAccountDeletionEnabled.title') }}</div>
                <p class="text-sm text-muted-foreground">{{ t('systemConfig.fields.selfServiceAccountDeletionEnabled.description') }}</p>
              </div>
              <Switch v-model="form.selfServiceAccountDeletionEnabled" />
            </label>
          </div>

          <div class="rounded-xl border p-4">
            <div class="font-medium">{{ t('systemConfig.fields.passwordLoginEnabled.title') }}</div>
            <p class="text-sm text-muted-foreground">{{ t('systemConfig.fields.passwordLoginEnabled.description') }}</p>
            <Badge
              class="mbs-3"
              variant="outline"
            >
              {{ settings?.passwordLoginEnabled ? t('common.state.enabled') : t('common.state.disabled') }}
            </Badge>
          </div>

          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p
              v-if="updatedAtLabel"
              class="text-sm text-muted-foreground"
            >
              {{ t('systemConfig.updatedAt', { value: updatedAtLabel }) }}
            </p>

            <div class="flex items-center gap-2">
              <Button
                variant="outline"
                :disabled="saving || !isDirty"
                @click="loadSettings"
              >
                {{ t('common.action.cancel') }}
              </Button>
              <Button
                :disabled="saving || !isDirty"
                @click="saveSettings"
              >
                <Spinner
                  v-if="saving"
                  class="me-2"
                />
                {{ t('common.action.save') }}
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{{ t('systemConfig.cards.effectivePolicy.title') }}</CardTitle>
          <CardDescription>{{ t('systemConfig.cards.effectivePolicy.description') }}</CardDescription>
        </CardHeader>
        <CardContent class="space-y-3">
          <Badge variant="outline">{{ t('systemConfig.policy.authMode', { value: t(`systemConfig.options.authMode.${form.authMode === 'single_user' ? 'singleUser' : 'multiUser'}`) }) }}</Badge>
          <Badge variant="outline">
            {{ t('systemConfig.policy.registrationMode', { value: t(`systemConfig.options.registrationMode.${form.registrationMode}`) }) }}
          </Badge>
          <Badge variant="outline">
            {{ t('systemConfig.policy.publicRegistration', { value: publicRegistrationEffective ? t('common.state.enabled') : t('common.state.disabled') }) }}
          </Badge>
          <Badge variant="outline">
            {{ t('systemConfig.policy.adminUserCreate', { value: form.adminUserCreateEnabled ? t('common.state.enabled') : t('common.state.disabled') }) }}
          </Badge>
          <Badge variant="outline">
            {{ t('systemConfig.policy.selfServiceAccountDeletion', { value: form.selfServiceAccountDeletionEnabled ? t('common.state.enabled') : t('common.state.disabled') }) }}
          </Badge>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
