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
