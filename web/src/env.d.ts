/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'

  const component: DefineComponent<Record<string, never>, Record<string, never>, unknown>
  export default component
}

interface ImportMetaEnv {
  readonly VITE_API_MOCKING?: boolean
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare const __VITE_API_MOCKING__: boolean
