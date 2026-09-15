import { useSearchParams } from 'react-router-dom'

import { SearchBox } from '../components/SearchBox'
import { fetchSearch } from '../lib/api'
import type { SearchResponse } from '../lib/types'
import { useAsync } from '../lib/useAsync'

export function Search() {
  const [searchParams] = useSearchParams()
  const q = (searchParams.get('q') ?? '').trim()

  const { data, error, loading } = useAsync<SearchResponse | undefined>(
    () => (q ? fetchSearch(q) : Promise.resolve(undefined)),
    [q],
  )

  return (
    <>
      <h1 className="page-heading">搜索</h1>
      <SearchBox />
      {!q ? <p className="state-empty">输入关键词后回车搜索。</p> : null}
      {q && loading ? <p className="state-loading">搜索中…</p> : null}
      {q && error ? <p className="state-error">搜索失败：{error}</p> : null}
      {q && !loading && !error && (data?.count ?? 0) === 0 ? (
        <p className="state-empty">没有匹配「{q}」的内容。</p>
      ) : null}
      {q && data && data.count > 0 ? (
        <>
          <p className="pagination-info">找到 {data.count} 条结果</p>
          {data.items.map((item) => (
            <article key={item.slug} className="search-result">
              <a className="post-item-title" href={item.url}>
                {item.title}
              </a>
              <p className="post-item-meta">
                <time dateTime={item.date}>{item.date_text}</time>
                {item.tags?.map((tag) => <span key={tag}>#{tag}</span>)}
              </p>
              {item.excerpt ? <p className="search-excerpt">{item.excerpt}</p> : null}
            </article>
          ))}
        </>
      ) : null}
    </>
  )
}
