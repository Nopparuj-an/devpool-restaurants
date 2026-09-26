# API reference

Base path `/api`. JSON in and out, UTC ISO-8601 timestamps (R-TIME-1). Errors look like `{"error": {"code", "message"}}`. For codes and statuses see [architecture.md](architecture.md#api-conventions). 🔒 = requires a session cookie.

The tests in `backend/internal/*/…_test.go` are the executable spec for these endpoints.

## Auth
| Method | Path | Body | Returns |
|---|---|---|---|
| POST | `/auth/signup` | `{email, password, display_name}` | `201` Account + session cookie. `409 email_taken` |
| POST | `/auth/login` | `{email, password}` | `200` Account + cookie. `401 invalid_credentials` |
| POST | `/auth/logout` | – | `204`, cookie cleared |
| GET 🔒 | `/me` | – | Account `{id, email, display_name, email_verified, has_password}` |
| PUT 🔒 | `/me/password` | `{current_password, new_password}` (current is ignored if the account has no password yet) | `204` |

## Restaurants
| Method | Path | Body / query | Returns |
|---|---|---|---|
| GET | `/restaurants` | `?sort=top_rated\|most_reviewed\|newest&q=&cuisine=&limit=&offset=` | `{restaurants: Summary[]}` |
| GET | `/restaurants/{id}` | – | Detail. Owners also get `is_owner: true` and `upcoming_reservations` |
| POST 🔒 | `/restaurants` | multipart: `data` = Input JSON, `images` = 1–10 files (the first is the cover) | `201` Detail |
| PUT 🔒 | `/restaurants/{id}` | Input JSON (owner only, `timezone` ignored) | Detail |
| DELETE 🔒 | `/restaurants/{id}` | – | `204` (cascades, R-REST-5) |
| GET 🔒 | `/me/restaurants` | – | `{restaurants: Summary[]}` |
| POST 🔒 | `/restaurants/{id}/images` | multipart `images` | `201 {images}` |
| DELETE 🔒 | `/restaurants/{id}/images/{imageID}` | – | `{images}`. Deleting the last image returns `422 R-REST-1` |
| PUT 🔒 | `/restaurants/{id}/images/{imageID}/cover` | – | `{images}` |

**Input:** `{name, description, cuisine, location, seats, cancel_cutoff_minutes?, max_reservation_minutes?, timezone, hours: [{weekday 0-6 (0=Sun), open "HH:MM", close "HH:MM"}]}`. `timezone` is taken from the browser: `Intl.DateTimeFormat().resolvedOptions().timeZone`.

**Summary:** `{id, name, cuisine, location, seats, rating (1 decimal or null), review_count, cover_url, owner: {id, display_name}}`.
**Detail:** Summary plus `{description, cancel_cutoff_minutes, max_reservation_minutes, timezone, hours, images: [{id, url, is_cover}], is_owner}`.

Image URLs are relative (`/images/restaurants/…`). Next.js rewrites `/images/*` to Garage's public web endpoint (ADR-0005).

## Reservations
| Method | Path | Body / query | Returns |
|---|---|---|---|
| GET | `/restaurants/{id}/availability` | `?from=&to=` (RFC 3339, max 7 days; defaults to the next 24h) | `{seats, limited_threshold, limited, slots: [{start, seats_left, limited}]}`. Returns 15-minute slots inside opening hours only (R-SEATS-1) |
| POST 🔒 | `/restaurants/{id}/reservations` | `{pax, starts_at, ends_at}` | `201` Reservation. `422 R-BOOK-1…4, 8`, `409 R-BOOK-5` |
| GET 🔒 | `/me/reservations` | – | `{reservations}`. Current ones first (soonest first), then past or cancelled (most recent first) |
| GET 🔒 | `/reservations/{id}` | – | Reservation (own only, otherwise `404`) |
| PUT 🔒 | `/reservations/{id}` | `{pax, starts_at, ends_at}` | Reservation. `409 R-EDIT-1` past the cutoff, `409 not_active` if cancelled |
| POST 🔒 | `/reservations/{id}/cancel` | – | Reservation. `409 R-CANCEL-1` past the cutoff |
| GET 🔒 | `/restaurants/{id}/reservations` | `?from=&to=` (defaults to the next 7 days) | Owner only. `{reservations}` with `customer: {id, display_name, email}` (R-PRIV-2) |

**Reservation:** `{id, pax, starts_at, ends_at, status: active|cancelled, state: upcoming|in_progress|completed|cancelled, modifiable_until, can_modify, restaurant: {id, name, cover_url}, customer?}`.

Timestamps in query strings must be URL-encoded (`+07:00` → `%2B07:00`), or just send UTC `Z` times.
