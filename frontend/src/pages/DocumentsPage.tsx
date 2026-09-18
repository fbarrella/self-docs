import { useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Document, Paginated, Section, Tag } from '../api/types'
import {
  Button,
  ButtonLink,
  DocumentCard,
  EmptyState,
  ImportDialog,
  SearchBar,
  SectionHeader,
  Skeleton,
  TagPill,
} from '../components'
import { useAsync } from '../hooks/useAsync'
import { routes } from '../routes'
import { sectionMeta } from '../sectionMeta'
import type { ListDocumentsParams } from '../api/client'
import { useState } from 'react'

export interface DocumentsPageProps {
  /** Restricts the listing to a single section; omitted for "all documents". */
  section?: Section
  title?: string
  description?: string
}

type SortKey = 'updated_at' | 'created_at' | 'title'

/**
 * DocumentsPage is the shared listing for Documents, Workflows, and Cheat
 * Sheets. Tag filtering, sorting, and pagination are reflected in the URL so
 * the view is shareable and survives reloads.
 */
export function DocumentsPage({ section, title, description }: DocumentsPageProps) {
  const [params, setParams] = useSearchParams()

  const activeTag = params.get('tag') ?? ''
  const sort = (params.get('sort') as SortKey) ?? 'updated_at'
  const order = params.get('order') === 'asc' ? 'asc' : 'desc'
  const page = Math.max(1, Number(params.get('page') ?? '1') || 1)
  const query = params.get('q') ?? ''

  const [importOpen, setImportOpen] = useState(false)
  // A null draft means "show the committed URL query"; editing sets a draft so
  // the input stays responsive while the URL only updates on submit/reset.
  const [draftQuery, setDraftQuery] = useState<string | null>(null)
  const localQuery = draftQuery ?? query

  const load = useCallback((): Promise<Paginated<Document>> => {
    const request: ListDocumentsParams = {
      sort,
      order,
      page,
      page_size: 12,
    }
    if (section) request.section = section
    if (activeTag) request.tag = activeTag
    if (query) request.q = query
    return api.documents.list(request)
  }, [section, activeTag, query, sort, order, page])

  const { data, error, loading, reload } = useAsync<Paginated<Document>>(load, [
    section,
    activeTag,
    query,
    sort,
    order,
    page,
  ])

  const meta = section ? sectionMeta[section] : undefined
  const pageTitle = title ?? meta?.title ?? 'All Documents'
  const pageDescription = description ?? meta?.description ?? 'Everything in your knowledge base.'

  function updateParam(key: string, value: string | undefined) {
    const next = new URLSearchParams(params)
    if (value) next.set(key, value)
    else next.delete(key)
    if (key !== 'page') next.delete('page')
    setParams(next, { replace: true })
  }

  function toggleTag(tag: string) {
    updateParam('tag', activeTag === tag ? undefined : tag)
  }

  function submitSearch(value: string) {
    setDraftQuery(null)
    updateParam('q', value.trim() || undefined)
  }

  const totalPages = data?.pagination.total_pages ?? 0
  const documents = data?.data ?? []

  return (
    <div className="container" style={{ paddingBlock: 'var(--space-7)' }}>
      <SectionHeader title={pageTitle} headingLevel="h1" />
      <p className="text-secondary">{pageDescription}</p>

      <div className="doc-toolbar">
        <div className="doc-filters">
          <ButtonLink to={routes.editor()} size="sm">
            New document
          </ButtonLink>
          <Button size="sm" variant="secondary" onClick={() => setImportOpen(true)}>
            Import
          </Button>
          {activeTag ? (
            <TagPill active onClick={() => toggleTag(activeTag)} count={undefined}>
              {activeTag} ✕
            </TagPill>
          ) : (
            <PopularTagFilters onSelect={toggleTag} />
          )}
        </div>

        <div className="doc-toolbar__sort">
          <label htmlFor="doc-sort">Sort</label>
          <select
            id="doc-sort"
            className="doc-toolbar__select"
            value={`${sort}:${order}`}
            onChange={(event) => {
              const [nextSort, nextOrder] = event.target.value.split(':')
              const next = new URLSearchParams(params)
              next.set('sort', nextSort)
              next.set('order', nextOrder)
              next.delete('page')
              setParams(next, { replace: true })
            }}
          >
            <option value="updated_at:desc">Recently updated</option>
            <option value="updated_at:asc">Least recently updated</option>
            <option value="created_at:desc">Newest first</option>
            <option value="created_at:asc">Oldest first</option>
            <option value="title:asc">Title A–Z</option>
            <option value="title:desc">Title Z–A</option>
          </select>
        </div>
      </div>

      <div style={{ marginBottom: 'var(--space-5)', maxWidth: '32rem' }}>
        <SearchBar
          value={localQuery}
          onChange={setDraftQuery}
          onSubmit={submitSearch}
          placeholder="Filter by title..."
          ariaLabel="Filter documents by title"
        />
      </div>

      <Button
        variant="ghost"
        size="sm"
        onClick={() => {
          setDraftQuery(null)
          setParams(new URLSearchParams(), { replace: true })
        }}
      >
        Reset filters
      </Button>

      {error && (
        <div className="dashboard-error" role="alert" style={{ marginTop: 'var(--space-4)' }}>
          <span>Could not load documents. {error.message}</span>
          <Button variant="secondary" size="sm" onClick={reload}>
            Retry
          </Button>
        </div>
      )}

      <div style={{ marginTop: 'var(--space-5)' }}>
        {loading && !data ? (
          <div className="doc-list">
            {Array.from({ length: 4 }).map((_, index) => (
              <Skeleton key={index} height={140} radius="var(--radius-lg)" />
            ))}
          </div>
        ) : documents.length > 0 ? (
          <>
            <div className="doc-list">
              {documents.map((doc) => (
                <DocumentCard key={doc.id} document={doc} onTagClick={toggleTag} />
              ))}
            </div>
            {totalPages > 1 && (
              <div className="doc-pagination">
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => updateParam('page', String(page - 1))}
                >
                  Previous
                </Button>
                <span className="doc-pagination__status">
                  Page {page} of {totalPages}
                </span>
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={page >= totalPages}
                  onClick={() => updateParam('page', String(page + 1))}
                >
                  Next
                </Button>
              </div>
            )}
          </>
        ) : (
          <EmptyState
            title="No documents found"
            description={
              activeTag || query
                ? 'Try clearing the filters or using a different search.'
                : 'Create your first document to get started.'
            }
            action={<ButtonLink to={routes.editor()}>New document</ButtonLink>}
          />
        )}
      </div>

      <ImportDialog
        open={importOpen}
        onClose={() => setImportOpen(false)}
        onImported={reload}
        defaultSection={section ?? 'workflow'}
      />
    </div>
  )
}

function PopularTagFilters({ onSelect }: { onSelect: (tag: string) => void }) {
  const { data } = useAsync<{ data: Tag[] }>(() => api.tags.popular(8), [])
  if (!data || data.data.length === 0) return null
  return (
    <>
      {data.data.map((tag) => (
        <TagPill key={tag.name} count={tag.count} onClick={() => onSelect(tag.name)}>
          {tag.name}
        </TagPill>
      ))}
    </>
  )
}
