# self-docs - Development Plan

Derived from `PRD.md` (features) and `DESIGN.md` (UI). The application is a
self-hosted personal knowledge base: React/Vite frontend, Go/Gin backend,
PostgreSQL, optional Redis, orchestrated with Docker Compose.

Each task has a stable ID (`T<phase>.<n>`). After completing a task, log it to
`progress.txt` using the `progress-log` skill, then stop and wait for the user
before starting the next task.

Task states: `[ ]` pending, `[~]` in progress, `[x]` done.

---

## Phase 0 - Foundation & Contracts

### T0.1 - Repository scaffolding and tooling
- **Goal:** Create the monorepo layout and baseline developer tooling.
- **Deliverables:** `frontend/`, `backend/`, `deploy/`, root `README.md`,
  root `.gitignore`, `.editorconfig`, `opencode.json` (if needed).
- **Details:** Go module init (`go mod init`), Vite React scaffold,
  npm scripts, formatting/lint config (ESLint/Prettier, `gofmt`).
- **Depends on:** none.
- **Verification:** `go build ./...` and `npm install` succeed.

### T0.2 - Shared API contract and data model design
- **Goal:** Freeze the REST contract and domain model before implementation.
- **Deliverables:** `docs/api.md` with endpoints, request/response JSON,
  status codes; `docs/data-model.md` describing entities.
- **Details:** Document types enum (`workflow`, `project_note`, `cheat_sheet`,
  `private`), parent/child hierarchy for project notes, tag relations,
  activity actions, master-password validation flow.
- **Depends on:** T0.1.
- **Verification:** Contract reviewed against PRD sections 3.1-3.4.

---

## Phase 1 - Backend Persistence

### T1.1 - PostgreSQL schema and migrations
- **Goal:** Implement the schemas required by PRD section 6.1.
- **Deliverables:** `backend/migrations/*.sql` for `documents`, `tags`,
  `document_tags`, `activity_logs` (plus a `settings` table for the master
  password hash).
- **Details:** UUID keys, `section`/`doc_type` enum, nullable `parent_id` self
  reference, `created_at`/`updated_at`, full-text search vector column on
  `documents` (title + content), indexes for `updated_at`, tags, hierarchy.
- **Depends on:** T0.2.
- **Verification:** `psql` apply against a local container; `\dt` lists all tables.

### T1.2 - Backend bootstrap: config, DB pool, migrations runner
- **Goal:** Stand up the Gin server with config and DB connectivity.
- **Deliverables:** `backend/cmd/server/main.go`, config loader (env vars),
  pgx/sqlx pool, migration runner, health endpoint.
- **Details:** Env: `DATABASE_URL`, `PORT`, `MASTER_PASSWORD_HASH`,
  `REDIS_URL` (optional), `CORS_ORIGINS`. Fail fast on missing required config.
- **Depends on:** T1.1.
- **Verification:** `GET /healthz` returns `200`; migrations run on startup.

### T1.3 - Domain models and repository layer
- **Goal:** Isolate all SQL behind repositories used by handlers.
- **Deliverables:** `backend/internal/model/*.go`,
  `backend/internal/repository/*.go` for documents, tags, activity logs.
- **Details:** CRUD + list-with-filters + full-text search queries + tag
  upsert/attach/detach + activity insert/list.
- **Depends on:** T1.2.
- **Verification:** Repository unit tests against a disposable test database.

---

## Phase 2 - Backend API (Go/Gin)

### T2.1 - Documents CRUD API
- **Goal:** Implement PRD 6.2 CRUD endpoints (section 3.1).
- **Deliverables:** `backend/internal/handler/document_handler.go`, routes in
  `backend/internal/router/router.go`.
- **Endpoints:** `POST /api/documents`, `GET /api/documents`,
  `GET /api/documents/:id`, `PUT /api/documents/:id`,
  `DELETE /api/documents/:id`.
- **Details:** Validate section/doc_type, enforce hierarchy rules for project
  notes, accept `tags[]`, write activity logs on create/update/delete.
- **Depends on:** T1.3.
- **Verification:** `curl`/HTTP test suite covering each verb and error cases.

### T2.2 - Tags API
- **Goal:** Support tag listing, popularity, and filtering (PRD 3.1, 3.4).
- **Deliverables:** tag handlers and routes.
- **Endpoints:** `GET /api/tags`, `GET /api/tags/popular`,
  `GET /api/tags/:tag/documents`.
