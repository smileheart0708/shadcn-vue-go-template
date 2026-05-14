<script setup lang="ts">
import BasicAppearanceCard from '@/components/settings/BasicAppearanceCard.vue'
import DataRefreshIntervalCard from '@/components/settings/DataRefreshIntervalCard.vue'
import { useSettingsPreferences } from '@/composables/settings/useSettingsPreferences'

const {
  theme,
  locale,
  localeOptions,
  pollingIntervalSliderValue,
  currentUserPollingIntervalSeconds,
  handleThemeChange,
  handleLocaleChange,
  handlePollingIntervalSliderChange,
  commitPollingInterval,
} = useSettingsPreferences()
</script>

<template>
  <BasicAppearanceCard
    :theme="theme"
    :locale="locale"
    :locale-options="localeOptions"
    @update:theme="handleThemeChange"
    @update:locale="handleLocaleChange"
  />

  <DataRefreshIntervalCard
    :slider-value="pollingIntervalSliderValue"
    :current-seconds="currentUserPollingIntervalSeconds"
    @update:slider-value="handlePollingIntervalSliderChange"
    @value-commit="commitPollingInterval"
  />
</template>
