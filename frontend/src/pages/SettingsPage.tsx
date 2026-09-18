import { useState } from 'react'
import { ApiError, api } from '../api/client'
import type { Tag } from '../api/types'
import { Button, Card, EmptyState, ErrorState, Input, SectionHeader, Skeleton } from '../components'
import { usePrivateSession } from '../context/privateSession'
import { fullNameOf, initialsOf, useProfile } from '../context/profile'
import { useToast } from '../context/toast'
import { useAsync } from '../hooks/useAsync'

/**
 * SettingsPage implements the global configuration surface (PRD 4): app info
 * and health, master-password management, and tag management.
 */
export function SettingsPage() {
  const settings = useAsync(() => api.settings.get(), [])

  return (
    <div className="container settings">
      <SectionHeader title="Settings" headingLevel="h1" />
      <p className="text-secondary">
        Manage your profile, the master password, tags, and app info.
      </p>

      <ProfileCard />

      <Card className="settings__section">
        <SectionHeader title="Application" headingLevel="h2" />
        {settings.loading && !settings.data ? (
          <Skeleton height={72} />
        ) : settings.error ? (
          <ErrorState
            title="Could not load settings"
            message={settings.error.message}
            onRetry={settings.reload}
          />
        ) : (
          <>
            <div className="settings__row">
              <span className="settings__label">Version</span>
              <span className="settings__value">{settings.data?.data.version ?? '—'}</span>
            </div>
            <div className="settings__row">
              <span className="settings__label">Redis cache</span>
              <span className="settings__value">
                {settings.data?.data.redis_enabled ? 'Enabled' : 'Disabled'}
              </span>
            </div>
            <div className="settings__row">
              <span className="settings__label">Master password</span>
              <span className="settings__status settings__status--ok">
                {settings.data?.data.master_password_set ? 'Configured' : 'Not configured'}
              </span>
            </div>
          </>
        )}
      </Card>

      <MasterPasswordCard onChanged={() => settings.reload()} />
      <TagsCard />
    </div>
  )
}

