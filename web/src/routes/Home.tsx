import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'

import { Pagination } from '../components/Pagination'
import { PostList } from '../components/PostList'
import { bootstrap, fetchPosts } from '../lib/api'
import type { PostsResponse } from '../lib/types'

type State = {
  data?: PostsResponse
  error?: string
  loading: boolean
}

export function Home() {
  const [searchParams] = useSearchParams()
  const rawPage = Number(searchParams.get('page') ?? '1')
  const page = Number.isFinite(rawPage) && rawPage >= 1 ? Math.floor(rawPage) : 1

  const initial = bootstrap.initial
  const [state, setState] = useState<State>(() =>
    page === 1 && initial ? { data: initial, loading: false } : { loading: true },
  )

  useEffect(() => {
    if (page === 1 && initial) {
      setState({ data: initial, loading: false })
      return
    }
    let cancelled = false
    setState({ loading: true })
    fetchPosts({ page })
      .then((data) => {
        if (!cancelled) setState({ data, loading: false })
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          setState({ error: error instanceof Error ? error.message : String(error), loading: false })
        }
      })
    return () => {
      cancelled = true
    }
  }, [page, initial])

  if (state.loading) {
    return <p className="state-loading">加载中…</p>
  }
  if (state.error) {
    return <p className="state-error">加载失败：{state.error}</p>
  }
  return (
    <>
      <PostList posts={state.data?.items ?? []} />
      <Pagination page={state.data?.page ?? 1} totalPages={state.data?.total_pages ?? 0} />
    </>
  )
}
