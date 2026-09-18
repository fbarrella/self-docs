import { Suspense, lazy } from 'react'
import { Navigate, Outlet, Route, Routes } from 'react-router-dom'
import { AppLayout } from './components/layout'
import { ErrorBoundary, MasterPasswordModal, OfflineBanner, Spinner } from './components'
import { PrivateSessionProvider } from './context/PrivateSessionContext'
import { ToastProvider } from './context/ToastContext'
import { DashboardPage } from './pages/DashboardPage'
import { DocumentView } from './pages/DocumentView'
import { DocumentsPage } from './pages/DocumentsPage'
import { KnowledgeBasePage } from './pages/KnowledgeBasePage'
import { SearchPage } from './pages/SearchPage'
import { SettingsPage } from './pages/SettingsPage'
import { routes } from './routes'

// The Markdown editor is heavy, so these editor-bearing routes load on demand.
const EditorPage = lazy(() =>
  import('./pages/EditorPage').then((module) => ({ default: module.EditorPage })),
)
const PrivateArchivePage = lazy(() =>
  import('./pages/PrivateArchivePage').then((module) => ({ default: module.PrivateArchivePage })),
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
      <OfflineBanner />
      <ErrorBoundary>
        <Outlet />
      </ErrorBoundary>
    </AppLayout>
  )
}

/**
 * App defines the SPA route map wrapped by the persistent layout shell.
 * Feature pages replace the placeholders in Phase 4.
 */
function App() {
  return (
    <ToastProvider>
      <PrivateSessionProvider>
        <MasterPasswordModal />
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
                <Suspense fallback={<RouteFallback />}>
                  <PrivateArchivePage />
                </Suspense>
              }
            />
            <Route path={routes.search} element={<SearchPage />} />
            <Route path={routes.settings} element={<SettingsPage />} />
            <Route path={`${routes.documents}/:id`} element={<DocumentView />} />
            <Route
              path={routes.editor()}
              element={
                <Suspense fallback={<RouteFallback />}>
                  <EditorPage />
                </Suspense>
              }
            />
            <Route
              path={`${routes.editor()}/:id`}
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
      </PrivateSessionProvider>
    </ToastProvider>
  )
}

export default App
