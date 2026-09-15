import { fetchArchive } from '../lib/api'
import { useAsync } from '../lib/useAsync'

export function Archive() {
  const { data, error, loading } = useAsync(() => fetchArchive(), [])

  if (loading) {
    return <p className="state-loading">加载中…</p>
  }
  if (error) {
    return <p className="state-error">加载失败：{error}</p>
  }
  const groups = data?.groups ?? []
  if (groups.length === 0) {
    return (
      <>
        <h1 className="page-heading">归档</h1>
        <p className="state-empty">还没有文章。</p>
      </>
    )
  }

  return (
    <>
      <h1 className="page-heading">归档</h1>
      {groups.map((group) => (
        <section key={group.year}>
          <h2 className="archive-year">
            {group.year}
            <span className="tag-count">{group.count} 篇</span>
          </h2>
          {group.months.map((month) => (
            <div key={`${group.year}-${month.month}`}>
              <h3 className="archive-month">
                {month.month_text}
                <span className="tag-count">{month.count} 篇</span>
              </h3>
              <ul className="archive-list">
                {month.items.map((post) => (
                  <li key={post.slug}>
                    <a href={post.url}>
                      <span className="archive-date">{post.date.slice(5)}</span>
                      {post.title}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </section>
      ))}
    </>
  )
}
