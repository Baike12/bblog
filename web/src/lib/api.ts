import type {
  ArchiveResponse,
  Bootstrap,
  PostsResponse,
  SearchResponse,
  Site,
  TagsResponse,
} from './types'

const fallbackSite: Site = {
  title: 'bblog',
  author: '',
  description: '',
  base_path: '/blog',
  base_url: '',
  language: 'zh-CN',
}

function readBootstrap(): Bootstrap {
  const node = document.getElementById('bblog-data')
  if (!node?.textContent) {
    return { site: fallbackSite, nav: [], route: 'home' }
  }
  try {
    const parsed = JSON.parse(node.textContent) as Bootstrap
    return { ...parsed, site: { ...fallbackSite, ...parsed.site }, nav: parsed.nav ?? [] }
  } catch {
    return { site: fallbackSite, nav: [], route: 'home' }
  }
}

export const bootstrap = readBootstrap()

export const basePath = bootstrap.site.base_path || ''

/** 拼接站点内绝对路径（含基础路径），用于 React Router 之外的链接。 */
export function sitePath(path: string): string {
  const normalized = path.startsWith('/') ? path : `/${path}`
  return `${basePath}${normalized}`
}

async function getJSON<T>(url: string): Promise<T> {
  const response = await fetch(url, { headers: { Accept: 'application/json' } })
  if (!response.ok) {
    let message = `请求失败（HTTP ${response.status}）`
    try {
      const payload = (await response.json()) as { error?: { message?: string } }
      if (payload.error?.message) {
        message = payload.error.message
      }
    } catch {
      // 响应不是 JSON，保留默认提示
    }
    throw new Error(message)
  }
  return (await response.json()) as T
}

export function fetchPosts(params: {
  page?: number
  size?: number
  tag?: string
  year?: string
}): Promise<PostsResponse> {
  const query = new URLSearchParams()
  if (params.page) query.set('page', String(params.page))
  if (params.size) query.set('size', String(params.size))
  if (params.tag) query.set('tag', params.tag)
  if (params.year) query.set('year', params.year)
  const suffix = query.toString() ? `?${query.toString()}` : ''
  return getJSON<PostsResponse>(sitePath(`/api/posts${suffix}`))
}

export function fetchTags(): Promise<TagsResponse> {
  return getJSON<TagsResponse>(sitePath('/api/tags'))
}

export function fetchArchive(): Promise<ArchiveResponse> {
  return getJSON<ArchiveResponse>(sitePath('/api/archive'))
}

export function fetchSearch(q: string, limit = 20): Promise<SearchResponse> {
  const query = new URLSearchParams({ q, limit: String(limit) })
  return getJSON<SearchResponse>(sitePath(`/api/search?${query.toString()}`))
}
