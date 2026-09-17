# self-docs

A self-hosted personal knowledge base for professional documentation, project
notes, guides, cheat sheets, and private notes. Built with a React/Vite
frontend, a Go/Gin backend, PostgreSQL, optional Redis, and Docker Compose.

The product requirements live in [`PRD.md`](./PRD.md), the UI specification in
[`DESIGN.md`](./DESIGN.md), and the task breakdown in
[`DEVELOPMENT_PLAN.md`](./DEVELOPMENT_PLAN.md).

## Repository layout

```
backend/    Go/Gin API, migrations, and internal packages
frontend/   React + Vite + TypeScript single-page app
deploy/     Dockerfiles, compose stack, and deployment assets
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

## Tooling

- Backend: `gofmt`/`go vet` (run via `go vet ./...`)
- Frontend: `oxlint` (`npm run lint`), Prettier (`npm run format`),
  TypeScript (`npm run typecheck`)

## Task workflow

Each task in `DEVELOPMENT_PLAN.md` is implemented, verified, and then logged to
`progress.txt` using the `progress-log` skill before moving to the next task.
