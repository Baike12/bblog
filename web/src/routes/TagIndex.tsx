import { TagCloud } from '../components/TagCloud'
import { fetchTags } from '../lib/api'
import { useAsync } from '../lib/useAsync'

export function TagIndex() {
  const { data, error, loading } = useAsync(() => fetchTags(), [])

  if (loading) {
    return <p className="state-loading">加载中…</p>
  }
  if (error) {
    return <p className="state-error">加载失败：{error}</p>
  }
  return (
    <>
      <h1 className="page-heading">标签</h1>
      <TagCloud tags={data?.tags ?? []} />
    </>
  )
}
