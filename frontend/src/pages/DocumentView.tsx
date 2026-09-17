import { useMemo } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { ApiError, api } from '../api/client'
import type { Document, DocumentNode } from '../api/types'
import {
  Button,
  Card,
  EmptyState,
  MarkdownView,
  SectionHeader,
  Spinner,
  TagPill,
  TreeNav,
  collectHeadings,
  findNodePath,
} from '../components'
import { useAsync } from '../hooks/useAsync'
import { routes } from '../routes'
import { sectionMeta } from '../sectionMeta'
import { formatAbsolute, relativeTime } from '../utils/time'

/**
 * DocumentView renders a single document (PRD 3.2): rendered Markdown, a table
 * of contents, tags, metadata, and — for Project Notes — breadcrumbs plus a
 * sidebar tree for navigating the hierarchy.
 */
export function DocumentView() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data, error, loading, reload } = useAsync<Document>(() => api.documents.get(id), [id])

  const isProjectNote = data?.section === 'project_note'

  const tree = useAsync<{ data: DocumentNode[] }>(
    () => (isProjectNote ? api.documents.tree('project_note') : Promise.resolve({ data: [] })),
    [isProjectNote],
  )

  const headings = useMemo(() => (data ? collectHeadings(data.content) : []), [data])
  const breadcrumbs = useMemo(
    () => (isProjectNote && tree.data ? findNodePath(tree.data.data, id) : []),
    [isProjectNote, tree.data, id],
  )

  if (loading) {
    return (
      <div className="container viewer">
        <div className="row">
          <Spinner size="lg" />
          <span>Loading document…</span>
        </div>
      </div>
    )
  }

  if (error || !data) {
    const notFound = error instanceof ApiError && error.status === 404
    return (
      <div className="container viewer">
        <EmptyState
          title={notFound ? 'Document not found' : 'Could not load document'}
          description={
            notFound
              ? 'It may have been deleted or is private.'
              : error instanceof ApiError
                ? error.message
                : 'Please try again.'
          }
          action={
            <div className="row">
              <Button variant="secondary" onClick={reload}>
                Retry
              </Button>
              <Button variant="ghost" onClick={() => navigate(routes.documents)}>
                Back to documents
              </Button>
            </div>
          }
        />
      </div>
    )
  }

  return (
    <div className="container viewer">
      {isProjectNote && breadcrumbs.length > 0 && (
        <nav className="breadcrumbs" aria-label="Breadcrumb">
          <Link to={routes.knowledgeBase}>Knowledge Base</Link>
          {breadcrumbs.map((node) => (
            <span key={node.id} style={{ display: 'contents' }}>
              <span className="breadcrumbs__sep" aria-hidden="true">
                /
              </span>
              {node.id === id ? (
                <span aria-current="page">{node.title}</span>
              ) : (
                <Link to={routes.document(node.id)}>{node.title}</Link>
              )}
            </span>
          ))}
        </nav>
      )}

      <div className="viewer__layout">
        <article>
          <div className="viewer__header">
            <div>
              <h1 className="viewer__title">{data.title}</h1>
              <div className="viewer__meta">
                <span>{sectionMeta[data.section].title}</span>
                <span aria-hidden="true">·</span>
                <span title={formatAbsolute(data.updated_at)}>
                  Updated {relativeTime(data.updated_at)}
                </span>
              </div>
            </div>
            <Button variant="secondary" onClick={() => navigate(routes.editor(data.id))}>
              Edit
            </Button>
          </div>

          {data.tags.length > 0 && (
            <div className="viewer__tags">
              {data.tags.map((tag) => (
                <TagPill key={tag}>{tag}</TagPill>
              ))}
            </div>
          )}

          <Card className="viewer__content">
            {data.content.trim() ? (
              <MarkdownView content={data.content} />
            ) : (
              <p className="text-secondary">This document has no content yet.</p>
            )}
          </Card>
        </article>

        <aside>
          {isProjectNote ? (
            <Card className="viewer__toc">
              <p className="viewer__toc-title">Project Notes</p>
              {tree.data && tree.data.data.length > 0 ? (
                <TreeNav nodes={tree.data.data} activeId={id} />
              ) : (
                <p className="text-secondary">No notes yet.</p>
              )}
            </Card>
          ) : (
            headings.length > 0 && (
              <Card className="viewer__toc">
                <SectionHeader title="On this page" headingLevel="h3" />
                <ul className="viewer__toc-list">
                  {headings.map((heading) => (
                    <li key={heading.id} data-level={heading.level}>
                      <a href={`#${heading.id}`}>{heading.text}</a>
                    </li>
                  ))}
                </ul>
              </Card>
            )
          )}
        </aside>
      </div>
    </div>
  )
}
