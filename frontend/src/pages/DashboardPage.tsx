import { Link, useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import type { Activity, Dashboard, Section } from '../api/types'
import {
  Button,
  ButtonLink,
  Card,
  EmptyState,
  IconBadge,
  ListRow,
  SearchBar,
  SectionHeader,
  SectionIcon,
  Skeleton,
  TagPill,
} from '../components'
import { useAsync } from '../hooks/useAsync'
import { routes } from '../routes'
import { sectionMeta } from '../sectionMeta'
import { relativeTime } from '../utils/time'
import { useState } from 'react'

const cardOrder: Section[] = ['workflow', 'project_note', 'cheat_sheet', 'private']

function activityPhrase(entry: Activity): string {
  const title = entry.document_title ?? 'a document'
  switch (entry.action) {
    case 'created':
      return `Created ${title}`
    case 'updated':
      return `Updated ${title}`
    case 'deleted':
      return `Deleted ${title}`
    case 'imported':
      return `Imported ${title}`
    case 'unlocked':
      return 'Unlocked the Private Archive'
    default:
      return title
  }
}

/** DashboardPage implements PRD 3.4 and DESIGN.md 2.2-2.4. */
export function DashboardPage() {
  const navigate = useNavigate()
  const [query, setQuery] = useState('')
  const { data, error, loading, reload } = useAsync<Dashboard>(() => api.dashboard(), [])

  function submitSearch(value: string) {
    const trimmed = value.trim()
    if (trimmed) navigate(`${routes.search}?q=${encodeURIComponent(trimmed)}`)
  }

  return (
    <div className="dashboard">
      <section className="dashboard-hero">
        <p className="dashboard-hero__greeting">Welcome back</p>
        <h1 className="dashboard-hero__title">Your Personal Knowledge Base.</h1>
        <p className="dashboard-hero__subtitle">
          A secure and organized hub for your docs, guides, and notes.
        </p>
        <div className="dashboard-hero__search">
          <SearchBar value={query} onChange={setQuery} onSubmit={submitSearch} />
        </div>
      </section>

      <div className="container">
        {error && (
          <div className="dashboard-error" role="alert">
            <span>Could not load the dashboard. {error.message}</span>
            <Button variant="secondary" size="sm" onClick={reload}>
              Retry
            </Button>
          </div>
        )}

        <section aria-label="Sections" className="dashboard-cards">
          {cardOrder.map((section) => {
            const meta = sectionMeta[section]
            const card = data?.cards.find((item) => item.id === section)
            return (
              <Card key={section} interactive>
                <IconBadge accent label={meta.title}>
                  <SectionIcon section={section} />
                </IconBadge>
                <h2 className="dashboard-card__title">{meta.title}</h2>
                <p className="dashboard-card__description">{meta.description}</p>
                <p className="dashboard-card__meta">
                  {card?.locked ? 'Password required' : `${card?.count ?? 0} documents`}
                </p>
                <ButtonLink to={meta.route} block>
                  {meta.buttonLabel}
                </ButtonLink>
              </Card>
            )
          })}
        </section>

        <div className="dashboard-lower">
          <Card className="dashboard-panel">
            <SectionHeader
              title="Recently Updated"
              action={<Link to={routes.documents}>View all</Link>}
              headingLevel="h2"
            />
            {loading && !data ? (
              <RecentSkeleton />
            ) : data && data.recently_updated.length > 0 ? (
              data.recently_updated.map((doc) => (
                <ListRow
                  key={doc.id}
                  icon={<span aria-hidden="true">📄</span>}
                  title={doc.title}
                  meta={relativeTime(doc.updated_at)}
                  action={<Link to={routes.document(doc.id)}>View</Link>}
                />
              ))
            ) : (
              <EmptyState title="No documents yet" description="Create your first document." />
            )}
          </Card>

          <div>
            <Card className="dashboard-panel dashboard-panel--tags">
              <SectionHeader title="Popular Tags" headingLevel="h2" />
              {loading && !data ? (
                <Skeleton height={24} />
              ) : data && data.popular_tags.length > 0 ? (
                <div className="dashboard-tags">
                  {data.popular_tags.map((tag) => (
                    <TagPill
                      key={tag.name}
                      count={tag.count}
                      onClick={() => submitSearch(tag.name)}
                    >
                      {tag.name}
                    </TagPill>
                  ))}
                </div>
              ) : (
                <p className="text-secondary">No tags yet.</p>
              )}
            </Card>

            <Card className="dashboard-panel">
              <SectionHeader title="Activity Feed" headingLevel="h2" />
              {loading && !data ? (
                <Skeleton height={48} />
              ) : data && data.activity.length > 0 ? (
                <div className="dashboard-activity">
                  {data.activity.map((entry) => (
                    <div key={entry.id} className="dashboard-activity__item">
                      <span aria-hidden="true">🕒</span>
                      <div>
                        <div className="dashboard-activity__text">{activityPhrase(entry)}</div>
                        <div className="dashboard-activity__time">
                          {relativeTime(entry.created_at)}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-secondary">No activity yet.</p>
              )}
            </Card>
          </div>
        </div>
      </div>
    </div>
  )
}

function RecentSkeleton() {
  return (
    <div className="stack">
      <Skeleton height={20} />
      <Skeleton height={20} width="80%" />
      <Skeleton height={20} width="60%" />
    </div>
  )
}
