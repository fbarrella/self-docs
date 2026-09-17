import { routes } from './routes'

export interface NavItem {
  label: string
  to: string
}

/** Primary header navigation (PRD 4, DESIGN.md 2.1). The bell is out of scope. */
export const primaryNav: NavItem[] = [
  { label: 'Home', to: routes.dashboard },
  { label: 'Documents', to: routes.documents },
  { label: 'Knowledge Base', to: routes.knowledgeBase },
  { label: 'Private Notes', to: routes.private },
  { label: 'Settings', to: routes.settings },
]
