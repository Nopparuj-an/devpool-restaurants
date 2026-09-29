# 0004 — Stored rating aggregate + Bayesian ranking

**Status:** Accepted (2026-09-26). Decision 1 (how the totals are kept) superseded by [0015](0015-rating-totals-by-trigger.md).

## Context
Two questions from the brief:
1. Should the average be recomputed on every page view, or stored and updated when a review is written?
2. Should a ★5.0 from 1 review rank above a ★4.8 from 300 reviews under "Highest rated"?

## Decision
1. **Stored aggregate.** `restaurants.rating_sum` and `review_count` are updated **in the same transaction** as a review insert/update/delete, e.g. `rating_sum = rating_sum - old + new`. Sorting and pagination then read indexed columns instead of aggregating every review on each list request.
   *Trade-off to explain:* computing on read (`AVG()` + `GROUP BY`) is always consistent and simpler, and at exam scale it would be fast enough. The stored approach avoids repeated aggregation on the hot list query, at the cost of keeping two writes consistent. The shared transaction handles that. A DB trigger is an alternative.
2. **Bayesian average for ranking:** `score = (v·R + m·C) / (v + m)`
   - `R` = restaurant average, `v` = its review count
   - `C` = mean rating across all reviews, `m` = prior weight (start with 5)
   - A restaurant with few reviews is pulled toward the global mean, and the effect fades as reviews accumulate.
   - The UI still **displays** the raw average and count. The score is only used for ordering.

## Worked example (from `TestTopRatedIsBayesian`)
Lucky has 1 × 5★. Solid has 10 reviews averaging 4.8★. Meh has 10 × 3★. The global mean is C ≈ 3.95, and m = 5:
- Solid = (48 + 5·3.95) / 15 ≈ **4.52**
- Lucky = (5 + 5·3.95) / 6 ≈ **4.13**
- Meh ≈ 3.32

Solid ranks above Lucky, even though Lucky's raw average is higher.

**Caveat worth explaining in the interview:** the answer depends on C. If *every* restaurant is rated about 4.8 (C ≈ 4.8), a single 5★ is still evidence of "above average". Lucky = (5 + 24) / 6 ≈ 4.83 would then beat Solid at 4.8. The formula is doing its job there. To make one review count for less, raise m (e.g. 10–20) or require a minimum count to enter the ranking.

## Consequences
- `C` changes whenever any review changes. Compute it in the list query (`SELECT AVG(rating) FROM reviews` is cheap) or cache it.
- Deleting a review must decrement the aggregate too.
