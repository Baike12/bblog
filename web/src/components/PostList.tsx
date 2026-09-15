import type { Post } from '../lib/types'

/**
 * 文章链接指向后端直出的页面，因此使用普通 <a> 触发整页跳转，
 * 不走 React Router 的客户端路由。
 */
export function PostList({ posts }: { posts: Post[] }) {
  if (posts.length === 0) {
    return <p className="state-empty">还没有文章。</p>
  }
  return (
    <ul className="post-list">
      {posts.map((post) => (
        <li key={post.slug} className="post-item">
          <a className="post-item-title" href={post.url}>
            {post.title}
          </a>
          <p className="post-item-meta">
            <time dateTime={post.date}>{post.date_text}</time>
            <span>约 {post.reading_minutes} 分钟</span>
            {post.updated_text ? <span>更新于 {post.updated_text}</span> : null}
          </p>
          {post.summary ? <p className="post-item-summary">{post.summary}</p> : null}
        </li>
      ))}
    </ul>
  )
}
