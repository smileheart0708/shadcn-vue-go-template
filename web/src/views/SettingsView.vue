<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import AccountSettingsTab from '@/components/settings/AccountSettingsTab.vue'
import BasicSettingsTab from '@/components/settings/BasicSettingsTab.vue'

const { t } = useI18n()
</script>

<template>
  <div class="flex flex-1 flex-col gap-4 p-4 lg:gap-6 lg:p-6">
    <div>
      <h1 class="text-2xl font-semibold">{{ t('settings.title') }}</h1>
      <p class="text-sm text-muted-foreground">
        {{ t('settings.description') }}
      </p>
    </div>

    <Tabs
      default-value="basic"
      class="space-y-4"
    >
      <TabsList>
        <TabsTrigger value="basic">{{ t('settings.tabs.basic') }}</TabsTrigger>
        <TabsTrigger value="account">{{
          t('settings.tabs.account')
        }}</TabsTrigger>
      </TabsList>

      <TabsContent
        value="basic"
        class="space-y-4"
      >
        <BasicSettingsTab />
      </TabsContent>

      <TabsContent
        value="account"
        class="space-y-4"
      >
        <AccountSettingsTab />
      </TabsContent>
    </Tabs>
  </div>
</template>
