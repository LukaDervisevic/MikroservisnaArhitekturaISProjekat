import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// The APISIX gateway does not send CORS headers (no `cors` plugin in the
// global rules), so the browser must never talk to :9080 cross-origin.
// Everything goes to same-origin /api/* and is proxied here instead.
const GATEWAY = process.env.VITE_GATEWAY_URL ?? 'http://localhost:9080'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: GATEWAY,
        changeOrigin: true,
      },
    },
  },
})
