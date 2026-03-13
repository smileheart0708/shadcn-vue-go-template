<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { APIError } from '@/lib/api/client'
import { useAuthStore } from '@/stores/auth'

import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'

const { t } = useI18n()

const props = defineProps<{ class?: HTMLAttributes['class'] }>()

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const email = ref('')
const password = ref('')
const isSubmitting = ref(false)

async function handleSubmit() {
  if (isSubmitting.value) {
    return
  }

  isSubmitting.value = true

  try {
    await authStore.login({ email: email.value, password: password.value })

    const redirectTarget = typeof route.query.redirect === 'string' && route.query.redirect.startsWith('/') ? route.query.redirect : { name: 'dashboard' as const }

    await router.push(redirectTarget)
  } catch (error) {
    const message = error instanceof APIError ? error.message : t('auth.signIn.loginFailed')
    toast.error(message)
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <div :class="cn('flex flex-col gap-6', props.class)">
    <form @submit.prevent="handleSubmit">
      <FieldGroup>
        <div class="flex flex-col items-center gap-2 text-center">
          <img
            src="/logo.svg"
            alt="Logo"
            class="size-8"
          />
          <h1 class="text-xl font-bold">{{ t('auth.signIn.title') }}</h1>
          <p class="text-muted-foreground text-sm">
            {{ t('auth.signIn.description') }}
          </p>
        </div>
        <Field>
          <FieldLabel for="email"> {{ t('common.field.email') }} </FieldLabel>
          <Input
            id="email"
            v-model="email"
            type="email"
            :placeholder="t('auth.signIn.emailPlaceholder')"
            autocomplete="username"
            required
          />
        </Field>
        <Field>
          <FieldLabel for="password"> {{ t('common.field.password') }} </FieldLabel>
          <Input
            id="password"
            v-model="password"
            type="password"
            :placeholder="t('auth.signIn.passwordPlaceholder')"
            autocomplete="current-password"
            required
          />
        </Field>
        <Field>
          <Button
            type="submit"
            :disabled="isSubmitting"
          >
            {{ isSubmitting ? t('auth.signIn.signingIn') : t('auth.signIn.submit') }}
          </Button>
        </Field>
      </FieldGroup>
    </form>
  </div>
</template>
