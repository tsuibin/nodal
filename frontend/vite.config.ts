import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

/**
 * Vite 配置：
 * - React 18 + TS；
 * - Tailwind CSS v4 通过官方 Vite 插件接入，CSS 里只需 @import "tailwindcss"；
 * - 开发期 /api 代理到本地 Go 服务（默认 :8080），生产由网关/同源反向代理处理。
 */
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 5173,
    host: '0.0.0.0',
    proxy: {
      '/api': {
        target: process.env.VITE_DEV_API_TARGET ?? 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
})
