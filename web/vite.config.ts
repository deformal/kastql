import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/graphql': 'http://localhost:8080',
      '/v1':      'http://localhost:8080',
      '/api':     'http://localhost:8080',
    },
  },
  optimizeDeps: {
    exclude: ['@graphiql/react/setup-workers/vite'],
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
