import { useCallback, useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { ToastContext, type Toast, type ToastContextValue, type ToastKind } from './toast'

const AUTO_DISMISS_MS = 5000

/**
 * ToastProvider renders a toast region and exposes helpers to push transient
 * success/error/info notifications. Toasts auto-dismiss and can be closed.
 */
export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])
  const nextId = useRef(1)

  const dismiss = useCallback((id: number) => {
    setToasts((current) => current.filter((toast) => toast.id !== id))
  }, [])

  const notify = useCallback(
    (toast: Omit<Toast, 'id'>) => {
      const id = nextId.current++
      setToasts((current) => [...current, { ...toast, id }])
      window.setTimeout(() => dismiss(id), AUTO_DISMISS_MS)
    },
    [dismiss],
  )

  const helpers = useMemo(
    () => ({
      success: (title: string, message?: string) =>
        notify({ kind: 'success' as ToastKind, title, message }),
      error: (title: string, message?: string) =>
        notify({ kind: 'error' as ToastKind, title, message }),
      info: (title: string, message?: string) =>
        notify({ kind: 'info' as ToastKind, title, message }),
    }),
    [notify],
  )

  const value = useMemo<ToastContextValue>(
    () => ({ toasts, notify, dismiss, ...helpers }),
    [toasts, notify, dismiss, helpers],
  )

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="toast-region" role="region" aria-label="Notifications">
        {toasts.map((toast) => (
          <div key={toast.id} className={`toast toast--${toast.kind}`} role="status">
            <div className="toast__body">
              <div className="toast__title">{toast.title}</div>
              {toast.message && <div className="toast__message">{toast.message}</div>}
            </div>
            <button
              type="button"
              className="toast__close"
              aria-label="Dismiss notification"
              onClick={() => dismiss(toast.id)}
            >
              ✕
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}