- **Details:** Popularity ordering with usage counts, exclude private docs.
- **Depends on:** T2.1.
- **Verification:** Tests asserting private documents never appear in tag results.

### T2.3 - Markdown import API
- **Goal:** Import existing `.md` files (PRD 3.1).
- **Deliverables:** `POST /api/documents/import` (multipart upload).
- **Details:** Accept single or multiple files, derive title from filename/H1,
  parse front-matter tags when present, return per-file results.
- **Depends on:** T2.1.
- **Verification:** Upload a sample `.md`; document is persisted with tags.

### T2.4 - Global full-text search API
- **Goal:** Implement PRD 3.3 using PostgreSQL `to_tsvector`.
- **Deliverables:** `GET /api/search?q=` handler.
- **Details:** Search title, tags, and content; rank results; **hard-exclude
  `private` section**; return highlighted snippets and section labels.
- **Depends on:** T2.1, T2.2.
- **Verification:** Test proves a private-only term returns zero results while
  a public term returns ranked hits.

### T2.5 - Private Archive access control
- **Goal:** Master-password validation and enforcement (PRD 3.2, 5).
- **Deliverables:** `POST /api/private/unlock`, auth middleware for private
  routes, hashing utilities (bcrypt/argon2).
- **Details:** Compare against stored hash; issue a short-lived signed session
  token (HttpOnly cookie or bearer); guard every private document read/write;
  keep private data out of home lists, tags, search, and activity feeds.
- **Depends on:** T1.2, T2.1.
- **Verification:** Unauthenticated private access returns `401`; wrong password
  rejected; correct password grants scoped access.

### T2.6 - Dashboard aggregation API
- **Goal:** One call powering the home page (PRD 3.4).
- **Deliverables:** `GET /api/dashboard`.
- **Details:** Returns navigation card metadata, recently updated (excluding
  private), popular tags, and recent activity feed; supports cache headers.
- **Depends on:** T2.1, T2.2.
- **Verification:** Test asserts private documents are absent from every field.

### T2.7 - Activity log API
- **Goal:** Feed of user actions (PRD 3.4).
- **Deliverables:** `GET /api/activity` (paginated).
- **Details:** Recorded by handlers across create/update/delete/import/unlock;
  readable phrases like "Updated Guide: Frontend".
- **Depends on:** T1.3, T2.1.
- **Verification:** Actions performed via API appear in chronological order.

### T2.8 - Backend hardening and tests
- **Goal:** Cross-cutting quality gate for the API.
- **Deliverables:** Validation middleware, centralized error responses,
  request logging, CORS config, rate limit on unlock, integration test suite.
- **Depends on:** T2.1-T2.7.
- **Verification:** `go test ./...` passes; `golangci-lint run` clean.

---

## Phase 3 - Frontend Foundation

### T3.1 - Vite/React app and routing
- **Goal:** Establish the SPA shell and route map.
- **Deliverables:** `frontend/src/main.tsx`, `App.tsx`, router config, API
  client module, env config for backend URL.
- **Routes:** `/` dashboard, `/documents`, `/knowledge-base` (project notes),
  `/cheat-sheets`, `/workflows`, `/private`, `/settings`.
- **Depends on:** T0.1, T0.2.
- **Verification:** `npm run build` succeeds; all routes render placeholders.

### T3.2 - Design system and theming
- **Goal:** Encode `DESIGN.md` tokens once and reuse everywhere.
- **Deliverables:** CSS variables / theme file, base typography (sans-serif),
  color tokens (off-white bg `#f8f9fa`, white cards, `#212529` text,
  `#6c757d` secondary, lime accent `#c4e349`, tag bg `#e9ecef`, border
  `#dee2e6`), radius/shadow/spacing scales.
- **Depends on:** T3.1.
- **Verification:** Story/demo page renders tokens swatches matching DESIGN.md.

### T3.3 - Base UI components
- **Goal:** Build reusable primitives from DESIGN.md 2.3-2.5.
- **Deliverables:** `Button` (pill, lime accent), `Card`, `TagPill`, `Input`,
  `SearchBar` (pill with inner green Search button), `ListRow`, `Modal`,
  `Avatar`, `Dropdown`, `IconBadge`, `SectionHeader`, `EmptyState`,
  `Skeleton`, `Spinner`.
- **Depends on:** T3.2.
- **Verification:** Component gallery renders all states (default/hover/disabled).

