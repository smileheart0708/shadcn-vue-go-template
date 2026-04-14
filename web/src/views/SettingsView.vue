<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import SettingsAccountTab from '@/components/settings/SettingsAccountTab.vue'
import SettingsNotificationsTab from '@/components/settings/SettingsNotificationsTab.vue'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Slider } from '@/components/ui/slider'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { usePollingTask } from '@/composables/usePollingTask'
import { localeNames, type AppLocale } from '@/plugins/i18n/locales'
import { useAuthStore } from '@/stores/auth'
import { useLocaleStore } from '@/stores/locale'
import { POLLING_INTERVAL_MAX_SECONDS, POLLING_INTERVAL_MIN_SECONDS, normalizePollingIntervalSeconds, usePollingStore } from '@/stores/polling'
import { useThemeStore } from '@/stores/theme'

const { t } = useI18n()
const authStore = useAuthStore()
const themeStore = useThemeStore()
const localeStore = useLocaleStore()
const pollingStore = usePollingStore()

const pollingIntervalSliderValue = ref([pollingStore.currentUserIntervalSeconds])
const currentUserPolling = usePollingTask({
  key: 'auth.current-user',
  intervalMs: () => pollingStore.currentUserIntervalMs,
  enabled: () => authStore.isAuthenticated,
  fetch: async ({ signal }) => authStore.fetchCurrentUser({ signal, backgroundRequest: true }),
  apply: (user) => {
    authStore.applyCurrentUser(user)
  },
})

const localeOptions = Object.entries(localeNames).map(([value, label]) => ({
  value,
  label,
}))
const currentUserPollingIntervalSeconds = computed(() => normalizePollingIntervalSeconds(pollingIntervalSliderValue.value[0] ?? pollingStore.currentUserIntervalSeconds))

watch(
  () => pollingStore.currentUserIntervalSeconds,
  (seconds) => {
    pollingIntervalSliderValue.value = [seconds]
  },
  { immediate: true },
)

watch(
  () => currentUserPolling.error.value,
  (error) => {
    if (error === null) {
      return
    }

    toast.error(t('common.feedback.networkError'))
  },
)

function isThemePreference(value: unknown): value is 'light' | 'dark' | 'system' {
  return value === 'light' || value === 'dark' || value === 'system'
}

function isAppLocale(value: unknown): value is AppLocale {
  return value === 'en-US' || value === 'zh-CN'
}

function handleThemeChange(value: unknown) {
  if (isThemePreference(value)) {
    themeStore.setTheme(value)
  }
}

function handleLocaleChange(value: unknown) {
  if (isAppLocale(value)) {
    localeStore.setLocale(value)
  }
}

function handlePollingIntervalSliderChange(value: number[] | undefined) {
  if (!Array.isArray(value) || value.length === 0) {
    return
  }

  const sliderValue = value[0]
  pollingIntervalSliderValue.value = [normalizePollingIntervalSeconds(sliderValue)]
}

function commitPollingInterval(value: number[] | undefined) {
  if (!Array.isArray(value) || value.length === 0) {
    return
  }

  const sliderValue = value[0]
  const nextSeconds = normalizePollingIntervalSeconds(sliderValue)
  pollingIntervalSliderValue.value = [nextSeconds]

  if (nextSeconds === pollingStore.currentUserIntervalSeconds) {
    return
  }

  pollingStore.setCurrentUserIntervalSeconds(nextSeconds)

  if (!currentUserPolling.running.value) {
    return
  }

  currentUserPolling.pause()
  currentUserPolling.resume()
}
</script>

<template>
  <div class="flex flex-1 flex-col gap-4 p-4 lg:gap-6 lg:p-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-semibold">{{ t('settings.title') }}</h1>
        <p class="text-sm text-muted-foreground">{{ t('settings.description') }}</p>
      </div>
    </div>

    <Tabs
      default-value="basic"
      class="space-y-4"
    >
      <TabsList>
        <TabsTrigger value="basic"> {{ t('settings.tabs.basic') }} </TabsTrigger>
        <TabsTrigger value="account"> {{ t('settings.tabs.account') }} </TabsTrigger>
        <TabsTrigger value="notifications"> {{ t('settings.tabs.notifications') }} </TabsTrigger>
      </TabsList>

      <TabsContent
        value="basic"
        class="space-y-4"
      >
        <Card>
          <CardHeader>
            <CardTitle>{{ t('settings.basic.theme') }}</CardTitle>
            <CardDescription>{{ t('settings.basic.themeDesc') }}</CardDescription>
          </CardHeader>
          <CardContent class="space-y-6">
            <div class="space-y-2">
              <Label>{{ t('settings.basic.colorTheme') }}</Label>
              <Tabs
                :model-value="themeStore.theme"
                @update:model-value="handleThemeChange"
              >
                <TabsList>
                  <TabsTrigger value="light">
                    {{ t('settings.basic.light') }}
                  </TabsTrigger>
                  <TabsTrigger value="dark">
                    {{ t('settings.basic.dark') }}
                  </TabsTrigger>
                  <TabsTrigger value="system">
                    {{ t('settings.basic.system') }}
                  </TabsTrigger>
                </TabsList>
              </Tabs>
            </div>

            <div class="space-y-2">
              <Label>{{ t('settings.basic.language') }}</Label>
              <Select
                :model-value="localeStore.locale"
                @update:model-value="handleLocaleChange"
              >
                <SelectTrigger>
                  <SelectValue :placeholder="t('settings.basic.selectLanguage')" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem
                    v-for="locale in localeOptions"
                    :key="locale.value"
                    :value="locale.value"
                  >
                    {{ locale.label }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>{{ t('settings.basic.dataRefreshInterval') }}</CardTitle>
            <CardDescription>{{ t('settings.basic.dataRefreshIntervalDesc', { seconds: currentUserPollingIntervalSeconds }) }}</CardDescription>
          </CardHeader>
          <CardContent>
            <Slider
              :model-value="pollingIntervalSliderValue"
              :min="POLLING_INTERVAL_MIN_SECONDS"
              :max="POLLING_INTERVAL_MAX_SECONDS"
              :step="1"
              @update:model-value="handlePollingIntervalSliderChange"
              @value-commit="commitPollingInterval"
            />
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent
        value="account"
        class="space-y-4"
      >
        <SettingsAccountTab />
      </TabsContent>

      <TabsContent
        value="notifications"
        class="space-y-4"
      >
        <SettingsNotificationsTab />
      </TabsContent>
    </Tabs>
  </div>
</template>
