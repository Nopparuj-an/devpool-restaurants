# Architecture

> Status: initial proposal. Items marked *(proposed)* are defaults that can still change. Record any change as an ADR.

## Components

```
Browser
  │  https://localhost  (same origin)
  ▼
Next.js (App Router)      ── /api/*  rewrite ──▶  Go API  ──▶  PostgreSQL
  • Server Components fetch the Go API,                  │
    forwarding the session cookie                        └──▶  Garage (S3) — restaurant images
  • Client Components for forms / interactive UI
```

- **Same origin:** Next.js rewrites `/api/*` to the Go service, so the browser only ever talks to one origin. The session cookie stays `SameSite=Lax; HttpOnly` with no CORS setup. *(proposed)*
- **Go API:** stdlib `net/http` routing, `pgx` for Postgres, embedded `goose` migrations. The rules engine is `backend/internal/booking` (pure functions, no I/O).
- **PostgreSQL:** relational data, the transactions and row locks used for booking, and `timestamptz`.
- **Garage:** S3-compatible object storage for images. See [ADR-0005](decisions/0005-garage-object-storage.md).
- **Local dev ([ADR-0009](decisions/0009-dev-stack-and-local-infra.md)):** `make env && make up && make garage-init` starts Postgres and Garage with data bind-mounted in `deployment/data/`. `make api` runs the Go API on the host.

## Authentication ([ADR-0002](decisions/0002-go-native-auth.md))

- Two login methods: **email + password** (bcrypt/argon2id) and **Google OAuth** (OIDC, authorization-code flow handled by Go).
- **Sessions:** the server stores sessions in Postgres. The browser holds an opaque random token in an `HttpOnly; Secure; SameSite=Lax` cookie, and Postgres stores only its hash. Logout deletes the row. There are no JWTs.
- **Account linking:** a Google login whose `email_verified=true` email matches an existing account is linked to that account. Google takes priority: the existing password and sessions are dropped, and the user can set a new password (ADR-0002).
- **Authorization:** middleware loads the account from the session. Handlers check ownership (`restaurant.owner_id == account.id`, `reservation.account_id == account.id`) for every mutation.

## Data model (sketch)

```
accounts(id, email UNIQUE, email_verified, display_name, created_at)
auth_identities(id, account_id → accounts, provider 'password'|'google',
                provider_subject, password_hash NULL, UNIQUE(provider, provider_subject))
sessions(token_hash PK, account_id → accounts, expires_at, created_at)

restaurants(id, owner_id → accounts, name, description, cuisine, location,
            seats CHECK (seats >= 1),
            cancel_cutoff_minutes DEFAULT 30 CHECK (>= 30 AND % 15 = 0),
            max_reservation_minutes DEFAULT 240 CHECK (>= 15 AND % 15 = 0),
            timezone NOT NULL,            -- from owner's browser at creation, hidden
            rating_sum INT DEFAULT 0, review_count INT DEFAULT 0,
            created_at, updated_at)
restaurant_hours(restaurant_id → restaurants ON DELETE CASCADE,
                 weekday 0-6, open_time TIME, close_time TIME,   -- close<open: next day; close=open: 24h
                 PRIMARY KEY (restaurant_id, weekday))
restaurant_images(id, restaurant_id → restaurants ON DELETE CASCADE,
                  object_key, is_cover, position)

reservations(id, restaurant_id → restaurants ON DELETE CASCADE,
             account_id → accounts, pax, starts_at timestamptz, ends_at timestamptz,
             status 'active'|'cancelled', created_at, updated_at,
             CHECK (ends_at > starts_at))
  INDEX (restaurant_id, starts_at, ends_at) WHERE status = 'active'

reviews(id, restaurant_id → restaurants ON DELETE CASCADE, account_id → accounts,
        rating CHECK (1..5), body, created_at, updated_at,
        UNIQUE (restaurant_id, account_id))
```

Deleting image objects from Garage when a restaurant is deleted happens in application code after the DB transaction commits. DB cascades can't reach S3.

## Booking write path ([ADR-0003](decisions/0003-booking-concurrency.md))

```
BEGIN
  SELECT … FROM restaurants WHERE id = $1 FOR UPDATE      -- serialize per restaurant
  validate R-BOOK-1..4, 8 (pax, grid+max duration, future, open time, horizon)
  SELECT pax, starts_at, ends_at FROM reservations
   WHERE restaurant_id = $1 AND status = 'active'
     AND starts_at < $new_end AND ends_at > $new_start
     AND id <> $self                                       -- edits only
  peak := sweep(events)                                   -- pure Go func, unit-tested
  if peak + pax > seats and not shrinkEdit → ROLLBACK, 409  -- R-EDIT-3
  INSERT / UPDATE reservation
COMMIT
```

## API conventions

| Situation | Status |
|---|---|
| Malformed JSON / unknown fields | `400` `bad_request` |
| Validation error (bad grid, past time, outside hours, pax < 1) | `422` with `{ "error": { "code": "R-BOOK-3", "message": "…" } }` |
| Not logged in | `401` |
| Logged in but not owner / not your reservation / reviewing own restaurant | `403` |
| Not found (also used for others' private resources if we want to hide existence) | `404` |
| Capacity exceeded, or past the cancel cutoff | `409` |
| Created | `201` + resource |

All timestamps in requests and responses are UTC ISO-8601 (`2026-10-11T05:00:00Z`). The frontend converts to the browser's timezone (ADR-0007).

Error `code`s reuse the rule IDs from [domain.md](domain.md). The frontend maps each code to a message.
