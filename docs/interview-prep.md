# Interview prep

The interview checks *foundation*: can you explain your own code and why. Keep answers tied to this repo (link the file once written).

## From the brief

| Question | Our answer (short) | Where |
|---|---|---|
| 6 seats left, two people book 6 at the same second — how many succeed, where is it guarded? | Exactly one. Tx locks the restaurant row (`FOR UPDATE`), re-checks peak load, inserts; the second waits, sees the new booking, gets 409. Guard is in Postgres, not the frontend or Go memory. **Proof:** `TestBookingWaitsForRestaurantLock` fails with 201 if you delete `FOR UPDATE`. Also a good story: the naive goroutine race test passed even without the lock. | ADR-0003 |
| Restaurant open 18:00–02:00 — how stored and checked? | Hours as wall-clock `TIME` per weekday + the restaurant's hidden tz (from owner's browser). `close < open` = next day, `close == open` = 24h. Expand today's and yesterday's shifts to UTC instants, merge adjacent ones, booking must fit in one merged period. Reservations are UTC instants so midnight is not special. | ADR-0007, R-HOURS-* |
| ★5.0 from 1 review vs ★4.8 from 300? | No. Rank by Bayesian average pulling low-count restaurants to the global mean; display raw average + count. | ADR-0004 |
| Compute average on read or store it? | Stored sum/count, recomputed from that restaurant's reviews by a DB trigger on every review change (reads are far more common than writes). Self-healing, covers SQL/cascade deletes, CHECK-guarded; row lock stops concurrent writers losing a review. | ADR-0004, ADR-0015 |
| Why can A=7, B=7, C=3 fit in 10 seats? | A and B don't overlap; capacity is the **peak** concurrent pax, not the sum. Sweep line. | R-BOOK-5 |

