import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { ApiError, api } from '../api/client'
import type { Profile } from '../api/types'
import { DEFAULT_PROFILE, ProfileContext, type ProfileContextValue } from './profile'

/**
 * ProfileProvider loads the user's display name from the settings API and
 * exposes it to the header avatar/menu and the settings page. It falls back to
 * the default John Doe when the backend is unreachable.
 */
export function ProfileProvider({ children }: { children: ReactNode }) {
  const [profile, setProfile] = useState<Profile>(DEFAULT_PROFILE)
  const [loading, setLoading] = useState(true)
  const [nonce, setNonce] = useState(0)

  useEffect(() => {
    let active = true
    api.settings
      .get()
      .then((response) => {
        if (active) setProfile(response.data.profile)
      })
      .catch(() => {
        if (active) setProfile(DEFAULT_PROFILE)
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [nonce])

  const reload = useCallback(() => setNonce((value) => value + 1), [])

  const updateProfile = useCallback(async (next: Profile) => {
    try {
      const response = await api.settings.updateProfile(next)
      setProfile(response.data)
      return response.data
    } catch (err) {
      if (err instanceof ApiError) throw err
      throw new Error('Failed to save the profile.')
    }
  }, [])

  const value = useMemo<ProfileContextValue>(
    () => ({ profile, loading, updateProfile, reload }),
    [profile, loading, updateProfile, reload],
  )

  return <ProfileContext.Provider value={value}>{children}</ProfileContext.Provider>
}
