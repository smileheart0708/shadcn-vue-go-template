<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Spinner } from '@/components/ui/spinner'
import type { ManagedUser } from '@/lib/api/admin-users'

defineProps<{
  target: ManagedUser | null
  pending: boolean
}>()

const emit = defineEmits<{
  confirm: []
}>()

const open = defineModel<boolean>('open', { required: true })

const { t } = useI18n()
</script>

<template>
  <AlertDialog v-model:open="open">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>
          {{
            target?.status === 'active'
              ? t('adminUsers.confirm.disableTitle')
              : t('adminUsers.confirm.enableTitle')
          }}
        </AlertDialogTitle>
        <AlertDialogDescription>
          {{
            target?.status === 'active'
              ? t('adminUsers.confirm.disableDescription', {
                  username: target?.username ?? '',
                })
              : t('adminUsers.confirm.enableDescription', {
                  username: target?.username ?? '',
                })
          }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel :disabled="pending">{{
          t('common.action.cancel')
        }}</AlertDialogCancel>
        <AlertDialogAction
          :disabled="pending"
          @click.prevent="emit('confirm')"
        >
          <Spinner
            v-if="pending"
            class="me-2"
          />
          {{
            target?.status === 'active'
              ? t('adminUsers.actions.disable')
              : t('adminUsers.actions.enable')
          }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>
