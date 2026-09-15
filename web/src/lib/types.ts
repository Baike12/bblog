export type Site = {
  title: string
  author: string
  description: string
  base_path: string
  base_url: string
  language: string
}

export type Post = {
  slug: string
  url: string
  title: string
  summary: string
  date: string
  date_text: string
  updated?: string
  updated_text?: string
  tags?: string[]
  reading_minutes: number
  word_count: number
  draft?: boolean
  private?: boolean
}

export type PostsResponse = {
  items: Post[]
  page: number
  size: number
  total: number
  total_pages: number
}

export type Tag = {
  name: string
  count: number
}

export type TagsResponse = {
  tags: Tag[]
}

export type ArchiveMonth = {
  month: number
  month_text: string
  count: number
  items: Post[]
}

export type ArchiveYear = {
  year: number
  count: number
  months: ArchiveMonth[]
}

export type ArchiveResponse = {
  groups: ArchiveYear[]
  total: number
}

export type SearchItem = Post & {
  score: number
  excerpt: string
}

export type SearchResponse = {
  q: string
  count: number
  items: SearchItem[]
}

export type NavItem = {
  slug: string
  url: string
  title: string
}

/** 后端注入到 HTML 的首屏数据。 */
export type Bootstrap = {
  site: Site
  nav: NavItem[]
  route: string
  initial?: PostsResponse
}

export type ApiError = {
  error: {
    code: string
    message: string
  }
}
