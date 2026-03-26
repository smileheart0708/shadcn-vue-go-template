<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Github } from 'lucide-vue-next'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Field, FieldDescription, FieldGroup, FieldLabel, FieldSeparator } from '@/components/ui/field'
import { Input } from '@/components/ui/input'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()

const { t } = useI18n()
const router = useRouter()

const name = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const isSubmitting = ref(false)

function handleSubmit() {
  if (isSubmitting.value) {
    return
  }

  if (password.value !== confirmPassword.value) {
    return
  }

  isSubmitting.value = true

  setTimeout(() => {
    isSubmitting.value = false
    void router.push('/login')
  }, 1000)
}
</script>

<template>
  <div :class="cn('flex flex-col gap-6', props.class)">
    <form @submit.prevent="handleSubmit">
      <FieldGroup>
        <div class="flex flex-col items-center gap-2 text-center">
          <h1 class="text-xl font-bold">{{ t('auth.signUp.title') }}</h1>
          <FieldDescription>
            {{ t('auth.signUp.description') }}
            <router-link
              to="/login"
              class="font-medium hover:underline"
            >
              {{ t('auth.signUp.signIn') }}
            </router-link>
          </FieldDescription>
        </div>
        <Field>
          <FieldLabel for="username"> {{ t('common.field.username') }} </FieldLabel>
          <Input
            id="username"
            v-model="name"
            type="text"
            :placeholder="t('auth.signUp.usernamePlaceholder')"
            autocomplete="username"
            required
          />
        </Field>
        <Field>
          <FieldLabel for="email"> {{ t('common.field.email') }} </FieldLabel>
          <Input
            id="email"
            v-model="email"
            type="email"
            :placeholder="t('auth.signUp.emailPlaceholder')"
            autocomplete="email"
            required
          />
        </Field>
        <Field>
          <FieldLabel for="password"> {{ t('common.field.password') }} </FieldLabel>
          <Input
            id="password"
            v-model="password"
            type="password"
            :placeholder="t('auth.signUp.passwordPlaceholder')"
            autocomplete="new-password"
            required
          />
        </Field>
        <Field>
          <FieldLabel for="confirmPassword"> {{ t('auth.signUp.confirmPassword') }} </FieldLabel>
          <Input
            id="confirmPassword"
            v-model="confirmPassword"
            type="password"
            :placeholder="t('auth.signUp.confirmPasswordPlaceholder')"
            autocomplete="new-password"
            required
          />
        </Field>
        <Field>
          <Button
            type="submit"
            :disabled="isSubmitting"
          >
            {{ isSubmitting ? t('auth.signUp.creating') : t('auth.signUp.submit') }}
          </Button>
        </Field>
        <FieldSeparator>{{ t('auth.signUp.or') }}</FieldSeparator>
        <Field>
          <Button
            variant="outline"
            type="button"
          >
            <Github class="size-4 mr-2" />
            {{ t('auth.signUp.continueWithGithub') }}
          </Button>
        </Field>
      </FieldGroup>
    </form>
    <FieldDescription class="px-6 text-center">
      {{ t('auth.signUp.termsPrefix') }}
      <a
        href="#"
        class="font-medium hover:underline"
        >{{ t('auth.signUp.terms') }}</a
      >
      {{ t('auth.signUp.termsAnd') }}
      <a
        href="#"
        class="font-medium hover:underline"
        >{{ t('auth.signUp.privacy') }}</a
      >
    </FieldDescription>
  </div>
</template>
