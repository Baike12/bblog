import { Link, Outlet } from 'react-router-dom'

import { bootstrap } from '../lib/api'
import { ThemeToggle } from './ThemeToggle'

export function Layout() {
  const { site, nav } = bootstrap

  return (
    <>
      <header className="site-header">
        <div className="wrap header-inner">
          <Link className="site-title" to="/">
            {site.title}
          </Link>
          <nav className="site-nav">
            <Link to="/tags/">标签</Link>
            <Link to="/archive/">归档</Link>
            <Link to="/search/">搜索</Link>
            {nav.map((item) => (
              // 独立页面由后端直出，不走前端路由，因此用普通链接
              <a key={item.slug} href={item.url}>
                {item.title}
              </a>
            ))}
            <ThemeToggle />
          </nav>
        </div>
      </header>
      <main className="wrap">
        <Outlet />
      </main>
      <footer className="site-footer">
        <div className="wrap footer-inner">
          <p>
            © {site.author} · <a href={`${site.base_path}/rss.xml`}>RSS</a>
          </p>
        </div>
      </footer>
    </>
  )
}
