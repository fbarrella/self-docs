import { useCallback, useRef, useState } from 'react'
import type { ChangeEvent, DragEvent } from 'react'
import { ApiError, api } from '../api/client'
import type { ImportResponse } from '../api/client'
import type { Section } from '../api/types'
import { Button, Input, Modal } from '.'
import { useToast } from '../context/toast'
import { sectionMeta } from '../sectionMeta'
import { routes } from '../routes'
import { useNavigate } from 'react-router-dom'

export interface ImportDialogProps {
  open: boolean
  onClose: () => void
  onImported?: () => void
  defaultSection?: Section
}

const importSections: Section[] = ['workflow', 'project_note', 'cheat_sheet']

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

/**
 * ImportDialog uploads one or more Markdown files to POST
 * /api/documents/import (PRD 3.1) with drag-and-drop, a removable file list,
 * a target section, optional default tags, and a per-file result summary.
 */
export function ImportDialog({
  open,
  onClose,
  onImported,
  defaultSection = 'workflow',
}: ImportDialogProps) {
  const navigate = useNavigate()
  const toast = useToast()
  const inputRef = useRef<HTMLInputElement>(null)

  const [files, setFiles] = useState<File[]>([])
  const [section, setSection] = useState<Section>(defaultSection)
  const [tags, setTags] = useState('')
  const [dragging, setDragging] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [result, setResult] = useState<ImportResponse | null>(null)
  const [error, setError] = useState<string | null>(null)

  const reset = useCallback(() => {
    setFiles([])
    setTags('')
    setSection(defaultSection)
    setResult(null)
    setError(null)
    setUploading(false)
    setDragging(false)
  }, [defaultSection])

  function close() {
    reset()
    onClose()
  }

  function addFiles(incoming: FileList | null) {
    if (!incoming) return
    const accepted = Array.from(incoming).filter((file) => file.name.toLowerCase().endsWith('.md'))
    if (accepted.length === 0) {
      setError('Only .md files can be imported.')
      return
    }
    setError(null)
    setFiles((current) => {
      const merged = [...current]
      for (const file of accepted) {
        if (
          !merged.some((existing) => existing.name === file.name && existing.size === file.size)
        ) {
          merged.push(file)
        }
      }
      return merged
    })
  }

  function handleDrop(event: DragEvent<HTMLDivElement>) {
    event.preventDefault()
    setDragging(false)
    addFiles(event.dataTransfer.files)
  }

  function handleInput(event: ChangeEvent<HTMLInputElement>) {
    addFiles(event.target.files)
    event.target.value = ''
  }

  async function upload() {
    if (files.length === 0) {
      setError('Add at least one .md file.')
      return
    }
    setUploading(true)
    setError(null)
    try {
      const form = new FormData()
      for (const file of files) form.append('files', file)
      form.append('section', section)
      if (tags.trim()) form.append('tags', tags.trim())
      const response = await api.documents.import(form)
      setResult(response)
      if (response.summary.created > 0) {
        toast.success(
          `Imported ${response.summary.created} file${response.summary.created === 1 ? '' : 's'}`,
          response.summary.skipped > 0 ? `${response.summary.skipped} skipped` : undefined,
        )
        onImported?.()
      } else {
        toast.info('Nothing imported', 'No valid .md files were found.')
      }
    } catch (err) {
      const message = err instanceof ApiError ? err.message : 'Import failed.'
      setError(message)
      toast.error('Import failed', message)
    } finally {
      setUploading(false)
    }
  }

  return (
    <Modal open={open} title="Import Markdown" onClose={close}>
      {result ? (
        <div>
          <div className="import-summary">
            <span className="import-summary__created">{result.summary.created} created</span>
            <span className="import-summary__skipped">{result.summary.skipped} skipped</span>
            <span className="import-summary__failed">{result.summary.failed} failed</span>
          </div>
          <div className="import-results">
            {result.data.map((entry) => (
              <div key={entry.filename} className="import-result-row">
                <span className="import-filelist__name">{entry.filename}</span>
                <span className="import-result-row__status">
                  {entry.status}
                  {entry.reason ? ` (${entry.reason})` : ''}
                </span>
              </div>
            ))}
          </div>
          <div className="ui-modal__actions">
            <Button variant="secondary" onClick={reset}>
              Import more
            </Button>
            <Button
              onClick={() => {
                close()
                navigate(section === 'workflow' ? routes.workflows : routes.documents)
              }}
            >
              Done
            </Button>
          </div>
        </div>
      ) : (
        <div>
          <div
            className={`import-dropzone${dragging ? ' import-dropzone--active' : ''}`}
            role="button"
            tabIndex={0}
            onClick={() => inputRef.current?.click()}
            onKeyDown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault()
                inputRef.current?.click()
              }
            }}
            onDragOver={(event) => {
              event.preventDefault()
              setDragging(true)
            }}
            onDragLeave={() => setDragging(false)}
            onDrop={handleDrop}
          >
            <span aria-hidden="true" style={{ fontSize: '1.75rem' }}>
              📥
            </span>
            <span className="import-dropzone__title">Drag &amp; drop .md files here</span>
            <span>or click to browse</span>
            <input
              ref={inputRef}
              type="file"
              accept=".md,text/markdown"
              multiple
              hidden
              onChange={handleInput}
            />
          </div>

          {files.length > 0 && (
            <ul className="import-filelist">
              {files.map((file) => (
                <li key={`${file.name}-${file.size}`} className="import-filelist__item">
                  <span className="import-filelist__name">{file.name}</span>
                  <span className="import-filelist__size">{formatSize(file.size)}</span>
                  <button
                    type="button"
                    className="import-remove"
                    aria-label={`Remove ${file.name}`}
                    onClick={() => setFiles((current) => current.filter((f) => f !== file))}
                  >
                    ✕
                  </button>
                </li>
              ))}
            </ul>
          )}

          <div className="stack" style={{ marginTop: 'var(--space-4)' }}>
            <div className="ui-input-wrap">
              <label className="ui-input-label" htmlFor="import-section">
                Target section
              </label>
              <select
                id="import-section"
                className="doc-toolbar__select"
                value={section}
                onChange={(event) => setSection(event.target.value as Section)}
              >
                {importSections.map((value) => (
                  <option key={value} value={value}>
                    {sectionMeta[value].title}
                  </option>
                ))}
              </select>
            </div>
            <Input
              label="Default tags"
              value={tags}
              onChange={(event) => setTags(event.target.value)}
              placeholder="e.g. imported, reference"
              hint="Applied to every imported file"
            />
          </div>

          {error && (
            <p
              className="editor__status editor__status--error"
              role="alert"
              style={{ marginTop: 'var(--space-3)' }}
            >
              {error}
            </p>
          )}

          <div className="ui-modal__actions">
            <Button variant="secondary" onClick={close} disabled={uploading}>
              Cancel
            </Button>
            <Button onClick={upload} disabled={uploading || files.length === 0}>
              {uploading ? 'Importing…' : `Import ${files.length || ''}`.trim()}
            </Button>
          </div>
        </div>
      )}
    </Modal>
  )
}
