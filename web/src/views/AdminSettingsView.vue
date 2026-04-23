<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { Switch } from '@/components/ui/switch'
import { getAPIErrorMessage } from '@/lib/api/error-messages'
import { getAdminSystemSettings, updateSystemSettings, type SystemSettings } from '@/lib/api/system-settings'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const authStore = useAuthStore()

const loading = ref(true)
const saving = ref(false)
const loadFailed = ref(false)
const settings = ref<SystemSettings | null>(null)
const form = ref<SystemSettings>({
  publicRegistrationEnabled: false,
  selfServiceAccountDeletionEnabled: false,
})

const isDirty = computed(() => {
  const currentSettings = settings.value
  if (!currentSettings) {
    return false
  }

  return (
    form.value.publicRegistrationEnabled !== currentSettings.publicRegistrationEnabled ||
    form.value.selfServiceAccountDeletionEnabled !== currentSettings.selfServiceAccountDeletionEnabled
  )
})

onMounted(() => {
  void loadSettings()
})

async function loadSettings() {
  loading.value = true

  try {
    const nextSettings = await getAdminSystemSettings()
    settings.value = nextSettings
    form.value = { ...nextSettings }
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
    publicRegistrationEnabled: form.value.publicRegistrationEnabled,
    selfServiceAccountDeletionEnabled: form.value.selfServiceAccountDeletionEnabled,
  })

  toast.promise(savePromise, {
    loading: t('systemConfig.feedback.saving'),
    success: () => t('systemConfig.feedback.saved'),
    error: (error: unknown) => getAPIErrorMessage(t, error, 'systemConfig.feedback.saveFailed'),
  })

  try {
    settings.value = await savePromise
    form.value = { ...settings.value }
    await authStore.refreshPublicState()
    try {
      await authStore.refreshViewer({ backgroundRequest: true })
    } catch (error: unknown) {
      console.warn('Failed to refresh viewer after saving account policies', error)
    }
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="flex flex-1 flex-col gap-6 p-4 lg:p-6">
    <section class="space-y-1">
      <h1 class="text-2xl font-semibold">{{ t('systemConfig.title') }}</h1>
      <p class="text-sm text-muted-foreground">{{ t('systemConfig.description') }}</p>
    </section>

    <Card v-if="loading">
      <CardHeader>
        <Skeleton class="h-6 w-32 rounded-md" />
      </CardHeader>
      <CardContent class="space-y-4">
        <Skeleton class="h-16 rounded-lg" />
        <Skeleton class="h-16 rounded-lg" />
        <div class="flex justify-end gap-2">
          <Skeleton class="h-10 w-20 rounded-md" />
          <Skeleton class="h-10 w-20 rounded-md" />
        </div>
      </CardContent>
    </Card>

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

    <Card v-else>
      <CardHeader>
        <CardTitle>{{ t('systemConfig.cardTitle') }}</CardTitle>
        <CardDescription>{{ t('systemConfig.cardDescription') }}</CardDescription>
      </CardHeader>
      <CardContent class="space-y-4">
        <label class="flex items-center justify-between gap-4 rounded-lg border p-4">
          <div class="space-y-1">
            <div class="font-medium">{{ t('systemConfig.fields.publicRegistrationEnabled.title') }}</div>
            <p class="text-sm text-muted-foreground">{{ t('systemConfig.fields.publicRegistrationEnabled.description') }}</p>
          </div>
          <Switch v-model="form.publicRegistrationEnabled" />
        </label>

        <label class="flex items-center justify-between gap-4 rounded-lg border p-4">
          <div class="space-y-1">
            <div class="font-medium">{{ t('systemConfig.fields.selfServiceAccountDeletionEnabled.title') }}</div>
            <p class="text-sm text-muted-foreground">{{ t('systemConfig.fields.selfServiceAccountDeletionEnabled.description') }}</p>
          </div>
          <Switch v-model="form.selfServiceAccountDeletionEnabled" />
        </label>

        <div class="flex justify-end gap-2">
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
      </CardContent>
    </Card>
  </div>
</template>
