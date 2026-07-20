<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import SettingsSectionCard from '@/components/settings/common/SettingsSectionCard.vue'
import { usePasswordSettings } from '@/composables/settings/usePasswordSettings'

const { t } = useI18n()
const { passwordForm, passwordError, isUpdatingPassword, updatePassword } =
  usePasswordSettings()
</script>

<template>
  <SettingsSectionCard
    :title="t('settings.account.password')"
    :description="t('settings.account.passwordDesc')"
    content-class="space-y-4"
    footer-class="justify-end"
  >
    <div class="space-y-2">
      <Label for="current-password">{{
        t('settings.account.currentPassword')
      }}</Label>
      <Input
        id="current-password"
        v-model="passwordForm.currentPassword"
        type="password"
        :placeholder="t('settings.account.currentPasswordPlaceholder')"
      />
    </div>

    <div class="grid gap-4 md:grid-cols-2">
      <div class="space-y-2">
        <Label for="new-password">{{
          t('settings.account.newPassword')
        }}</Label>
        <Input
          id="new-password"
          v-model="passwordForm.newPassword"
          type="password"
          :placeholder="t('settings.account.newPasswordPlaceholder')"
        />
      </div>

      <div class="space-y-2">
        <Label for="confirm-password">{{
          t('settings.account.confirmPassword')
        }}</Label>
        <Input
          id="confirm-password"
          v-model="passwordForm.confirmPassword"
          type="password"
          :placeholder="t('settings.account.confirmPasswordPlaceholder')"
        />
      </div>
    </div>

    <p
      v-if="passwordError !== ''"
      class="text-sm text-destructive"
    >
      {{ passwordError }}
    </p>

    <template #footer>
      <Button
        variant="outline"
        :disabled="isUpdatingPassword"
        @click="updatePassword"
      >
        {{
          isUpdatingPassword
            ? t('settings.account.updatingPassword')
            : t('settings.account.updatePassword')
        }}
      </Button>
    </template>
  </SettingsSectionCard>
</template>
