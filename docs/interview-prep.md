# Interview prep

The interview checks *foundation*: can you explain your own code and why. Keep answers tied to this repo (link the file once written).

## From the brief

| Question | Our answer (short) | Where |
|---|---|---|
| 6 seats left, two people book 6 at the same second — how many succeed, where is it guarded? | Exactly one. Tx locks the restaurant row (`FOR UPDATE`), re-checks peak load, inserts; the second waits, sees the new booking, gets 409. Guard is in Postgres, not the frontend or Go memory. | ADR-0003 |
| Restaurant open 18:00–02:00 — how stored and checked? | Hours as wall-clock `TIME` per weekday + the restaurant's hidden tz (from owner's browser). `close < open` = next day, `close == open` = 24h. Expand today's and yesterday's shifts to UTC instants, merge adjacent ones, booking must fit in one merged period. Reservations are UTC instants so midnight is not special. | ADR-0007, R-HOURS-* |
| ★5.0 from 1 review vs ★4.8 from 300? | No. Rank by Bayesian average pulling low-count restaurants to the global mean; display raw average + count. | ADR-0004 |
| Compute average on read or store it? | Stored sum/count updated in the same tx as the review. Trade-off vs on-read explained in ADR. | ADR-0004 |
| Why can A=7, B=7, C=3 fit in 10 seats? | A and B don't overlap; capacity is the **peak** concurrent pax, not the sum. Sweep line. | R-BOOK-5 |

## Next.js topics
- Server vs Client Components: which of our pages are which, and why (list/detail = server; booking form = client).
- Data fetching & forms: how Server Components call the Go API with the cookie; Server Actions vs client fetch for mutations.
- Login/session: where the session lives (Postgres row, opaque HttpOnly cookie), why not localStorage/JWT.

## Go topics
- HTTP handlers & status codes: our table in architecture.md (401 vs 403 vs 404 vs 409 vs 422).
- Preventing double booking: ADR-0003, and why an in-process mutex is insufficient.
- Time & timezone: `timestamptz`, `time.LoadLocation`, `time/tzdata`, half-open intervals.
- SQL & transactions: isolation level we run at (READ COMMITTED) and why the row lock makes it safe; what happens on rollback.

## Justify our choices
- Postgres (transactions, row locks, `timestamptz`, constraints) · Go-native auth over Keycloak (ADR-0002) · Garage over MinIO/disk (ADR-0005) · 15-min grid (ADR-0001).
