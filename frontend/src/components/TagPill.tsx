import type { ButtonHTMLAttributes, ReactNode } from 'react'

export interface TagPillProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  active?: boolean
  count?: number
  children: ReactNode
}

/**
 * TagPill renders a tag. When rendered as a button (onClick provided) it gains
 * hover/active affordances used for tag filtering (DESIGN.md 2.4).
 */
export function TagPill({ active = false, count, className, children, ...rest }: TagPillProps) {
  const isButton = typeof rest.onClick === 'function' || rest.type !== undefined
  const classes = [
    'ui-tag-pill',
    isButton ? 'ui-tag-pill--button' : '',
    active ? 'ui-tag-pill--active' : '',
    className ?? '',
  ]
    .filter(Boolean)
    .join(' ')

  const content = (
    <>
      <span>#{children}</span>
      {count !== undefined && <span className="ui-tag-pill__count">{count}</span>}
    </>
  )

  if (isButton) {
    return (
      <button type="button" className={classes} {...rest}>
        {content}
      </button>
    )
  }
  return <span className={classes}>{content}</span>
}
