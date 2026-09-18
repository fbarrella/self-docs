# self-docs

A self-hosted personal knowledge base for professional documentation, project
notes, guides, cheat sheets, and private notes. Built with a React/Vite
frontend, a Go/Gin backend, PostgreSQL, optional Redis, and Docker Compose.

The product requirements live in [`PRD.md`](./PRD.md), the UI specification in
[`DESIGN.md`](./DESIGN.md), and the task breakdown in
[`DEVELOPMENT_PLAN.md`](./DEVELOPMENT_PLAN.md). The frozen contracts are
[`docs/data-model.md`](./docs/data-model.md),
[`docs/api.md`](./docs/api.md), and the
[private vault security review](./docs/security-review.md).

## Features

- Four content pillars: Workflows & Guides, Project Notes (nested), Cheat
  Sheets, and a password-protected Private Archive.
- Markdown authoring with live preview, plus a Markdown file importer.
- Global full-text search (PostgreSQL `to_tsvector`) that never includes
  private content.
- Dashboard with navigation cards, recently updated, popular tags, and an
  activity feed.
- Single-user, local-first: no accounts, only a master password for the vault.
- Optional Redis caching that degrades gracefully when unavailable.

## Repository layout

```
backend/    Go/Gin API, migrations, Dockerfile, and seed command
frontend/   React + Vite + TypeScript SPA, Dockerfile, nginx config, E2E tests
docker-compose.yml  Full local stack (db, backend, frontend, optional cache)
docs/       API contract, data model, and security review
progress.txt  Append-only log of completed tasks
```

## Quick start (Docker Compose)

The whole stack (PostgreSQL, Go API, nginx-served SPA) runs with one command:

```bash
cp .env.example .env   # optional; defaults work out of the box
docker compose up --build
```

Open <http://localhost:8080>. The frontend serves the SPA and proxies `/api` to
the backend, so the browser only talks to one origin. PostgreSQL data lives in
the `self-docs_db-data` volume and survives restarts.

To load sample documents (guides, cheat sheets, and a nested Project Notes
tree) into a fresh instance:

```bash
docker compose exec backend /usr/local/bin/seed
```

The seed is idempotent, so it is safe to run more than once.

### Optional Redis cache

```bash
REDIS_URL=redis://cache:6379 docker compose --profile cache up --build
```

The app runs normally without Redis; when it is set but unreachable, the
backend logs a warning and continues without caching.

## Configuration

All variables are documented in [`.env.example`](./.env.example). The most
important ones:

| Variable | Purpose | Default |
| --- | --- | --- |
| `APP_PORT` | Host port for the web app | `8080` |
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | Database credentials | `selfdocs` |
| `POSTGRES_PORT` | Host port for direct DB access | `5432` |
| `MASTER_PASSWORD_HASH` | bcrypt hash to seed the vault password at first boot | empty |
| `REDIS_URL` | Enables caching when set | empty |
| `CORS_ORIGINS` | Allowed browser origins | `http://localhost:8080` |
| `TRUSTED_PROXIES` | CIDRs whose forwarded headers are trusted | empty |
| `MAX_IMPORT_BYTES` | Per-file import size cap | 2 MiB |

Generate a master-password hash with:

```bash
htpasswd -bnBC 10 "" 'your-password' | tr -d ':\n'
```

Leave `MASTER_PASSWORD_HASH` empty to configure the password later from the
Settings page.

## Local development

### Backend

```bash
cd backend
cp .env.example .env   # set DATABASE_URL
go mod tidy
go run ./cmd/server
```

The server applies migrations on startup and listens on `PORT` (default 8080).

### Frontend

```bash
cd frontend
npm install
npm run dev
```

Set `VITE_API_BASE_URL` (see `frontend/.env.example`) to point at the backend.
During development the default `http://localhost:8080` is used.

## Tooling

- Backend: `gofmt`, `go vet ./...`, `golangci-lint run ./...` (config in
  `backend/.golangci.yml`), and `go test ./...`.
- Integration tests: set `TEST_DATABASE_URL` to a disposable PostgreSQL
  database to run the repository, handler, and router suites; they are skipped
  otherwise. The suites serialize themselves via an advisory lock. Set
  `TEST_REDIS_URL` to also exercise the cache integration tests.
- Frontend: `oxlint` (`npm run lint`), Prettier (`npm run format`),
  TypeScript (`npm run typecheck`).
- E2E smoke tests (Playwright): start the stack, then run the suite against it.

  ```bash
  docker compose up --build -d
  cd frontend
  npm run test:e2e          # targets http://localhost:8080 by default
  # override the target when needed:
  BASE_URL=http://localhost:8080 npm run test:e2e
  ```

  Coverage: dashboard rendering, create/view/edit/delete, tag filtering,
  global search, Markdown import, and the private archive unlock flow. Set
  `E2E_MASTER_PASSWORD` to the stack's master password when it is already
  configured.

## Backup and restore

PostgreSQL stores all data in the `self-docs_db-data` volume. To back it up:

```bash
docker compose exec db pg_dump -U selfdocs selfdocs > backup.sql
```

To restore into a fresh stack:

```bash
docker compose exec -T db psql -U selfdocs selfdocs < backup.sql
```

The optional Redis cache holds only disposable, invalidatable data; no backup
is needed.

## Troubleshooting

- **Blank dashboard / "Cannot reach the backend" banner.** The SPA is up but
  the API is not. Check `docker compose logs backend` and `docker compose ps`;
  the backend healthcheck gates the frontend.
- **Imports return 403.** Ensure the request is same-origin through the
  frontend (the backend strips same-host `Origin` headers automatically). When
  serving the SPA from a different origin, add it to `CORS_ORIGINS`.
- **Changes do not appear immediately.** The dashboard and popular tags are
  cached for up to 5 minutes (when Redis is enabled); document writes
  invalidate them automatically.
- **Locked out of the vault.** Rate limiting blocks unlock after 5 failed
  attempts within 15 minutes; wait for the window or restart the backend (the
  limiter is in-memory).

## Task workflow

Each task in `DEVELOPMENT_PLAN.md` is implemented, verified, and then logged to
`progress.txt` using the `progress-log` skill before moving to the next task.
