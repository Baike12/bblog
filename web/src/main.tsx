import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import { App } from './App'
import './styles/index.css'

const container = document.getElementById('root')
if (container) {
  // 使用 createRoot 而非 hydrateRoot：服务端在 #root 内预渲染了链接骨架供爬虫读取，
  // 首次渲染时会被整体替换，因此不存在 hydration 不匹配问题。
  createRoot(container).render(
    <StrictMode>
      <App />
    </StrictMode>,
  )
}
