import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    // Proxy das chamadas de API pro backend Go durante o desenvolvimento.
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
