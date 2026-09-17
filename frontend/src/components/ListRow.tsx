import type { ReactNode } from 'react'

export interface ListRowProps {
  icon?: ReactNode
  title: ReactNode
  meta?: ReactNode
  action?: ReactNode
}

/** ListRow is a horizontal row for the Recently Updated list (DESIGN.md 2.4). */
export function ListRow({ icon, title, meta, action }: ListRowProps) {
  return (
    <div className="ui-list-row">
      {icon && <span className="ui-list-row__icon">{icon}</span>}
      <span className="ui-list-row__main">
        <span className="ui-list-row__title">{title}</span>
      </span>
      {meta && <span className="ui-list-row__meta">{meta}</span>}
      {action && <span className="ui-list-row__action">{action}</span>}
    </div>
  )
}
