import { useState } from 'react'
import { Link } from 'react-router-dom'
import type { DocumentNode } from '../api/types'
import { routes } from '../routes'

interface TreeNavProps {
  nodes: DocumentNode[]
  activeId?: string
  onNavigate?: () => void
}

/** TreeNav renders the collapsible Project Notes tree (PRD 3.2). */
export function TreeNav({ nodes, activeId, onNavigate }: TreeNavProps) {
  return (
    <ul className="kb__tree">
      {nodes.map((node) => (
        <TreeItem key={node.id} node={node} activeId={activeId} onNavigate={onNavigate} />
      ))}
    </ul>
  )
}

function TreeItem({
  node,
  activeId,
  onNavigate,
}: { node: DocumentNode } & Omit<TreeNavProps, 'nodes'>) {
  const [open, setOpen] = useState(true)
  const hasChildren = node.children.length > 0
  const isActive = node.id === activeId

  return (
    <li>
      <div className="kb__tree-item">
        {hasChildren ? (
          <button
            type="button"
            className={`kb__tree-toggle${open ? ' kb__tree-toggle--open' : ''}`}
            aria-label={open ? `Collapse ${node.title}` : `Expand ${node.title}`}
            aria-expanded={open}
            onClick={() => setOpen((value) => !value)}
          >
            <svg
              width="12"
              height="12"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2.5"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <polyline points="9 6 15 12 9 18" />
            </svg>
          </button>
        ) : (
          <span className="kb__tree-toggle" aria-hidden="true" />
        )}
        <Link
          to={routes.document(node.id)}
          className={`kb__tree-link${isActive ? ' kb__tree-link--active' : ''}`}
          onClick={onNavigate}
          title={node.title}
        >
          {node.title}
        </Link>
      </div>
      {hasChildren && open && (
        <ul>
          {node.children.map((child) => (
            <TreeItem key={child.id} node={child} activeId={activeId} onNavigate={onNavigate} />
          ))}
        </ul>
      )}
    </li>
  )
}
