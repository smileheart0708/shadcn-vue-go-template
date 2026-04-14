<script setup lang="ts">
import { computed, reactive, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AdminUser } from '@/lib/api/admin-users'
import { Button } from '@/components/ui/button'
import { Dialog, DialogDescription, DialogFooter, DialogHeader, DialogScrollContent, DialogTitle } from '@/components/ui/dialog'
import { Field, FieldDescription, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Spinner } from '@/components/ui/spinner'

const props = defineProps<{
  open: boolean
  mode: 'create' | 'edit'
  user: AdminUser | null
  pending: boolean
  canAssignAdminRole: boolean
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  submit: [payload: { username: string; email: string | null; password?: string; role?: number | null }]
}>()

const { t } = useI18n()

const usernameId = useId()
const emailId = useId()
const passwordId = useId()
const roleId = useId()

const form = reactive({
  username: '',
  email: '',
  password: '',
  role: '0',
})

const dialogTitle = computed(() => (props.mode === 'create' ? t('adminUsers.dialog.createTitle') : t('adminUsers.dialog.editTitle')))
const dialogDescription = computed(() => (props.mode === 'create' ? t('adminUsers.dialog.createDescription') : t('adminUsers.dialog.editDescription')))
const submitLabel = computed(() => (props.mode === 'create' ? t('adminUsers.dialog.createSubmit') : t('adminUsers.dialog.editSubmit')))

watch(
  () => [props.open, props.mode, props.user] as const,
  () => {
    form.username = props.user?.username ?? ''
    form.email = props.user?.email ?? ''
    form.password = ''
    form.role = String(props.user?.role ?? 0)
  },
  { immediate: true },
)

function handleSubmit() {
  const email = form.email.trim()

  emit('submit', {
    username: form.username.trim(),
    email: email === '' ? null : email,
    password: props.mode === 'create' ? form.password : undefined,
    role: props.canAssignAdminRole ? Number(form.role) : null,
  })
}
</script>

<template>
  <Dialog
    :open="props.open"
    @update:open="(value) => emit('update:open', value)"
  >
    <DialogScrollContent class="sm:max-w-xl">
      <DialogHeader>
        <DialogTitle>{{ dialogTitle }}</DialogTitle>
        <DialogDescription>{{ dialogDescription }}</DialogDescription>
      </DialogHeader>

      <form
        class="space-y-6"
        @submit.prevent="handleSubmit"
      >
        <FieldGroup>
          <Field>
            <FieldLabel :for="usernameId">{{ t('common.field.username') }}</FieldLabel>
            <Input
              :id="usernameId"
              v-model="form.username"
              autocomplete="username"
              :placeholder="t('adminUsers.dialog.usernamePlaceholder')"
              required
            />
          </Field>

          <Field>
            <FieldLabel :for="emailId">{{ t('common.field.email') }}</FieldLabel>
            <Input
              :id="emailId"
              v-model="form.email"
              autocomplete="email"
              :placeholder="t('adminUsers.dialog.emailPlaceholder')"
              type="email"
            />
          </Field>

          <Field v-if="props.mode === 'create'">
            <FieldLabel :for="passwordId">{{ t('common.field.password') }}</FieldLabel>
            <Input
              :id="passwordId"
              v-model="form.password"
              autocomplete="new-password"
              :placeholder="t('adminUsers.dialog.passwordPlaceholder')"
              type="password"
              minlength="8"
              required
            />
            <FieldDescription>{{ t('adminUsers.dialog.passwordHint') }}</FieldDescription>
          </Field>

          <Field v-if="props.canAssignAdminRole">
            <FieldLabel :for="roleId">{{ t('adminUsers.table.role') }}</FieldLabel>
            <Select v-model="form.role">
              <SelectTrigger :id="roleId">
                <SelectValue :placeholder="t('adminUsers.filters.rolePlaceholder')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="0">{{ t('common.userRole.0') }}</SelectItem>
                <SelectItem value="1">{{ t('common.userRole.1') }}</SelectItem>
              </SelectContent>
            </Select>
          </Field>
        </FieldGroup>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            :disabled="props.pending"
            @click="emit('update:open', false)"
          >
            {{ t('common.action.cancel') }}
          </Button>
          <Button
            type="submit"
            :disabled="props.pending"
          >
            <Spinner
              v-if="props.pending"
              class="me-2"
            />
            {{ submitLabel }}
          </Button>
        </DialogFooter>
      </form>
    </DialogScrollContent>
  </Dialog>
</template>
