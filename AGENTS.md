# AGENTS.md

Restaurant reservation + review app (PEA DevPool 2026 final exam). Next.js frontend, Go backend, Postgres, Garage (S3).

**Start with [`docs/README.md`](docs/README.md).** It is the wiki index and the source of truth.

- Business rules: `docs/domain.md`. Every rule has an ID (`R-BOOK-5`) that is reused in code comments, tests, and API error codes.
- Rules are enforced in Go (`backend/internal/booking` is pure logic, with no I/O). The frontend may check them again but is never the only guard.
- Missing or unclear rule → add it to `docs/open-questions.md` and ask. Don't decide product behavior silently.
- Decisions that constrain future work → new ADR in `docs/decisions/`.

## Commands

```sh
make env          # first time: create deployment/.env with random secrets
make up           # Postgres + Garage (data in deployment/data/)
make garage-init  # first time: Garage layout, key, bucket, website mode
make seed         # demo data (no-op if already seeded)
make api          # run Go API on :8080 (applies migrations)
make test         # backend tests
```
