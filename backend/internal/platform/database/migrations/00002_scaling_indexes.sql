-- +goose Up
-- Indexes for browsing thousands of restaurants and reviews (docs/backend.md#scaling).

-- Trigram indexes let `ILIKE '%thai%'` search use an index instead of scanning every row.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX restaurants_name_trgm_idx ON restaurants USING gin (name gin_trgm_ops);
CREATE INDEX restaurants_cuisine_trgm_idx ON restaurants USING gin (cuisine gin_trgm_ops);

-- Sort orders of the restaurant list ("Most reviewed", "New").
CREATE INDEX restaurants_review_count_idx ON restaurants (review_count DESC, name);
CREATE INDEX restaurants_created_idx ON restaurants (created_at DESC, id DESC);

-- A restaurant's reviews, newest first, one page at a time.
CREATE INDEX reviews_restaurant_updated_idx ON reviews (restaurant_id, updated_at DESC, id DESC);

-- +goose Down
DROP INDEX reviews_restaurant_updated_idx;
DROP INDEX restaurants_created_idx;
DROP INDEX restaurants_review_count_idx;
DROP INDEX restaurants_cuisine_trgm_idx;
DROP INDEX restaurants_name_trgm_idx;
