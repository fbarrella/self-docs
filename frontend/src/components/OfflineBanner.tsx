import { useEffect, useState } from 'react'
import { api } from '../api/client'

/**
 * OfflineBanner polls the backend health endpoint and shows a non-blocking
 * banner when the API is unreachable, so users understand why data is missing
 * instead of seeing blank screens.
 */
export function OfflineBanner() {
  const [offline, setOffline] = useState(false)

  useEffect(() => {
    let active = true

    async function check() {
      try {
        await api.health()
        if (active) setOffline(false)
      } catch {
        if (active) setOffline(true)
      }
    }

    void check()
    const timer = window.setInterval(check, 20000)
    return () => {
      active = false
      window.clearInterval(timer)
    }
  }, [])

  if (!offline) return null

  return (
    <div className="offline-banner" role="status">
      <span aria-hidden="true">⚠️</span>
      <span>Cannot reach the backend. Showing limited data — retrying automatically…</span>
    </div>
  )
}
