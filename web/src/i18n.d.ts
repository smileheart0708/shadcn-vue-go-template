import 'vue-i18n'
import type { MessageSchema } from '@/locales/schema'

declare module 'vue-i18n' {
  export interface DefineLocaleMessage {
    app: MessageSchema['app']
    auth: MessageSchema['auth']
    common: MessageSchema['common']
    route: MessageSchema['route']
  }
}
