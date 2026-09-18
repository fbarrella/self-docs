import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { ApiError, api } from '../api/client'
import { clearPrivateToken, getPrivateToken, setPrivateToken } from '../api/privateToken'
import {
  PrivateSessionContext,
  type PrivateSessionValue,
  type PrivateStatus,
  type UnlockResult,
} from './privateSession'

/**
 * PrivateSessionProvider tracks the Private Archive lock state, exposes the
 * unlock/lock actions, and controls the master-password modal. The dashboard
 * card, header menu, and /private route all consult this context.
 */
export function PrivateSessionProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<PrivateStatus>(() =>
    getPrivateToken() ? 'unlocked' : 'locked',
  )
  const [unlockOpen, setUnlockOpen] = useState(false)

  const requestUnlock = useCallback(() => {
    if (getPrivateToken()) {
      setStatus('unlocked')
      return
    }
    setUnlockOpen(true)
  }, [])

  const closeUnlock = useCallback(() => setUnlockOpen(false), [])

  const unlock = useCallback(async (password: string): Promise<UnlockResult> => {
    try {
      const response = await api.private.unlock(password)
      setPrivateToken(response.data.token, response.data.expires_at)
      setStatus('unlocked')
      setUnlockOpen(false)
      return { ok: true }
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 429) {
          return { ok: false, rateLimited: true, error: err.message }
        }
        if (err.status === 401) {
          return { ok: false, error: 'Incorrect password. Please try again.' }
        }
        return { ok: false, error: err.message }
      }
      return { ok: false, error: 'Unlock failed. Is the backend running?' }
    }
  }, [])

  const lock = useCallback(async () => {
    try {
      await api.private.lock()
    } catch {
      // Even if the request fails, clear local state.
    }
    clearPrivateToken()
    setStatus('locked')
  }, [])

  // Reflect token expiry: re-check when the tab regains focus.
  useEffect(() => {
    function refresh() {
      setStatus(getPrivateToken() ? 'unlocked' : 'locked')
    }
    window.addEventListener('focus', refresh)
    return () => window.removeEventListener('focus', refresh)
  }, [])

  const value = useMemo<PrivateSessionValue>(
    () => ({ status, unlockOpen, requestUnlock, closeUnlock, unlock, lock }),
    [status, unlockOpen, requestUnlock, closeUnlock, unlock, lock],
  )

  return <PrivateSessionContext.Provider value={value}>{children}</PrivateSessionContext.Provider>
}
