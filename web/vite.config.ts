import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  // 站点挂在 /blog/ 路径下，资源引用必须带这个前缀
  base: '/blog/',
  plugins: [react(), tailwindcss()],
  build: {
    // 产物直接进入 Go 的 go:embed 目录
    outDir: '../internal/web/dist',
    emptyOutDir: true,
    // 生成 manifest，后端据此解析带哈希的资源文件名
    manifest: true,
    sourcemap: false,
  },
})
