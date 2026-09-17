import type { ButtonHTMLAttributes, ReactNode } from 'react'

export type ButtonVariant = 'accent' | 'secondary' | 'ghost' | 'danger'
export type ButtonSize = 'sm' | 'md' | 'lg'

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
  size?: ButtonSize
  block?: boolean
  loading?: boolean
  children: ReactNode
}

/**
 * Button is the pill-shaped action control. The lime `accent` variant is the
 * primary call to action (DESIGN.md 2.3).
 */
export function Button({
  variant = 'accent',
  size = 'md',
  block = false,
  loading = false,
  disabled,
  className,
  children,
  ...rest
}: ButtonProps) {
  const classes = [
    'ui-button',
    `ui-button--${variant}`,
    size !== 'md' ? `ui-button--${size}` : '',
    block ? 'ui-button--block' : '',
    className ?? '',
  ]
    .filter(Boolean)
    .join(' ')

  return (
    <button className={classes} disabled={disabled || loading} {...rest}>
      {loading && <span className="ui-spinner" aria-hidden="true" />}
      {children}
    </button>
  )
}
