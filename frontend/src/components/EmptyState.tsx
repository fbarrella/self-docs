import type { ReactNode } from 'react'

export interface EmptyStateProps {
  title: string
  description?: string
  icon?: ReactNode
  action?: ReactNode
}

/** EmptyState communicates that a collection has no items yet. */
export function EmptyState({ title, description, icon, action }: EmptyStateProps) {
  return (
    <div className="ui-empty-state">
      {icon}
      <span className="ui-empty-state__title">{title}</span>
      {description && <span>{description}</span>}
      {action}
    </div>
  )
}