### T3.4 - Layout shell: Header, Footer, responsive navigation
- **Goal:** Implement DESIGN.md 2.1, 2.5, and section 3 responsiveness.
- **Deliverables:** `Header` (logo left, nav center/right: Home, Documents,
  Knowledge Base, Private Notes, Settings; avatar dropdown with global settings
  only; no bell icon), `Footer` (3 columns: brand/copyright/legal, Portal
  links, Useful links), mobile hamburger menu.
- **Depends on:** T3.3.
- **Verification:** Visual check at desktop/tablet/mobile breakpoints.

---

## Phase 4 - Frontend Features

### T4.1 - Dashboard page
- **Goal:** Implement PRD 3.4 and DESIGN.md 2.2-2.4.
- **Deliverables:** `Dashboard` page with Welcome hero ("Welcome back",
  "Your Personal Knowledge Base.", subtitle, global search input), 4 navigation
  cards (Workflows & Guides, Project Notes, Cheat Sheets, Private Archive),
  Recently Updated list, Popular Tags pills, Activity Feed.
- **Details:** Consume `GET /api/dashboard`; responsive grid 4/2/1; two-column
  lower area stacking on mobile; private content never shown.
- **Depends on:** T2.6, T3.4.
- **Verification:** Matches reference layout; private docs absent.

### T4.2 - Document listing pages
- **Goal:** Browsable, filterable lists per section.
- **Deliverables:** `Documents`, `Workflows`, `CheatSheets` pages; tag filtering;
  sort by updated date.
- **Depends on:** T2.1, T2.2, T3.4.
- **Verification:** Filtering by tag updates the list via API.

### T4.3 - Markdown editor (create/edit)
- **Goal:** In-app document authoring (PRD 3.1).
- **Deliverables:** `Editor` page embedding MDEditor/Milkdown with live preview,
  title field, section selector, tag input, save/update/delete actions.
- **Depends on:** T2.1, T2.2, T3.3.
- **Verification:** Create and edit a document end-to-end; content persists.

### T4.4 - Markdown import UI
- **Goal:** Client for PRD 3.1 import.
- **Deliverables:** Import dialog with drag-and-drop, file list, progress, and
  result summary wired to `POST /api/documents/import`.
- **Depends on:** T2.3, T3.3.
- **Verification:** Drag a `.md` file, confirm it appears in the target section.

### T4.5 - Document viewer and Project Notes hierarchy
- **Goal:** Read documents and navigate nested project notes (PRD 3.2).
- **Deliverables:** `DocumentView` with rendered Markdown, TOC, tags, metadata;
  `KnowledgeBase` page with collapsible tree/sidebar and breadcrumbs.
- **Depends on:** T2.1, T3.3.
- **Verification:** Deep-linked nested note renders with correct breadcrumb.

### T4.6 - Global search experience
- **Goal:** Frontend for PRD 3.3.
- **Deliverables:** Search page/results with title, section, tag, and content
  snippets; query in URL; debounced input; zero-state messaging.
- **Depends on:** T2.4, T3.3.
- **Verification:** Searching a public term returns hits; a private-only term
  returns none even when unlocked.

### T4.7 - Private Archive UI and master-password modal
- **Goal:** PRD 3.2 private section.
- **Deliverables:** Private card/route interception opening a password modal;
  unlock call to `/api/private/unlock`; session handling; locked/error states;
  private document list and viewer once unlocked.
- **Depends on:** T2.5, T3.3.
- **Verification:** Clicking the card or navigating directly prompts for the
  password; wrong password shows an error; correct password reveals content.

### T4.8 - Settings page
- **Goal:** Global configuration surface (PRD 4, header dropdown).
- **Deliverables:** `Settings` page to change master password, manage/delete
  tags, and view app info/health.
- **Depends on:** T2.5, T3.4.
- **Verification:** Changing the master password invalidates the old one.

### T4.9 - State management, API error handling, UX polish
- **Goal:** Consistent data/loading/error behavior across the SPA.
- **Deliverables:** Data-fetching layer (React Query or equivalent), toasts,
  optimistic updates, error boundaries, skeletons for lists/search.
- **Depends on:** T4.1-T4.8.
- **Verification:** Backend offline shows graceful errors, not blank screens.

---

## Phase 5 - Infrastructure & Deployment

