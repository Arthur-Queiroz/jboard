import { createApp } from 'vue'
// Fontes auto-hospedadas (via @fontsource): funcionam offline no Tauri e
// respeitam a CSP 'self' (sem CDN do Google). Só o eixo de peso (variable).
import '@fontsource-variable/bricolage-grotesque/wght.css'
import '@fontsource-variable/hanken-grotesk/wght.css'
import App from './App.vue'
import './style.css'

createApp(App).mount('#app')
