import { Link } from 'react-router-dom'

import type { Tag } from '../lib/types'

export function TagCloud({ tags }: { tags: Tag[] }) {
  if (tags.length === 0) {
    return <p className="state-empty">还没有标签。</p>
  }
  return (
    <ul className="tag-cloud">
      {tags.map((tag) => (
        <li key={tag.name}>
          <Link className="tag-item" to={`/tags/${encodeURIComponent(tag.name)}/`}>
            {tag.name}
            <span className="tag-count">{tag.count}</span>
          </Link>
        </li>
      ))}
    </ul>
  )
}
