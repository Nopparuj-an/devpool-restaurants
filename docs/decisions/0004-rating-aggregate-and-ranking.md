# 0004 — Stored rating aggregate + Bayesian ranking

**Status:** Accepted (2026-09-26)

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

## Consequences
- `C` changes whenever any review changes. Compute it in the list query (`SELECT AVG(rating) FROM reviews` is cheap) or cache it.
- Deleting a review must decrement the aggregate too.
