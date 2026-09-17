export interface SpinnerProps {
  size?: 'sm' | 'lg'
  label?: string
}

/** Spinner indicates an in-progress operation. */
export function Spinner({ size = 'sm', label = 'Loading' }: SpinnerProps) {
  const classes = ['ui-spinner', size === 'lg' ? 'ui-spinner--lg' : ''].filter(Boolean).join(' ')
  return <span className={classes} role="status" aria-label={label} />
}
