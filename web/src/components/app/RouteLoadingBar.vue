<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRouteLoadingState } from '@/router/route-loading'

const isRouteLoading = useRouteLoadingState()
const { t } = useI18n()
</script>

<template>
  <Transition
    enter-active-class="transition-opacity duration-150 ease-linear"
    enter-from-class="opacity-0"
    leave-active-class="transition-opacity duration-150 ease-linear"
    leave-to-class="opacity-0"
  >
    <div
      v-if="isRouteLoading"
      class="pointer-events-none fixed inset-x-0 inset-bs-0 z-70 h-0.5 overflow-hidden"
      role="status"
      aria-live="polite"
    >
      <span class="sr-only">{{ t('common.state.loading') }}</span>
      <span
        class="absolute inset-y-0 inset-s-0 w-[34%] rounded-full bg-[linear-gradient(90deg,transparent_0%,var(--primary)_45%,var(--primary)_55%,transparent_100%)] motion-reduce:w-full motion-reduce:animate-none"
        style="animation: route-loading-sweep 1.05s linear infinite"
      />
      <span
        class="absolute -inset-y-1.5 inset-s-0 w-[34%] rounded-full bg-[linear-gradient(90deg,transparent_0%,var(--primary)_45%,var(--primary)_55%,transparent_100%)] opacity-[0.35] blur-[10px] motion-reduce:hidden"
        style="animation: route-loading-sweep 1.05s linear infinite"
      />
    </div>
  </Transition>
</template>
