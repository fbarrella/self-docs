# self-docs - REST API Contract

Frozen contract for the Go/Gin backend. Base path: `/api`. Content type:
`application/json; charset=utf-8` unless noted. This document is the source of
truth for both backend handlers and the frontend API client.

## 1. Conventions

### Conventions summary

- **Auth:** none for public endpoints. Private Archive endpoints require a
  session token obtained from `POST /api/private/unlock`.
- **IDs:** UUID v4 strings.
- **Timestamps:** RFC 3339 / ISO 8601 UTC (e.g. `2026-09-17T16:52:00Z`).
- **Pagination:** `?page=1&page_size=20` for collections; responses include a
  `pagination` object.
- **Errors:** consistent envelope (section 2).
- **Naming:** request/response bodies use `snake_case`.

## 2. Error envelope

All non-2xx responses:

```json
{
  "error": {
    "code": "validation_error",
    "message": "title must not be empty",
    "details": [{ "field": "title", "issue": "required" }]
  }
}
```

### Error codes

| HTTP | `code`             | When                                           |
| ---- | ------------------ | ---------------------------------------------- |
| 400  | `validation_error` | Malformed or missing fields                    |
| 400  | `bad_request`      | Invalid query parameters or body shape         |
| 401  | `unauthorized`     | Missing/invalid master-password session        |
| 403  | `forbidden`        | Action not permitted for the current session   |
| 404  | `not_found`        | Document, tag, or route does not exist         |
| 409  | `conflict`         | Slug collision or invalid hierarchy move       |
| 413  | `payload_too_large`| Import exceeds the configured size limit       |
| 422  | `unprocessable`    | Semantically invalid (e.g. cycle in note tree) |
| 429  | `rate_limited`     | Too many unlock attempts                       |
| 500  | `internal_error`   | Unexpected server failure                      |

## 3. Shared schemas

### Document

```json
{
  "id": "0f6b3c2e-6c4a-4f7e-9b1a-2d5e8f7a1c33",
  "title": "Git Commands",
  "slug": "git-commands",
  "content": "# Git Commands\n...",
  "excerpt": "Reference for common Git commands.",
  "section": "cheat_sheet",
  "parent_id": null,
  "position": 0,
  "tags": ["Git", "CLI"],
  "created_at": "2026-09-17T14:00:00Z",
  "updated_at": "2026-09-17T16:30:00Z"
}
```

- `section` ∈ `workflow | project_note | cheat_sheet | private`.
- `tags` is an array of tag display names.
- `content` is omitted from list responses unless `include=content` is passed.
- Private documents are only ever returned to an unlocked session on
  `/api/private/*` endpoints.

### Tag

```json
{ "name": "Git", "count": 12 }
```

`count` is present only in list/popular responses.

### Pagination

```json
{ "page": 1, "page_size": 20, "total": 42, "total_pages": 3 }
```

### Activity entry

```json
{
  "id": "8a1d...",
  "action": "updated",
  "document_id": "0f6b...",
  "document_title": "Frontend Guide",
  "section": "workflow",
  "actor": "user",
  "created_at": "2026-09-17T16:31:00Z"
}
```

## 4. Documents API

### 4.1 Create document

`POST /api/documents`

Request:

```json
{
  "title": "Git Commands",
  "content": "# Git Commands\n...",
  "excerpt": "Reference for common Git commands.",
  "section": "cheat_sheet",
  "parent_id": null,
  "position": 0,
  "tags": ["Git", "CLI"]
}
```

- `title`, `section` required. `content` defaults to `""`.
- `parent_id` only accepted when `section = "project_note"`; parent must also
  be a `project_note`.
- `tags` are upserted by normalized name and attached.

Responses: `201` with the created `Document`. `400`, `409` (slug), `422`.

### 4.2 List documents

`GET /api/documents`

