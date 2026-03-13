import { createI18n } from 'vue-i18n'
import enUS from '@/locales/en-US'
import type { MessageSchema } from '@/locales/schema'
import zhCN from '@/locales/zh-CN'

export const supportedLocales = ['zh-CN', 'en-US'] as const

export type AppLocale = (typeof supportedLocales)[number]

export const localeNames: Record<AppLocale, string> = {
  'zh-CN': '简体中文',
  'en-US': 'English',
}

const STORAGE_KEY = 'app.locale'
const fallbackLocale: AppLocale = 'en-US'
const supportedLocaleSet = new Set<string>(supportedLocales)
const localeAliases: Readonly<Record<string, AppLocale>> = {
  en: 'en-US',
  'en-us': 'en-US',
  zh: 'zh-CN',
  'zh-cn': 'zh-CN',
}

const messages: { [locale in AppLocale]: MessageSchema } = { 'zh-CN': zhCN, 'en-US': enUS }

function isSupportedLocale(locale: string): locale is AppLocale {
  return supportedLocaleSet.has(locale)
}

function normalizeLocale(locale: string): AppLocale | null {
  return localeAliases[locale.toLowerCase()] ?? null
}

function resolveInitialLocale(): AppLocale {
  if (typeof window !== 'undefined') {
    const persistedLocale = window.localStorage.getItem(STORAGE_KEY)

    if (persistedLocale && isSupportedLocale(persistedLocale)) {
      return persistedLocale
    }
  }

  if (typeof navigator !== 'undefined') {
    for (const candidate of navigator.languages) {
      const normalizedLocale = normalizeLocale(candidate)

      if (normalizedLocale) {
        return normalizedLocale
      }
    }
  }

  return 'zh-CN'
}

function applyDocumentLanguage(locale: AppLocale) {
  if (typeof document !== 'undefined') {
    document.documentElement.lang = locale
  }
}

const locale = resolveInitialLocale()

applyDocumentLanguage(locale)

export const i18n = createI18n<[MessageSchema], AppLocale>({
  legacy: false,
  locale,
  fallbackLocale,
  globalInjection: true,
  messages,
})

export function setLocale(locale: AppLocale) {
  i18n.global.locale = locale
  applyDocumentLanguage(locale)

  if (typeof window !== 'undefined') {
    window.localStorage.setItem(STORAGE_KEY, locale)
  }
}

export default i18n
