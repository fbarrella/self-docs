import type { Section } from './api/types'
import { routes } from './routes'

export interface SectionMeta {
  section: Section
  title: string
  description: string
  route: string
  buttonLabel: string
}

/**
 * Section metadata for the dashboard navigation cards (PRD 3.2 / DESIGN.md
 * 2.3). Icons are rendered as SVG by the SectionIcon component.
 */
export const sectionMeta: Record<Section, SectionMeta> = {
  workflow: {
    section: 'workflow',
    title: 'Workflows & Guides',
    description: 'Standard procedures and step-by-step guides.',
    route: routes.workflows,
    buttonLabel: 'Explore Guides',
  },
  project_note: {
    section: 'project_note',
    title: 'Project Notes',
    description: 'Hierarchical documentation with nested pages.',
    route: routes.knowledgeBase,
    buttonLabel: 'Open Notes',
  },
  cheat_sheet: {
    section: 'cheat_sheet',
    title: 'Cheat Sheets',
    description: 'Quick references, commands, and code snippets.',
    route: routes.cheatSheets,
    buttonLabel: 'View Sheets',
  },
  private: {
    section: 'private',
    title: 'Private Archive',
    description: 'Confidential notes protected by the master password.',
    route: routes.private,
    buttonLabel: 'Unlock Vault',
  },
}
