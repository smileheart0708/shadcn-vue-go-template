<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

const { t } = useI18n()

const notifications = ref({
  emailNotifications: true,
  pushNotifications: true,
  weeklyDigest: false,
  securityAlerts: true,
})
const saved = ref(false)

let savedTimer: number | null = null

function saveSettings() {
  saved.value = true

  if (savedTimer !== null) {
    window.clearTimeout(savedTimer)
  }

  savedTimer = window.setTimeout(() => {
    saved.value = false
    savedTimer = null
  }, 2000)
}

onBeforeUnmount(() => {
  if (savedTimer !== null) {
    window.clearTimeout(savedTimer)
  }
})
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle>{{ t('settings.notifications.title') }}</CardTitle>
      <CardDescription>{{ t('settings.notifications.desc') }}</CardDescription>
    </CardHeader>
    <CardContent class="space-y-6">
      <div class="flex items-center justify-between">
        <div class="space-y-0.5">
          <Label>{{ t('settings.notifications.email') }}</Label>
          <p class="text-sm text-muted-foreground">{{ t('settings.notifications.emailDesc') }}</p>
        </div>
        <Switch v-model="notifications.emailNotifications" />
      </div>

      <div class="flex items-center justify-between">
        <div class="space-y-0.5">
          <Label>{{ t('settings.notifications.push') }}</Label>
          <p class="text-sm text-muted-foreground">{{ t('settings.notifications.pushDesc') }}</p>
        </div>
        <Switch v-model="notifications.pushNotifications" />
      </div>

      <div class="flex items-center justify-between">
        <div class="space-y-0.5">
          <Label>{{ t('settings.notifications.digest') }}</Label>
          <p class="text-sm text-muted-foreground">{{ t('settings.notifications.digestDesc') }}</p>
        </div>
        <Switch v-model="notifications.weeklyDigest" />
      </div>

      <div class="flex items-center justify-between">
        <div class="space-y-0.5">
          <Label>{{ t('settings.notifications.security') }}</Label>
          <p class="text-sm text-muted-foreground">{{ t('settings.notifications.securityDesc') }}</p>
        </div>
        <Switch v-model="notifications.securityAlerts" />
      </div>
    </CardContent>
    <CardFooter class="justify-end">
      <Button @click="saveSettings">
        {{ saved ? t('settings.saved') : t('settings.save') }}
      </Button>
    </CardFooter>
  </Card>
</template>
