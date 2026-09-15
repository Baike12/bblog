import { toggleTheme } from '../lib/theme'

export function ThemeToggle() {
  return (
    <button
      type="button"
      className="theme-toggle"
      aria-label="切换深浅色"
      onClick={() => toggleTheme()}
    >
      ◐
    </button>
  )
}
