<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'

defineProps<{
  mode: 'create' | 'edit'
  pending: boolean
}>()

const emit = defineEmits<{
  submit: []
}>()

const open = defineModel<boolean>('open', { required: true })
const form = defineModel<{
  username: string
  email: string
  password: string
}>('form', { required: true })

const { t } = useI18n()
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent>
      <DialogHeader>
        <DialogTitle>{{
          mode === 'create'
            ? t('adminUsers.dialog.createTitle')
            : t('adminUsers.dialog.editTitle')
        }}</DialogTitle>
        <DialogDescription>{{
          mode === 'create'
            ? t('adminUsers.dialog.createDescription')
            : t('adminUsers.dialog.editDescription')
        }}</DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-2">
        <Input
          v-model="form.username"
          :placeholder="t('adminUsers.dialog.usernamePlaceholder')"
        />
        <Input
          v-model="form.email"
          :placeholder="t('adminUsers.dialog.emailPlaceholder')"
        />
        <Input
          v-if="mode === 'create'"
          v-model="form.password"
          type="password"
          :placeholder="t('adminUsers.dialog.passwordPlaceholder')"
        />
      </div>

      <DialogFooter>
        <Button
          variant="outline"
          :disabled="pending"
          @click="open = false"
        >
          {{ t('common.action.cancel') }}
        </Button>
        <Button
          :disabled="pending"
          @click="emit('submit')"
        >
          <Spinner
            v-if="pending"
            class="me-2"
          />
          {{
            mode === 'create'
              ? t('adminUsers.dialog.createSubmit')
              : t('adminUsers.dialog.editSubmit')
          }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
