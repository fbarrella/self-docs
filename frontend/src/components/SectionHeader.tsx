import type { ReactNode } from 'react'

export interface SectionHeaderProps {
  title: string
  action?: ReactNode
  headingLevel?: 'h1' | 'h2' | 'h3' | 'h4'
}

/** SectionHeader titles a content block with an optional trailing action. */
export function SectionHeader({ title, action, headingLevel: Heading = 'h3' }: SectionHeaderProps) {
  return (
    <div className="ui-section-header">
      <Heading className="ui-section-header__title">{title}</Heading>
      {action && <span className="ui-section-header__action">{action}</span>}
    </div>
  )
}
