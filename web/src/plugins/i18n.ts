import { createI18n } from 'vue-i18n'
import enUS from '@/locales/en-US'
import type { MessageSchema } from '@/locales/schema'
import zhCN from '@/locales/zh-CN'
import { applyDocumentLanguage, fallbackLocale, localeNames, persistLocale, resolveInitialLocale, supportedLocales, type AppLocale } from '@/plugins/i18n/locales'

const messages: Record<AppLocale, MessageSchema> = { 'zh-CN': zhCN, 'en-US': enUS }

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
  persistLocale(locale)
}

export default i18n
export { fallbackLocale, localeNames, supportedLocales }
export type { AppLocale }
