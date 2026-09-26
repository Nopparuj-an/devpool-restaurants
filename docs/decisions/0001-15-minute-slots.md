# 0001 — Reservations use a 15-minute grid

**Status:** Accepted (2026-09-26)

## Context
Free-form times make validation, UI pickers, and "seats left" displays harder, and they add edge cases (12:07–12:53) that don't help anyone.

## Decision
Reservation start/end, opening/closing times, and the cancel cutoff are all multiples of 15 minutes in restaurant local time. The backend rejects anything off the grid (R-BOOK-2).

## Consequences
- UI time pickers are simple dropdowns.
- "Seats left" can be shown per 15-minute slot for a day (~96 buckets max).
- The capacity check still uses the general sweep algorithm. The grid is an input constraint, not a storage model, so it can be relaxed later without a migration.
