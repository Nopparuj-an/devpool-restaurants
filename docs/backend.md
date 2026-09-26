# Backend map

How the Go API is laid out, so you can find things without reading code first. The layout is hexagonal ("ports and adapters"), one folder per feature, using Gin ([ADR-0011](decisions/0011-gin-hexagonal-layout.md)).

## One shape for every feature

Every feature folder (`auth`, `restaurant`, `reservation`, `review`) has the same parts:

| File or folder | What's in it | Knows about |
|---|---|---|
| `routes.go` | Every URL of the feature and the handler it calls. **Start here.** | Gin |
| `handler.go` (+ `*_handler.go`) | Reads the request, calls the service, writes JSON. No business rules. | Gin, service |
| `module.go` | Wiring: repository → service → handler | everything in the feature |
| `model/` | Types (request, response, domain) and the feature's errors | nothing |
| `service/` | **Business rules.** Declares `Service` (what handlers call) and `Repository` (what it needs from storage) as interfaces. | model, `booking`, interfaces only |
| `repository/` | SQL that implements the service's `Repository` interface | Postgres (pgx) |
| `*_test.go` next to `routes.go` | API tests for the feature, run over real HTTP and a real database | |

```
HTTP request
   │
   ▼
routes.go ──► handler.go ──► service.Service ──► service.Repository ◄── repository/ ──► Postgres
 (URL)        (HTTP in/out)   (rules, tx)         (interface = port)      (SQL = adapter)
```

Dependencies point inwards. Services don't import Gin or pgx, so the rules can be read, and tested, without HTTP or SQL.

## The tree

```
backend/
  bruno/                     Bruno collection: every endpoint, runnable end to end
  cmd/
    api/main.go              start the server: config → database → server.New
    seed/main.go             demo data, through the same services as the API
  internal/
    server/server.go         composition root: Gin engine, middleware, every module's routes
    booking/                 PURE seat and time rules, no I/O (R-BOOK-*, R-HOURS-*, R-EDIT-*, R-CANCEL-*)
      rules.go               CheckNew / CheckEdit / CheckCancel
      peak.go                PeakLoad (the sweep), LimitedThreshold
      hours.go               opening hours → open periods, overnight and 24h
      booking.go             Interval, Reservation, Restaurant, 15-minute grid
    auth/                    accounts, password + Google login, sessions (ADR-0002)
      routes.go  handler.go  google_handler.go  middleware.go  module.go
      model/ service/ repository/
    restaurant/              restaurant CRUD, hours, photos (R-REST-*)
      routes.go  handler.go  image_handler.go  module.go
      model/ service/ (validate.go = input rules)  repository/ (rules.go = the booking lock)
    reservation/             book / change / cancel, availability, owner table
      routes.go  handler.go  module.go  model/ service/ repository/
    review/                  ratings and reviews (R-REVIEW-*); repository has RecomputeRatings
      routes.go  handler.go  module.go  model/ service/ repository/
    admin/                   moderation: users, restaurants, reversible bans (R-ADMIN-*)
      routes.go  handler.go  module.go  model/ service/ repository/
    platform/                shared plumbing, no business rules
      apperr/                errors with a Kind (NotFound, Conflict, …), no HTTP
      imageproc/             shrink photos to ≤1600 px JPEG, reject decompression bombs
      web/                   Gin helpers: Kind → status code, JSON decode, path IDs
      database/              pool, migrations/, WithinTx + Conn (transaction in ctx)
      storage/               S3 (Garage) image store + in-memory fake
      config/                environment variables
    testutil/
      apitest/               full API on a throwaway DB, JSON client for tests
      testdb/                creates and drops a test database per test
```

## One request, end to end

`POST /api/restaurants/7/reservations` with `{pax, starts_at, ends_at}`:

1. `server/server.go`: the `/api` group runs `auth` middleware, which loads the account from the session cookie.
2. `reservation/routes.go`: `customer.POST("/restaurants/:id/reservations", h.Create)` sits behind `auth.RequireLogin()`.
3. `reservation/handler.go` → `Create`: decodes the JSON and calls `svc.Create(me, 7, input)`.
4. `reservation/service/reservation_service.go` → `Create`, inside `tx.WithinTx`:
   1. `repo.RestaurantRules(7, lock=true)` locks the restaurant row (`restaurant/repository/rules.go`, ADR-0003).
   2. `repo.Overlapping(...)` loads the active bookings in that time range.
   3. `booking.CheckNew(...)` runs the pure rules (`booking/rules.go`).
   4. `repo.Insert(...)` writes the booking, and the transaction commits.
