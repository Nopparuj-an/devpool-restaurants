# 0006 — Owner edits don't touch bookings; delete cascades

**Status:** Accepted (2026-09-26)

## Decision
- Changing seats, opening hours, or cancel cutoff affects **new** bookings and **later edits** only. Existing reservations are left as they are, even if they now fall outside the hours or exceed the new seat count (R-REST-4).
- Deleting a restaurant **hard-deletes** its reservations, reviews, and images (R-REST-5).

## Consequences
- A restaurant can be temporarily "over capacity" after the owner lowers seats. That is accepted.
- In that state, a normal capacity check could reject an edit that only *reduces* a booking. R-EDIT-3 fixes this: edits that don't grow the booking's pax or time window skip the capacity check.
- Customers with upcoming bookings at a deleted restaurant lose them silently. The owner must confirm the deletion in a dialog showing the count of upcoming reservations. Soft delete or notifications could be revisited later.
