export const supportedLocales = ['zh-CN', 'en-US'] as const

export type AppLocale = (typeof supportedLocales)[number]

export const localeDefinitions: Record<AppLocale, { label: string; htmlLang: string }> = {
  'zh-CN': { label: '简体中文', htmlLang: 'zh-CN' },
  'en-US': { label: 'English', htmlLang: 'en-US' },
}

export const localeNames: Record<AppLocale, string> = {
  'zh-CN': localeDefinitions['zh-CN'].label,
  'en-US': localeDefinitions['en-US'].label,
}

export const fallbackLocale: AppLocale = 'en-US'

const STORAGE_KEY = 'app.locale'
const defaultLocale: AppLocale = 'zh-CN'
const supportedLocaleSet = new Set<string>(supportedLocales)
const localeAliases: Readonly<Partial<Record<string, AppLocale>>> = {
  en: 'en-US',
  'en-us': 'en-US',
  zh: 'zh-CN',
  'zh-cn': 'zh-CN',
}

export function isSupportedLocale(locale: string): locale is AppLocale {
  return supportedLocaleSet.has(locale)
}

export function normalizeLocale(locale: string): AppLocale | null {
  const normalizedLocale = locale.toLowerCase()
  const exactMatch = supportedLocales.find((candidate) => candidate.toLowerCase() === normalizedLocale)
  const primaryLocale = normalizedLocale.split('-')[0]

  if (exactMatch) {
    return exactMatch
  }

  return localeAliases[normalizedLocale] ?? localeAliases[primaryLocale] ?? null
}

export function resolveInitialLocale(): AppLocale {
  if (typeof window !== 'undefined') {
    const persistedLocale = window.localStorage.getItem(STORAGE_KEY)

    if (persistedLocale !== null && isSupportedLocale(persistedLocale)) {
      return persistedLocale
    }
  }

  if (typeof navigator !== 'undefined') {
    const browserLocales = navigator.languages.length > 0 ? navigator.languages : [navigator.language]

    for (const candidate of browserLocales) {
      const normalizedLocale = normalizeLocale(candidate)

      if (normalizedLocale !== null) {
        return normalizedLocale
      }
    }
  }

  return defaultLocale
}

export function applyDocumentLanguage(locale: AppLocale) {
  if (typeof document === 'undefined') {
    return
  }

  document.documentElement.lang = localeDefinitions[locale].htmlLang
}

export function persistLocale(locale: AppLocale) {
  if (typeof window === 'undefined') {
    return
  }

  window.localStorage.setItem(STORAGE_KEY, locale)
}
