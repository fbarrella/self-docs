import type { FormEvent } from 'react'

export interface SearchBarProps {
  value: string
  onChange: (value: string) => void
  onSubmit: (value: string) => void
  placeholder?: string
  buttonLabel?: string
  ariaLabel?: string
}

/**
 * SearchBar is the pill-shaped global search with an inner green Search button
 * (DESIGN.md 2.2).
 */
export function SearchBar({
  value,
  onChange,
  onSubmit,
  placeholder = 'Search your documentation...',
  buttonLabel = 'Search',
  ariaLabel = 'Global search',
}: SearchBarProps) {
  function handleSubmit(event: FormEvent) {
    event.preventDefault()
    onSubmit(value)
  }

  return (
    <form className="ui-searchbar" role="search" onSubmit={handleSubmit}>
      <svg
        className="ui-searchbar__icon"
        width="18"
        height="18"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <circle cx="11" cy="11" r="7" />
        <line x1="21" y1="21" x2="16.65" y2="16.65" />
      </svg>
      <input
        className="ui-searchbar__input"
        type="search"
        value={value}
        placeholder={placeholder}
        aria-label={ariaLabel}
        onChange={(event) => onChange(event.target.value)}
      />
      <button className="ui-searchbar__button" type="submit">
        {buttonLabel}
      </button>
    </form>
  )
}
