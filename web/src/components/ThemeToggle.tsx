import { toggleTheme } from '../lib/theme'

/** 细线图标，与站点「纸与墨」的排版基调一致：静置时不喧哗。 */
const iconProps = {
  viewBox: '0 0 24 24',
  fill: 'none',
  stroke: 'currentColor',
  strokeWidth: 1.6,
  strokeLinecap: 'round' as const,
  strokeLinejoin: 'round' as const,
  'aria-hidden': true,
}

export function ThemeToggle() {
  return (
    <button
      type="button"
      className="theme-toggle"
      aria-label="切换深浅色"
      onClick={() => toggleTheme()}
    >
      <svg className="icon-moon" {...iconProps}>
        <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
      </svg>
      <svg className="icon-sun" {...iconProps}>
        <circle cx="12" cy="12" r="4.5" />
        <path d="M12 1.5v2M12 20.5v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1.5 12h2M20.5 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" />
      </svg>
    </button>
  )
}
