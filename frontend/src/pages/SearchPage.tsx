import { Fragment, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Paginated, SearchResult, Section } from '../api/types'
import { Button, Card, EmptyState, SectionHeader, Skeleton, TagPill } from '../components'
import { useAsync } from '../hooks/useAsync'
import { routes } from '../routes'
import { sectionMeta } from '../sectionMeta'
import { relativeTime } from '../utils/time'

/**
 * highlightSnippet converts the server's `<mark>` highlighted snippet into
 * React nodes so it can be rendered without dangerouslySetInnerHTML.
 */
function highlightSnippet(snippet: string) {
  const parts = snippet.split(/(<mark>.*?<\/mark>)/gi)
  return parts.map((part, index) => {
    const match = /^<mark>(.*?)<\/mark>$/i.exec(part)
    if (match) return <mark key={index}>{match[1]}</mark>
    return <Fragment key={index}>{part}</Fragment>
  })
}

/** SearchPage implements the global full-text search UX (PRD 3.3). */
export function SearchPage() {
  const [params, setParams] = useSearchParams()
  const query = params.get('q') ?? ''
  const page = Math.max(1, Number(params.get('page') ?? '1') || 1)

  // Track the query the draft was initialized from so external navigation
  // (e.g., header or dashboard search) resets the input during render, which
  // avoids a setState-in-effect.
  const [draft, setDraft] = useState(query)
  const [syncedQuery, setSyncedQuery] = useState(query)
  if (query !== syncedQuery) {
    setSyncedQuery(query)
    setDraft(query)
  }

  const { data, error, loading, reload } = useAsync<Paginated<SearchResult> & { query: string }>(
    () => api.search({ q: query, page, page_size: 20 }),
    [query, page],
  )

  function commit(value: string) {
    const next = new URLSearchParams()
    if (value.trim()) next.set('q', value.trim())
    setParams(next, { replace: false })
  }

  const results = data?.data ?? []
  const totalPages = data?.pagination.total_pages ?? 0

  return (
    <div className="container search-page">
      <div className="search-page__header">
        <SectionHeader title="Search" headingLevel="h1" />
        <form
          className="search-page__form"
          role="search"
          onSubmit={(event) => {
            event.preventDefault()
            commit(draft)
          }}
        >
          <div className="ui-searchbar">
            <svg
              className="ui-searchbar__icon"
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <circle cx="11" cy="11" r="7" />
              <line x1="21" y1="21" x2="16.65" y2="16.65" />
            </svg>
            <input
              className="ui-searchbar__input"
              type="search"
              value={draft}
              placeholder="Search your documentation..."
              aria-label="Global search"
              onChange={(event) => setDraft(event.target.value)}
            />
            <button className="ui-searchbar__button" type="submit">
              Search
            </button>
          </div>
        </form>
        {query && !loading && data && (
          <p className="text-secondary">
            {data.pagination.total} result{data.pagination.total === 1 ? '' : 's'} for{' '}
            <strong>{query}</strong>
          </p>
        )}
      </div>

      {error && (
        <div className="dashboard-error" role="alert">
          <span>Search failed. {error.message}</span>
          <Button variant="secondary" size="sm" onClick={reload}>
            Retry
          </Button>
        </div>
      )}

      {!query ? (
        <div className="search-empty">
          <EmptyState
            title="Search your knowledge base"
            description="Find documents by title, tag, or content. Private notes are never included."
          />
        </div>
      ) : loading && !data ? (
        <div className="search-results">
          {Array.from({ length: 3 }).map((_, index) => (
            <Skeleton key={index} height={120} radius="var(--radius-lg)" />
          ))}
        </div>
      ) : results.length > 0 ? (
        <>
          <div className="search-results">
            {results.map((result) => (
              <Card key={result.id} interactive className="search-result">
                <Link to={routes.document(result.id)}>
                  <div className="search-result__header">
                    <h2 className="search-result__title">{result.title}</h2>
                    <span className="search-result__section">
                      {sectionMeta[result.section as Section]?.title ?? result.section}
                    </span>
                  </div>
                  {result.snippet && (
                    <p className="search-result__snippet">{highlightSnippet(result.snippet)}</p>
                  )}
                  <div className="search-result__footer">
                    <div className="search-result__tags">
                      {result.tags.slice(0, 4).map((tag) => (
                        <TagPill key={tag}>{tag}</TagPill>
                      ))}
                    </div>
                    <span className="search-result__time">{relativeTime(result.updated_at)}</span>
                  </div>
                </Link>
              </Card>
            ))}
          </div>

          {totalPages > 1 && (
            <div className="doc-pagination">
              <Button
                variant="secondary"
                size="sm"
                disabled={page <= 1}
                onClick={() => {
                  const next = new URLSearchParams(params)
                  next.set('page', String(page - 1))
                  setParams(next)
                }}
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
                onClick={() => {
                  const next = new URLSearchParams(params)
                  next.set('page', String(page + 1))
                  setParams(next)
                }}
              >
                Next
              </Button>
            </div>
          )}
        </>
      ) : (
        <div className="search-empty">
          <EmptyState
            title="No results found"
            description={`Nothing matches "${query}". Try different keywords or check your spelling.`}
          />
        </div>
      )}
    </div>
  )
}
