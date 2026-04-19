<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Field, FieldDescription, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { getAPIErrorMessage } from '@/lib/api/error-messages'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()

const username = ref('owner')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const submitting = ref(false)
const setupCompleted = computed(() => authStore.installState?.setupCompleted === true)

async function submit() {
  if (submitting.value || setupCompleted.value) {
    return
  }
  if (password.value !== confirmPassword.value) {
    toast.error(t('auth.signUp.passwordMismatch'))
    return
  }

  submitting.value = true
  const setupPromise = authStore.completeSetup({
    username: username.value.trim(),
    email: email.value.trim() === '' ? null : email.value.trim(),
    password: password.value,
  })

  toast.promise(setupPromise, {
    loading: t('setup.creating'),
    success: () => t('setup.success'),
    error: (error: unknown) => getAPIErrorMessage(t, error, 'setup.failed'),
  })

  try {
    await setupPromise
    await router.push({ name: 'dashboard' })
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="grid min-h-screen place-items-center bg-linear-to-br from-slate-50 via-white to-stone-100 p-6 dark:from-slate-950 dark:via-slate-950 dark:to-slate-900">
    <Card class="w-full max-w-xl border-border/70 shadow-lg">
      <CardHeader class="space-y-2">
        <CardTitle class="text-2xl">{{ t('setup.title') }}</CardTitle>
        <CardDescription>{{ t('setup.description') }}</CardDescription>
      </CardHeader>
      <CardContent>
        <form @submit.prevent="submit">
          <FieldGroup>
            <Field>
              <FieldLabel for="setup-username">{{ t('common.field.username') }}</FieldLabel>
              <Input
                id="setup-username"
                v-model="username"
                autocomplete="username"
                required
              />
            </Field>
            <Field>
              <FieldLabel for="setup-email">{{ t('common.field.email') }}</FieldLabel>
              <Input
                id="setup-email"
                v-model="email"
                autocomplete="email"
                type="email"
              />
              <FieldDescription>{{ t('setup.emailHint') }}</FieldDescription>
            </Field>
            <Field>
              <FieldLabel for="setup-password">{{ t('common.field.password') }}</FieldLabel>
              <Input
                id="setup-password"
                v-model="password"
                autocomplete="new-password"
                minlength="8"
                required
                type="password"
              />
            </Field>
            <Field>
              <FieldLabel for="setup-confirm-password">{{ t('common.field.confirmPassword') }}</FieldLabel>
              <Input
                id="setup-confirm-password"
                v-model="confirmPassword"
                autocomplete="new-password"
                minlength="8"
                required
                type="password"
              />
            </Field>
            <Field>
              <Button
                type="submit"
                :disabled="submitting || setupCompleted"
              >
                {{ submitting ? t('setup.creating') : t('setup.submit') }}
              </Button>
            </Field>
          </FieldGroup>
        </form>
      </CardContent>
    </Card>
  </div>
</template>
