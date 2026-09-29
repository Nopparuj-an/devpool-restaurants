-- +goose Up
-- Per-star review counts (R-REVIEW-8), kept by the same triggers as the
-- totals (ADR-0015): rating_counts[k] is the number of k-star reviews that
-- count, so the breakdown and the filtered total cost nothing to read.
ALTER TABLE restaurants ADD COLUMN rating_counts INT[] NOT NULL DEFAULT '{0,0,0,0,0}';

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION recompute_ratings(ids BIGINT[]) RETURNS void LANGUAGE plpgsql AS $$
BEGIN
    PERFORM 1 FROM restaurants WHERE id = ANY (ids) ORDER BY id FOR NO KEY UPDATE;
    UPDATE restaurants r SET rating_sum = s.sum, review_count = s.n, rating_counts = s.counts
    FROM (
        SELECT r2.id, coalesce(sum(rv.rating), 0) AS sum, count(rv.id) AS n,
            ARRAY[count(*) FILTER (WHERE rv.rating = 1), count(*) FILTER (WHERE rv.rating = 2),
                  count(*) FILTER (WHERE rv.rating = 3), count(*) FILTER (WHERE rv.rating = 4),
                  count(*) FILTER (WHERE rv.rating = 5)]::INT[] AS counts
        FROM restaurants r2
        LEFT JOIN (reviews rv JOIN accounts a ON a.id = rv.account_id AND a.banned_at IS NULL)
            ON rv.restaurant_id = r2.id
        WHERE r2.id = ANY (ids)
        GROUP BY r2.id
    ) s
    WHERE s.id = r.id
        AND (r.rating_sum, r.review_count, r.rating_counts) IS DISTINCT FROM (s.sum, s.n, s.counts);
END $$;
-- +goose StatementEnd

SELECT recompute_ratings(ARRAY(SELECT id FROM restaurants));
-- The counts must add up to the totals.
ALTER TABLE restaurants ADD CONSTRAINT restaurants_rating_counts_check CHECK (
    cardinality(rating_counts) = 5
    AND rating_counts[1] >= 0 AND rating_counts[2] >= 0 AND rating_counts[3] >= 0
    AND rating_counts[4] >= 0 AND rating_counts[5] >= 0
    AND rating_counts[1] + rating_counts[2] + rating_counts[3] + rating_counts[4] + rating_counts[5] = review_count
    AND rating_counts[1] + 2 * rating_counts[2] + 3 * rating_counts[3] + 4 * rating_counts[4] + 5 * rating_counts[5] = rating_sum
);

-- A restaurant's reviews with one star rating, newest or oldest first.
CREATE INDEX reviews_restaurant_rating_updated_idx ON reviews (restaurant_id, rating, updated_at DESC, id DESC);

-- +goose Down
DROP INDEX reviews_restaurant_rating_updated_idx;
ALTER TABLE restaurants DROP CONSTRAINT restaurants_rating_counts_check;
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION recompute_ratings(ids BIGINT[]) RETURNS void LANGUAGE plpgsql AS $$
BEGIN
    PERFORM 1 FROM restaurants WHERE id = ANY (ids) ORDER BY id FOR NO KEY UPDATE;
    UPDATE restaurants r SET rating_sum = s.sum, review_count = s.n
    FROM (
        SELECT r2.id, coalesce(sum(rv.rating), 0) AS sum, count(rv.id) AS n
        FROM restaurants r2
        LEFT JOIN (reviews rv JOIN accounts a ON a.id = rv.account_id AND a.banned_at IS NULL)
            ON rv.restaurant_id = r2.id
        WHERE r2.id = ANY (ids)
        GROUP BY r2.id
    ) s
    WHERE s.id = r.id AND (r.rating_sum, r.review_count) IS DISTINCT FROM (s.sum, s.n);
END $$;
-- +goose StatementEnd
ALTER TABLE restaurants DROP COLUMN rating_counts;