### T5.1 - Backend Dockerfile
- **Goal:** Reproducible Go build image.
- **Deliverables:** `backend/Dockerfile` (multi-stage, static binary,
  non-root runtime).
- **Depends on:** T2.8.
- **Verification:** `docker build` succeeds; container starts and serves `/healthz`.

### T5.2 - Frontend Dockerfile
- **Goal:** Serve the built SPA.
- **Deliverables:** `frontend/Dockerfile` (build stage + nginx static serving
  with SPA fallback and `/api` proxy).
- **Depends on:** T3.1.
- **Verification:** `docker build` succeeds; app loads and reaches the API.

### T5.3 - Docker Compose orchestration
- **Goal:** One-command local stack (PRD 2).
- **Deliverables:** `docker-compose.yml` with services `frontend`, `backend`,
  `db` (PostgreSQL + volume + healthcheck), optional `cache` (Redis),
  and `.env.example`.
- **Details:** Startup ordering via `depends_on` healthchecks; persistent
  volume for Postgres; seed migration; documented ports.
- **Depends on:** T1.1, T5.1, T5.2.
- **Verification:** `docker compose up --build` yields a working app at the
  documented URL.

### T5.4 - Redis caching layer (optional/recommended)
- **Goal:** PRD 2 and 5 performance for search and dashboard.
- **Deliverables:** Redis client with graceful fallback when unset; cached
  dashboard response and popular tags with invalidation on document writes.
- **Depends on:** T2.6, T5.3.
- **Verification:** Second dashboard request is served from cache; writes
  invalidate the cache; app runs normally with Redis disabled.

---

## Phase 6 - Quality, Security & Documentation

### T6.1 - Responsive and accessibility audit
- **Goal:** Satisfy DESIGN.md section 3 and basic a11y.
- **Deliverables:** Breakpoint fixes (cards 4/2/1, stacked lower area,
  hamburger nav), keyboard navigation, focus states, ARIA labels, contrast pass.
- **Depends on:** Phase 4.
- **Verification:** Manual audit at desktop/tablet/mobile; keyboard-only runthrough.

### T6.2 - Security review of the private vault
- **Goal:** Harden PRD 5 security requirements.
- **Deliverables:** Documented review of password hashing, token lifetime,
  leak checks across endpoints, and a design note for future encryption at rest.
- **Depends on:** T2.5, T4.7, T2.8.
- **Verification:** Checklist shows no endpoint leaks private metadata.

### T6.3 - End-to-end smoke tests
- **Goal:** Validate the critical user journeys.
- **Deliverables:** E2E suite (Playwright) covering create/edit, import, search,
  private unlock, and dashboard rendering.
- **Depends on:** T5.3.
- **Verification:** E2E suite passes against the compose stack.

### T6.4 - Documentation and seed content
- **Goal:** Make the project self-explanatory and demo-ready.
- **Deliverables:** Root `README.md` (setup, env vars, run, backup),
  `docs/` updates, seed script with sample documents/tags.
- **Depends on:** T5.3.
- **Verification:** A new user follows README to a running instance.

---

## Recommended Implementation Order

1. **Phase 0** (T0.1 -> T0.2) - freeze layout and contract.
2. **Phase 1** (T1.1 -> T1.2 -> T1.3) - persistence first.
3. **Phase 2** (T2.1 -> T2.2 -> T2.3 -> T2.4 -> T2.5 -> T2.6 -> T2.7 -> T2.8)
   - complete, tested API before UI.
4. **Phase 3** (T3.1 -> T3.2 -> T3.3 -> T3.4) - shell and design system.
5. **Phase 4** (T4.1 ... T4.9) - features, dashboard first, then authoring,
   reading, search, private, settings, polish.
6. **Phase 5** (T5.1, T5.2 -> T5.3 -> T5.4) - containerize, then cache.
7. **Phase 6** (T6.1 -> T6.2 -> T6.3 -> T6.4) - audit, test, document.

## Critical Paths & Notes

- **Private isolation is cross-cutting:** enforced in T2.4, T2.5, T2.6, T2.7
  and re-verified in T4.6, T4.7, T6.2.
- **Redis is optional:** T5.4 must degrade gracefully if Redis is unavailable.
- **No conventional login:** single-user local-first (PRD 5); only the private
  vault requires the master password.
- **Workflow:** after each task, invoke the `progress-log` skill to append an
  entry to `progress.txt`, then pause for user confirmation before the next task.
