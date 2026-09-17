import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError } from '../api/client'

export interface AsyncState<T> {
  data: T | undefined
  error: ApiError | Error | undefined
  loading: boolean
  reload: () => void
}

/**
 * useAsync runs an async loader on mount and whenever its dependencies change,
 * tracking loading/error/data. It ignores results from stale or unmounted
 * requests and exposes a reload function.
 */
export function useAsync<T>(
  loader: (signal: AbortSignal) => Promise<T>,
  deps: unknown[] = [],
): AsyncState<T> {
  const [data, setData] = useState<T | undefined>(undefined)
  const [error, setError] = useState<ApiError | Error | undefined>(undefined)
  const [loading, setLoading] = useState(true)
  const [nonce, setNonce] = useState(0)
  const loaderRef = useRef(loader)
  loaderRef.current = loader

  useEffect(() => {
    const controller = new AbortController()
    let active = true

    setLoading(true)
    setError(undefined)

    loaderRef
      .current(controller.signal)
      .then((result) => {
        if (!active) return
        setData(result)
      })
      .catch((err: unknown) => {
        if (!active || controller.signal.aborted) return
        setError(err instanceof Error ? err : new Error(String(err)))
      })
      .finally(() => {
        if (active) setLoading(false)
      })

    return () => {
      active = false
      controller.abort()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, nonce])

  const reload = useCallback(() => setNonce((value) => value + 1), [])

  return { data, error, loading, reload }
}
