-- +goose Up
-- Admin impersonation (R-ADMIN-7, ADR-0014): a session can be a stand-in
-- that an admin opened as another user. It stays valid only while that
-- admin is still an admin and not banned.
ALTER TABLE sessions ADD COLUMN impersonator_id BIGINT REFERENCES accounts ON DELETE CASCADE;

-- Public profiles list a user's reviews, newest first.
CREATE INDEX reviews_account_updated_idx ON reviews (account_id, updated_at DESC, id DESC);

-- +goose Down
DROP INDEX reviews_account_updated_idx;
ALTER TABLE sessions DROP COLUMN impersonator_id;
