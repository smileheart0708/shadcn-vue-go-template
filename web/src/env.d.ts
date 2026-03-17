/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_MOCKING?: boolean
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare const __VITE_API_MOCKING__: boolean

