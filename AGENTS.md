# AGENTS.md

Restaurant reservation + review app (PEA DevPool 2026 final exam). Next.js frontend, Go backend, Postgres, Garage (S3).

**Start with [`docs/README.md`](docs/README.md).** It is the wiki index and the source of truth.

- Business rules: `docs/domain.md`. Every rule has an ID (`R-BOOK-5`) that is reused in code comments, tests, and API error codes.
- Rules are enforced in Go (`backend/internal/booking` is pure logic, with no I/O). The frontend may check them again but is never the only guard.
- Backend layout: **`docs/backend.md`**. Each feature is `routes.go → handler.go → service/ (rules, ports) → repository/ (SQL)`. Keep Gin out of services and SQL out of services.
- Missing or unclear rule → add it to `docs/open-questions.md` and ask. Don't decide product behavior silently.
- Decisions that constrain future work → new ADR in `docs/decisions/`.
- When you add or change an API route, update `docs/api.md` **and** the Bruno collection in `backend/bruno/` (check with `npx @usebruno/cli run --env example` from that folder).

## Commands

```sh
make env          # first time: create deployment/.env with random secrets
make up           # Postgres + Garage (data in deployment/data/)
make garage-init  # first time: Garage layout, key, bucket, website mode
make seed         # demo data (no-op if already seeded)
make seed-bulk N=10000   # load-test data; make seed-bulk-remove to undo
make api          # run Go API on :8080 (applies migrations)
make test         # backend tests
make app-up       # API + web in containers (web on :3000); app compose is separate from infra
```

Frontend: `cd frontend && pnpm dev`. Next.js 16 differs from older versions; read `frontend/AGENTS.md` first. UI conventions and copy rules: `docs/design.md`; the gallery is at `/design`. Never import plain values from a `"use client"` module into server code (see docs/design.md).
