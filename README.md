# Restaurants

A restaurant reservation and review community built for the PEA DevPool 2026 final exam. Every account can open its own restaurants and book or review anyone else's.

- **Backend:** Go (stdlib `net/http`, pgx, goose), PostgreSQL 18
- **Images:** Garage (S3-compatible), served publicly
- **Frontend:** Next.js 16. The design gallery is at `/design`

Design notes, business rules and decisions live in **[docs/](docs/README.md)**.

## Run it

### With Docker only

Requirements: Docker (with Compose), `make`, `openssl`.

```sh
make env          # create deployment/.env with random secrets (first time only)
make up           # infra: Postgres + Garage; data is kept in deployment/data/
make garage-init  # create the image bucket (first time only)
make app-up       # build + start the API and web containers
make app-seed     # demo accounts, restaurants, bookings, reviews
```

Open **http://localhost:3000**. The infra and the app are separate compose files (`deployment/docker-compose.yml` and `deployment/docker-compose.app.yml`), so `make app-down` stops only the app.

### For development (hot reload)

Additional requirements: Go 1.27+, Node 24 with pnpm (`corepack enable`).

```sh
make env && make up && make garage-init   # same infra as above
make seed                                 # demo data
make seed-bulk N=10000                    # optional load-test data (make seed-bulk-remove to undo)
make api                                  # API on http://localhost:8080
cd frontend && pnpm install && pnpm dev   # web on http://localhost:3000
```

Endpoints are listed in [docs/api.md](docs/api.md).

To start over, run `make down && rm -rf deployment/data`, then repeat the steps above from `make up`.

### Google login (optional)

Create an OAuth client (type "Web application") in Google Cloud Console with the redirect URI `http://localhost:3000/api/auth/google/callback`. Then set `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` in `deployment/.env`. When they're empty, only email/password login is offered.

## Test accounts

All passwords are `password123`.

| Email | Role in the seed data |
|---|---|
| `somchai@example.com` | Owns ครัวริมคลอง, ส้มตำหน้าตลาด, ก๋วยเตี๋ยวเรือหน้าวัด |
| `malee@example.com` | Owns Midnight Moo Kra Ta (18:00–02:00) and Sabai 24h Café (open 24h) |
| `alice@example.com` | Customer with upcoming and past bookings and verified reviews |
| `bob@example.com`, `carol@example.com`, `dan@example.com` | Customers |

ครัวริมคลอง is nearly full at lunch on the next open day, so it shows **Limited Seats Left**.

## Tests

```sh
make test
```

- Pure rule tests: `backend/internal/booking` (capacity sweep, opening hours incl. overnight and 24h, edit and cancel rules).
- API integration tests run against a throwaway database on the compose Postgres, one per test. They include the brief's seat examples and a lock test that fails if the booking row lock is removed.
- UI: `cd frontend && pnpm e2e` drives the real app in headless Chrome or Edge (sign up, book, change, cancel, review, create and edit a restaurant, log out). It needs the dev setup running.

## Layout

```
backend/     Go API; booking rules in internal/booking (pure, no I/O)
frontend/    Next.js app; screens in components/screens, design gallery at /design
deployment/  compose files (infra + app), Garage config, .env, bind-mounted data/
docs/        wiki: requirements, domain rules, architecture, ADRs, API
```
