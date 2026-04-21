<script setup lang="ts">
import type { Component } from 'vue'
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { Check, Rocket, User } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
import { Stepper, StepperIndicator, StepperItem, StepperSeparator, StepperTitle, StepperTrigger } from '@/components/ui/stepper'
import { getAPIErrorMessage } from '@/lib/api/error-messages'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()

type TransitionDirection = 'forward' | 'backward'

interface SetupStep {
  step: number
  titleKey: 'setup.step1' | 'setup.step2'
  icon: Component
}

const steps: SetupStep[] = [
  { step: 1, titleKey: 'setup.step1', icon: User },
  { step: 2, titleKey: 'setup.step2', icon: Check },
]

const totalSteps = steps.length

const username = ref('admin')
const password = ref('')
const confirmPassword = ref('')
const submitting = ref(false)
const setupCompleted = computed(() => authStore.installState?.setupCompleted === true)
const currentStep = ref(1)
const transitionDirection = ref<TransitionDirection>('forward')

const transitionClasses = computed(() => {
  if (transitionDirection.value === 'backward') {
    return {
      enterActive: 'transition duration-300 ease-out motion-reduce:transition-none',
      enterFrom: '-translate-x-8 opacity-0 motion-reduce:translate-x-0 motion-reduce:opacity-100',
      enterTo: 'translate-x-0 opacity-100 motion-reduce:translate-x-0 motion-reduce:opacity-100',
      leaveActive: 'transition duration-200 ease-in motion-reduce:transition-none',
      leaveFrom: 'translate-x-0 opacity-100 motion-reduce:translate-x-0 motion-reduce:opacity-100',
      leaveTo: 'translate-x-8 opacity-0 motion-reduce:translate-x-0 motion-reduce:opacity-100',
    }
  }

  return {
    enterActive: 'transition duration-300 ease-out motion-reduce:transition-none',
    enterFrom: 'translate-x-8 opacity-0 motion-reduce:translate-x-0 motion-reduce:opacity-100',
    enterTo: 'translate-x-0 opacity-100 motion-reduce:translate-x-0 motion-reduce:opacity-100',
    leaveActive: 'transition duration-200 ease-in motion-reduce:transition-none',
    leaveFrom: 'translate-x-0 opacity-100 motion-reduce:translate-x-0 motion-reduce:opacity-100',
    leaveTo: '-translate-x-8 opacity-0 motion-reduce:translate-x-0 motion-reduce:opacity-100',
  }
})

function isStepSelectable(step: number) {
  if (step < 1 || step > totalSteps) {
    return false
  }

  if (setupCompleted.value) {
    return step === totalSteps
  }

  return step <= currentStep.value
}

watch(
  setupCompleted,
  (completed) => {
    if (completed) {
      currentStep.value = totalSteps
    }
  },
  { immediate: true },
)

async function submit() {
  if (submitting.value || setupCompleted.value) {
    return
  }

  if (password.value !== confirmPassword.value) {
    toast.error(t('setup.passwordMismatch'))
    return
  }

  submitting.value = true
  const setupPromise = authStore.completeSetup({
    username: username.value.trim(),
    password: password.value,
  })

  toast.promise(setupPromise, {
    loading: t('setup.creating'),
    success: () => t('setup.success'),
    error: (error: unknown) => getAPIErrorMessage(t, error, 'setup.failed'),
  })

  try {
    await setupPromise
    transitionDirection.value = 'forward'
    currentStep.value = totalSteps
  } catch {
    // Error is handled by toast.promise
  } finally {
    submitting.value = false
  }
}

async function goToDashboard() {
  await router.push({ name: 'dashboard' })
}
</script>

<template>
  <div class="min-h-screen bg-background px-6 py-10 text-foreground">
    <div class="mx-auto flex min-h-[calc(100vh-5rem)] w-full max-w-2xl items-center">
      <div class="w-full space-y-8">
        <div class="space-y-2 text-center">
          <h1 class="text-3xl font-semibold tracking-tight sm:text-4xl">
            {{ t('setup.title') }}
          </h1>
        </div>

        <Stepper
          v-model="currentStep"
          :linear="false"
          class="flex w-full items-start gap-3"
        >
          <StepperItem
            v-for="step in steps"
            :key="step.step"
            :step="step.step"
            :disabled="!isStepSelectable(step.step)"
            class="relative flex flex-1 flex-col items-center justify-center gap-2 text-center transition-opacity data-disabled:pointer-events-none data-disabled:opacity-50"
          >
            <StepperSeparator
              v-if="step.step !== totalSteps"
              class="absolute inset-s-[calc(50%+1.5rem)] inset-e-[calc(-50%+1.5rem)] inset-bs-4 block h-px rounded-full bg-border/70"
            />

            <StepperTrigger as-child>
              <StepperIndicator class="size-10 shadow-sm">
                <component
                  :is="step.icon"
                  class="size-4"
                />
              </StepperIndicator>
            </StepperTrigger>

            <StepperTitle class="text-sm font-medium text-foreground">
              {{ t(step.titleKey) }}
            </StepperTitle>
          </StepperItem>
        </Stepper>

        <div class="relative overflow-hidden rounded-2xl border border-border/60 bg-card/70 p-6 shadow-sm sm:p-8">
          <Transition
            mode="out-in"
            :enter-active-class="transitionClasses.enterActive"
            :enter-from-class="transitionClasses.enterFrom"
            :enter-to-class="transitionClasses.enterTo"
            :leave-active-class="transitionClasses.leaveActive"
            :leave-from-class="transitionClasses.leaveFrom"
            :leave-to-class="transitionClasses.leaveTo"
          >
            <section
              :key="currentStep"
              class="flex min-h-96 flex-col gap-6"
            >
              <template v-if="currentStep === 1">
                <h2 class="text-xl font-semibold tracking-tight">{{ t('setup.step1') }}</h2>

                <form
                  class="flex flex-1 flex-col"
                  @submit.prevent="submit"
                >
                  <FieldGroup class="flex flex-1 flex-col gap-5">
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

                    <Field class="mbs-auto pbs-2">
                      <Button
                        type="submit"
                        size="lg"
                        class="w-full"
                        :disabled="submitting || setupCompleted"
                      >
                        <Spinner
                          v-if="submitting"
                          class="me-2"
                        />
                        <Rocket
                          v-else
                          class="me-2 size-4"
                        />
                        {{ submitting ? t('setup.creating') : t('setup.submit') }}
                      </Button>
                    </Field>
                  </FieldGroup>
                </form>
              </template>

              <template v-else>
                <div class="flex flex-1 flex-col items-center justify-center gap-8 py-6 text-center sm:py-10">
                  <div class="flex size-20 items-center justify-center rounded-full bg-primary/10 text-primary">
                    <Check class="size-10" />
                  </div>

                  <div class="space-y-3">
                    <h2 class="text-3xl font-bold tracking-tight">{{ t('setup.success') }}</h2>
                  </div>

                  <Button
                    size="lg"
                    class="w-full px-8 sm:w-auto"
                    @click="goToDashboard"
                  >
                    {{ t('setup.complete') }}
                  </Button>
                </div>
              </template>
            </section>
          </Transition>
        </div>
      </div>
    </div>
  </div>
</template>
