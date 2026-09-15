import { Link } from 'react-router-dom'

export function NotFound() {
  return (
    <section className="not-found">
      <h1>404</h1>
      <p>这个地址没有对应的内容。</p>
      <p>
        <Link to="/">返回首页</Link>
      </p>
    </section>
  )
}
