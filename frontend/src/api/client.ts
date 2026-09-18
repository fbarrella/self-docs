/**
 * API client for the self-docs backend. The base URL comes from
 * VITE_API_BASE_URL and defaults to the local backend. All private requests
 * include credentials so the HttpOnly session cookie travels with the request.
 */
import type {
  Activity,
  ApiErrorBody,
  Dashboard,
  Document,
  DocumentNode,
  ErrorDetail,
  Paginated,
  SearchResult,
  Section,
  Tag,
} from './types'

export const API_BASE_URL: string = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'

export const PRIVATE_COOKIE_PATH = '/api/private'

/** ApiError carries the parsed error envelope from the backend. */
export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly details: ErrorDetail['details']

  constructor(status: number, detail: ErrorDetail) {
    super(detail.message)
    this.name = 'ApiError'
    this.status = status
    this.code = detail.code
    this.details = detail.details
  }

  get isUnauthorized(): boolean {
    return this.status === 401
  }
}

// Imported lazily to avoid a module cycle at load time.
import { clearPrivateToken, getPrivateToken } from './privateToken'

type QueryValue = string | number | boolean | undefined | null

function buildQuery(params: Record<string, QueryValue | QueryValue[]>): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') continue
    if (Array.isArray(value)) {
      for (const item of value) {
        if (item !== undefined && item !== null && item !== '') {
          search.append(key, String(item))
        }
      }
    } else {
      search.append(key, String(value))
    }
  }
  const qs = search.toString()
  return qs ? `?${qs}` : ''
}

interface RequestOptions {
  method?: string
  body?: unknown
  signal?: AbortSignal
  /** Attach the private session bearer token when available. */
  private?: boolean
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, signal, private: isPrivate } = options

  const isFormData = body instanceof FormData
  const requestBody = isFormData ? body : body !== undefined ? JSON.stringify(body) : undefined

  const headers: Record<string, string> = {}
  if (body !== undefined && !isFormData) headers['Content-Type'] = 'application/json'
  if (isPrivate) {
    const token = getPrivateToken()
    if (token) headers.Authorization = `Bearer ${token}`
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    method,
    credentials: 'include',
    headers: Object.keys(headers).length > 0 ? headers : undefined,
    body: requestBody,
    signal,
  })

  if (response.status === 401 && isPrivate) {
    clearPrivateToken()
  }

  if (response.status === 204) {
    return undefined as T
  }

  const text = await response.text()
  let parsed: unknown = undefined
  if (text) {
    try {
      parsed = JSON.parse(text)
    } catch {
      parsed = undefined
    }
  }

  if (!response.ok) {
    const apiBody = parsed as ApiErrorBody | undefined
    const detail: ErrorDetail = apiBody?.error ?? {
      code: 'internal_error',
      message: response.statusText || 'Request failed',
    }
    throw new ApiError(response.status, detail)
  }

  return parsed as T
}

export interface ListDocumentsParams {
  section?: Section | Section[]
  tag?: string | string[]
  parent_id?: string
  q?: string
  include?: 'content'
  sort?: 'updated_at' | 'created_at' | 'title'
  order?: 'asc' | 'desc'
  page?: number
  page_size?: number
}

export interface CreateDocumentPayload {
  title: string
  content?: string
  excerpt?: string | null
  section: Section
  parent_id?: string | null
  position?: number
  tags?: string[]
}

export interface UpdateDocumentPayload {
  title?: string
  content?: string
  excerpt?: string | null
  section?: Section
  parent_id?: string | null
  position?: number
  tags?: string[]
}

export interface ImportResult {
  filename: string
  status: 'created' | 'skipped' | 'failed'
  document_id?: string
  reason?: string
}

export interface ImportResponse {
  data: ImportResult[]
  summary: { created: number; skipped: number; failed: number }
}

export const api = {
  health: () => request<{ status: string; database: string; redis: string }>('/healthz'),

  documents: {
    list: (params: ListDocumentsParams = {}) =>
      request<Paginated<Document>>(`/api/documents${buildQuery({ ...params })}`),
    get: (id: string) => request<Document>(`/api/documents/${id}`),
    create: (payload: CreateDocumentPayload) =>
      request<Document>('/api/documents', { method: 'POST', body: payload }),
    update: (id: string, payload: UpdateDocumentPayload) =>
      request<Document>(`/api/documents/${id}`, { method: 'PUT', body: payload }),
    remove: (id: string) => request<void>(`/api/documents/${id}`, { method: 'DELETE' }),
    tree: (section: Section = 'project_note') =>
      request<{ data: DocumentNode[] }>(`/api/documents/tree${buildQuery({ section })}`),
    import: (form: FormData) =>
      request<ImportResponse>('/api/documents/import', { method: 'POST', body: form }),
  },

  tags: {
    list: (
      params: {
        q?: string
        sort?: 'name' | 'count'
        order?: 'asc' | 'desc'
        page?: number
        page_size?: number
      } = {},
    ) => request<Paginated<Tag>>(`/api/tags${buildQuery({ ...params })}`),
    popular: (limit = 12) => request<{ data: Tag[] }>(`/api/tags/popular${buildQuery({ limit })}`),
    documents: (name: string, params: { page?: number; page_size?: number } = {}) =>
      request<Paginated<Document>>(
        `/api/tags/${encodeURIComponent(name)}/documents${buildQuery({ ...params })}`,
      ),
    remove: (name: string) =>
      request<void>(`/api/tags/${encodeURIComponent(name)}`, { method: 'DELETE' }),
  },

  settings: {
    get: () =>
      request<{
        data: {
          version: string
          redis_enabled: boolean
          master_password_set: boolean
          private_session_open: boolean
        }
      }>('/api/settings', { private: true }),
    changeMasterPassword: (currentPassword: string, newPassword: string) =>
      request<void>('/api/settings/master-password', {
        method: 'PUT',
        body: { current_password: currentPassword, new_password: newPassword },
        private: true,
      }),
  },

  search: (params: { q: string; section?: Section[]; page?: number; page_size?: number }) =>
    request<Paginated<SearchResult> & { query: string }>(`/api/search${buildQuery({ ...params })}`),

  dashboard: () => request<Dashboard>('/api/dashboard'),

  activity: (params: { page?: number; page_size?: number } = {}) =>
    request<Paginated<Activity>>(`/api/activity${buildQuery({ ...params })}`),

  private: {
    unlock: (password: string) =>
      request<{ data: { token: string; expires_at: string } }>('/api/private/unlock', {
        method: 'POST',
        body: { password },
      }),
    lock: () => request<void>('/api/private/lock', { method: 'POST', private: true }),
    list: (params: { include?: 'content'; page?: number; page_size?: number } = {}) =>
      request<Paginated<Document>>(`/api/private/documents${buildQuery({ ...params })}`, {
        private: true,
      }),
    get: (id: string) => request<Document>(`/api/private/documents/${id}`, { private: true }),
    create: (payload: CreateDocumentPayload) =>
      request<Document>('/api/private/documents', { method: 'POST', body: payload, private: true }),
    update: (id: string, payload: UpdateDocumentPayload) =>
      request<Document>(`/api/private/documents/${id}`, {
        method: 'PUT',
        body: payload,
        private: true,
      }),
    remove: (id: string) =>
      request<void>(`/api/private/documents/${id}`, { method: 'DELETE', private: true }),
  },
}
