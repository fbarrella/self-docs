/**
 * User profile context and helpers, kept separate from the provider component
 * for fast-refresh compatibility.
 */
import { createContext, useContext } from 'react'
import type { Profile } from '../api/types'

export interface ProfileContextValue {
  profile: Profile
  loading: boolean
  updateProfile: (profile: Profile) => Promise<Profile>
  reload: () => void
}

export const DEFAULT_PROFILE: Profile = { first_name: 'John', last_name: 'Doe' }

/** initials derives up to two uppercase initials from a profile name. */
export function initialsOf(profile: Profile): string {
  const first = profile.first_name.trim().charAt(0)
  const last = profile.last_name.trim().charAt(0)
  const initials = `${first}${last}`.toUpperCase()
  return initials || 'JD'
}

/** fullName joins the profile parts, falling back to the default name. */
export function fullNameOf(profile: Profile): string {
  const name = [profile.first_name, profile.last_name]
    .map((part) => part.trim())
    .filter(Boolean)
    .join(' ')
  return name || `${DEFAULT_PROFILE.first_name} ${DEFAULT_PROFILE.last_name}`
}

export const ProfileContext = createContext<ProfileContextValue | null>(null)

/** useProfile accesses the user profile context. */
export function useProfile(): ProfileContextValue {
  const context = useContext(ProfileContext)
  if (!context) {
    throw new Error('useProfile must be used within ProfileProvider')
  }
  return context
}
