import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: '../src/embed/frontend',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/ws': {
        target: 'ws://localhost:3578',
        ws: true,
      },
      '/api': 'http://localhost:3578',
      '/_ready': 'http://localhost:3578',
    },
  },
})