Query: `section` (repeatable), `tag` (repeatable), `parent_id`, `q`
(title-only filter), `include=content`, `sort=updated_at|created_at|title`,
`order=asc|desc`, `page`, `page_size`.

Response `200`:

```json
{
  "data": [ /* Document without content by default */ ],
  "pagination": { "page": 1, "page_size": 20, "total": 42, "total_pages": 3 }
}
```

- Never returns `private` documents.

### 4.3 Get document

`GET /api/documents/:id`

Response `200` with a full `Document` (content included). `404` if missing or
if the document is private and the session is locked.

### 4.4 Update document

`PUT /api/documents/:id`

Request: any subset of create fields plus `tags` (replaces the tag set when
present).

Response `200` with the updated `Document`. `400`, `404`, `409`, `422`.

### 4.5 Delete document

`DELETE /api/documents/:id`

Response `204`. Cascades children for project notes and removes tag links.

### 4.6 Document tree (Project Notes)

`GET /api/documents/tree?section=project_note`

Response `200`: recursive nodes.

```json
{
  "data": [
    {
      "id": "0f6b...",
      "title": "Frontend",
      "slug": "frontend",
      "section": "project_note",
      "position": 0,
      "children": [
        { "id": "1a2b...", "title": "Routing", "slug": "routing",
          "section": "project_note", "position": 0, "children": [] }
      ]
    }
  ]
}
```

### 4.7 Import Markdown

`POST /api/documents/import` — `multipart/form-data`

Fields:

- `files` — one or more `.md` files (required).
- `section` — target section (required; `private` requires a session).
- `parent_id` — optional, for importing into a Project Notes folder.
- `tags` — optional comma-separated default tags applied to every file.

Response `200`:

```json
{
  "data": [
    { "filename": "git.md", "status": "created", "document_id": "0f6b..." },
    { "filename": "notes.md", "status": "skipped", "reason": "empty_file" }
  ],
  "summary": { "created": 1, "skipped": 1, "failed": 0 }
}
```

- Title is derived from YAML front-matter `title`, then the first `# heading`,
  then the filename.
- Front-matter `tags` are merged with the `tags` field.
- `413` when a file or the total upload exceeds the configured limit.

## 5. Tags API

### 5.1 List tags

`GET /api/tags?q=&sort=name|count&order=asc|desc&page=&page_size=`

Response `200`: `{ "data": [Tag...], "pagination": {...} }`.
Counts exclude private documents.

### 5.2 Popular tags

`GET /api/tags/popular?limit=12`

Response `200`: `{ "data": [{ "name": "Git", "count": 12 }, ...] }`.

### 5.3 Documents by tag

`GET /api/tags/:name/documents?page=&page_size=`

Response `200`: paginated `Document` list (no content). `404` if the tag is
unknown.

## 6. Search API

`GET /api/search?q=&section=&page=&page_size=`

Response `200`:

```json
{
  "data": [
    {
      "id": "0f6b...",
      "title": "Git Commands",
      "slug": "git-commands",
      "section": "cheat_sheet",
      "tags": ["Git", "CLI"],
      "snippet": "...<mark>git</mark> rebase...",
      "rank": 0.87,
      "updated_at": "2026-09-17T16:30:00Z"
    }
  ],
  "pagination": { "page": 1, "page_size": 20, "total": 1, "total_pages": 1 },
  "query": "git rebase"
}
```

- Full-text over title + content + tags using `websearch_to_tsquery`.
- **Always excludes `private` documents, regardless of session state**
  (PRD 3.3).
- `snippet` is a server-generated highlighted excerpt.

## 7. Private Archive API

### 7.1 Unlock

`POST /api/private/unlock`

Request: `{ "password": "..." }`

Response `200`:

```json
{ "data": { "token": "opaque-session-token", "expires_at": "2026-09-17T17:07:00Z" } }
```

- Token lifetime: 15 minutes, sliding (each private request refreshes it).
- Response `401` on wrong password; `429` after 5 failed attempts within 15
  minutes. Failed and successful unlocks are rate-limited separately.

