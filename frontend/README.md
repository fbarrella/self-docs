# self-docs frontend

React + Vite + TypeScript single-page application for the self-docs personal
knowledge base. UI specifications are in [`../DESIGN.md`](../DESIGN.md).

## Scripts

- `npm run dev` — start the Vite dev server
- `npm run build` — type-check and produce a production build in `dist/`
- `npm run preview` — preview the production build
- `npm run lint` — run oxlint
- `npm run format` — format with Prettier
- `npm run format:check` — verify formatting
- `npm run typecheck` — run the TypeScript compiler without emitting

## Environment

Copy `.env.example` to `.env` and adjust the backend URL when it differs from
the default.
