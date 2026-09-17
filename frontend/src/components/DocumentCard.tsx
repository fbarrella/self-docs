import { Link } from 'react-router-dom'
import type { Document } from '../api/types'
import { Card, TagPill } from '.'
import { routes } from '../routes'
import { relativeTime } from '../utils/time'

export interface DocumentCardProps {
  document: Document
  onTagClick?: (tag: string) => void
}

/** DocumentCard summarizes a document in the listing grid. */
export function DocumentCard({ document, onTagClick }: DocumentCardProps) {
  return (
    <Card interactive className="doc-card">
      <Link to={routes.document(document.id)} className="doc-card__link">
        <h3 className="doc-card__title">{document.title}</h3>
        <p className="doc-card__excerpt">
          {document.excerpt ?? firstLine(document.content) ?? 'No preview available.'}
        </p>
      </Link>
      <div className="doc-card__footer">
        <div className="doc-card__tags">
          {document.tags.slice(0, 3).map((tag) => (
            <TagPill
              key={tag}
              onClick={
                onTagClick
                  ? (event) => {
                      event.preventDefault()
                      onTagClick(tag)
                    }
                  : undefined
              }
            >
              {tag}
            </TagPill>
          ))}
        </div>
        <span className="doc-card__meta">{relativeTime(document.updated_at)}</span>
      </div>
    </Card>
  )
}

function firstLine(content: string): string | undefined {
  const line = content
    .split('\n')
    .map((value) => value.trim())
    .find((value) => value && !value.startsWith('#'))
  return line?.slice(0, 160)
}