### 7.2 Private documents

All require header `Authorization: Bearer <token>` or the
`selfdocs_private` HttpOnly cookie.

- `GET /api/private/documents` — list private documents.
- `POST /api/private/documents` — create a private document.
- `GET /api/private/documents/:id` — read one.
- `PUT /api/private/documents/:id` — update one.
- `DELETE /api/private/documents/:id` — delete one.

Same shapes as the public document endpoints. `401` when the session is
missing, expired, or invalid.

### 7.3 Lock

`POST /api/private/lock`

Response `204`. Invalidates the current session token.

### 7.4 Private isolation guarantees

- Private documents never appear in `/api/documents`, `/api/tags*`,
  `/api/search`, `/api/dashboard`, or `/api/activity`.
- Activity rows are not created for private mutations.
- `POST /api/documents` with `section = "private"` returns `403`; use the
  `/api/private/documents` endpoint instead.

## 8. Dashboard API

`GET /api/dashboard`

Response `200`:

```json
{
  "cards": [
    { "id": "workflow", "title": "Workflows & Guides",
      "count": 8, "route": "/workflows" },
    { "id": "project_note", "title": "Project Notes",
      "count": 14, "route": "/knowledge-base" },
    { "id": "cheat_sheet", "title": "Cheat Sheets",
      "count": 6, "route": "/cheat-sheets" },
    { "id": "private", "title": "Private Archive",
      "count": null, "route": "/private", "locked": true }
  ],
  "recently_updated": [
    { "id": "0f6b...", "title": "Frontend Guide",
      "section": "workflow", "updated_at": "2026-09-17T16:30:00Z" }
  ],
  "popular_tags": [{ "name": "Git", "count": 12 }],
  "activity": [ /* Activity entry... */ ]
}
```

- `locked` state is per session; the private card shows `locked: true` and
  omits counts (it stays `false` when the private session is unlocked).
- `recently_updated`, `popular_tags`, and `activity` exclude private content.
- Cached in Redis when configured (PRD 5); cache is invalidated on document
  mutations.

## 9. Activity API

`GET /api/activity?page=&page_size=`

Response `200`: `{ "data": [ActivityEntry...], "pagination": {...} }`.
Chronological, newest first. Private actions are never recorded or returned.

## 10. Settings API

- `GET /api/settings` — `{ "data": { "version": "0.1.0", "redis_enabled": false } }`.
- `PUT /api/settings/master-password` — body:
  `{ "current_password": "...", "new_password": "..." }`. Requires a valid
  private session or a correct `current_password`. Response `204`.
- `GET /healthz` — unauthenticated liveness probe:
  `{ "status": "ok", "database": "ok", "redis": "disabled" }`.

## 11. CORS and transport

- Allowed origins come from `CORS_ORIGINS`; default `http://localhost:5173`.
- `Authorization` header and credentials are allowed.
- All responses are JSON except `204` responses and file uploads.

## 12. Frontend route map

Routes consumed by the SPA, aligned with the header navigation in `DESIGN.md`.

| Route            | Page            | API                                     |
| ---------------- | --------------- | --------------------------------------- |
| `/`              | Dashboard       | `GET /api/dashboard`                    |
| `/documents`     | All documents   | `GET /api/documents`                    |
| `/workflows`     | Workflows       | `GET /api/documents?section=workflow`   |
| `/knowledge-base`| Project Notes   | `GET /api/documents/tree`               |
| `/cheat-sheets`  | Cheat Sheets    | `GET /api/documents?section=cheat_sheet`|
| `/private`       | Private Archive | `/api/private/*`                        |
| `/search`        | Search results  | `GET /api/search`                       |
| `/settings`      | Settings        | `GET /api/settings`                     |
| `/documents/:id` | Viewer          | `GET /api/documents/:id`                |
| `/editor/:id?`   | Create/Edit     | documents CRUD / import                 |
