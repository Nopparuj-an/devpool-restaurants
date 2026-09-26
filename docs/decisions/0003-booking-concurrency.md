# 0003 — Per-restaurant row lock for booking writes

**Status:** Accepted (2026-09-26)

## Context
Say 6 seats are left and two customers each request 6 at the same instant. If both read the load before either inserts, both pass and the restaurant ends up with 12 over capacity. A check-then-insert must be atomic with respect to other writes for the same restaurant.

## Options
| Option | Pros | Cons |
|---|---|---|
| **`SELECT … FROM restaurants WHERE id=$1 FOR UPDATE` at the start of the tx** | Simple, easy to explain, only blocks bookings for the *same* restaurant | Serializes that restaurant's bookings (fine at this scale) |
| `pg_advisory_xact_lock(restaurant_id)` | Same effect without touching the row | Less obvious to readers |
| `SERIALIZABLE` isolation + retry on `40001` | No explicit lock | Needs a retry loop, and failures are harder to reason about in an interview |
| Exclusion constraint | Great for "no overlap" | Can't express "sum of pax ≤ seats" |
| Check only in app memory / a mutex | — | Breaks with more than one API instance, and the frontend isn't trustworthy anyway |

## Decision
Row lock on the restaurant (`FOR UPDATE`) inside one transaction that runs the full validation, the peak-load sweep, and the insert/update. The second of two concurrent requests waits, then sees the committed booking and gets `409`.

## Consequences
- Seat edits by the owner also lock the same row, so they are serialized with bookings.
- Tests (`backend/internal/reservation/reservation_test.go`):
  - `TestBookingWaitsForRestaurantLock` is the **deterministic proof**. A test transaction holds the lock, and a booking request must block, then get `409` once the holder commits. With `FOR UPDATE` removed, this test fails with `201` (overbooking). That was verified on 2026-09-26.
  - `TestConcurrentLastSeats` is a smoke test: 12 goroutines race for the last 6 seats and exactly one wins. **Lesson learned:** it also passed with the lock removed (0 failures in 20 runs), because the requests rarely overlap in time. A race test that passes proves nothing unless you've seen it fail without the fix.
- Lock order is always restaurant, then reservation (create, edit, cancel), so two writers can't deadlock.
