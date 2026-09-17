import { Link } from 'react-router-dom'
import { routes } from '../../routes'

function BrandMark() {
  return (
    <svg
      width="20"
      height="20"
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
    </svg>
  )
}

/**
 * Footer implements DESIGN.md 2.5: a brand/copyright/legal column, a Portal
 * links column, and a Useful links column.
 */
export function Footer() {
  return (
    <footer className="site-footer">
      <div className="container site-footer__grid">
        <div>
          <span className="site-footer__brand">
            <BrandMark />
            <span>self-docs</span>
          </span>
          <p className="site-footer__copyright">© {new Date().getFullYear()} self-docs</p>
          <div className="site-footer__legal">
            <a href="#privacy">Privacy Policy</a>
            <a href="#terms">Terms</a>
            <a href="#contact">Contact</a>
          </div>
        </div>

        <div>
          <h4 className="site-footer__heading">Portal links</h4>
          <ul className="site-footer__links">
            <li>
              <a href="/api/healthz">Admin Dashboard</a>
            </li>
            <li>
              <a href="/api/healthz">Portal API</a>
            </li>
          </ul>
        </div>

        <div>
          <h4 className="site-footer__heading">Useful links</h4>
          <ul className="site-footer__links">
            <li>
              <Link to={routes.documents}>All Documents</Link>
            </li>
            <li>
              <Link to={routes.settings}>Settings</Link>
            </li>
          </ul>
        </div>
      </div>
    </footer>
  )
}
