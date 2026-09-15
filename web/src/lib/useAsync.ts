import { useCallback, useEffect, useState } from 'react'

type AsyncState<T> = {
  data?: T
  error?: string
  loading: boolean
}

/**
 * 极简异步数据钩子：依赖变化时重新请求，卸载或依赖变化后丢弃旧响应。
 */
export function useAsync<T>(fn: () => Promise<T>, deps: unknown[]): AsyncState<T> & {
  reload: () => void
} {
  const [state, setState] = useState<AsyncState<T>>({ loading: true })
  const [nonce, setNonce] = useState(0)

  const reload = useCallback(() => setNonce((value) => value + 1), [])

  useEffect(() => {
    let cancelled = false
    setState((prev) => ({ data: prev.data, loading: true }))
    fn()
      .then((data) => {
        if (!cancelled) {
          setState({ data, loading: false })
        }
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          const message = error instanceof Error ? error.message : String(error)
          setState({ error: message, loading: false })
        }
      })
    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, nonce])

  return { ...state, reload }
}
