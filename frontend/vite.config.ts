import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/api/terminal/ws': {
        target: 'ws://localhost:9000',
        ws: true,
      },
      '/api/ws': {
        target: 'ws://localhost:9000',
        ws: true,
      },
      '/api': {
        target: 'http://localhost:9000',
        changeOrigin: true,
      },
    },
  },
})
