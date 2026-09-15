export type Theme = 'light' | 'dark'

const storageKey = 'bblog-theme'

export function currentTheme(): Theme {
  const attr = document.documentElement.getAttribute('data-theme')
  return attr === 'dark' ? 'dark' : 'light'
}

export function preferredTheme(): Theme {
  try {
    const stored = localStorage.getItem(storageKey)
    if (stored === 'dark' || stored === 'light') {
      return stored
    }
  } catch {
    // localStorage 不可用时退回系统偏好
  }
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export function applyTheme(theme: Theme): void {
  document.documentElement.setAttribute('data-theme', theme)
  try {
    localStorage.setItem(storageKey, theme)
  } catch {
    // 忽略写入失败
  }
  syncGiscus(theme)
}

/** giscus 主题通过 postMessage 同步，避免评论区与站点主题不一致。 */
export function syncGiscus(theme: Theme): void {
  const frame = document.querySelector<HTMLIFrameElement>('iframe.giscus-frame')
  if (!frame?.contentWindow) {
    return
  }
  frame.contentWindow.postMessage(
    { giscus: { setConfig: { theme: theme === 'dark' ? 'dark' : 'light' } } },
    'https://giscus.app',
  )
}

export function toggleTheme(): Theme {
  const next: Theme = currentTheme() === 'dark' ? 'light' : 'dark'
  applyTheme(next)
  return next
}
