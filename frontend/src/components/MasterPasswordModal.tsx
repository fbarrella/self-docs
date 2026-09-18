import { useState } from 'react'
import { Button, Input, Modal } from '../components'
import { usePrivateSession } from '../context/privateSession'

/**
 * MasterPasswordModal prompts for the Private Archive master password. It
 * renders from the session context so any locked entry point (dashboard card,
 * header menu, /private route) can trigger it.
 */
export function MasterPasswordModal() {
  const { unlockOpen, closeUnlock, unlock } = usePrivateSession()
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  function reset() {
    setPassword('')
    setError(null)
    setSubmitting(false)
  }

  async function handleSubmit() {
    if (!password) {
      setError('Enter the master password.')
      return
    }
    setSubmitting(true)
    setError(null)
    const result = await unlock(password)
    setSubmitting(false)
    if (!result.ok) {
      setError(result.error ?? 'Unlock failed.')
      return
    }
    reset()
  }

  return (
    <Modal
      open={unlockOpen}
      title="Unlock Private Archive"
      onClose={() => {
        reset()
        closeUnlock()
      }}
    >
      <p className="text-secondary" style={{ marginTop: 0 }}>
        The Private Archive is protected. Enter the master password to view your confidential notes.
      </p>
      <form
        onSubmit={(event) => {
          event.preventDefault()
          void handleSubmit()
        }}
      >
        <Input
          label="Master password"
          type="password"
          value={password}
          autoFocus
          autoComplete="current-password"
          onChange={(event) => setPassword(event.target.value)}
          error={error ?? undefined}
        />
        <div className="ui-modal__actions">
          <Button
            type="button"
            variant="secondary"
            onClick={() => {
              reset()
              closeUnlock()
            }}
            disabled={submitting}
          >
            Cancel
          </Button>
          <Button type="submit" disabled={submitting}>
            {submitting ? 'Unlocking…' : 'Unlock'}
          </Button>
        </div>
      </form>
    </Modal>
  )
}