## Next.js topics
- Server vs Client Components: we chose a client-side app (ADR-0013). `page.tsx` files are server components that only set metadata; everything that fetches is a client component. Know what we gave up: server-rendered HTML and SEO for restaurant pages, and a spinner on first load.
- Data fetching & forms: TanStack Query for reads (`useQuery`, `useInfiniteQuery` for endless scroll, `keepPreviousData` so a new filter doesn't flash a spinner); writes are plain `fetch` calls followed by `invalidateQueries()`. Why not Server Actions: the backend is Go, so the browser talks to its API directly.
- Login/session: where the session lives (Postgres row, opaque HttpOnly cookie), why not localStorage/JWT, and how a path-based `/api` proxy keeps the cookie first-party without CORS.

## Go topics
- HTTP handlers & status codes: our table in architecture.md (401 vs 403 vs 404 vs 409 vs 422).
- Preventing double booking: ADR-0003, and why an in-process mutex is insufficient.
- Time & timezone: `timestamptz`, `time.LoadLocation`, `time/tzdata`, half-open intervals.
- SQL & transactions: isolation level we run at (READ COMMITTED) and why the row lock makes it safe; what happens on rollback.

## Timezone details
- `OpenPeriods` starts one day early because yesterday's shift may run past midnight into today. It builds each shift with `time.Date(..., loc)`, converts to UTC, and merges touching shifts. `Covers` requires the booking `[start, end)` to fit in one merged period.
- Why merge: a 24h restaurant or back-to-back shifts would otherwise reject a booking that spans two of them.
- DST is handled by `time.Date` in a `*time.Location`. Thailand has none, but the code is right where it exists.
- `time/tzdata` is embedded so `LoadLocation` works in slim images without `/usr/share/zoneinfo`.
- `LoadLocation` accepts `""` and `"Local"`; validation rejects them because they mean nothing to other viewers.
- Half-open `[start, end)`: a booking ending at 19:00 doesn't clash with one starting at 19:00.
- The 15-min grid is checked on UTC instants. Every real UTC offset is a multiple of 15 min.
- Trade-off: a viewer in another timezone sees hours converted to their own time. Accepted, since the audience is all in Asia/Bangkok.

## Booking and rating details
- Why not a Go mutex: it protects one process only. The lock is in Postgres, so any number of API instances are safe.
- Why lock the restaurant row: the new reservation doesn't exist yet, so there's nothing else to lock. It serializes bookings per restaurant only.
- `WithinTx` stores the tx in the `context`; repositories call `Conn(ctx)` and join it. Nested calls join the outer tx. An error from `fn` rolls back, nil commits.
- `booking` is pure logic with no I/O, and `now` is injected, so rules are unit-tested without a database or a real clock.
- Bayesian average: `(sum + m·C) / (count + m)`, `C` = mean of all visible reviews, `m` = `bayesianPrior`.
- Review `Upsert` locks the restaurant first so concurrent writers can't lose a review. Owner can't review their own restaurant; one review per person per restaurant.

## Auth
- Go-native auth, no Keycloak (ADR-0002). Two identity types: `password` and `google`.
- Session = Postgres row. Cookie holds a random opaque token; the DB stores only its SHA-256 hash, so a leaked DB gives no working sessions.
- Cookie flags: `HttpOnly` (XSS can't read it), `Secure` (HTTPS only), `SameSite=Lax` (cross-site POSTs don't carry it, which blocks CSRF).
- Why not JWT: it can't be revoked before expiry without a server-side list, which is a session table anyway. Sessions make logout, bans, and password changes instant.
- Middleware: `Session()` loads the account on every request and clears stale cookies. `RequireLogin()` gives 401 (`session_expired` if a dead cookie was sent, else `unauthorized`). `RequireAdmin()` gives 403.
- Google: authorization-code flow with PKCE, `state`, `nonce` in a 10-minute HttpOnly `google_flow` cookie. ID token verified with go-oidc. Requires `email_verified=true`. `next` limited to relative paths to stop open redirects.
- Pre-hijack: attacker signs up with the victim's email first, victim later uses Google and gets linked. Fix: first Google link deletes the password identity and all sessions, and marks the email verified. Test: `google_handler_test.go`.
- Impersonation (ADR-0014): a session row with `admin_id` and a short TTL.

## React Query details
- `staleTime` 30 s (5 min for `/me`), so going back to a page doesn't always refetch.
- Retry: not on 4xx, up to twice on network/5xx.
- A 401 on any read sets the `me` query to `null`, so the app shows logged out when a session expires mid-use.
- `useRequireAccount` redirects to `/login?next=...`. It only picks the screen; the API refuses the data anyway.
- Server data lives in the query cache; local UI state (form fields, dialogs) stays in `useState`.
- Google button is a plain link to the server-side flow. `GET /api/auth/providers` hides it when Google isn't configured.
- Same-origin `/api` via `proxy.ts` means no CORS and no `SameSite=None`.

## Scaling (1,000+ users or restaurants)
- **How does the list scale?** Every list is paged in SQL and returns a total. Search runs in SQL on trigram indexes, not in the browser. Measured with 10,006 restaurants and ~50k reviews: every list endpoint p95 ≤ 12.5 ms (docs/backend.md#scaling).
- **What breaks first?** "Top rated" scores every row before sorting, and deep offset pages. The fixes are a stored, indexed score column and keyset paging. Both are explained in docs/backend.md.
- **Photos?** Resized to ≤1600 px JPEG in the browser *and* on the server, so storage per restaurant stays small whatever the client does. Decompression bombs are rejected before decoding.

## Architecture questions
- **Why interfaces for repositories?** The service says what it needs (the port). The SQL is swappable and the rules don't depend on Postgres. Each feature's `Repository` interface doubles as a list of every query that feature runs.
- **Where does the transaction start?** In the service (`WithinTx`). Repositories only join it through the context. The booking lock needs the lock, the overlap read, and the insert in one transaction, and only the service knows that.
- **Why no DI container?** Four modules wired in `server.New` fit on one screen (ADR-0011).

## Justify our choices
- Postgres (transactions, row locks, `timestamptz`, constraints) · Go-native auth over Keycloak (ADR-0002) · Garage over MinIO/disk (ADR-0005) · 15-min grid (ADR-0001).