5. If a rule fails, the service returns a `booking.Violation` or an `apperr.Error`. `platform/web` turns it into `409 {"error":{"code":"R-BOOK-5",…}}`.

## Where do I change…

| I want to change… | Go to |
|---|---|
| A URL, or which ones need login | `<feature>/routes.go` |
| The JSON a client sends or gets back | `<feature>/model/` |
| An error message or code | `<feature>/model/errors.go` (restaurant, auth) or the bottom of `<feature>/model/*.go` |
| A business rule about seats, hours, cutoffs | `booking/` (pure, with table tests in `booking_test.go`) |
| Any other business rule (ownership, reviews, linking Google…) | `<feature>/service/` |
| Restaurant input validation (lengths, cutoff steps, hours) | `restaurant/service/validate.go` |
| A SQL query | `<feature>/repository/` |
| The database schema | `platform/database/migrations/` (add a new numbered file) |
| How an error kind maps to a status code | `platform/web/web.go` |
| Cookies, sessions, who is logged in | `auth/middleware.go`, `auth/handler.go` |
| Wiring a new feature in | its `module.go`, then one line in `server/server.go` |
| Who is admin | the database: `make admin EMAIL=…` / `make admin-revoke EMAIL=…` |
| Trying an endpoint by hand | `backend/bruno/` in Bruno. Update it whenever you change routes |
| Demo data | `cmd/seed/main.go`; bulk load data in `cmd/seed/bulk.go` |
| How photos are resized | `platform/imageproc` (server), `frontend/lib/image.ts` (browser) |

## Endpoints → code

| Method and path | Handler | Service method |
|---|---|---|
| `POST /auth/signup`, `/auth/login`, `/auth/logout` | `auth/handler.go` `Signup`, `Login`, `Logout` | `auth` `Signup`, `Login`, `CreateSession`, `DeleteSession` |
| `GET /auth/providers` | `auth/handler.go` `Providers` | none |
| `GET /auth/google/start`, `/auth/google/callback` | `auth/google_handler.go` | `auth` `LoginWithGoogle` |
| `GET /me`, `PUT /me`, `PUT /me/password` | `auth/handler.go` `Me`, `UpdateProfile`, `SetPassword` | `auth` `UpdateProfile`, `SetPassword` |
| `GET /restaurants`, `GET /restaurants/:id` | `restaurant/handler.go` `List`, `Get` | `restaurant` `List`, `Get` |
| `POST`, `PUT`, `DELETE /restaurants[/:id]`, `GET /me/restaurants` | `restaurant/handler.go` `Create`, `Update`, `Delete`, `Mine` | `restaurant` same names |
| `POST /restaurants/:id/images`, `DELETE …/images/:imageID`, `PUT …/cover` | `restaurant/image_handler.go` | `restaurant` `AddImages`, `DeleteImage`, `SetCover` |
| `GET /restaurants/:id/availability` | `reservation/handler.go` `Availability` | `reservation` `Availability` |
| `POST /restaurants/:id/reservations` | `reservation/handler.go` `Create` | `reservation` `Create` |
| `GET /restaurants/:id/reservations` (owner) | `reservation/handler.go` `ListForOwner` | `reservation` `ListForOwner` |
| `GET /me/reservations`, `GET`, `PUT /reservations/:id`, `POST …/cancel` | `reservation/handler.go` | `reservation` `ListMine`, `Get`, `Update`, `Cancel` |
| `GET /restaurants/:id/reviews`, `GET`, `PUT`, `DELETE …/reviews/me` | `review/handler.go` | `review` `List`, `Mine`, `Upsert`, `Delete` |
| `GET /admin/users[/:id]`, `POST …/ban`, `…/unban` | `admin/handler.go` | `admin` `ListUsers`, `GetUser`, `BanUser`, `UnbanUser` |
| `GET /admin/restaurants`, `POST …/ban`, `…/unban` | `admin/handler.go` | `admin` `ListRestaurants`, `BanRestaurant`, `UnbanRestaurant` |

Request and response shapes are in [api.md](api.md).

## Rules → code

