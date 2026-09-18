import { useEffect, useState } from 'react'
import MDEditor from '@uiw/react-md-editor/nohighlight'
import '@uiw/react-md-editor/markdown-editor.css'
import { ApiError, api } from '../api/client'
import type { Document, Paginated } from '../api/types'
import {
  Button,
  Card,
  EmptyState,
  Input,
  MarkdownView,
  SectionHeader,
  Skeleton,
  TagPill,
} from '../components'
import { usePrivateSession } from '../context/privateSession'
import { useAsync } from '../hooks/useAsync'
import { relativeTime } from '../utils/time'

/** PrivateArchivePage implements the locked Private Archive UX (PRD 3.2). */
export function PrivateArchivePage() {
  const { status, requestUnlock, lock } = usePrivateSession()

  // Prompt for the password as soon as the locked route is visited.
  useEffect(() => {
    if (status === 'locked') requestUnlock()
  }, [status, requestUnlock])

  if (status === 'locked') {
    return (
      <div className="container private-page">
        <Card className="private-locked">
          <div style={{ fontSize: '2.5rem' }} aria-hidden="true">
            🔒
          </div>
          <SectionHeader title="Private Archive is locked" headingLevel="h1" />
          <p className="text-secondary">
            This section contains confidential notes. Unlock it with the master password.
          </p>
          <Button onClick={requestUnlock}>Unlock Private Archive</Button>
        </Card>
      </div>
    )
  }

  return <UnlockedArchive onLock={lock} />
}

function UnlockedArchive({ onLock }: { onLock: () => Promise<void> }) {
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [mode, setMode] = useState<'view' | 'edit' | 'create'>('view')

  const { data, error, loading, reload } = useAsync<Paginated<Document>>(
    () => api.private.list(),
    [],
  )
  const documents = data?.data ?? []
  // Selection falls back to the first document, so no state sync is needed.
  const selected = documents.find((doc) => doc.id === selectedId) ?? documents[0] ?? null
  const errorMessage = error
    ? error instanceof ApiError
      ? error.message
      : 'Failed to load private documents.'
    : null

  return (
    <div className="container private-page">
      <div className="private-page__header">
        <div>
          <SectionHeader title="Private Archive" headingLevel="h1" />
          <span className="private-badge">🔒 Unlocked</span>
        </div>
        <div className="row">
          <Button
            size="sm"
            onClick={() => {
              setMode('create')
              setSelectedId(null)
            }}
          >
            New private note
          </Button>
          <Button size="sm" variant="secondary" onClick={() => void onLock()}>
            Lock
          </Button>
        </div>
      </div>

      {error && (
        <div className="dashboard-error" role="alert">
          <span>{errorMessage}</span>
          <Button variant="secondary" size="sm" onClick={() => reload()}>
            Retry
          </Button>
        </div>
      )}

      <div className="private-grid">
        <Card className="private-list">
          {loading ? (
            <div className="stack" style={{ padding: 'var(--space-4)' }}>
              <Skeleton height={18} />
              <Skeleton height={18} width="80%" />
              <Skeleton height={18} width="60%" />
            </div>
          ) : documents.length > 0 ? (
            documents.map((doc) => (
              <button
                key={doc.id}
                type="button"
                className={`private-list__item${
                  doc.id === selectedId && mode === 'view' ? ' private-list__item--active' : ''
                }`}
                onClick={() => {
                  setSelectedId(doc.id)
                  setMode('view')
                }}
              >
                <span className="private-list__title">{doc.title}</span>
                <span className="private-list__meta">{relativeTime(doc.updated_at)}</span>
              </button>
            ))
          ) : (
            <p className="kb__empty">No private notes yet.</p>
          )}
        </Card>

        <div>
          {mode === 'create' ? (
            <PrivateEditor
              onCancel={() => setMode('view')}
              onSaved={async (doc) => {
                await reload()
                setSelectedId(doc.id)
                setMode('view')
              }}
            />
          ) : mode === 'edit' && selected ? (
            <PrivateEditor
              initial={selected}
              onCancel={() => setMode('view')}
              onSaved={async (doc) => {
                await reload()
                setSelectedId(doc.id)
                setMode('view')
              }}
              onDeleted={async () => {
                setSelectedId(null)
                setMode('view')
                await reload()
              }}
            />
          ) : selected ? (
            <PrivateView
              document={selected}
              onEdit={() => setMode('edit')}
              onDeleted={async () => {
                setSelectedId(null)
                await reload()
              }}
            />
          ) : (
            !loading && (
              <EmptyState
                title="No note selected"
                description="Choose a note from the list or create a new private note."
                action={<Button onClick={() => setMode('create')}>New private note</Button>}
              />
            )
          )}
        </div>
      </div>
    </div>
  )
}

