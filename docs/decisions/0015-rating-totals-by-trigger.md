# 0015 — Rating totals recomputed by database triggers

**Status:** Accepted (2026-09-29). Supersedes decision 1 of [ADR-0004](0004-rating-aggregate-and-ranking.md) (how the stored totals are kept) and the rating part of [ADR-0012](0012-admin-bans.md). The Bayesian ranking in ADR-0004 is unchanged.

## Context
`restaurants.rating_sum` and `review_count` were kept by the Go review service, which added and subtracted deltas (`rating_sum + new − old`), and by the admin ban flow, which rebuilt them. Any change made outside those code paths left them wrong for good. Examples: deleting an account in SQL (its reviews go by cascade), fixing a review by hand, or a future feature that forgets the update. The reviews list also uses `review_count` as its paging total, so drift breaks paging too. Deltas never correct themselves, and they can go negative.

Computing the totals on read would always be right, but "Top rated" would have to aggregate every review on every list request, and reads far outnumber review writes.

## Decision
- **Recompute, don't add deltas.** Whenever a restaurant's reviews change, its totals are rebuilt from its reviews: one indexed `sum`/`count` over that restaurant's reviews by accounts that aren't banned (R-ADMIN-3). Writes cost a little more, reads stay as cheap as before, and any drift is corrected by the restaurant's next change.
- **In the database, not the app** (migration `00005_rating_triggers.sql`). Statement-level `AFTER INSERT/UPDATE/DELETE` triggers on `reviews` call `recompute_ratings(ids)` once per affected restaurant, whoever made the change, including foreign-key cascades. A text-only edit changes nothing. A row trigger on `accounts.banned_at` handles ban and unban. The Go code no longer touches the totals.
- **Lock, then aggregate.** `recompute_ratings` locks the restaurant rows (in id order) before it aggregates, in a separate statement. Without this, two concurrent writers each count without the other's uncommitted review, and the last one to commit loses a review (`TestTotalsConcurrentWriters` shows this). The review service already holds this row lock, so the app path waits on nothing new. We use a row lock rather than advisory locks, because a bulk insert touching 10,000 restaurants would take 10,000 advisory locks and could run out of lock-table slots.
- **Guard and repair.** A CHECK constraint (`0 ≤ review_count`, `review_count ≤ rating_sum ≤ 5 · review_count`) turns impossible totals into an error instead of stored data. `make ratings-recompute` rebuilds every restaurant, for what triggers can't see: `TRUNCATE`, or a session with `session_replication_role = replica`.

## Consequences
- A review write does one extra aggregate over that restaurant's reviews (under 1 ms for 150 reviews). Deleting an account recomputes each restaurant it reviewed once, not once per review.
- Business logic now lives partly in SQL. The "banned reviewers don't count" rule is written in `recompute_ratings`, next to the list query that hides them, and nowhere in Go.
- Bulk loads get totals for free, and seeds no longer compute them.
- Totals that are wrong but within the valid range (e.g. edited by hand) stay wrong until that restaurant's next review change or a `make ratings-recompute`.
