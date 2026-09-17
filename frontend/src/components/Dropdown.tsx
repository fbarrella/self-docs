import { useEffect, useId, useRef, useState } from 'react'
import type { ReactNode } from 'react'

export interface DropdownItem {
  label: string
  onSelect: () => void
  icon?: ReactNode
}

export interface DropdownProps {
  trigger: (props: { open: boolean; toggle: () => void }) => ReactNode
  items: DropdownItem[]
  align?: 'left' | 'right'
}

/**
 * Dropdown renders a trigger plus a menu. It closes on outside click, Escape,
 * and item selection, and exposes basic menu semantics.
 */
export function Dropdown({ trigger, items, align = 'right' }: DropdownProps) {
  const [open, setOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)
  const menuId = useId()

  useEffect(() => {
    if (!open) return

    function handlePointer(event: MouseEvent) {
      if (!containerRef.current?.contains(event.target as Node)) setOpen(false)
    }
    function handleKey(event: KeyboardEvent) {
      if (event.key === 'Escape') setOpen(false)
    }

    document.addEventListener('mousedown', handlePointer)
    document.addEventListener('keydown', handleKey)
    return () => {
      document.removeEventListener('mousedown', handlePointer)
      document.removeEventListener('keydown', handleKey)
    }
  }, [open])

  return (
    <div className="ui-dropdown" ref={containerRef}>
      {trigger({ open, toggle: () => setOpen((value) => !value) })}
      {open && (
        <div
          className="ui-dropdown__menu"
          role="menu"
          id={menuId}
          style={align === 'left' ? { right: 'auto', left: 0 } : undefined}
        >
          {items.map((item) => (
            <button
              key={item.label}
              type="button"
              role="menuitem"
              className="ui-dropdown__item"
              onClick={() => {
                setOpen(false)
                item.onSelect()
              }}
            >
              {item.icon}
              {item.label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
