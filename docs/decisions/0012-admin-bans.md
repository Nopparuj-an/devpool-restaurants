# 0012 — Admins and reversible bans

**Status:** Accepted (2026-09-26)

## Context
Admins need to moderate users and restaurants. Admin rights come from the database only. Bans must hide content and remove it from ratings, and must be undoable.

## Decision
- `accounts.is_admin` is set by operators (`make admin EMAIL=…`). There is no API to grant it, which removes a whole class of privilege-escalation bugs.
- A ban is a timestamp plus a reason (`banned_at`, `ban_reason`) on the account or restaurant. Nothing is deleted. "Hidden" is computed in queries (`restaurant/repository.Visible`), so unbanning is simply clearing the timestamp.
- Ratings are stored totals (ADR-0004). When a user is banned or unbanned, `RecomputeRatings` rebuilds the totals of every restaurant they reviewed from the reviews that still count, in the same transaction. We rebuild instead of adding and subtracting, so repeated or out-of-order bans can't make the totals drift.
- Banning a user deletes their sessions. The session query also ignores banned accounts, in case a session is created in a race.
- Restaurant bans don't touch the stored totals. Hidden restaurants simply drop out of lists and out of the global mean C.
- Admins can't ban themselves or other admins, so there is always someone who can unban.

## Consequences
- Every public query has to include the visibility condition. This is centralized in `Visible` and `LoadRules(includeHidden)` and covered by `admin_test.go`.
- A banned user's future bookings keep their seats. That's intended, so unbanning restores everything, but an admin may want to contact the restaurants. Automatic cancellation could be added later.
