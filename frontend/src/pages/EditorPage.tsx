import { useCallback, useMemo, useState } from 'react'
import MDEditor from '@uiw/react-md-editor/nohighlight'
import '@uiw/react-md-editor/markdown-editor.css'
import { useNavigate, useParams } from 'react-router-dom'
import { ApiError, api } from '../api/client'
import type { Document, Section } from '../api/types'
import { Button, Card, Input, Spinner, TagPill } from '../components'
import { useAsync } from '../hooks/useAsync'
import { routes } from '../routes'
import { sectionMeta } from '../sectionMeta'

const sectionOptions: Section[] = ['workflow', 'project_note', 'cheat_sheet']

function parseTags(raw: string): string[] {
  const seen = new Set<string>()
  const tags: string[] = []
  for (const part of raw.split(',')) {
    const tag = part.trim()
    if (!tag) continue
    const key = tag.toLowerCase()
    if (seen.has(key)) continue
    seen.add(key)
    tags.push(tag)
  }
  return tags
}

/**
 * EditorPage creates or edits a document (PRD 3.1). New documents render an
 * empty form; an existing document is loaded first and the form is remounted
 * (keyed by id) so its initial state always matches the fetched data.
 */
export function EditorPage() {
  const { id } = useParams<{ id: string }>()

  if (!id) {
    return <EditorForm />
  }
  return <EditorLoader id={id} />
}

function EditorLoader({ id }: { id: string }) {
  const navigate = useNavigate()
  const { data, error, loading, reload } = useAsync<Document>(() => api.documents.get(id), [id])

  if (loading) {
    return (
      <div className="container editor">
        <div className="row">
          <Spinner size="lg" />
          <span>Loading document…</span>
        </div>
      </div>
    )
  }

  if (error || !data) {
    return (
      <div className="container editor">
        <div className="dashboard-error" role="alert">
          <span>{error instanceof ApiError ? error.message : 'Failed to load the document.'}</span>
          <div className="row">
            <Button variant="secondary" size="sm" onClick={reload}>
              Retry
            </Button>
            <Button variant="ghost" size="sm" onClick={() => navigate(routes.documents)}>
              Back to documents
            </Button>
          </div>
        </div>
      </div>
    )
  }

  return <EditorForm key={id} initial={data} />
}

function EditorForm({ initial }: { initial?: Document }) {
  const navigate = useNavigate()
  const isEditing = Boolean(initial)

  const [title, setTitle] = useState(initial?.title ?? '')
  const [content, setContent] = useState(initial?.content ?? '')
  const [section, setSection] = useState<Section>(initial?.section ?? 'workflow')
  const [tagInput, setTagInput] = useState('')
  const [tags, setTags] = useState<string[]>(initial?.tags ?? [])
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [status, setStatus] = useState<{ kind: 'error' | 'success'; message: string } | null>(null)

  const tagPreview = useMemo(() => parseTags(tagInput), [tagInput])

  function addTags() {
    if (tagPreview.length === 0) return
    setTags((current) => {
      const merged = [...current]
      for (const tag of tagPreview) {
        if (!merged.some((existing) => existing.toLowerCase() === tag.toLowerCase())) {
          merged.push(tag)
        }
      }
      return merged
    })
    setTagInput('')
  }

  function removeTag(tag: string) {
    setTags((current) => current.filter((value) => value !== tag))
  }

  const save = useCallback(async () => {
    if (!title.trim()) {
      setStatus({ kind: 'error', message: 'Title is required.' })
      return
    }
    setSaving(true)
    setStatus(null)
    try {
      const payload = { title: title.trim(), content, section, tags }
      const doc =
        isEditing && initial
          ? await api.documents.update(initial.id, payload)
          : await api.documents.create(payload)
      setStatus({ kind: 'success', message: 'Saved.' })
      if (!isEditing) navigate(routes.document(doc.id), { replace: true })
    } catch (err) {
      setStatus({
        kind: 'error',
        message: err instanceof ApiError ? err.message : 'Failed to save the document.',
      })
    } finally {
      setSaving(false)
    }
  }, [title, content, section, tags, isEditing, initial, navigate])

  const remove = useCallback(async () => {
    if (!initial) return
    if (!window.confirm('Delete this document? This cannot be undone.')) return
    setDeleting(true)
    setStatus(null)
    try {
      await api.documents.remove(initial.id)
      navigate(routes.documents, { replace: true })
    } catch (err) {
      setStatus({
        kind: 'error',
        message: err instanceof ApiError ? err.message : 'Failed to delete the document.',
      })
      setDeleting(false)
    }
  }, [initial, navigate])

  return (
    <div className="container editor">
      <div className="editor__header">
        <h1 className="editor__heading">{isEditing ? 'Edit document' : 'New document'}</h1>
        <div className="editor__actions">
          {status && (
            <span className={`editor__status editor__status--${status.kind}`} role="status">
              {status.message}
            </span>
          )}
          {isEditing && (
            <Button variant="danger" onClick={remove} disabled={deleting || saving}>
              {deleting ? 'Deleting…' : 'Delete'}
            </Button>
          )}
          <Button onClick={save} disabled={saving || deleting}>
            {saving ? 'Saving…' : isEditing ? 'Save changes' : 'Create document'}
          </Button>
        </div>
      </div>

      <div className="editor__grid">
        <aside className="editor__sidebar">
          <Card>
            <div className="editor__field">
              <Input
                label="Title"
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                placeholder="Document title"
                required
              />
              <div className="ui-input-wrap">
                <label className="ui-input-label" htmlFor="editor-section">
                  Section
                </label>
                <select
                  id="editor-section"
                  className="doc-toolbar__select"
                  value={section}
                  onChange={(event) => setSection(event.target.value as Section)}
                >
                  {sectionOptions.map((value) => (
                    <option key={value} value={value}>
                      {sectionMeta[value].title}
                    </option>
                  ))}
                </select>
              </div>
            </div>
          </Card>

          <Card>
            <p className="editor__panel-title">Tags</p>
            <Input
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
            {tagPreview.length > 0 && (
              <div className="editor__chips">
                {tagPreview.map((tag) => (
                  <TagPill key={tag}>{tag}</TagPill>
                ))}
              </div>
            )}
            {tags.length > 0 && (
              <div className="editor__chips">
                {tags.map((tag) => (
                  <TagPill key={tag} onClick={() => removeTag(tag)}>
                    {tag} ✕
                  </TagPill>
                ))}
              </div>
            )}
          </Card>
        </aside>

        <div className="editor__markdown">
          <MDEditor
            value={content}
            onChange={(value) => setContent(value ?? '')}
            height={560}
            preview="live"
            visibleDragbar
            textareaProps={{ placeholder: 'Write your Markdown here…' }}
          />
        </div>
      </div>
    </div>
  )
}
