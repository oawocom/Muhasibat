import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Build to ../backend/web so the Go binary can embed the SPA (single container).
export default defineConfig({
  plugins: [react()],
  base: '/',
  build: {
    outDir: '../backend/web',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: { '/api': 'http://localhost:8080' },
  },
})
