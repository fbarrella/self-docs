import { routes } from './routes'

export interface NavItem {
  label: string
  to: string
}

/**
 * Primary header navigation (PRD 4, DESIGN.md 2.1). The bell is out of scope,
 * and Settings lives only in the user dropdown.
 */
export const primaryNav: NavItem[] = [
  { label: 'Home', to: routes.dashboard },
  { label: 'Documents', to: routes.documents },
  { label: 'Workflows & Guides', to: routes.workflows },
  { label: 'Knowledge Base', to: routes.knowledgeBase },
  { label: 'Private Notes', to: routes.private },
]
