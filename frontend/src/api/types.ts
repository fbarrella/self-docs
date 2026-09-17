/**
 * Shared API types mirroring docs/api.md. These are the contract between the
 * SPA and the Go backend.
 */

export type Section = 'workflow' | 'project_note' | 'cheat_sheet' | 'private'

export type ActivityAction = 'created' | 'updated' | 'deleted' | 'imported' | 'unlocked'

export interface Document {
  id: string
  title: string
  slug: string
  content: string
  excerpt: string | null
  section: Section
  parent_id: string | null
  position: number
  tags: string[]
  created_at: string
  updated_at: string
}

export interface DocumentNode {
  id: string
  title: string
  slug: string
  section: Section
  position: number
  children: DocumentNode[]
}

export interface SearchResult {
  id: string
  title: string
  slug: string
  section: Section
  tags: string[]
  snippet: string
  rank: number
  updated_at: string
}

export interface Tag {
  name: string
  count?: number
}

export interface Activity {
  id: string
  action: ActivityAction
  document_id: string | null
  document_title: string | null
  section: Section | null
  actor: string
  created_at: string
}

export interface Pagination {
  page: number
  page_size: number
  total: number
  total_pages: number
}

export interface Paginated<T> {
  data: T[]
  pagination: Pagination
}

export interface DashboardCard {
  id: Section
  title: string
  count: number | null
  route: string
  locked: boolean
}

export interface DashboardRecent {
  id: string
  title: string
  section: Section
  updated_at: string
}

export interface Dashboard {
  cards: DashboardCard[]
  recently_updated: DashboardRecent[]
  popular_tags: Tag[]
  activity: Activity[]
}

export interface ErrorDetail {
  code: string
  message: string
  details?: { field: string; issue: string }[]
}

export interface ApiErrorBody {
  error: ErrorDetail
}
