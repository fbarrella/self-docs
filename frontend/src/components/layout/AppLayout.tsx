import type { ReactNode } from 'react'
import { useLocation } from 'react-router-dom'
import { Footer } from './Footer'
import { Header } from './Header'

/**
 * AppLayout wraps every route with the persistent Header and Footer. A skip
 * link precedes the header for keyboard users.
 */
export function AppLayout({ children }: { children: ReactNode }) {
  const { pathname } = useLocation()

  return (
    <div className="app-shell">
      <a className="visually-hidden" href="#main-content">
        Skip to content
      </a>
      <Header />
      <main className="app-main" id="main-content" key={pathname}>
        {children}
      </main>
      <Footer />
    </div>
  )
}

export { Header, Footer }
