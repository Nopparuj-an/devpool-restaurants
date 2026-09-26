# Restaurants

A restaurant reservation and review community built for the PEA DevPool 2026 final exam. Every account can open its own restaurants and book or review anyone else's.

- **Backend:** Go (stdlib `net/http`, pgx, goose), PostgreSQL 18
- **Images:** Garage (S3-compatible), served publicly
- **Frontend:** Next.js *(in progress)*

Design notes, business rules and decisions live in **[docs/](docs/README.md)**.

## Run it locally

Requirements: Docker (with Compose), Go 1.26+, `make`, `openssl`.

```sh
make env          # create deployment/.env with random secrets (first time only)
make up           # start Postgres + Garage; data is kept in deployment/data/
make garage-init  # create the image bucket (first time only)
make seed         # demo accounts, restaurants, bookings, reviews
make api          # API on http://localhost:8080 (applies migrations on start)
```

Check it: `curl localhost:8080/api/restaurants`. Endpoints are listed in [docs/api.md](docs/api.md).

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

## Layout

```
backend/     Go API; booking rules in internal/booking (pure, no I/O)
frontend/    Next.js app (in progress)
deployment/  docker compose, Garage config, .env, bind-mounted data/
docs/        wiki: requirements, domain rules, architecture, ADRs, API
```
