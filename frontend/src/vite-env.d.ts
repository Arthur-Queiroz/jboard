/// <reference types="vite/client" />

interface ImportMetaEnv {
  // URL absoluta da API pro build desktop; ausente na web (usa proxy '/api').
  readonly VITE_JBOARD_API_BASE?: string
  // Token Bearer injetado no build quando a API está protegida.
  readonly VITE_JBOARD_API_TOKEN?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}
