import { useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'

/** 搜索框：提交后跳转到 /search/?q=… */
export function SearchBox() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const [value, setValue] = useState(searchParams.get('q') ?? '')

  return (
    <form
      className="search-form"
      role="search"
      onSubmit={(event) => {
        event.preventDefault()
        const q = value.trim()
        navigate(q ? `/search/?q=${encodeURIComponent(q)}` : '/search/')
      }}
    >
      <input
        className="search-input"
        type="search"
        name="q"
        value={value}
        placeholder="搜索标题、标签与正文"
        aria-label="搜索关键词"
        onChange={(event) => setValue(event.target.value)}
      />
      <button className="search-button" type="submit">
        搜索
      </button>
    </form>
  )
}
