<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/auth/useAuth'
import { APIError } from '@/lib/api/client'

import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()

const router = useRouter()
const route = useRoute()
const auth = useAuth()

const email = ref('')
const password = ref('')
const errorMessage = ref('')
const isSubmitting = ref(false)

async function handleSubmit() {
  if (isSubmitting.value) {
    return
  }

  errorMessage.value = ''
  isSubmitting.value = true

  try {
    await auth.login({
      email: email.value,
      password: password.value,
    })

    const redirectTarget
      = typeof route.query.redirect === 'string' && route.query.redirect.startsWith('/')
        ? route.query.redirect
        : { name: 'dashboard' as const }

    await router.push(redirectTarget)
  }
  catch (error) {
    errorMessage.value = error instanceof APIError
      ? error.message
      : 'Login failed. Please try again.'
  }
  finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <div :class="cn('flex flex-col gap-6', props.class)">
    <form @submit.prevent="handleSubmit">
      <FieldGroup>
        <div class="flex flex-col items-center gap-2 text-center">
          <img src="/logo.svg" alt="Logo" class="size-8" />
          <h1 class="text-xl font-bold">
            Welcome back
          </h1>
          <p class="text-muted-foreground text-sm">
            Sign in with the account configured on the Go server.
          </p>
        </div>
        <Field>
          <FieldLabel for="email">
            Email
          </FieldLabel>
          <Input
            id="email"
            v-model="email"
            type="email"
            placeholder="m@example.com"
            autocomplete="username"
            required
          />
        </Field>
        <Field>
          <FieldLabel for="password">
            Password
          </FieldLabel>
          <Input
            id="password"
            v-model="password"
            type="password"
            placeholder="Enter your password"
            autocomplete="current-password"
            required
          />
        </Field>
        <Field v-if="errorMessage">
          <FieldError :errors="[errorMessage]" />
        </Field>
        <Field>
          <Button type="submit" :disabled="isSubmitting">
            {{ isSubmitting ? 'Signing in...' : 'Login' }}
          </Button>
        </Field>
      </FieldGroup>
    </form>
  </div>
</template>
