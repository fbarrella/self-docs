import { Navigate, Route, Routes } from 'react-router-dom'
import { Gallery } from './pages/Gallery'
import { PlaceholderPage } from './pages/PlaceholderPage'
import { routes } from './routes'

/**
 * App defines the SPA route map. Feature pages replace the placeholders in
 * Phase 4; the shell, navigation targets, and 404 handling live here.
 */
function App() {
  return (
    <Routes>
      <Route
        path={routes.dashboard}
        element={
          <PlaceholderPage
            title="Dashboard"
            description="Welcome back — dashboard coming in Phase 4."
          />
        }
      />
      <Route
        path={routes.documents}
        element={
          <PlaceholderPage title="Documents" description="All documents will be listed here." />
        }
      />
      <Route
        path={routes.workflows}
        element={
          <PlaceholderPage
            title="Workflows & Guides"
            description="Step-by-step procedures and guides."
          />
        }
      />
      <Route
        path={routes.knowledgeBase}
        element={
          <PlaceholderPage title="Project Notes" description="Nested project documentation." />
        }
      />
      <Route
        path={routes.cheatSheets}
        element={
          <PlaceholderPage title="Cheat Sheets" description="Quick references and snippets." />
        }
      />
      <Route
        path={routes.private}
        element={
          <PlaceholderPage title="Private Archive" description="Master-password protected notes." />
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
      <Route
        path={`${routes.documents}/:id`}
        element={<PlaceholderPage title="Document" description="Document viewer." />}
      />
      <Route
        path={`${routes.editor}/:id?`}
        element={<PlaceholderPage title="Editor" description="Create or edit a document." />}
      />
      {import.meta.env.DEV && <Route path="/_gallery" element={<Gallery />} />}
      <Route path="*" element={<Navigate to={routes.dashboard} replace />} />
    </Routes>
  )
}

export default App
