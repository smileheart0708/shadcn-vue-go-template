<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { RouterLink } from 'vue-router'
import { ShieldCheck } from 'lucide-vue-next'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { getAPIErrorMessage } from '@/lib/api/error-messages'
import { getAdminSystemSettings, updateRegistrationMode } from '@/lib/api/system-settings'

const { t, locale } = useI18n()

const loading = ref(true)
const saving = ref(false)
const loadFailed = ref(false)
const registrationMode = ref<'disabled' | 'password'>('disabled')
const savedRegistrationMode = ref<'disabled' | 'password'>('disabled')
const updatedAt = ref<string | null>(null)

const isDirty = computed(() => registrationMode.value !== savedRegistrationMode.value)
const updatedAtLabel = computed(() => {
  if (!updatedAt.value) {
    return null
  }

  return new Intl.DateTimeFormat(locale.value, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(updatedAt.value))
})

onMounted(() => {
  void loadSettings()
})

async function loadSettings() {
  loading.value = true

  try {
    const settings = await getAdminSystemSettings()
    registrationMode.value = settings.registrationMode
    savedRegistrationMode.value = settings.registrationMode
    updatedAt.value = settings.updatedAt
    loadFailed.value = false
  } catch (error) {
    loadFailed.value = true
    toast.error(getAPIErrorMessage(t, error, 'systemConfig.feedback.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function saveRegistrationMode() {
  if (!isDirty.value || saving.value) {
    return
  }

  saving.value = true

  const savePromise = updateRegistrationMode(registrationMode.value)
  toast.promise(savePromise, {
    loading: t('systemConfig.feedback.saving'),
    success: () => t('systemConfig.feedback.saved'),
    error: (error: unknown) => getAPIErrorMessage(t, error, 'systemConfig.feedback.saveFailed'),
  })

  try {
    const settings = await savePromise
    registrationMode.value = settings.registrationMode
    savedRegistrationMode.value = settings.registrationMode
    updatedAt.value = settings.updatedAt
  } catch {
    return
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
          <Skeleton class="h-10 w-36 rounded-md" />
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
          <CardTitle>{{ t('systemConfig.registration.title') }}</CardTitle>
          <CardDescription>{{ t('systemConfig.registration.description') }}</CardDescription>
        </CardHeader>
        <CardContent class="space-y-6">
          <RadioGroup
            v-model="registrationMode"
            class="gap-3"
          >
            <label class="flex cursor-pointer items-start gap-3 rounded-xl border p-4 transition-colors hover:bg-muted/40">
              <RadioGroupItem
                value="disabled"
                class="mbs-1"
              />
              <div class="space-y-1">
                <div class="font-medium">{{ t('systemConfig.registration.options.disabled.title') }}</div>
                <p class="text-sm text-muted-foreground">{{ t('systemConfig.registration.options.disabled.description') }}</p>
              </div>
            </label>

            <label class="flex cursor-pointer items-start gap-3 rounded-xl border p-4 transition-colors hover:bg-muted/40">
              <RadioGroupItem
                value="password"
                class="mbs-1"
              />
              <div class="space-y-1">
                <div class="font-medium">{{ t('systemConfig.registration.options.password.title') }}</div>
                <p class="text-sm text-muted-foreground">{{ t('systemConfig.registration.options.password.description') }}</p>
              </div>
            </label>
          </RadioGroup>

          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p
              v-if="updatedAtLabel"
              class="text-sm text-muted-foreground"
            >
              {{ t('systemConfig.registration.updatedAt', { value: updatedAtLabel }) }}
            </p>
            <div class="flex items-center gap-2">
              <Button
                variant="outline"
                :disabled="saving || !isDirty"
                @click="registrationMode = savedRegistrationMode"
              >
                {{ t('common.action.cancel') }}
              </Button>
              <Button
                :disabled="saving || !isDirty"
                @click="saveRegistrationMode"
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
          <CardTitle>{{ t('systemConfig.observability.title') }}</CardTitle>
          <CardDescription>{{ t('systemConfig.observability.description') }}</CardDescription>
        </CardHeader>
        <CardContent class="flex h-full items-end">
          <Button as-child>
            <RouterLink :to="{ name: 'system-logs' }">
              {{ t('systemConfig.observability.cta') }}
            </RouterLink>
          </Button>
        </CardContent>
      </Card>
    </div>
  </div>
</template>
