<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRouteLoadingState } from '@/router/route-loading'

const isRouteLoading = useRouteLoadingState()
const { t } = useI18n()
</script>

<template>
  <Transition name="route-loading">
    <div
      v-if="isRouteLoading"
      class="route-loading-bar"
      role="status"
      aria-live="polite"
    >
      <span class="sr-only">{{ t('common.state.loading') }}</span>
      <span class="route-loading-bar__track" />
    </div>
  </Transition>
</template>
