import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

const THEME_STORAGE_KEY = 'app.theme'
const SYSTEM_THEME_MEDIA_QUERY = '(prefers-color-scheme: dark)'

const THEME_OPTIONS = {
  light: 'light',
  dark: 'dark',
  system: 'system',
} as const

export type ThemePreference = (typeof THEME_OPTIONS)[keyof typeof THEME_OPTIONS]
type ResolvedTheme = Exclude<ThemePreference, 'system'>

function isThemePreference(value: string | null): value is ThemePreference {
  return value === THEME_OPTIONS.light || value === THEME_OPTIONS.dark || value === THEME_OPTIONS.system
}

function readStoredTheme(): ThemePreference {
  const storedTheme = window.localStorage.getItem(THEME_STORAGE_KEY)
  return isThemePreference(storedTheme) ? storedTheme : THEME_OPTIONS.system
}

function getSystemTheme(): ResolvedTheme {
  return window.matchMedia(SYSTEM_THEME_MEDIA_QUERY).matches ? THEME_OPTIONS.dark : THEME_OPTIONS.light
}

function applyTheme(theme: ResolvedTheme) {
  document.documentElement.classList.toggle('dark', theme === THEME_OPTIONS.dark)
  document.documentElement.style.colorScheme = theme
}

export const useThemeStore = defineStore('theme', () => {
  const initialized = ref(false)
  const theme = ref<ThemePreference>(THEME_OPTIONS.system)
  const systemTheme = ref<ResolvedTheme>(THEME_OPTIONS.light)

  const resolvedTheme = computed<ResolvedTheme>(() => (theme.value === THEME_OPTIONS.system ? systemTheme.value : theme.value))

  let mediaQueryList: MediaQueryList | null = null

  function persistTheme(nextTheme: ThemePreference) {
    window.localStorage.setItem(THEME_STORAGE_KEY, nextTheme)
  }

  function syncResolvedTheme() {
    applyTheme(resolvedTheme.value)
  }

  function setTheme(nextTheme: ThemePreference) {
    theme.value = nextTheme
    persistTheme(nextTheme)
    syncResolvedTheme()
  }

  function initialize() {
    if (initialized.value) {
      return
    }

    theme.value = readStoredTheme()
    systemTheme.value = getSystemTheme()
    syncResolvedTheme()

    mediaQueryList ??= window.matchMedia(SYSTEM_THEME_MEDIA_QUERY)
    mediaQueryList.addEventListener('change', (event) => {
      systemTheme.value = event.matches ? THEME_OPTIONS.dark : THEME_OPTIONS.light

      if (theme.value === THEME_OPTIONS.system) {
        syncResolvedTheme()
      }
    })

    initialized.value = true
  }

  return { initialized, theme, resolvedTheme, setTheme, initialize }
})
