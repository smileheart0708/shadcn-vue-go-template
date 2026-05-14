import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from 'vue-sonner'
import { usePollingTask } from '@/composables/usePollingTask'
import { useAuthStore } from '@/stores/auth'
import { normalizePollingIntervalSeconds, usePollingStore } from '@/stores/polling'
import { useLocaleStore } from '@/stores/locale'
import { useThemeStore } from '@/stores/theme'
import { getLocaleOptions, isAppLocale, isThemePreference } from '@/utils/settings/preferences'

export function useSettingsPreferences() {
  const { t } = useI18n()
  const authStore = useAuthStore()
  const themeStore = useThemeStore()
  const localeStore = useLocaleStore()
  const pollingStore = usePollingStore()

  const pollingIntervalSliderValue = ref([pollingStore.currentUserIntervalSeconds])
  const localeOptions = getLocaleOptions()
  const currentUserPollingIntervalSeconds = computed(() => normalizePollingIntervalSeconds(pollingIntervalSliderValue.value[0] ?? pollingStore.currentUserIntervalSeconds))

  const currentUserPolling = usePollingTask({
    key: 'auth.current-user',
    intervalMs: () => pollingStore.currentUserIntervalMs,
    enabled: () => authStore.isAuthenticated,
    fetch: async ({ signal }) => authStore.fetchViewer({ signal, backgroundRequest: true }),
    apply: (viewer) => {
      authStore.applyViewer(viewer)
    },
  })

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

  return {
    theme: computed(() => themeStore.theme),
    locale: computed(() => localeStore.locale),
    localeOptions,
    pollingIntervalSliderValue,
    currentUserPollingIntervalSeconds,
    handleThemeChange,
    handleLocaleChange,
    handlePollingIntervalSliderChange,
    commitPollingInterval,
  }
}