function PrivateView({
  document,
  onEdit,
  onDeleted,
}: {
  document: Document
  onEdit: () => void
  onDeleted: () => Promise<void>
}) {
  const [deleting, setDeleting] = useState(false)

  async function remove() {
    if (!window.confirm('Delete this private note? This cannot be undone.')) return
    setDeleting(true)
    try {
      await api.private.remove(document.id)
      await onDeleted()
    } finally {
      setDeleting(false)
    }
  }

  return (
    <article>
      <div className="viewer__header">
        <div>
          <h2 className="viewer__title">{document.title}</h2>
          <div className="viewer__meta">
            <span>Updated {relativeTime(document.updated_at)}</span>
          </div>
        </div>
        <div className="row">
          <Button variant="secondary" onClick={onEdit}>
            Edit
          </Button>
          <Button variant="danger" onClick={remove} disabled={deleting}>
            {deleting ? 'Deleting…' : 'Delete'}
          </Button>
        </div>
      </div>
      {document.tags.length > 0 && (
        <div className="viewer__tags">
          {document.tags.map((tag) => (
            <TagPill key={tag}>{tag}</TagPill>
          ))}
        </div>
      )}
      <Card className="viewer__content">
        {document.content.trim() ? (
          <MarkdownView content={document.content} />
        ) : (
          <p className="text-secondary">This note has no content yet.</p>
        )}
      </Card>
    </article>
  )
}

function PrivateEditor({
  initial,
  onCancel,
  onSaved,
  onDeleted,
}: {
  initial?: Document
  onCancel: () => void
  onSaved: (doc: Document) => void | Promise<void>
  onDeleted?: () => void | Promise<void>
}) {
  const [title, setTitle] = useState(initial?.title ?? '')
  const [content, setContent] = useState(initial?.content ?? '')
  const [tags, setTags] = useState<string[]>(initial?.tags ?? [])
  const [tagInput, setTagInput] = useState('')
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function addTags() {
    const parsed = tagInput
      .split(',')
      .map((tag) => tag.trim())
      .filter(Boolean)
    if (parsed.length === 0) return
    setTags((current) => {
      const merged = [...current]
      for (const tag of parsed) {
        if (!merged.some((existing) => existing.toLowerCase() === tag.toLowerCase())) {
          merged.push(tag)
        }
      }
      return merged
    })
    setTagInput('')
  }

  async function save() {
    if (!title.trim()) {
      setError('Title is required.')
      return
    }
    setSaving(true)
    setError(null)
    try {
      const payload = { title: title.trim(), content, section: 'private' as const, tags }
      const doc = initial
        ? await api.private.update(initial.id, payload)
        : await api.private.create(payload)
      await onSaved(doc)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Failed to save the note.')
    } finally {
      setSaving(false)
    }
  }

  async function remove() {
    if (!initial || !onDeleted) return
    if (!window.confirm('Delete this private note? This cannot be undone.')) return
    setDeleting(true)
    try {
      await api.private.remove(initial.id)
      await onDeleted()
    } finally {
      setDeleting(false)
    }
  }

  return (
    <div className="private-editor">
      <Card>
        <div className="stack">
          <Input
            label="Title"
            value={title}
            onChange={(event) => setTitle(event.target.value)}
            placeholder="Private note title"
          />
          <Input
            label="Tags"
            value={tagInput}
            onChange={(event) => setTagInput(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                event.preventDefault()
                addTags()
              }
            }}
            placeholder="Add tags, comma separated"
            hint="Press Enter to add"
          />
          {tags.length > 0 && (
            <div className="editor__chips">
              {tags.map((tag) => (
                <TagPill
                  key={tag}
                  onClick={() => setTags((current) => current.filter((value) => value !== tag))}
                >
                  {tag} ✕
                </TagPill>
              ))}
            </div>
          )}
        </div>
      </Card>

      <div className="editor__markdown">
        <MDEditor
          value={content}
          onChange={(value) => setContent(value ?? '')}
          height={420}
          preview="live"
          visibleDragbar
        />
      </div>

      {error && (
        <p className="editor__status editor__status--error" role="alert">
          {error}
        </p>
      )}

      <div className="private-editor__actions">
        <Button variant="secondary" onClick={onCancel} disabled={saving || deleting}>
          Cancel
        </Button>
        {initial && onDeleted && (
          <Button variant="danger" onClick={remove} disabled={saving || deleting}>
            {deleting ? 'Deleting…' : 'Delete'}
          </Button>
        )}
        <Button onClick={save} disabled={saving || deleting}>
          {saving ? 'Saving…' : initial ? 'Save changes' : 'Create note'}
        </Button>
      </div>
    </div>
  )
}