| Rule ([domain.md](domain.md)) | Enforced in |
|---|---|
| R-BOOK-1…5, 8, R-EDIT-*, R-CANCEL-1 | `booking/rules.go` (`checkShape`, `checkCapacity`, `checkCutoff`) |
| R-BOOK-5 peak load | `booking/peak.go` `PeakLoad` |
| R-BOOK-6 concurrency | `restaurant/repository/rules.go` `LoadRules(lock=true)` inside `reservation/service` `WithinTx` |
| R-HOURS-* | `booking/hours.go` (checking), `restaurant/service/validate.go` (input) |
| R-REST-1…6 | `restaurant/service/` (`validate.go`, `restaurant_service.go`) |
| R-REVIEW-* | `review/service/review_service.go`, plus "verified" in `review/repository` |
| R-SEATS-1 | `booking/peak.go` `LimitedThreshold`, `reservation/service` `Availability` |
| R-PRIV-1/2 | `reservation/service` `derive`, `review/service` `hideEmail` |
| R-ADMIN-1 | `auth/middleware.go` `RequireAdmin`, `admin/routes.go` |
| R-ADMIN-3 | `admin/service` `BanUser`, `auth` login (ErrBanned), `auth/repository` session query, `review/repository` `RecomputeRatings` |
| R-ADMIN-4 | `restaurant/repository` `Visible` (lists, C), `restaurant/service` `Get`, `restaurant/repository/rules.go` `LoadRules(includeHidden)` |
| R-ADMIN-5 | `admin/service` (guards, unban) |
| R-TIME-* | `cmd/api/main.go` (UTC), `platform/database` (session timezone) |

## Conventions

- **Errors.** Services return `apperr.Error` (Kind, Code, Message) or `booking.Violation`, and `platform/web` picks the status code. Handlers never write error JSON themselves; they `return err`.
- **Transactions.** A service wraps work in `tx.WithinTx(ctx, …)`. Repositories call `db.Conn(ctx)`, which is the transaction when there is one. SQL never decides where a transaction starts or ends. **Inside `WithinTx`, always use the ctx it passes in.** A closure that captures the outer ctx runs its SQL outside the transaction, and it waits forever if the transaction holds a row lock it needs. That happened once in `admin/service` and was caught by the tests.
- **Handlers** have the signature `func(c *gin.Context) error` and are wrapped with `web.Handle` in `routes.go`.
- **The logged-in account** comes from `auth.MustAccount(c)` behind `RequireLogin`, or `auth.ViewerID(c)` on public routes (0 = visitor).
- **Tests.** Pure rules: `booking/booking_test.go`. Each feature's API: `<feature>/<feature>_test.go` (real HTTP and DB, run with `make test`). Google linking: `auth/service/auth_service_test.go`.

## Scaling

**Every list is paged or bounded.** Restaurants, reviews and my bookings take `limit`/`offset` and return `total`. The restaurant list computes its total in the same query with `count(*) OVER ()`. Availability is capped at 7 days (672 slots), the owner table at 31 days. Search (`q`, name or cuisine) runs in SQL. The web app pages the home list 24 at a time and loads reviews and past bookings with "Show more".

**Indexes** (`migrations/00002_scaling_indexes.sql`): pg_trgm GIN indexes make `ILIKE '%…%'` search use an index. There are b-tree indexes for "Most reviewed", "New", and each restaurant's reviews. The booking overlap query uses `reservations_active_time_idx` from the first migration.

**Measured** with `make seed-bulk N=10000` on the dev laptop, through the API (20 requests each):

| Request | 1,006 restaurants | 10,006 restaurants (~50k reviews) |
|---|---|---|
| `/restaurants` (top rated, page 1) | p95 5.3 ms | p95 12.5 ms |
| `/restaurants?sort=most_reviewed` | p95 6.0 ms | p95 8.8 ms |
| `/restaurants?sort=newest&offset=9950` (last page) | p95 3.8 ms | p95 10.5 ms |
| `/restaurants?q=sabai` (trigram index) | p95 5.0 ms | p95 4.9 ms |
| `/restaurants/:id/reviews` | p95 3.7 ms | p95 3.8 ms |
| `/restaurants/:id/availability` (24h) | p95 3.9 ms | p95 3.6 ms |

`EXPLAIN ANALYZE` confirms the plans: search is a BitmapOr of both trigram indexes (0.7 ms), and "Most reviewed" reads 50 rows from its index (0.03 ms).

**Known limits, and the next step if they matter:**
- "Top rated" computes the Bayesian score for every row, then sorts. That's fine at 10k (12 ms). Around 100k+, store the score in a column updated with the rating totals, and index it.
- Offset paging reads and skips earlier rows, so page 400 costs more than page 1. Keyset ("after id X") paging would fix that, at the price of no "Page N of M".
- Photos are served straight from Garage, and the API never streams them.

`make seed-bulk N=…` adds generated users, restaurants (one photo each) and about 5N reviews. `make seed-bulk-remove` removes all of it, including the photos.
