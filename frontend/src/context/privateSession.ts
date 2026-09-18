/**
 * Private Archive session types and context, kept separate from the provider
 * component so fast refresh works for the component module.
 */
import { createContext, useContext } from 'react'

export type PrivateStatus = 'locked' | 'unlocked'

export interface UnlockResult {
  ok: boolean
  error?: string
  rateLimited?: boolean
}

export interface PrivateSessionValue {
  status: PrivateStatus
  unlockOpen: boolean
  requestUnlock: () => void
  closeUnlock: () => void
  unlock: (password: string) => Promise<UnlockResult>
  lock: () => Promise<void>
}

export const PrivateSessionContext = createContext<PrivateSessionValue | null>(null)

/** usePrivateSession accesses the Private Archive session context. */
export function usePrivateSession(): PrivateSessionValue {
  const context = useContext(PrivateSessionContext)
  if (!context) {
    throw new Error('usePrivateSession must be used within PrivateSessionProvider')
  }
  return context
}
