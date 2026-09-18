import type { Section } from '../api/types'

/**
 * SectionIcon renders a line-icon for each content pillar. Icons inherit
 * `currentColor`, so the badge controls their color (white on the accent
 * background per DESIGN.md 2.3).
 */
export function SectionIcon({ section }: { section: Section }) {
  switch (section) {
    case 'workflow':
      // Book with a bookmark: step-by-step guides.
      return (
        <svg
          width="26"
          height="26"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          <path d="M5 4.5A1.5 1.5 0 0 1 6.5 3H18a1 1 0 0 1 1 1v15a1 1 0 0 1-1 1H6.5A1.5 1.5 0 0 1 5 18.5Z" />
          <path d="M5 17.5A1.5 1.5 0 0 1 6.5 16H19" />
          <path d="M9 3v7l2-1.5L13 10V3" />
        </svg>
      )
    case 'project_note':
      // Nested tree: hierarchical project documentation.
      return (
        <svg
          width="26"
          height="26"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          <rect x="9" y="3" width="6" height="4" rx="1" />
          <rect x="3" y="17" width="6" height="4" rx="1" />
          <rect x="15" y="17" width="6" height="4" rx="1" />
          <path d="M12 7v4" />
          <path d="M6 17v-2a1 1 0 0 1 1-1h10a1 1 0 0 1 1 1v2" />
        </svg>
      )
    case 'cheat_sheet':
      // Lightning bolt over code brackets: quick snippets.
      return (
        <svg
          width="26"
          height="26"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          <polyline points="8 6 4 12 8 18" />
          <polyline points="16 6 20 12 16 18" />
          <path d="M12.8 8.5 11.2 12h2.4l-1.6 3.5" />
        </svg>
      )
    case 'private':
      // Padlock: master-password protected archive.
      return (
        <svg
          width="26"
          height="26"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          <rect x="4.5" y="10.5" width="15" height="10" rx="2" />
          <path d="M8 10.5V7.5a4 4 0 0 1 8 0v3" />
          <circle cx="12" cy="15.5" r="1.4" />
        </svg>
      )
    default:
      return null
  }
}
