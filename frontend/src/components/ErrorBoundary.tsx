import { Component } from 'react'
import type { ErrorInfo, ReactNode } from 'react'
import { Button } from '.'

interface ErrorBoundaryProps {
  children: ReactNode
}

interface ErrorBoundaryState {
  error: Error | null
}

/**
 * ErrorBoundary prevents a render error in one page from blanking the whole
 * SPA. It shows a recoverable fallback and offers a full reload.
 */
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // Surface the error for local debugging without crashing the app.
    console.error('Unhandled UI error:', error, info.componentStack)
  }

  reset = () => this.setState({ error: null })

  render() {
    if (this.state.error) {
      return (
        <div className="container">
          <div className="error-state" role="alert">
            <span style={{ fontSize: '2.5rem' }} aria-hidden="true">
              ⚠️
            </span>
            <span className="error-state__title">Something went wrong</span>
            <p className="error-state__message">{this.state.error.message}</p>
            <div className="row">
              <Button variant="secondary" onClick={this.reset}>
                Try again
              </Button>
              <Button onClick={() => window.location.reload()}>Reload</Button>
            </div>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}
