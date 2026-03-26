import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

const CURRENT_USER_INTERVAL_STORAGE_KEY = 'app.polling.current-user-interval-seconds'

export const POLLING_INTERVAL_MIN_SECONDS = 5
export const POLLING_INTERVAL_MAX_SECONDS = 60
export const POLLING_INTERVAL_DEFAULT_SECONDS = 10

export function normalizePollingIntervalSeconds(value: number) {
  if (!Number.isFinite(value)) {
    return POLLING_INTERVAL_DEFAULT_SECONDS
  }

  return Math.min(POLLING_INTERVAL_MAX_SECONDS, Math.max(POLLING_INTERVAL_MIN_SECONDS, Math.round(value)))
}

function readStoredCurrentUserIntervalSeconds() {
  if (typeof window === 'undefined') {
    return POLLING_INTERVAL_DEFAULT_SECONDS
  }

  const rawValue = window.localStorage.getItem(CURRENT_USER_INTERVAL_STORAGE_KEY)

  if (rawValue === null) {
    return POLLING_INTERVAL_DEFAULT_SECONDS
  }

  return normalizePollingIntervalSeconds(Number(rawValue))
}

function persistCurrentUserIntervalSeconds(value: number) {
  if (typeof window === 'undefined') {
    return
  }

  window.localStorage.setItem(CURRENT_USER_INTERVAL_STORAGE_KEY, String(value))
}

export const usePollingStore = defineStore('polling', () => {
  const currentUserIntervalSeconds = ref(readStoredCurrentUserIntervalSeconds())

  persistCurrentUserIntervalSeconds(currentUserIntervalSeconds.value)

  const currentUserIntervalMs = computed(() => currentUserIntervalSeconds.value * 1000)

  function setCurrentUserIntervalSeconds(value: number) {
    const nextValue = normalizePollingIntervalSeconds(value)

    if (currentUserIntervalSeconds.value === nextValue) {
      persistCurrentUserIntervalSeconds(nextValue)
      return
    }

    currentUserIntervalSeconds.value = nextValue
    persistCurrentUserIntervalSeconds(nextValue)
  }

  return {
    currentUserIntervalSeconds,
    currentUserIntervalMs,
    setCurrentUserIntervalSeconds,
  }
})
