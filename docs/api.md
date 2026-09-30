# API reference

Base path `/api`. JSON in and out, UTC ISO-8601 timestamps (R-TIME-1). Errors look like `{"error": {"code", "message"}}`. For codes and statuses see [architecture.md](architecture.md#api-conventions). 🔒 = requires a session cookie.

The tests in `backend/internal/*/…_test.go` are the executable spec for these endpoints.

## Auth
Every 🔒 route answers `401` with a code that says why: `unauthorized` (no session cookie: not logged in), `session_expired` (a cookie was sent but it no longer works: expired, logged out elsewhere, or the account was banned; the server clears it). Wrong login details are `401 invalid_credentials` on `/auth/login`.

| Method | Path | Body | Returns |
|---|---|---|---|
| POST | `/auth/signup` | `{email, password, display_name}` | `201` Account + session cookie. `409 email_taken` |
| POST | `/auth/login` | `{email, password}` | `200` Account + cookie. `401 invalid_credentials` |
| POST | `/auth/logout` | – | `204`, cookie cleared |
| GET 🔒 | `/me` | – | Account `{id, email, display_name, email_verified, has_password, is_admin, impersonator?: {id, display_name}}` |
| PUT 🔒 | `/me` | `{display_name}` (1 to 80 characters; the email can't change) | Account |
| GET | `/auth/providers` | – | `{password: true, google: bool}`. Hide the Google button when it's false |
| GET | `/auth/google/start?next=/path` | – | `302` to Google. Use as a plain link, not fetch |
| GET | `/auth/google/callback` | (from Google) | `302` to `next` with a session, or to `/login?error=<code>` |
| PUT 🔒 | `/me/password` | `{new_password}` (the session is the proof; no current password) | `204`. `403 R-ADMIN-7` while impersonating |
| POST 🔒 | `/auth/impersonate/stop` | – | Admin's Account + a new admin session cookie. `409 not_impersonating` otherwise |

## Restaurants
| Method | Path | Body / query | Returns |
|---|---|---|---|
| GET | `/restaurants` | `?sort=top_rated\|most_reviewed\|newest&order=desc\|asc&q=&cuisine=&owner_id=&limit=&offset=` (`order=asc` reverses the sort, default `desc`; `q` matches name or cuisine; limit ≤ 100, default 50). With `owner_id`, the owner and admins also get hidden ones | `{restaurants: Summary[], total}` |
| GET | `/restaurants/{id}` | – | Detail. `can_manage` for the owner and admins, who also get `upcoming_reservations` |
| POST 🔒 | `/restaurants` | multipart: `data` = Input JSON, `images` = 1 to 10 files, 10 MB each (the first is the cover) | `201` Detail |
| PUT 🔒 | `/restaurants/{id}` | Input JSON (owner or admin, `timezone` ignored) | Detail |
| DELETE 🔒 | `/restaurants/{id}` | – | `204` (cascades, R-REST-5) |
| GET 🔒 | `/me/restaurants` | `?limit=&offset=` | `{restaurants: Summary[], total}` |
| POST 🔒 | `/restaurants/{id}/images` | multipart `images` | `201 {images}` |
| DELETE 🔒 | `/restaurants/{id}/images/{imageID}` | – | `{images}`. Deleting the last image returns `422 R-REST-1` |
| PUT 🔒 | `/restaurants/{id}/images/{imageID}/cover` | – | `{images}` |

**Input:** `{name, description, cuisine, location, seats, cancel_cutoff_minutes?, max_reservation_minutes?, timezone, hours: [{weekday 0-6 (0=Sun), open "HH:MM", close "HH:MM"}]}`. `timezone` is taken from the browser: `Intl.DateTimeFormat().resolvedOptions().timeZone`.

**Summary:** `{id, name, cuisine, location, seats, rating (1 decimal or null), review_count, cover_url, owner: {id, display_name}}`. JSON drops trailing zeros (`5`, not `5.0`), so display it with `rating.toFixed(1)`.
**Detail:** Summary plus `{description, cancel_cutoff_minutes, max_reservation_minutes, timezone, hours, images: [{id, url, is_cover}], is_owner, can_manage}`. Every change to a restaurant or its photos is allowed for the owner and admins (R-REST-2, R-ADMIN-6).

Photos larger than 1600 px or 1 MB are resized to 1600 px and stored as JPEG (`platform/imageproc`). Images over 40 megapixels are rejected. The web app already resizes photos in the browser before upload (`lib/image.ts`).

Image URLs are relative (`/images/restaurants/…`). Next.js rewrites `/images/*` to Garage's public web endpoint (ADR-0005).

## Reservations
| Method | Path | Body / query | Returns |
|---|---|---|---|
| GET | `/restaurants/{id}/availability` | `?from=&to=` (RFC 3339, max 7 days; defaults to the next 24h) | `{seats, limited_threshold, limited, slots: [{start, seats_left, limited}]}`. Returns 15-minute slots inside opening hours only (R-SEATS-1) |
| POST 🔒 | `/restaurants/{id}/reservations` | `{pax, starts_at, ends_at}` | `201` Reservation. `422 R-BOOK-1…4, 8`, `409 R-BOOK-5` |
| GET 🔒 | `/me/reservations` | `?limit=&offset=` (limit ≤ 100, default 50) | `{reservations, total}`. Current ones first (soonest first), then past or cancelled (most recent first) |
| GET 🔒 | `/reservations/{id}` | – | Reservation (own only, otherwise `404`) |
| PUT 🔒 | `/reservations/{id}` | `{pax, starts_at, ends_at}` | Reservation. `409 R-EDIT-1` past the cutoff, `409 not_active` if cancelled |
| POST 🔒 | `/reservations/{id}/cancel` | – | Reservation. `409 R-CANCEL-1` past the cutoff |
| GET 🔒 | `/restaurants/{id}/reservations` | `?from=&to=` (defaults to the next 7 days, at most 31) | Owner or admin. `{reservations}` with `customer: {id, display_name, email}` (R-PRIV-2) |

**Reservation:** `{id, pax, starts_at, ends_at, status: active|cancelled, state: upcoming|in_progress|completed|cancelled, modifiable_until, can_modify, restaurant: {id, name, cover_url}, customer?}`.

Timestamps in query strings must be URL-encoded (`+07:00` → `%2B07:00`), or just send UTC `Z` times.

## Admin
Admin session required (R-ADMIN-1): 401 when logged out, 403 `admin_only` for other accounts. List queries take `?q=&status=active|banned&limit=&offset=` (limit ≤ 500, default 50).

| Method | Path | Body | Returns |
|---|---|---|---|
| GET | `/admin/users` | – | `{users: AdminUser[], total}`. `q` matches email or name |
| GET | `/admin/users/{id}` | – | AdminUser + `owned_restaurants: AdminRestaurant[]` |
| POST | `/admin/users/{id}/ban` | `{reason?}` | AdminUser. `409 R-ADMIN-5` for yourself or an admin |
| POST | `/admin/users/{id}/unban` | – | AdminUser |
| PUT | `/admin/users/{id}` | `{display_name}` | AdminUser (R-ADMIN-6) |
| POST | `/admin/users/delete` | `{ids: [1–500 ids]}` | `{deleted: n}` (R-ADMIN-8). Hard delete; their restaurants, bookings and reviews cascade, photos are removed. Missing ids are skipped. All or nothing: `409 R-ADMIN-8` if any id is yourself or an admin. `422` for 0 or more than 500 ids |
| POST | `/admin/users/{id}/impersonate` | – | `204` + a session cookie as the user (R-ADMIN-7). `409 R-ADMIN-7` for yourself, an admin or a banned user |
| GET | `/admin/restaurants` | – | `{restaurants: AdminRestaurant[], total}`. `q` matches name, cuisine or owner email. Includes hidden ones |
| POST | `/admin/restaurants/{id}/ban` | `{reason?}` | `204` |
| POST | `/admin/restaurants/{id}/unban` | – | `204` |
| POST | `/admin/restaurants/delete` | `{ids: [1–500 ids]}` | `{deleted: n}` (R-ADMIN-6, R-REST-5): bookings, reviews and photos go too. Missing ids are skipped |

**AdminUser:** `{id, email, display_name, is_admin, banned_at, ban_reason, created_at, restaurants, reviews, reservations}` (the last three are counts).
**AdminRestaurant:** `{id, name, cuisine, location, rating, review_count, banned_at, ban_reason, created_at, owner: {id, display_name, email, banned}}`.

Hidden restaurants (R-ADMIN-4) return 404 to everyone except their owner and admins. For those two, Summary and Detail carry `banned`, `owner_banned` and `ban_reason`. `Account` has `is_admin`. Logging in to a banned account returns `403 account_banned`.

A Bruno collection with every endpoint is in `backend/bruno/` (see the README).

## Profiles
| Method | Path | Query | Returns |
|---|---|---|---|
| GET | `/users/{id}` | – | `{id, display_name, created_at, restaurant_count, review_count, banned?}`. No email. `404` for banned accounts unless you're an admin (R-PROFILE-2) |
| GET | `/users/{id}/reviews` | `?limit=&offset=` (limit ≤ 100, default 20) | `{reviews: [{id, rating, body, verified, created_at, updated_at, restaurant: {id, name, cover_url}, hidden?}], total}`, newest first |

A profile's restaurants: `GET /restaurants?owner_id={id}`.

## Reviews
| Method | Path | Body / query | Returns |
|---|---|---|---|
| GET | `/restaurants/{id}/reviews` | `?limit=&offset=&rating=&sort=` (limit ≤ 100, default 20; `rating` 1–5 filters to that star rating; `sort` newest (default) or oldest, by last update) | `{reviews, total, rating_counts}` (R-REVIEW-8). `total` counts the matches; `rating_counts` is `{"1": n, …, "5": n}` whatever the filter. `422` for another rating or sort |
| GET 🔒 | `/restaurants/{id}/reviews/me` | – | My Review, or `404` |
| PUT 🔒 | `/restaurants/{id}/reviews/me` | `{rating 1-5, body}` | `201` when created, `200` when updated (one per account, R-REVIEW-2). `403 R-REVIEW-3` on your own restaurant |
| DELETE 🔒 | `/restaurants/{id}/reviews/me` | – | `204` |

**Review:** `{id, rating, body, verified, author: {id, display_name, email?}, created_at, updated_at}`. `email` is only present for the restaurant's owner (R-PRIV-2).
