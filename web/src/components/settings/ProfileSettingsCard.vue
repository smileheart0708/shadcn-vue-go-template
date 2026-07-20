<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import EditProfileDialog from '@/components/settings/EditProfileDialog.vue'
import SettingsSectionCard from '@/components/settings/common/SettingsSectionCard.vue'
import { useProfileSettings } from '@/composables/settings/useProfileSettings'

const { t } = useI18n()

const {
  currentIdentity,
  avatarFallbackText,
  avatarImageSrc,
  editDialogOpen,
  editForm,
  editAvatarFallbackText,
  editAvatarImageSrc,
  profileError,
  isSavingProfile,
  roleLabel,
  roleBadgeVariant,
  openAvatarPicker,
  saveProfile,
} = useProfileSettings()
</script>

<template>
  <SettingsSectionCard
    :title="t('settings.account.profile')"
    :description="t('settings.account.profileDesc')"
    content-class="space-y-4"
  >
    <div
      class="flex flex-col gap-4 rounded-lg border p-4 sm:flex-row sm:items-center sm:justify-between"
    >
      <div class="flex items-center gap-4">
        <Avatar class="rounded-full block-12 inline-12">
          <AvatarImage
            v-if="avatarImageSrc !== null"
            :src="avatarImageSrc"
            :alt="currentIdentity?.username ?? ''"
          />
          <AvatarFallback class="rounded-full">{{
            avatarFallbackText
          }}</AvatarFallback>
        </Avatar>
        <div class="space-y-1">
          <div class="flex flex-wrap items-center gap-2">
            <p class="font-medium">{{ currentIdentity?.username }}</p>
            <Badge :variant="roleBadgeVariant">{{ roleLabel }}</Badge>
            <Badge
              :variant="
                currentIdentity?.status === 'disabled' ? 'secondary' : 'outline'
              "
            >
              {{
                currentIdentity?.status === 'disabled'
                  ? t('common.state.disabled')
                  : t('settings.account.statusActive')
              }}
            </Badge>
          </div>
          <p class="text-sm text-muted-foreground">
            {{ currentIdentity?.email ?? t('settings.account.emailNotSet') }}
          </p>
        </div>
      </div>

      <EditProfileDialog
        v-model:open="editDialogOpen"
        v-model:username="editForm.username"
        v-model:email="editForm.email"
        :avatar-image-src="editAvatarImageSrc"
        :avatar-fallback-text="editAvatarFallbackText"
        :profile-error="profileError"
        :is-saving-profile="isSavingProfile"
        @choose-avatar="openAvatarPicker"
        @save-profile="saveProfile"
      />
    </div>
  </SettingsSectionCard>
</template>
