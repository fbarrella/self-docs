import type { InputHTMLAttributes, TextareaHTMLAttributes } from 'react'

let inputIdCounter = 0

function nextId(provided?: string): string {
  if (provided) return provided
  inputIdCounter += 1
  return `ui-input-${inputIdCounter}`
}

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string
  hint?: string
  error?: string
}

/** Input is a labelled text field with hint and error states. */
export function Input({ label, hint, error, id, className, ...rest }: InputProps) {
  const inputId = nextId(id)
  const classes = ['ui-input', error ? 'ui-input--error' : '', className ?? '']
    .filter(Boolean)
    .join(' ')

  return (
    <div className="ui-input-wrap">
      {label && (
        <label className="ui-input-label" htmlFor={inputId}>
          {label}
        </label>
      )}
      <input
        id={inputId}
        className={classes}
        aria-invalid={error ? true : undefined}
        aria-describedby={hint || error ? `${inputId}-hint` : undefined}
        {...rest}
      />
      {(error || hint) && (
        <span
          id={`${inputId}-hint`}
          className={`ui-input-hint${error ? ' ui-input-hint--error' : ''}`}
        >
          {error ?? hint}
        </span>
      )}
    </div>
  )
}

export interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string
  hint?: string
  error?: string
}

/** Textarea mirrors Input for multi-line content. */
export function Textarea({ label, hint, error, id, className, ...rest }: TextareaProps) {
  const inputId = nextId(id)
  const classes = ['ui-input', 'ui-textarea', error ? 'ui-input--error' : '', className ?? '']
    .filter(Boolean)
    .join(' ')

  return (
    <div className="ui-input-wrap">
      {label && (
        <label className="ui-input-label" htmlFor={inputId}>
          {label}
        </label>
      )}
      <textarea
        id={inputId}
        className={classes}
        aria-invalid={error ? true : undefined}
        aria-describedby={hint || error ? `${inputId}-hint` : undefined}
        {...rest}
      />
      {(error || hint) && (
        <span
          id={`${inputId}-hint`}
          className={`ui-input-hint${error ? ' ui-input-hint--error' : ''}`}
        >
          {error ?? hint}
        </span>
      )}
    </div>
  )
}
