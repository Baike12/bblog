import { useParams, useSearchParams } from 'react-router-dom'

import { Pagination } from '../components/Pagination'
import { PostList } from '../components/PostList'
import { fetchPosts } from '../lib/api'
import { useAsync } from '../lib/useAsync'

const pageSize = 20

export function TagDetail() {
  const { tag = '' } = useParams()
  const [searchParams] = useSearchParams()
  const rawPage = Number(searchParams.get('page') ?? '1')
  const page = Number.isFinite(rawPage) && rawPage >= 1 ? Math.floor(rawPage) : 1

  const { data, error, loading } = useAsync(
    () => fetchPosts({ tag, page, size: pageSize }),
    [tag, page],
  )

  return (
    <>
      <h1 className="page-heading">标签：{tag}</h1>
      {loading ? <p className="state-loading">加载中…</p> : null}
      {error ? <p className="state-error">加载失败：{error}</p> : null}
      {!loading && !error ? (
        <>
          <PostList posts={data?.items ?? []} />
          <Pagination
            page={data?.page ?? 1}
            totalPages={data?.total_pages ?? 0}
            preserve={{ tag }}
          />
        </>
      ) : null}
    </>
  )
}
