-- +goose Up
-- Admins and reversible bans (docs/domain.md#admin). Admin rights are granted
-- in the database only: UPDATE accounts SET is_admin = true WHERE email = '…'
-- (or `make admin EMAIL=…`).

ALTER TABLE accounts
    ADD COLUMN is_admin   BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN banned_at  TIMESTAMPTZ,              -- NULL = not banned
    ADD COLUMN ban_reason TEXT NOT NULL DEFAULT '';

ALTER TABLE restaurants
    ADD COLUMN banned_at  TIMESTAMPTZ,
    ADD COLUMN ban_reason TEXT NOT NULL DEFAULT '';

-- The admin user list: search by email or name, newest first.
CREATE INDEX accounts_email_trgm_idx ON accounts USING gin (email gin_trgm_ops);
CREATE INDEX accounts_display_name_trgm_idx ON accounts USING gin (display_name gin_trgm_ops);
CREATE INDEX accounts_created_idx ON accounts (created_at DESC, id DESC);

-- +goose Down
DROP INDEX accounts_created_idx;
DROP INDEX accounts_display_name_trgm_idx;
DROP INDEX accounts_email_trgm_idx;
ALTER TABLE restaurants DROP COLUMN ban_reason, DROP COLUMN banned_at;
ALTER TABLE accounts DROP COLUMN ban_reason, DROP COLUMN banned_at, DROP COLUMN is_admin;
