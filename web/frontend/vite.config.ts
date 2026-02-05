import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  base: '/panel/',
  server: {
    port: 3000,
    proxy: {
      '/panel/api': {
        target: 'http://localhost:2053',
        changeOrigin: true,
        ws: true,
      },
      '/panel/ws': {
        target: 'http://localhost:2053',
        changeOrigin: true,
        ws: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    sourcemap: false,
  },
})
