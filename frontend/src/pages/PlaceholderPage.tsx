import type { ReactNode } from 'react'

interface PlaceholderPageProps {
  title: string
  description: string
  children?: ReactNode
}

/**
 * PlaceholderPage is a temporary scaffold used until each feature page is
 * implemented in Phase 4. It keeps the routing shell verifiable.
 */
export function PlaceholderPage({ title, description, children }: PlaceholderPageProps) {
  return (
    <section className="placeholder">
      <h1>{title}</h1>
      <p>{description}</p>
      {children}
    </section>
  )
}
