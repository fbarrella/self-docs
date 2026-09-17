import { Suspense, lazy } from 'react'
import { Navigate, Outlet, Route, Routes } from 'react-router-dom'
import { AppLayout } from './components/layout'
import { Spinner } from './components'
import { DashboardPage } from './pages/DashboardPage'
import { DocumentView } from './pages/DocumentView'
import { DocumentsPage } from './pages/DocumentsPage'
import { KnowledgeBasePage } from './pages/KnowledgeBasePage'
import { PlaceholderPage } from './pages/PlaceholderPage'
import { routes } from './routes'

// The Markdown editor is heavy, so it loads on demand.
const EditorPage = lazy(() =>
  import('./pages/EditorPage').then((module) => ({ default: module.EditorPage })),
)
const Gallery = lazy(() =>
  import('./pages/Gallery').then((module) => ({ default: module.Gallery })),
)

function RouteFallback() {
  return (
    <div className="container" style={{ paddingBlock: 'var(--space-8)' }}>
      <div className="row">
        <Spinner size="lg" />
        <span>Loading…</span>
      </div>
    </div>
  )
}

/** ShellRoute wraps nested routes with the persistent Header/Footer layout. */
function ShellRoute() {
  return (
    <AppLayout>
      <Outlet />
    </AppLayout>
  )
}

/**
 * App defines the SPA route map wrapped by the persistent layout shell.
 * Feature pages replace the placeholders in Phase 4.
 */
function App() {
  return (
    <Routes>
      <Route element={<ShellRoute />}>
        <Route path={routes.dashboard} element={<DashboardPage />} />
        <Route path={routes.documents} element={<DocumentsPage />} />
        <Route path={routes.workflows} element={<DocumentsPage section="workflow" />} />
        <Route path={routes.knowledgeBase} element={<KnowledgeBasePage />} />
        <Route path={routes.cheatSheets} element={<DocumentsPage section="cheat_sheet" />} />
        <Route
          path={routes.private}
          element={
            <PlaceholderPage
              title="Private Archive"
              description="Master-password protected notes."
            />
          }
        />
        <Route
          path={routes.search}
          element={<PlaceholderPage title="Search" description="Global full-text search." />}
        />
        <Route
          path={routes.settings}
          element={<PlaceholderPage title="Settings" description="Global application settings." />}
        />
        <Route path={`${routes.documents}/:id`} element={<DocumentView />} />
        <Route
          path={`${routes.editor}/:id?`}
          element={
            <Suspense fallback={<RouteFallback />}>
              <EditorPage />
            </Suspense>
          }
        />
        {import.meta.env.DEV && (
          <Route
            path="/_gallery"
            element={
              <Suspense fallback={<RouteFallback />}>
                <Gallery />
              </Suspense>
            }
          />
        )}
      </Route>
      <Route path="*" element={<Navigate to={routes.dashboard} replace />} />
    </Routes>
  )
}

export default App
