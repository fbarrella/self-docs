import type { ReactNode } from 'react'

export interface IconBadgeProps {
  children: ReactNode
  round?: boolean
  accent?: boolean
  label?: string
}

/**
 * IconBadge is the rounded-square icon container used on navigation cards and
 * list rows (DESIGN.md 2.3).
 */
export function IconBadge({ children, round = false, accent = false, label }: IconBadgeProps) {
  const classes = [
    'ui-icon-badge',
    round ? 'ui-icon-badge--round' : '',
    accent ? 'ui-icon-badge--accent' : '',
  ]
    .filter(Boolean)
    .join(' ')
  return (
    <span
      className={classes}
      role={label ? 'img' : undefined}
      aria-label={label}
      aria-hidden={!label}
    >
      {children}
    </span>
  )
}
