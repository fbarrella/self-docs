import type { ReactNode } from 'react'
import { Button } from '.'

export interface ErrorStateProps {
  title?: string
  message?: string
  onRetry?: () => void
  action?: ReactNode
}

/** ErrorState is the shared recoverable error surface for data failures. */
export function ErrorState({
  title = 'Something went wrong',
  message = 'We could not load this content. Please try again.',
  onRetry,
  action,
}: ErrorStateProps) {
  return (
    <div className="error-state" role="alert">
      <span style={{ fontSize: '2.5rem' }} aria-hidden="true">
        ⚠️
      </span>
      <span className="error-state__title">{title}</span>
      <p className="error-state__message">{message}</p>
      <div className="row">
        {onRetry && (
          <Button variant="secondary" onClick={onRetry}>
            Retry
          </Button>
        )}
        {action}
      </div>
    </div>
  )
}
