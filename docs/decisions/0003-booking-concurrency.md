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
- Required tests: a unit test for `peakLoad()` (brief examples, back-to-back bookings, edit excluding self) and an integration test that fires N concurrent goroutines at the last seats and asserts that exactly one succeeds.
