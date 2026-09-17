import { useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import type { DocumentNode } from '../api/types'
import { Button, Card, EmptyState, SectionHeader, Spinner, TreeNav } from '../components'
import { useAsync } from '../hooks/useAsync'
import { routes } from '../routes'

/**
 * KnowledgeBasePage is the Project Notes landing page (PRD 3.2): a collapsible
 * tree of the hierarchy that links into the document viewer.
 */
export function KnowledgeBasePage() {
  const navigate = useNavigate()
  const { data, error, loading, reload } = useAsync<{ data: DocumentNode[] }>(
    () => api.documents.tree('project_note'),
    [],
  )

  const roots = data?.data ?? []

  return (
    <div className="container kb">
      <div className="kb__header">
        <SectionHeader
          title="Project Notes"
          headingLevel="h1"
          action={
            <Button size="sm" onClick={() => navigate(routes.editor())}>
              New note
            </Button>
          }
        />
        <p className="text-secondary">Nested documentation for your projects.</p>
      </div>

      {error && (
        <div className="dashboard-error" role="alert">
          <span>Could not load project notes. {error.message}</span>
          <Button variant="secondary" size="sm" onClick={reload}>
            Retry
          </Button>
        </div>
      )}

      <div className="kb__layout" style={{ marginTop: 'var(--space-5)' }}>
        <Card className="kb__sidebar">
          {loading && !data ? (
            <div className="row">
              <Spinner />
              <span>Loading…</span>
            </div>
          ) : roots.length > 0 ? (
            <TreeNav nodes={roots} />
          ) : (
            <p className="kb__empty">No project notes yet.</p>
          )}
        </Card>

        <div>
          {loading && !data ? null : roots.length > 0 ? (
            <EmptyState
              title="Select a note"
              description="Choose a page from the tree to read it, or create a new note."
              action={<Button onClick={() => navigate(routes.editor())}>Create a note</Button>}
            />
          ) : (
            <EmptyState
              title="No project notes yet"
              description="Project Notes support nesting, so you can build a full documentation tree."
              action={
                <Button onClick={() => navigate(routes.editor())}>Create your first note</Button>
              }
            />
          )}
        </div>
      </div>
    </div>
  )
}
