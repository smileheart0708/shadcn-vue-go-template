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

<style scoped>
.route-loading-bar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: transparent;
  z-index: 9999;
  pointer-events: none;
}

.route-loading-bar__track {
  display: block;
  height: 100%;
  width: 100%;
  background: linear-gradient(to right, transparent, oklch(var(--primary)) 30%, oklch(var(--primary)) 60%, transparent);
  background-size: 200% 100%;
  animation: route-loading-bar__indeterminate 1.5s ease-in-out infinite;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

@keyframes route-loading-bar__indeterminate {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}

.route-loading-enter-active,
.route-loading-leave-active {
  transition: opacity 0.2s ease;
}

.route-loading-enter-from,
.route-loading-leave-to {
  opacity: 0;
}
</style>
