/** Route paths used across the SPA (docs/api.md section 12). */
export const routes = {
  dashboard: '/',
  documents: '/documents',
  workflows: '/workflows',
  knowledgeBase: '/knowledge-base',
  cheatSheets: '/cheat-sheets',
  private: '/private',
  search: '/search',
  settings: '/settings',
  document: (id: string) => `/documents/${id}`,
  editor: (id?: string) => (id ? `/editor/${id}` : '/editor'),
} as const
