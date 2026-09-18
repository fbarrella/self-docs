# self-docs

A self-hosted personal knowledge base for professional documentation, project
notes, guides, cheat sheets, and private notes. Built with a React/Vite
frontend, a Go/Gin backend, PostgreSQL, optional Redis, and Docker Compose.

The product requirements live in [`PRD.md`](./PRD.md), the UI specification in
[`DESIGN.md`](./DESIGN.md), and the task breakdown in
[`DEVELOPMENT_PLAN.md`](./DEVELOPMENT_PLAN.md). The frozen contracts are
[`docs/data-model.md`](./docs/data-model.md) and
[`docs/api.md`](./docs/api.md).

## Repository layout

```
backend/    Go/Gin API, migrations, Dockerfile, and internal packages
frontend/   React + Vite + TypeScript SPA, Dockerfile, and nginx config
docker-compose.yml  Full local stack (db, backend, frontend, optional cache)
docs/       API contract and data model documentation
progress.txt  Append-only log of completed tasks
```

## Prerequisites

- Go 1.24+
- Node.js 20+ and npm
- Docker and Docker Compose (for the containerized stack)

## Local development

### Backend

```bash
cd backend
go mod tidy
go build ./...
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

## Docker Compose (recommended)

The whole stack (PostgreSQL, Go API, nginx-served SPA) runs with one command:

```bash
cp .env.example .env   # optional; defaults work out of the box
docker compose up --build
```

Then open http://localhost:8080. The frontend serves the SPA and proxies `/api`
to the backend, so the browser only talks to one origin. PostgreSQL data is
stored in the `self-docs_db-data` volume and survives restarts.

Optional Redis cache:

```bash
REDIS_URL=redis://cache:6379 docker compose --profile cache up --build
```

Set `MASTER_PASSWORD_HASH` in `.env` to seed the Private Archive master
password at first boot (see `.env.example` for how to generate it), or configure
it later from the Settings page. Useful overrides: `APP_PORT`, `POSTGRES_PORT`,
`POSTGRES_USER`/`POSTGRES_PASSWORD`/`POSTGRES_DB`.

## Tooling

- Backend: `gofmt`, `go vet ./...`, `golangci-lint run ./...` (config in
  `backend/.golangci.yml`), and `go test ./...`.
- Integration tests: set `TEST_DATABASE_URL` to a disposable PostgreSQL
  database to run the repository, handler, and router test suites; they are
  skipped otherwise. The suites serialize themselves via an advisory lock.
- Frontend: `oxlint` (`npm run lint`), Prettier (`npm run format`),
  TypeScript (`npm run typecheck`)

## Task workflow

Each task in `DEVELOPMENT_PLAN.md` is implemented, verified, and then logged to
`progress.txt` using the `progress-log` skill before moving to the next task.
