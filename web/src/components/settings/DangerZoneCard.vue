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
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'
import SettingsSectionCard from '@/components/settings/common/SettingsSectionCard.vue'
import { useDeleteAccountFlow } from '@/composables/settings/useDeleteAccountFlow'

const { t } = useI18n()
const {
  deleteDialogOpen,
  deleteCountdown,
  deleteAccountConfirmed,
  canDeleteAccount,
  canOpenDeleteDialog,
  canSubmitDelete,
  isDeletingAccount,
  confirmDelete,
} = useDeleteAccountFlow()
</script>

<template>
  <SettingsSectionCard
    v-if="canDeleteAccount"
    class="border-destructive/60"
    :title="t('settings.account.dangerZone')"
    :description="t('settings.account.dangerZoneDesc')"
    title-class="text-destructive"
    content-class="space-y-3"
  >
    <div class="flex items-center gap-2">
      <Checkbox
        id="delete-account"
        v-model="deleteAccountConfirmed"
      />
      <Label for="delete-account">{{
        t('settings.account.dangerZoneConfirm')
      }}</Label>
    </div>

    <template #footer>
      <AlertDialog v-model:open="deleteDialogOpen">
        <AlertDialogTrigger as-child>
          <Button
            variant="destructive"
            :disabled="!canOpenDeleteDialog"
          >
            {{ t('settings.account.deleteAccount') }}
          </Button>
        </AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{{
              t('settings.account.deleteAccount')
            }}</AlertDialogTitle>
            <AlertDialogDescription>{{
              t('settings.account.deleteAccountConfirm')
            }}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel :disabled="isDeletingAccount">{{
              t('common.action.cancel')
            }}</AlertDialogCancel>
            <!-- 让确认动作在对话框自动关闭前先进入提交分支。 -->
            <AlertDialogAction
              as-child
              :disabled="!canSubmitDelete || isDeletingAccount"
              @click.prevent="confirmDelete"
            >
              <Button
                variant="destructive"
                :disabled="!canSubmitDelete || isDeletingAccount"
              >
                <Spinner
                  v-if="isDeletingAccount"
                  class="me-2"
                />
                {{
                  deleteCountdown > 0
                    ? `${deleteCountdown}s`
                    : t('settings.account.deleteAccount')
                }}
              </Button>
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </template>
  </SettingsSectionCard>
</template>
