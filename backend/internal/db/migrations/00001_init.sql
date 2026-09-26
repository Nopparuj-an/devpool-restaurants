-- +goose Up
-- Initial schema. See docs/architecture.md#data-model and docs/domain.md for the rules.

CREATE TABLE accounts (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email          TEXT NOT NULL,
    email_verified BOOLEAN NOT NULL DEFAULT false, -- true once proven via Google (ADR-0002)
    display_name   TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX accounts_email_key ON accounts (lower(email));

CREATE TABLE auth_identities (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id       BIGINT NOT NULL REFERENCES accounts ON DELETE CASCADE,
    provider         TEXT NOT NULL CHECK (provider IN ('password', 'google')),
    provider_subject TEXT NOT NULL, -- account id for 'password', Google "sub" for 'google'
    password_hash    TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_subject),
    UNIQUE (account_id, provider),
    CHECK ((provider = 'password') = (password_hash IS NOT NULL))
);

CREATE TABLE sessions (
    token_hash BYTEA PRIMARY KEY, -- sha256 of the cookie token; the token itself is never stored
    account_id BIGINT NOT NULL REFERENCES accounts ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX sessions_account_id_idx ON sessions (account_id);

CREATE TABLE restaurants (
    id                      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    owner_id                BIGINT NOT NULL REFERENCES accounts ON DELETE CASCADE,
    name                    TEXT NOT NULL,
    description             TEXT NOT NULL,
    cuisine                 TEXT NOT NULL,
    location                TEXT NOT NULL,
    seats                   INT NOT NULL CHECK (seats >= 1),                                          -- R-REST-1
    cancel_cutoff_minutes   INT NOT NULL DEFAULT 30 CHECK (cancel_cutoff_minutes >= 30 AND cancel_cutoff_minutes % 15 = 0), -- R-REST-3
    max_reservation_minutes INT NOT NULL DEFAULT 240 CHECK (max_reservation_minutes >= 15 AND max_reservation_minutes % 15 = 0), -- R-REST-6
    timezone                TEXT NOT NULL, -- IANA name from the owner's browser, hidden (R-TIME-5)
    rating_sum              INT NOT NULL DEFAULT 0, -- ADR-0004
    review_count            INT NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX restaurants_owner_id_idx ON restaurants (owner_id);

CREATE TABLE restaurant_hours (
    restaurant_id BIGINT NOT NULL REFERENCES restaurants ON DELETE CASCADE,
    weekday       SMALLINT NOT NULL CHECK (weekday BETWEEN 0 AND 6), -- 0 = Sunday, like Go's time.Weekday
    open_time     TIME NOT NULL CHECK (extract(minute FROM open_time)::int % 15 = 0 AND extract(second FROM open_time) = 0),
    close_time    TIME NOT NULL CHECK (extract(minute FROM close_time)::int % 15 = 0 AND extract(second FROM close_time) = 0),
    -- close < open: ends next day; close = open: 24h (R-HOURS-3)
    PRIMARY KEY (restaurant_id, weekday)
);

CREATE TABLE restaurant_images (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    restaurant_id BIGINT NOT NULL REFERENCES restaurants ON DELETE CASCADE,
    object_key    TEXT NOT NULL UNIQUE, -- restaurants/{restaurant_id}/{uuid}.{ext} (ADR-0005)
    is_cover      BOOLEAN NOT NULL DEFAULT false,
    position      INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX restaurant_images_one_cover ON restaurant_images (restaurant_id) WHERE is_cover;

CREATE TABLE reservations (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    restaurant_id BIGINT NOT NULL REFERENCES restaurants ON DELETE CASCADE, -- R-REST-5
    account_id    BIGINT NOT NULL REFERENCES accounts ON DELETE CASCADE,
    pax           INT NOT NULL CHECK (pax >= 1),
    starts_at     TIMESTAMPTZ NOT NULL,
    ends_at       TIMESTAMPTZ NOT NULL,
    status        TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'cancelled')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (ends_at > starts_at)
);
-- Overlap lookups for the capacity check (R-BOOK-5).
CREATE INDEX reservations_active_time_idx ON reservations (restaurant_id, starts_at, ends_at) WHERE status = 'active';
CREATE INDEX reservations_account_id_idx ON reservations (account_id, starts_at);

CREATE TABLE reviews (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    restaurant_id BIGINT NOT NULL REFERENCES restaurants ON DELETE CASCADE,
    account_id    BIGINT NOT NULL REFERENCES accounts ON DELETE CASCADE,
    rating        SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5), -- R-REVIEW-1
    body          TEXT NOT NULL CHECK (body <> ''),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (restaurant_id, account_id) -- R-REVIEW-2
);

-- +goose Down
DROP TABLE reviews;
DROP TABLE reservations;
DROP TABLE restaurant_images;
DROP TABLE restaurant_hours;
DROP TABLE restaurants;
DROP TABLE sessions;
DROP TABLE auth_identities;
DROP TABLE accounts;