function ProfileCard() {
  const { profile, updateProfile } = useProfile()
  const toast = useToast()
  const [firstName, setFirstName] = useState(profile.first_name)
  const [lastName, setLastName] = useState(profile.last_name)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<{ kind: 'error' | 'success'; text: string } | null>(null)

  // Adopt the loaded profile once it arrives (profile loading is async), and
  // when it changes elsewhere. Compared field-by-field to avoid clobbering
  // in-progress edits on every render.
  const [syncedProfile, setSyncedProfile] = useState(profile)
  if (profile !== syncedProfile) {
    setSyncedProfile(profile)
    setFirstName(profile.first_name)
    setLastName(profile.last_name)
  }

  async function submit() {
    setSaving(true)
    setMessage(null)
    try {
      const saved = await updateProfile({ first_name: firstName, last_name: lastName })
      setFirstName(saved.first_name)
      setLastName(saved.last_name)
      setMessage({ kind: 'success', text: 'Profile saved.' })
      toast.success('Profile saved')
    } catch (err) {
      const text = err instanceof ApiError ? err.message : 'Failed to save the profile.'
      setMessage({ kind: 'error', text })
      toast.error('Could not save profile', text)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card className="settings__section">
      <SectionHeader title="Profile" headingLevel="h2" />
      <p className="text-secondary">
        Your name is shown in the header. Leave both fields blank to use the default name.
      </p>
      <form
        className="settings__form"
        onSubmit={(event) => {
          event.preventDefault()
          void submit()
        }}
      >
        <Input
          label="First name"
          value={firstName}
          placeholder="John"
          autoComplete="given-name"
          onChange={(event) => setFirstName(event.target.value)}
        />
        <Input
          label="Last name"
          value={lastName}
          placeholder="Doe"
          autoComplete="family-name"
          onChange={(event) => setLastName(event.target.value)}
        />
        <p className="settings__label">
          Preview: <strong>{fullNameOf({ first_name: firstName, last_name: lastName })}</strong> (
          {initialsOf({ first_name: firstName, last_name: lastName })})
        </p>
        {message && (
          <p className={`settings__message settings__message--${message.kind}`} role="status">
            {message.text}
          </p>
        )}
        <div className="settings__actions">
          <Button type="submit" disabled={saving}>
            {saving ? 'Saving…' : 'Save profile'}
          </Button>
        </div>
      </form>
    </Card>
  )
}

function MasterPasswordCard({ onChanged }: { onChanged: () => void }) {
  const { status, requestUnlock } = usePrivateSession()
  const toast = useToast()
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<{ kind: 'error' | 'success'; text: string } | null>(null)

  async function submit() {
    setMessage(null)
    if (next.length < 8) {
      setMessage({ kind: 'error', text: 'New password must be at least 8 characters.' })
      return
    }
    if (next !== confirm) {
      setMessage({ kind: 'error', text: 'New password and confirmation do not match.' })
      return
    }
    setSaving(true)
    try {
      await api.settings.changeMasterPassword(current, next)
      setCurrent('')
      setNext('')
      setConfirm('')
      setMessage({
        kind: 'success',
        text: 'Master password changed. Existing private sessions were locked.',
      })
      toast.success('Master password changed')
      onChanged()
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setMessage({ kind: 'error', text: 'Current password is incorrect.' })
      } else {
        setMessage({
          kind: 'error',
          text: err instanceof ApiError ? err.message : 'Failed to change the password.',
        })
      }
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card className="settings__section">
      <SectionHeader title="Master password" headingLevel="h2" />
      <p className="text-secondary">
        Changing the master password locks any currently unlocked Private Archive sessions.
      </p>
      {status === 'locked' && (
        <Button variant="secondary" size="sm" onClick={requestUnlock}>
          Unlock to skip the current password
        </Button>
      )}
      <form
        className="settings__form"
        onSubmit={(event) => {
          event.preventDefault()
          void submit()
        }}
      >
        <Input
          label="Current password"
          type="password"
          autoComplete="current-password"
          value={current}
          onChange={(event) => setCurrent(event.target.value)}
          hint="Not required while the Private Archive is unlocked"
        />
        <Input
          label="New password"
          type="password"
          autoComplete="new-password"
          value={next}
          onChange={(event) => setNext(event.target.value)}
          hint="At least 8 characters"
        />
        <Input
          label="Confirm new password"
          type="password"
          autoComplete="new-password"
          value={confirm}
          onChange={(event) => setConfirm(event.target.value)}
        />
        {message && (
          <p className={`settings__message settings__message--${message.kind}`} role="status">
            {message.text}
          </p>
        )}
        <div className="settings__actions">
          <Button type="submit" disabled={saving}>
            {saving ? 'Saving…' : 'Change password'}
          </Button>
        </div>
      </form>
    </Card>
  )
}

function TagsCard() {
  const tags = useAsync<{ data: Tag[] }>(
    () => api.tags.list({ sort: 'count', order: 'desc', page_size: 100 }),
    [],
  )
  const toast = useToast()
  const [message, setMessage] = useState<string | null>(null)

  async function remove(name: string) {
    if (!window.confirm(`Delete the tag "${name}"? It will be removed from all documents.`)) return
    try {
      await api.tags.remove(name)
      setMessage(`Deleted "${name}".`)
      toast.success('Tag deleted', name)
      tags.reload()
    } catch (err) {
      const text = err instanceof ApiError ? err.message : 'Failed to delete the tag.'
      setMessage(text)
      toast.error('Could not delete tag', text)
    }
  }

  const items = tags.data?.data ?? []

  return (
    <Card className="settings__section">
      <SectionHeader title="Tags" headingLevel="h2" />
      <p className="text-secondary">Deleting a tag detaches it from every document.</p>
      {message && <p className="settings__message">{message}</p>}
      {tags.loading && !tags.data ? (
        <Skeleton height={64} />
      ) : items.length > 0 ? (
        <div className="settings__tag-list">
          {items.map((tag) => (
            <div key={tag.name} className="settings__tag-row">
              <span>
                <span className="settings__tag-name">#{tag.name}</span>{' '}
                <span className="settings__tag-count">{tag.count ?? 0} docs</span>
              </span>
              <Button variant="ghost" size="sm" onClick={() => remove(tag.name)}>
                Delete
              </Button>
            </div>
          ))}
        </div>
      ) : (
        <EmptyState
          title="No tags yet"
          description="Tags appear here as you add them to documents."
        />
      )}
    </Card>
  )
}
