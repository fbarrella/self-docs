import type { HTMLAttributes, ReactNode } from 'react'

export interface CardProps extends HTMLAttributes<HTMLDivElement> {
  interactive?: boolean
  children: ReactNode
}

/** Card is a white surface container with rounded corners and a soft shadow. */
export function Card({ interactive = false, className, children, ...rest }: CardProps) {
  const classes = ['ui-card', interactive ? 'ui-card--interactive' : '', className ?? '']
    .filter(Boolean)
    .join(' ')
  return (
    <div className={classes} {...rest}>
      {children}
    </div>
  )
}
