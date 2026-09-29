-- +goose Up
-- Rating totals are kept by the database (ADR-0015). Any change to reviews,
-- including cascades from deleting an account or restaurant and edits made
-- by hand in SQL, recomputes the affected restaurants' rating_sum and
-- review_count from their reviews. Only reviews by accounts that aren't
-- banned count (R-ADMIN-3), so a ban or unban recomputes too.

-- recompute_ratings rebuilds the totals of the given restaurants. It locks
-- their rows first (in id order, so two callers can't deadlock), then
-- aggregates in a separate statement: in READ COMMITTED that statement takes
-- a fresh snapshot, so it sees a concurrent writer's review once that writer
-- commits. The review service already holds this row lock, so the app path
-- waits on nothing new. `make ratings-recompute` runs it for every restaurant.
-- +goose StatementBegin
CREATE FUNCTION recompute_ratings(ids BIGINT[]) RETURNS void LANGUAGE plpgsql AS $$
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

-- One call per statement, not per row: deleting an account with 150 reviews
-- recomputes each restaurant it reviewed once. Postgres allows transition
-- tables on single-event triggers only, hence one function per event.
-- +goose StatementBegin
CREATE FUNCTION reviews_inserted() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    PERFORM recompute_ratings(ARRAY(SELECT DISTINCT restaurant_id FROM new_rows));
    RETURN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION reviews_updated() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    -- Only rows whose rating or restaurant changed; a text edit costs nothing.
    PERFORM recompute_ratings(ARRAY(
        SELECT o.restaurant_id FROM old_rows o JOIN new_rows n ON n.id = o.id
        WHERE (o.rating, o.restaurant_id) IS DISTINCT FROM (n.rating, n.restaurant_id)
        UNION
        SELECT n.restaurant_id FROM old_rows o JOIN new_rows n ON n.id = o.id
        WHERE (o.rating, o.restaurant_id) IS DISTINCT FROM (n.rating, n.restaurant_id)));
    RETURN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION reviews_deleted() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    PERFORM recompute_ratings(ARRAY(SELECT DISTINCT restaurant_id FROM old_rows));
    RETURN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION account_ban_changed() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    PERFORM recompute_ratings(ARRAY(SELECT restaurant_id FROM reviews WHERE account_id = NEW.id));
    RETURN NULL;
END $$;
-- +goose StatementEnd

-- Repair anything that drifted before this migration, then guard the range:
-- ratings are 1 to 5, so the sum lies between the count and five times it.
SELECT recompute_ratings(ARRAY(SELECT id FROM restaurants));
ALTER TABLE restaurants ADD CONSTRAINT restaurants_rating_totals_check
    CHECK (review_count >= 0 AND rating_sum BETWEEN review_count AND 5 * review_count);

CREATE TRIGGER reviews_ratings_insert AFTER INSERT ON reviews
    REFERENCING NEW TABLE AS new_rows FOR EACH STATEMENT EXECUTE FUNCTION reviews_inserted();
CREATE TRIGGER reviews_ratings_update AFTER UPDATE ON reviews
    REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows FOR EACH STATEMENT EXECUTE FUNCTION reviews_updated();
CREATE TRIGGER reviews_ratings_delete AFTER DELETE ON reviews
    REFERENCING OLD TABLE AS old_rows FOR EACH STATEMENT EXECUTE FUNCTION reviews_deleted();
CREATE TRIGGER accounts_ratings_ban AFTER UPDATE OF banned_at ON accounts
    FOR EACH ROW WHEN ((OLD.banned_at IS NULL) <> (NEW.banned_at IS NULL))
    EXECUTE FUNCTION account_ban_changed();

-- +goose Down
DROP TRIGGER accounts_ratings_ban ON accounts;
DROP TRIGGER reviews_ratings_delete ON reviews;
DROP TRIGGER reviews_ratings_update ON reviews;
DROP TRIGGER reviews_ratings_insert ON reviews;
ALTER TABLE restaurants DROP CONSTRAINT restaurants_rating_totals_check;
DROP FUNCTION account_ban_changed();
DROP FUNCTION reviews_deleted();
DROP FUNCTION reviews_updated();
DROP FUNCTION reviews_inserted();
DROP FUNCTION recompute_ratings(BIGINT[]);
