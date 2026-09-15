import { Link, useSearchParams } from 'react-router-dom'

type Props = {
  page: number
  totalPages: number
  /** 额外保留的查询参数，例如标签页的 tag。 */
  preserve?: Record<string, string>
}

export function Pagination({ page, totalPages, preserve }: Props) {
  const [searchParams] = useSearchParams()

  if (totalPages <= 1) {
    return null
  }

  const linkTo = (target: number) => {
    const params = new URLSearchParams(searchParams)
    for (const [key, value] of Object.entries(preserve ?? {})) {
      params.set(key, value)
    }
    params.set('page', String(target))
    return `?${params.toString()}`
  }

  return (
    <nav className="pagination">
      {page > 1 ? (
        <Link to={linkTo(page - 1)}>← 上一页</Link>
      ) : (
        <span className="pagination-info" />
      )}
      <span className="pagination-info">
        第 {page} / {totalPages} 页
      </span>
      {page < totalPages ? (
        <Link to={linkTo(page + 1)}>下一页 →</Link>
      ) : (
        <span className="pagination-info" />
      )}
    </nav>
  )
}
