import { ref } from 'vue'
import { defineStore } from 'pinia'
import { i18n, setLocale as applyLocale, type AppLocale } from '@/plugins/i18n'

export const useLocaleStore = defineStore('locale', () => {
  const locale = ref<AppLocale>(i18n.global.locale)

  function setLocale(nextLocale: AppLocale) {
    if (locale.value === nextLocale) {
      return
    }

    locale.value = nextLocale
    applyLocale(nextLocale)
  }

  return { locale, setLocale }
})
