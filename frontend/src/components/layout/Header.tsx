import { useState } from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import { Avatar, Dropdown } from '../index'
import type { DropdownItem } from '../Dropdown'
import { usePrivateSession } from '../../context/privateSession'
import { primaryNav } from '../../navigation'
import { routes } from '../../routes'

function BrandMark() {
  return (
    <svg
      className="site-header__brand-icon"
      width="22"
      height="22"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M4 4.5A1.5 1.5 0 0 1 5.5 3H14l4 4v13.5A1.5 1.5 0 0 1 16.5 22h-11A1.5 1.5 0 0 1 4 20.5Z" />
      <path d="M14 3v4h4" />
      <line x1="8" y1="12" x2="16" y2="12" />
      <line x1="8" y1="16" x2="16" y2="16" />
    </svg>
  )
}

function Caret({ open }: { open: boolean }) {
  return (
    <svg
      className={`site-header__caret${open ? ' site-header__caret--open' : ''}`}
      width="14"
      height="14"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <polyline points="6 9 12 15 18 9" />
    </svg>
  )
}

/**
 * Header implements DESIGN.md 2.1: logo left, primary nav center, avatar
 * dropdown right (global settings only, no notification bell), collapsing to a
 * hamburger menu on small screens.
 */
export function Header() {
  const [menuOpen, setMenuOpen] = useState(false)
  const navigate = useNavigate()
  const { status, requestUnlock, lock } = usePrivateSession()

  function closeMenu() {
    setMenuOpen(false)
  }

  const userMenuItems: DropdownItem[] = [
    { label: 'Global settings', onSelect: () => navigate(routes.settings) },
    status === 'unlocked'
      ? { label: 'Lock Private Archive', onSelect: () => void lock() }
      : { label: 'Unlock Private Archive', onSelect: requestUnlock },
  ]

  return (
    <header className="site-header">
      <div className="container site-header__inner">
        <NavLink to={routes.dashboard} className="site-header__brand" onClick={closeMenu}>
          <BrandMark />
          <span>self-docs</span>
        </NavLink>

        <nav className="site-nav" aria-label="Primary">
          {primaryNav.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === routes.dashboard}
              className={({ isActive }) =>
                `site-nav__link${isActive ? ' site-nav__link--active' : ''}`
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>

        <div className="site-header__user">
          <Dropdown
            items={userMenuItems}
            trigger={({ open, toggle }) => (
              <button
                type="button"
                className="site-header__user-trigger"
                onClick={toggle}
                aria-haspopup="menu"
                aria-expanded={open}
                aria-label="User menu"
              >
                <Avatar initials="AK" size="sm" label="User" />
                <span className="site-header__user-name">A.K.</span>
                <Caret open={open} />
              </button>
            )}
          />
          <button
            type="button"
            className="site-header__hamburger"
            aria-label={menuOpen ? 'Close menu' : 'Open menu'}
            aria-expanded={menuOpen}
            onClick={() => setMenuOpen((value) => !value)}
          >
            <svg
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              aria-hidden="true"
            >
              {menuOpen ? (
                <>
                  <line x1="6" y1="6" x2="18" y2="18" />
                  <line x1="18" y1="6" x2="6" y2="18" />
                </>
              ) : (
                <>
                  <line x1="3" y1="6" x2="21" y2="6" />
                  <line x1="3" y1="12" x2="21" y2="12" />
                  <line x1="3" y1="18" x2="21" y2="18" />
                </>
              )}
            </svg>
          </button>
        </div>
      </div>

      <div className="container">
        <nav
          className={`site-mobile-nav${menuOpen ? ' site-mobile-nav--open' : ''}`}
          aria-label="Primary mobile"
          hidden={!menuOpen}
        >
          {primaryNav.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === routes.dashboard}
              onClick={closeMenu}
              className={({ isActive }) =>
                `site-mobile-nav__link${isActive ? ' site-mobile-nav__link--active' : ''}`
              }
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
      </div>
    </header>
  )
}
